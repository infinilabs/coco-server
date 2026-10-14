# 统一 Pipeline 架构 — 最终设计文档

| | |
|---|---|
| **版本** | v2.0 **Final**（2026-10-10 定稿） |
| **关系** | 本文为**权威定稿**；详细规格见 [SPEC](./SPEC-unified-pipeline.md)、任务清单见 [WORKPLAN](./WORKPLAN-unified-pipeline.md)、初版评审稿见 [DESIGN v1](./DESIGN-unified-pipeline.md)（已被本文取代，保留供对照） |
| **涉及仓库** | `infini.sh/framework` · `coco` · `logpilot` · `gateway` · `ui-common` |
| **规模** | framework +24/-15 文件（含 output.go/tap.go/metrics_persist.go/lanes 端点）；coco Go +8/-45；ui-common 新增 1 包；logpilot -7/+7；gateway **零改动** |

---

## 0. 执行摘要

四个产品共享同一个 pipeline 引擎（framework `core/pipeline`），但数据约定、调试能力、观测能力各自分裂，且 Coco 链存在逐处理器全量序列化的性能税。本方案以**统一记录模型**为核心，分八个设计决策（D1-D8）、九个阶段（P1-P9）完成统一：

**一句话架构**：队列用字节换持久（磁盘、at-least-once），通道用指针换实时（内存、at-most-once），链内只有一种载体 `*event.Event`（Fields 唯一权威、原地修改、零拷贝）；处理器面向记录作用域编程，放错位置由契约编译器在编译期拦截；dry-run/指标/日志是同一次处理器调用的三个投影；多 Gateway 调度复用配置中心通道 + 既有租约锁。

**核心收益**：全产品处理器自由混编（37 个 enterprise 处理器对 Coco 即插即用）；数据搬运成本降一个数量级（~10 次全量序列化 → 2~3 次且向量子树 0 次）；调试与观测三产品同语义同 UI；Gateway 零代码改动继承全部能力。

---

## 1. 背景与问题（现状事实，均已在代码核实）

| # | 问题 | 证据 |
|---|---|---|
| 1 | **数据约定分裂**：Coco 处理器吃 `[]queue.Message`（Document JSON），其余吃 `*event.Event`；混排时记录类处理器静默空转 | `coco` 13 个处理器文件 vs `framework/core/pipeline/record.go` |
| 2 | **调试分裂**：两套 dry-run 互不相干（连函数名都撞车）；无副作用截获（dry-run 可能真写生产队列） | `coco/modules/pipeline/dryrun.go` vs `logpilot/.../api_dryrun.go` |
| 3 | **观测缺失**：framework 仅 3 个扁平计数器；Logpilot 自建 Stage 指标绑死 LogStream；日志无 trace 关联 | `processor.go:277-281` |
| 4 | **性能税**：链内逐处理器全量 JSON 往返（含 MB 级向量）；`indexing_merge` 为取 `_id` 全量解析 | `indexing_merge.go:275` |
| 5 | **动态生效缺口**：ORM 管道更新不触发热载（仅文件通道可热重载） | `pipeline.go` update handler 仅落库 |

## 2. 最终架构总览

### 2.1 三层模型

```
            跨进程/需持久                 进程内实时                  链内
        ┌────────────────┐        ┌────────────────┐      ┌────────────────┐
概念    │  Queue 队列     │        │  Lane 通道      │      │  Context 作用域 │
载体    │  bytes         │ codec  │ *event.Event   │ 指针 │ *event.Event   │
        │  Message.Data  │◄──────►│ 指针(记录即消息) │◄────►│ (原地修改)      │
语义    │ at-least-once  │        │ at-most-once   │      │ 零拷贝          │
        │ offset/重放    │        │ 有界/背压       │      │                │
处理器  │ consumer       │        │ lane_consumer  │      │ for_each 内的   │
        │ output(queue)  │        │ output(lane)   │      │ 全体处理器       │
判据    │ 崩溃不可重算/  │        │ 确定性/可重算/  │      │ 同一条链内       │
        │ LLM 昂贵       │        │ 低延迟          │      │                │
        └────────────────┘        └────────────────┘      └────────────────┘
领域层: core.Document 等退化为 codec 投影(document.*) + documentview 视图
HTTP 层: Filter(fasthttp.RequestCtx) 契约独立存在,不纳入统一
```

### 2.2 设计原则（五条）

1. **注册动态、绑定编译期**——注册随时发生，管道编译时快照实现引用，热重载后重编译切换；
2. **Fields 是唯一权威**——领域对象是投影视图（`documentview.Materialize/Project`），不存在第二份真相；
3. **字节换持久、指针换实时**——Queue 与 Lane 是正交契约，接口签名（`[]byte` vs `*event.Event`）锁死语义；
4. **一次埋点三个投影**——dry-run/指标/日志共享同一个 `StepEvent` 执行事件模型；
5. **每步 opt-in、存量零破坏**——契约、记录模式、lane、观测全部增量启用。

## 3. 设计决策（最终形态）

### D1 记录模型与 Codec

- 统一到 `*event.Event`（不统一到 Document——它是领域类型之一；不统一到 Message——它是传输信封）；
- **`coco_document` codec**：整树投影 `Fields["document"]`（mapstr 点路径可寻址，grok/mutate/`when` 直接操作；Document 加字段零映射维护），元数据进 Labels；`coco_attachment` 同模式；
- **`_codec` 提示由宿主打**（for_each 解码循环/类型化入口盖 `metadata.labels._codec`），出口编码三级解析：记录提示 > lane.namespace 主 codec > 默认 otel——编解码往返对称，输出即"某 codec 的标准 Encode 产物"，自动继承其全部既有消费者；
- **保真规则**：命名空间 codec 只序列化自身子树，子树外根级字段为链内瞬态数据（dry-run 编码边界步骤可见）；
- 处理器两层契约：`Process`（逐条）+ 可选 `ProcessBatch`（批段）。

### D2 契约与作用域（编译期正确性）

- **`RecordContract{Scope, Namespace, RequiredFields, Granularity}`**，处理器可选实现 `ContractProvider`；不声明 = unconstrained（存量零影响）；
- **`NewPipelineWithScope`**：作用域跟踪编译，错配即编译失败（带步序号/处理器名/双端描述）；运行期 `CheckContract` 兜底交 `on_failure`；`strict_contract` 开关（默认关，P8 分批收紧）；
- **granularity 三档**：`record`（默认）/ `batch_capable`（两路皆可，**必须自带单条兜底**）/ `batch_only`（仅批段有意义，只能作宿主直接成员，嵌套位置编译错误）；
- **段式执行（stage-wise）**：顺序即书写顺序——编译按书写序切交替段（连续逐条处理器=逐条段、批量处理器=批段），逐段整批推进；批处理器**看到前面逐条处理器的修改**；早段 drop 排除出后续段、批段失败全批打标、clone 流入后续段。**废弃现行两阶段行为**（`BatchProcessor` 全生态零实现，零风险窗口修正）；
- catalog 透出 `scope`/`record_contract`/`granularity`，画布据此过滤与标注。

### D3 Lane 通道（进程内实时层）

**命名定稿（封版）**：用户可见词汇只有 **lane（通道）** 一个——段落 `lane:`、处理器 `lane_consumer`、统一出口 `output`、Go 包 `core/lane`。否决：channel/stream/bus/buffer/relay/events_buffer（后者命名了 queue 与 lane **共有**的属性（都是 events）而非**区分**的维度（内存/持久、指针/字节）——名字该承诺契约，不该复述内容物）。对仗：入口 `consumer` ↔ `lane_consumer`（宿主），出口统一为 `output`（步骤）。

- **`lane_consumer` = 链宿主**（订阅 lane 攒批 `batch{max,flush_ms}` → 段式执行子链，契约同 for_each）；`output{lane:…}` = 子链内普通步骤（Emit 即返回本链继续）；
- **背压三策略，`block` 默认**；`spill` 仅限"快慢路径跑等价链"的快车道形态（快路径含必需步骤时 spill 会绕过它=正确性 bug，且掩盖背压）；`spill` 无回灌、路径终身制、溢出记录反而更持久（落盘即持久）、顺序分叉（强顺序用 block）；
- **AckSet 提交门**（跨 lane 的 commit 正确性）：`output{lane:…, ack:true}` 回执挂入 ctx AckSet；consumer 等回执**全部落定**才提交 offset（成功→`output_queue`/失败→`dead_letter`/显式 drop 均算落定），未决或超时（`ack_timeout_ms`）不提交→磁盘队列重放。**offset 六细则**：回执携带 (slice,offset)；v1 整批 all-or-nothing（前缀提交留作优化）；超时=子链失败=复用现有重试语义（零新状态机）；on_failure 三档均算落定；clone 不带回执；代价=上游提交等下游落定（吞吐耦合，与 durable-first 的取舍点）。**lane 零状态——at-most-once 被 queue offset 的 at-least-once 兜住，保证完全寄生在 offset 上**；
- **三种正确姿势**：durable-first（默认，子链内先 `output{queue:…}`，lane 仅旁路）/ ack 回执（端到端 at-least-once）/ 内联（需严格耦合时不解耦）；
- **动态 lane**：运行期 `Create/Delete/GetOrCreate` + `owner` 级联清理（复用 configs-center 孤儿模式）；**一条动态管道一条 lane 绝不共享**（背压天然隔离，慢流只堵自己）；内存三护栏（惰性分配/动态小容量 256~512/全局预算）；动态管道默认仍走队列，lane 为 opt-in；
- `consumer` 桥接糖：`codec`+`output_lane` 把磁盘→内存三明治压成一块。

### D4 零拷贝与性能

| 级 | 内容 | 效果 |
|---|---|---|
| L0 | 记录模型（链内指针传递原地修改） | 全量序列化 ~10 → 3 次 |
| L1 | Event 池化、编码缓冲池、jsoniter、for_each 记录级并行 | 分配降 50-80%，CPU 型链随核扩展 |
| L2 | **冷热分离**：`document_chunk`（向量）以 `json.RawMessage` 原字节寄生，embedding 处理器是唯一访问者 | 最重字段 0 次序列化 |
| L3 | 类型化入口（`DecodeFromDocument`）+ Lane | 进程内 0~1 次 |
| L4 | `indexing_merge` `_id` 部分解析；二进制 codec 变体（按流选） | 边界编码 2-4× |

验收基准：1MB 文档 K=4 链数据搬运 CPU 降 ≥80%；1KB 事件并行吞吐 ≥3×。诚实边界：LLM 链单条延迟由 API 决定，第一杠杆是批量化（`fetch_max_messages`）。

### D5 可观测与在线调试

- **`StepEvent`/`ExecutionObserver`** 埋点（一处，observers 空=零开销）：dry-run 全量投影、指标聚合投影、飞行记录采样投影（1/N + 错误全采）；
- **统一端点，产品 wrapper 全部删除**：`POST /pipeline/dry-run`（唯一定义，副作用截获修安全隐患，`Report` 含逐步快照/字段 diff/截获写入）、`POST /pipeline/ai-generate`（`ChainGenerator` 产品注册，未注册 501）；删除 Coco studio 两个端点 + Logpilot dry-run/parsing-test/ai-generate/debug 五个端点；
- 标准指标：每处理器六指标 + `StageStats`（通用化 Logpilot 模型，carrier 含 `queue:`/`lane:`）；
- 在线操控：`_inject`/`_pause`/`_resume`/`flight`/`stages` 端点 + `tap` 采样分流 + trace_id 贯穿（`pipelinelog` 助手）；lane 运行/堆积可查（`/pipeline/lanes`，健康判读：depth 涨+in>out=消费慢、pending_acks 高+ack_timeouts 增=下游卡死）；
- **观测四象限闭环（持久化平面）**：指标——`pipeline_metrics.persistence{interval,retention}` 周期快照 StageStats 落 `pipeline_stage_stats` 索引 + `_search` 端点（D-7 已决：framework 内置）；飞行记录——错误事件全量落 `pipeline_flight_errors`、全量追踪按需开窗（`trace_persist: 30m` 自动关）、`flight?window=` 同端点跨界查询。

### D6 共享 UI：`@infinilabs/pipeline-ui`

新建 `ui-common/packages/Pipeline`（coco/web 已有 `@infinilabs/*` 消费先例）：画布（React Flow）/处理器面板（契约+granularity 过滤、徽章、拖放即时红叉/黄牌）/SchemaForm/DryRunDrawer/AiGeneratePanel/FlightTimeline/StageStatsPanel/InjectDialog + TS client。**编辑与调试交互只有一份实现**，Coco 与 Logpilot 消费同一包；产品差异收窄为 props 数据绑定（链存哪/样本从哪来/外壳页面）。

### D7 分布式调度与治理（多 Gateway）

- **控制面/数据面分离**：控制面维护 `PipelineDeployment`（pipeline+placement+rollout）期望态，调和循环幂等；数据面 Gateway 只认本地配置（**执行引擎不动**）；
- **placement 三策略**：`singleton`（锁竞争+TTL 接管）/`all`（每台各跑）/`sharded:N`（kafka 共享队列+消费者组）——前两者零新增依赖；
- **协调四原语**：租约单例（已有 `locker.Hold`，锁 TTL=接管上界）/ 版本协商（`_sync` 回报 generation/state）/ 广播操控（generation 防脑裂）/ 排水隔离（drain 提交 offset 后停、cordon 不接新任务）；
- **与配置中心的配合**（framework `modules/configs`，非 Logpilot 私有）：**零替代，意图层叠加在传输层**——deployment 翻译为 `ManagedConfig` 行（三种定向：单实例/全部/分组），`_sync` 差分下发，行删除=实例删本地文件；rollout=版本化行更新；瞬态操控走实例代理不进配置；仅需三处小扩展（`_sync` 回报 runtime 字段/selector 静态解析/代理通用化）；
- 监测：StageStats 带 instance_id、`_sync` 即心跳、**锁接管事件=故障切换告警**、cluster/overview 矩阵；
- 不做：强一致调度器、磁盘队列跨机复制、v1 自动重均衡。

### D8 热生效补齐（现状缺口）

- ORM 管道更新触发任务重建（复用 fsnotify 同款路径，drain 当前批后换编译产物）；
- `process_documents` 版本感知编译缓存（消除逐消息 GetV2+NewPipeline）。

## 4. API 与 DSL 速览

**新增端点**（全部增量）：`GET /pipeline/codecs`、`GET /pipeline/lanes[/name]`（通道清单与实时指标：depth/in·out rate/drops/spills/emit_blocks/pending_acks/ack_timeouts/owner/消费者任务）、`POST /pipeline/dry-run`、`POST /pipeline/ai-generate`、`POST /pipeline/task/:id/_inject|_pause|_resume`、`GET /pipeline/task/:id/flight|stages`（flight 带 `?window=` 查持久化索引）、`POST /pipeline/metrics/_search`（持久化指标搜索）、Fleet API（`/pipeline/deployments/*`、`/pipeline/instances/:id/_drain|_cordon|proxy`、`/pipeline/cluster/overview|stages`）。

**DSL 文法**（一屏）：

```
pipeline := entry+                       # 有序数组,段式执行=书写顺序
entry    := NAME ":" config              # 单键处理器
          | "if" ":" cond, "then:" pipeline, "else:" pipeline?
config   := { 处理器字段..., "when" ":" cond? }
宿主     := consumer{codec?,output_lane?,processor:} | for_each{codec,parallel?,processor:}
          | lane_consumer{lane,batch,output_queue?,output_codec?,processor:}
          | if/then/else                 # 嵌套即递归完整 pipeline
cond     := equals{路径:值} | contains | range | exists | and/or/not   # record 域对记录字段,batch 域对上下文参数
```

典型链（Coco 文档富化）：

```yaml
- consumer:
    queue_selector: { keys: [indexing_documents] }
    processor:                       # 持久化动作必须在 consumer 子链内(offset 语义)
      - for_each:
          codec: coco_document
          parallel: 8
          processor:
            - redact: {...}          # enterprise 逐条处理器,原地改 Fields
            - extract_entities: {}   # coco 批段处理器,一次 LLM 调用
            - document_embedding: {}
      - output: { queue: documents_processed }
```

## 5. 迁移路线（P1-P9）

| 阶段 | 内容 | 破坏性 |
|---|---|---|
| P1 地基 | codec v2 + `coco_document` + documentview + Observer 埋点 | 零 |
| P2 契约 | RecordContract/granularity + 段式 executor + catalog | 零（opt-in） |
| P3 双模迁移 | 15 处理器 + 26 连接器/webhook（41 注册点）+ process_documents `mode:records` | 零 |
| P4 性能 | L1 池化/jsoniter/并行 → L2 冷热分离 → L4 部分解析 | jsoniter 需门禁 |
| P5 Lane | core/lane + 处理器对 + AckSet + 类型化入口 + 动态平面 | 零（新平面） |
| P6 观测 | dryrun 包 + 统一端点（删产品端点）+ 指标 + flight/inject/pause | dry-run 响应结构变化 |
| P7 UI | `@infinilabs/pipeline-ui` 共享包 + 两产品集成 | 前端同步切换 |
| P8 收尾 | 扫描/迁移工具 + 删旧路径 + strict_contract 默认开 | ⚠️ 按产品节奏 |
| P9 分布式 | 协调原语 + deployment/调和 + Fleet API + 集群 UI | 零 |

关键路径 P1→P2→P3→P8；P4/P5/P6/P7 并行；P9 依赖 P6。Gateway 全程零代码改动。

**存量配置零影响声明（P1-P7）**：已有 pipeline 配置（gateway 生产流配置、coco.yml、ORM 存储链）在此期间**一个字节都不用改**——依据：strict_contract 默认 false；BatchProcessor 零实现故段式执行无行为可变；queue_output/lane_output 保留注册别名；lane/ack/parallel 等均为新增键（旧配置惰性）；Coco 双模默认 messages。两个非配置破坏（产品 dry-run/AI 端点删除、响应结构统一）由两产品前端同版本切换消化，入 changelog。**P8 为唯一触碰存量时点**（D-1 已决一次切）：audit 扫描 → 机械迁移 + 语义等价复验 → 单次翻转；翻转后未迁移的 messages 式管道与"静默空转"组合将编译失败（audit 提前暴露）。jsoniter 热路径替换的极端 case 字节差异由行为对比门禁拦截（#70）。

## 6. 待决项（详见 WORKPLAN；**9 项已关闭，开放 4 项**）

**本轮已决 7 项**：D-1 strict_contract **一次切换**（audit→迁移→单次翻转，P8 简化）；D-2 codec 落位**独立轻量 module `infini.sh/coco-codecs`**（依赖面=framework+领域类型，为远程执行预留）；D-6 jsoniter **仅热路径局部替换**（codec/documentview/dryrun，全局保持 stdlib）；D-8 lane **并入 PipeModule**（三产品免接线）；D-9 共享包**源码引用**（file:/workspace，不发 npm）；D-11 控制面 **framework 全包**（产品只做 UI/实例对接）；D-12 sharded **v1 暂不支持**（生产无 kafka 后端，设计保留任务冻结）。另有 D-7（指标持久化内置，101a）、D-13（统一 output 已采纳）此前已关闭。

**仍开放**：D-3 `document.*` 路径命名冻结规则；D-4 `for_each.parallel` 默认值；D-5 LLM mock 录制存储与脱敏；D-10 性能基准环境与数字确认。

## 7. 决策日志（评审过程中推翻/修正的关键决策）

| # | 原方案 | 最终方案 | 触发原因 |
|---|---|---|---|
| 1 | dry-run 产品端点改薄壳 | **framework 统一端点，产品端点删除** | "不需要额外 wrapper"——统一接口与 UI，交互一致 |
| 2 | 画布移植到 coco/web | **共享组件包 `@infinilabs/pipeline-ui`** | 单一实现强于复制，ui-common 是现成落点 |
| 3 | 内存队列（memory queue） | **Lane 通道**（后定名） | "队列"承诺 offset/重放/持久，名字必须广告真契约 |
| 4 | relay + relay_in/relay_out 双词 | **lane 单词**（lane_consumer/lane_output） | 一个概念一个词；in/out 视角不明 |
| 5 | 背压默认 spill（洪峰落盘） | **block 默认，spill 收窄为等价快车道** | spill 会绕过快路径必需步骤（脱敏）=正确性 bug；掩盖背压致磁盘无界 |
| 6 | 批量两阶段（batch 先行）+ 顺序规范化补丁 | **段式执行，顺序即书写顺序** | 两阶段下批处理器看不到前面逐条处理器的修改；BatchProcessor 零实现=零风险窗口 |
| 7 | relay_in 平铺 + queue_output 平铺 | **宿主形态 + 内置 output_queue 持久桥** | 平铺=逐条退化无批量；queue_output 读 messages 在 record 域是死配置 |
| 8 | 示例中 queue_output 在 consumer 外 | **必须在 consumer 子链内** | offset 仅在子链成功后推进——在外=提交先于落盘的丢数窗口 |
| 9 | lane 无提交语义（Emit 即终点） | **AckSet 提交门**（可选 ack） | "成功失败怎么反馈、offset 怎么正确 commit"——保证借自上游 offset，lane 零状态 |
| 10 | lane 静态配置 | **动态平面 + per-pipeline 隔离** | 动态管道（按流）场景；共享 lane=语义错误 |
| 11 | RecordContract{scope,namespace,fields} | **+ granularity 三档** | "单条/批量配置清楚么、界面上清楚么"——粒度声明式+编译期拦截 |
| 12 | 分布式新造调度器 | **叠加配置中心（零替代）+ 既有租约锁** | configs-center 是 framework 级模块，协议全现成 |
| 13 | `queue_output`/`lane_output` 双出口处理器 + `to: type:name` 打包引用 | **统一 `output` 步骤 + one-of 目标字段**（`queue: <名>`/`lane: <名>`，Unpack 即校验）；旧名保留为注册别名但**不入文档** | 双名费解；打包字符串拼写错误不可枚举、工具黑盒；别名入文档会诱导用户继续旧写法 |
| 14 | `tee`（Unix 行话） | **`tap`**（网络 TAP：并联监听、复制不截流） | tee 非命令行用户不可读；tap 是观测领域标准词；`observe`/`inspect` 否决——超额承诺（处理器不做观测/检查，只旁路复制）、用途导向覆盖不了双跑/审计场景、observe 与 Observer 子系统撞词、inspect 应留给未来的记录校验处理器（检查并产生结果才是它的契约）；`sample` 已被 enterprise 占用且语义相反（丢弃式采样 vs 旁路复制） |

## 8. 附录

### 8.1 术语表

| 术语 | 定义 |
|---|---|
| 队列 Queue | 持久化落盘消息存储；bytes；at-least-once；offset/重放 |
| 通道 Lane | 进程内有界记录通道；`*event.Event` 指针（记录即消息）；at-most-once |
| 记录 Record | 链内唯一载体 `*event.Event`，`Fields` 唯一权威 |
| 记录作用域 | 宿主（for_each/lane_consumer）内的执行上下文，`CurrentRecord` 可用 |
| 命名空间 | codec 产出的记录族标识（document/attachment/""），契约匹配依据 |
| 契约 | 处理器声明的作用域/命名空间/必需字段/粒度依赖，编译期校验 |
| 段式执行 | 按书写序切逐条段/批段交替推进，顺序=书写顺序 |
| 冷热分离 | 重子树（向量）以 RawMessage 原字节寄生，唯一访问者才解码 |
| AckSet 提交门 | 回执集合；consumer 等全部落定才提交 offset |
| 飞行记录 | 采样式执行事件环形缓冲，错误全采 |
| 执行事件 StepEvent | 一次处理器调用的标准描述，dry-run/指标/日志的共同投影源 |
| 部署 PipelineDeployment | 控制面期望态（pipeline+placement+rollout），翻译为配置行下发 |

### 8.2 配套文档

- **[SPEC-unified-pipeline.md](./SPEC-unified-pipeline.md)**——API/配置/数据结构/DSL 完整规范（§1-§8 + Lane 速查/回执提交门/对照速查三个附录）
- **[WORKPLAN-unified-pipeline.md](./WORKPLAN-unified-pipeline.md)**——130+ 编号任务、文件级影响清单、测试门禁、D-1~D-12 待决项
- 证据索引（代码行号）见 SPEC 附录与 DESIGN v1 §附录 B
