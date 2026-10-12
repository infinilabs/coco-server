# 文档级增量编译 op 队列 — 立项设计(2026-10-12)

> 状态:**立项待排期**(工程量:预计 2-3 个批次;依赖:无新外部组件)。
> 背景:WeKnora 自动 Wiki 对照(2026-10 会话)收敛出的最后一块未落机制。本文件是单独立项的设计稿,落地记录回填 [WORKPLAN-knowledge-hub.md](WORKPLAN-knowledge-hub.md)。

## 一、问题:coco 现状的编译是 KB 级粗粒度

现状(已落地部分,不重复造):

- **触发**:W11 联络器检测文档内容实变 → `MarkKBForCompile(kb)` 标脏 → 5 分钟 trailing-edge debounce → 逐脏 KB 跑 `runCompilePipeline`。
- **成本挡板**:MAP 阶段有批指纹缓存(内容+模型身份),未变文档零 LLM;但 REDUCE/PLAN/提议生成仍是**全量重跑**——千文档 KB 改一篇,整库 reconcile 一遍,提议集全量重生(靠锚定 ID 幂等折叠,不重复入库,但计算与写放大是 O(全库))。

差距一句话:**触发已经能感知"哪个文档变了",执行却仍然"哪个库脏了整库重来"**。

## 二、WeKnora 机制拆解(v0.8.2 源码已核实)

| 机制 | 实现 | 值得借的 |
|---|---|---|
| 持久 op 队列 | `task_pending_ops` 表:`TaskPendingOp{TaskType, Scope, ScopeID, Op(ingest/retract), DedupKey, Payload, FailCount, ClaimedAt}` | (TaskType, Scope, ScopeID) 三元组即队列身份;**队列不保证去重,消费侧按 DedupKey 折叠**——职责划分干净 |
| 消费协议 | PeekBatch → 服务侧按 DedupKey 折叠 → 处理 → DeleteByIDs;失败 IncrFailCount,超限入死信 | 消费即删,行级 fail 计数 |
| 死信 | `task_dead_letter` 表(可重试/取消/清空+审计) | 失败 op 不静默蒸发 |
| 启动恢复 | `recoverPendingWikiTasks`:启动扫描未排空队列,**按 lane 重建触发器** | 重启不丢编译任务,不靠人重点 |
| 互斥 | 外部互斥(`wiki:active:<kbID>` SetNX),`ClaimedAt` 列预留未来行锁并行 | v1 外部互斥够用 |
| 目录规划 | `planBatchTaxonomy` **一趟批量规划**(大批分块),锚点=既有目录池(先裁剪到相关子集),只对无目录的页面应用 | 已定档后置,见下文"不立项项" |

## 三、coco 对应设计

### 3.1 op 模型与存储

```go
// core/wiki_op.go
type WikiCompileOp struct {
    orm.ORMObjectBase
    KbID      string `json:"kb_id" elastic_mapping:"kb_id:{type:keyword}"`
    Op        string `json:"op"`        // ingest | retract
    DocID     string `json:"doc_id" elastic_mapping:"doc_id:{type:keyword}"`
    Reason    string `json:"reason,omitempty"`   // content-changed | document-deleted | manual
    FailCount int    `json:"fail_count,omitempty"`
    EnqueuedAt *time.Time `json:"enqueued_at,omitempty"`
}
// DedupKey = doc_id(同文档同 KB 的 op 折叠为一条,覆盖 reason 取最新)
```

**存储选 badger KV,不建 ES 索引**:op 是瞬时指令不是审计事实,丢了可全量重扫兜底;单二进制纪律下不引 Redis/Asynq(第五篇已明记不抄)。KV bucket `/wiki/compileops/<kbID>`,值为 op 数组(JSON)。审计诉求由治理队列(提议)与 `pipeline_runs` 覆盖,不需要 op 层审计。

### 3.2 生产者(三处,均为既有钩子的接线)

1. W11 联络器内容实变(现调 `MarkKBForCompile` 处改投 op);
2. 文档删除(现 stale 提议路径顺带投 retract);
3. 手动 `POST /wiki/kb/:id/_compile` 直投 ingest 全量 op(兼容现语义)。

### 3.3 消费 Lane(替换 KB 脏标记)

```
tick(30s) → 拉有 op 的 kbID 集 → 逐 KB(单飞互斥,kv 租约 /wiki/compilelock/<kbID>):
  ops := peekBatch(kbID)                    // 全量取出,DedupKey=doc_id 折叠
  docs := loadDocs(ops.docIDs)              // 只加载 op 覆盖的文档
  MAP(docs)                                  // 批指纹缓存照旧,未变块零 LLM
  pages := contributionIndex(docIDs)         // 见 3.4
  REDUCE+PLAN(pages ∪ 其邻接页)              // 只 reconcile 受影响页
  提议照旧入治理队列(锚定 ID 幂等不变)
  deleteBatch(ops)                           // 成功即删
  失败: FailCount++,≥5 → 死信 bucket + notifyOwner
```

**人工闸门不破**:增量化的只是"算出提议"这一段,提议-only 契约(编译永不直写页面)原样保留。

### 3.4 贡献倒排(P2 的关键前置)

现状态:文章找来源靠 `Sources[].doc_id` **内存扫描**(上限 50 篇)。增量 REDUCE 需要反向:文档 → 受影响页。建持久倒排 `coco_wiki_contrib-v2`:

```go
type WikiContribution struct {
    orm.ORMObjectBase                       // ID = md5("contrib|kb|doc|page")
    KbID  string `json:"kb_id" elastic_mapping:"kb_id:{type:keyword}"`
    DocID string `json:"doc_id" elastic_mapping:"doc_id:{type:keyword}"`
    PageID string `json:"page_id" elastic_mapping:"page_id:{type:keyword}"` // article 或 entity
    Updated *time.Time `json:"updated,omitempty"`
}
```

写入点=每次文章 Sources 变更处(crud PostUpdate/编译 apply/实体 retract),与 Sources 同事务语义(best-effort + 启动对账一次)。**这是全设计里唯一的新索引**,也是 P1(纯替换触发)可以后置它的原因。

### 3.5 启动恢复

启动时扫 `/wiki/compileops/*`:非空 lane → 重建消费触发器(与 WeKnora `recoverPendingWikiTasks` 同构)。debounce 计时器内存态,重启即失——op 持久化让"丢计时器"无害。

### 3.6 死信

`/wiki/opdead/<kbID>` bucket,逐条 {op, error, failed_at, fail_count}。v1 无独立 UI:治理队列 KB 卡片显示"编译死信 N 条"横幅 + `DELETE /wiki/kb/:id/_opdead` 清空。独立死信面板等真用上再排(避免 W7 Mock 式提前建面)。

## 四、分期与验收

| 期 | 内容 | 验收口径 |
|---|---|---|
| P1 | op 队列 + 消费 lane 替换 KB 脏标记(3.1-3.3,3.5-3.6;MAP 仍全库但有指纹缓存) | 千文档 KB 改一篇:MAP LLM 调用=仅该文档;重启不丢 op;同文档 10 op 折叠为 1;双触发不并发编译(租约) |
| P2 | 贡献倒排 + 增量 REDUCE/PLAN(3.4) | 提议集只覆盖受影响页∪其邻接;全库提议数不回归(锚定 ID 折叠兜底);倒排对账启动跑通 |
| P3 | 死信面 + 编译观测(每 pass 处理 op 数/跳过数/LLM 调用数进 run 记录) | 死信可清;观测数字与日志一致 |

## 五、明确不做(防 cargo-cult)

- **不引 Redis/Asynq/NATS**(第五篇定档:Easysearch+badger 是唯一状态真相);
- **LLM 自动 patch 直写页面**(治理闸门红线,提议-only);
- **taxonomy LLM 批量规划**(WeKnora planBatchTaxonomy)——**已定档后置,不随本项目**:coco 目录是规则驱动(部门→维度→实体类型,确定性收敛,toc_layout+dimensions 已覆盖企业布局),LLM 规划的价值在"无锚点自由生长目录"场景,coco 恰恰处处有锚点。若未来开放自由目录,参考其两点:一趟批量规划(勿逐页并行发明)+ 锚点池先裁剪到相关子集再进 prompt。
