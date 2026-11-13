# 统一 Pipeline 架构设计与开发方案

| | |
|---|---|
| **文档状态** | Draft v1.0 — 待评审 |
| **日期** | 2026-10-10 |
| **涉及仓库** | `infini.sh/framework`、`infini.sh/coco`、`infini.sh/logpilot`、`infini.sh/gateway` |
| **评审方式** | 逐节评审，重点见 §12 评审要点清单 |

---

## 1. 背景与问题

Coco、Logpilot、Gateway 三个产品共享同一个 pipeline 引擎（`infini.sh/framework/core/pipeline`，三个产品的 go.mod 均为 `replace infini.sh/framework => ../framework`），处理器注册表也是同一个。**编译注册层面早已互通**：Coco 的 `coco.yml` 在混用 framework 的 `consumer` 与自有 `process_documents`；enterprise 的 37 个处理器（grok/mutate/geoip/...）已经编译进三个产品的二进制；Logpilot 编译的流管道正在 Gateway 上运行（`gateway/config/stream-*-pipelines.yml`）。

但在四个方面存在系统性分裂：

1. **数据约定分裂**：Coco 处理器消费 `[]queue.Message`（payload 为 `core.Document` JSON，context key `"messages"`），Logpilot/enterprise 处理器消费 `*event.Event`（`pipeline.CurrentRecord`，原地修改 `Fields`）。混排时记录类处理器**静默空转**（无报错、无效果）。
2. **调试能力分裂**：Coco 与 Logpilot 各养了一套 dry-run（连函数名都撞车，各有一个 `buildNamedProcessor`），模型不同、互不共享；Gateway 没有 dry-run。
3. **运行时可观测缺失**：framework 仅有 3 个扁平计数器；Logpilot 自建了队列阶段的 Stage 指标（绑死 LogStream 模型）；日志无记录级关联；无 trace。
4. **性能税**：Coco 富化链每个处理器对全量 Document（含 MB 级 embedding 向量 JSON）做一次完整编解码，K 个处理器的链发生 ~K+3 次全量序列化；`indexing_merge` 为取 `_id` 全量解析每条文档。

## 2. 目标与非目标

### 2.1 目标

- **G1 数据统一**：链内在途载体统一为 `*event.Event`，任何产品的处理器可在同一条链混排且语义正确。
- **G2 契约保障**：处理器与记录结构的依赖在编译期校验、运行期兜底、编排期可视化限制。
- **G3 性能**：全量序列化次数从 ~K+3 降至 2~3，最重字段（embedding）降至 0 次；关键路径分配池化。
- **G4 调试统一**：dry-run / 注入 / 暂停 / 飞行记录成为 framework 能力，三产品同语义。
- **G5 观测统一**：一次处理器调用定义唯一执行事件，指标/日志/dry-run 是它的三个投影。
- **G6 接口与 UI 统一**：dry-run/AI 生成收敛为 framework 统一端点（产品 wrapper 端点全部删除）；画布、表单、调试抽屉以共享组件包 `@infinilabs/pipeline-ui` 单一实现，两产品交互一致。
- **G7 分布式调度治理**：管道可调度到多 Gateway 实例，具备 placement 策略、版本化 rollout、集群级操控/监测/排水的统一能力（复用 locker 单例与 managed-configs 通道）。

### 2.2 非目标（明确不做）

- ❌ 统一 HTTP Filter 契约（`fasthttp.RequestCtx` 是网关同步转发的正当记录形态，与数据管道正交）。
- ❌ 更改磁盘队列文件格式（存量三种 payload 共存：Document JSON / otel 信封 / fasthttp 字节，`queue.Message.Data` 保持 opaque）。
- ❌ COW MapStr / 带失效追踪的 `Event.Private` 缓存（正确性陷阱，收益被冷热分离覆盖）。
- ❌ Lane 的持久化伪装（不加 offset/重放；要持久走 `spill` 明着降级）。
- ❌ Lane 广播语义（v1 只做工作队列：一条消息一个订阅者）。
- ❌ 处理器单步执行（暂停 + 注入 + dry-run 组合已覆盖）。
- ❌ 一次性全量二进制化队列 payload（改为按流的 codec 选项）。

## 3. 现状分析（事实基线）

> 以下均已在代码中核实，标注文件位置，作为评审的事实起点。

### 3.1 引擎共享现状

| 组件 | 位置 | 说明 |
|---|---|---|
| 处理器接口 | `framework/core/pipeline/processor.go` | `Processor{Name() string; Process(*Context) error}`；可选 `BatchProcessor{ProcessBatch(ctx, []*event.Event)}` |
| 注册表 | `framework/core/pipeline/register.go` | 进程级全局注册；`RegisterProcessorPlugin*`（bare name）、`RegisterDomainProcessor*`（`category:name` 别名）、`WithConfigMetadata` 变体（从 `config:` tag 提取 schema）；重名 panic |
| 管道配置 | `framework/core/pipeline/config.go` | `PipelineConfigV2`（ORM 对象，`Processors []map[string]interface{}`） |
| 运行宿主 | `framework/modules/pipeline/module.go` | `PipeModule`：YAML 加载 + fsnotify 热重载 + ORM CRUD（`/pipelines/*`）+ catalog（`GET /pipeline/processors`） |
| 记录约定 | `framework/core/pipeline/record.go` | `RecordContextKey="record"`、`CurrentRecord(ctx)`、`MarkDropped`、failure tags |
| 批量拆分 | `framework/modules/pipeline/for_each.go` | 消息→记录的边界；batch pass（`ProcessBatch`）先于 record pass；`on_failure: ignore\|tag\|fail`；编译期校验 codec 名 |
| Codec 注册表 | `framework/modules/pipeline/for_each_codec.go` | `RecordCodec{Name/Decode/Encode}`；mutex map（可动态调用）；默认 `otel`；重名 panic |

### 3.2 数据约定分裂（核心问题）

| | Coco 处理器（15 个） | Logpilot / enterprise 处理器（37+ 个） |
|---|---|---|
| 上下文键 | `"messages"` → `[]queue.Message` | `"record"` → `*event.Event`（`CurrentRecord`） |
| payload | `core.Document` / `core.Attachment` JSON | `Fields util.MapStr` 原地修改 |
| 包依赖 | `infini.sh/coco/core`、`modules/llm` 等（仅 Coco 二进制内可运行） | 纯 framework |
| 放错链的行为 | 类型断言失败 → warn + 空转 | `CurrentRecord` 取不到 → no-op（如 mutate 只打 "no current record, skip"） |

### 3.3 Gateway 定位

- 10 个数据处理器（`gateway/pipeline/`）：全部走 framework `Processor` 契约，但都是记录无关的 worker 循环（payload 为编码的 fasthttp 请求字节）。
- ~55 个 HTTP 过滤器（`gateway/proxy/`）：framework `Filter(fasthttp.RequestCtx)` 契约，独立机制。
- `main.go:49-56` blank-import framework plugins + enterprise；`main.go:89` 注册 `PipeModule`。
- **本方案对 Gateway 自有代码零改动**，它通过 framework 升级自动继承。

### 3.4 性能问题清单

| 问题 | 位置 | 量级 |
|---|---|---|
| 链内逐处理器全量 JSON 往返 | coco 13 个处理器文件 | K+3 次全量；1MB 文档 ≈ 70ms CPU + ~30MB 垃圾/条 |
| embedding 向量无读者仍全程编解码 | `core/document.go` `Chunks[].Embedding`（1024 维 float32 × N） | 占文档体积大部分 |
| `_id` 提取全量解析 | `framework/plugins/elastic/indexing_merge/indexing_merge.go:275` | 每文档一次全量 Unmarshal |
| 无 Event 池化 | framework（仅 Context 有 `AcquireContext/ReleaseContext`） | 高吞吐路径分配压力 |
| `for_each` 记录处理串行 | `for_each.go:197` 单 goroutine | CPU 型链不随核数扩展 |
| JSON 库为 stdlib | `framework/core/util/bytes.go:288` | ~100-200MB/s，jsoniter 2-4× |

### 3.5 调试/观测现状

| 能力 | Coco | Logpilot | framework |
|---|---|---|---|
| dry-run | `modules/pipeline/dryrun.go`（stepSnapshot，messages 快照） | `plugins/logpilot/api_dryrun.go`（dryRunProbe，Fields 快照 + diff + 采集仿真） | ❌ |
| 指标 | ❌ | `PipelineStageStats`/`PipelineBPEvent`（队列深度/lag/速率，绑 LogStream，存 ES） | 3 个计数器 |
| 注入/操控 | ❌ | `_inject`、`debug/chain`、`debug/queue`、实例代理 | Context 有 pause/resume 但未暴露 API；`processHistory` 内部数组未利用 |
| UI | `web/src/pages/pipeline/`（表单式 list/new/edit） | `processing-canvas.tsx`（React Flow 画布）+ `schema-form.tsx` + `debug-panel.tsx` | catalog 无作用域信息 |

### 3.6 已识别的安全隐患

**dry-run 可能真写生产队列**：两套 dry-run 均未截获 `output_queue` 类副作用，样本试跑时若配置指向真实队列会实际投递。本方案 D5 修复。

## 4. 总体架构

### 4.1 分层模型（最终形态）

```
              跨进程/需持久                进程内实时                  链内
           ┌────────────────┐       ┌────────────────┐      ┌────────────────┐
 概念      │  Queue 队列     │       │  Lane 通道     │      │  Context 作用域 │
           │ (core/queue)   │       │ (core/lane)   │      │ (core/pipeline)│
 载体      │  bytes         │ codec │ *event.Event   │ 指针 │ *event.Event   │
           │  Message.Data  │◄─────►│ 指针(记录即消息) │◄────►│ (记录作用域)    │
 语义      │ at-least-once  │       │ at-most-once   │      │ 原地修改        │
           │ offset/重放    │ spill │ 有界/背压       │      │                │
 处理器    │ consumer       │       │ lane_consumer       │      │ for_each 内的   │
           │ queue_output   │       │ lane_output      │      │ 全体处理器       │
 判据      │ 崩溃不可重算/  │       │ 确定性/廉价/    │      │ 同一条链内       │
           │ LLM 昂贵阶段    │       │ 可重算阶段      │      │                │
           └────────────────┘       └────────────────┘      └────────────────┘

 领域层: core.Document 等不再链内直传,退化为 codec 投影(Fields["document.*"]) + documentview 视图
 HTTP 层: Filter(fasthttp.RequestCtx) 契约独立存在,不纳入统一(见非目标)
```

### 4.2 一条文档的统一数据流（Coco 富化链示例）

```
连接器 Fetch ──► Document ──codec──► queue(indexing_documents, Document JSON, 格式不变)
                                              │
                          consumer ──► for_each(codec: coco_document)
                                              │ 解码一次,进入记录作用域(namespace=document)
                 ┌────────────────────────────┤
                 │ grok/mutate/redact   改 Fields,零额外编解码(通用处理器)
                 │ extract_entities     Materialize→改→Project(领域处理器)
                 │ embedding            ProcessBatch,一次 API 调用批量;只重编码 chunks 子树
                 └────────────────────────────┤
                                              │ 编码一次
                          queue_output ──► queue(documents_processed) ──► bulk_indexing
```

### 4.3 设计原则

1. **注册动态、绑定编译期**：codec/处理器注册随时可发生；管道编译时快照解析到的实现引用，热重载后重编译才切换。
2. **Fields 是唯一权威**：领域对象是记录的投影视图，不存在第二份真相；通用处理器改 Fields，领域处理器 Materialize/Project。
3. **字节换持久、指针换实时**：Queue 与 Lane 是正交契约，接口签名（`[]byte` vs `*event.Event`）锁死语义，`spill` 是唯一单向桥。
4. **一次埋点三个投影**：dry-run / 指标 / 日志共享同一个执行事件模型。
5. **每步 opt-in、存量零破坏**：契约、记录模式、lane、观测全部增量启用，未声明者行为不变。

## 5. 详细设计

### D1 记录模型与 Codec 体系

**D1.1 统一到 `*event.Event`**（不统一到 `Document`——它是领域类型之一；不统一到 `queue.Message`——它是传输信封）。

**D1.2 `coco_document` codec**（新增，`coco/modules/pipeline/document_codec.go`，约 100 行）：

```go
type DocumentCodec struct{}
func (DocumentCodec) Name() string      { return "coco_document" }
func (DocumentCodec) Namespace() string { return "document" }

func (DocumentCodec) Decode(data []byte) (*event.Event, error) {
    doc := core.Document{}
    if err := util.FromJSONBytes(data, &doc); err != nil { return nil, err }
    rec := &event.Event{Timestamp: doc.GetUpdatedTime()}
    rec.Fields = util.MapStr{"document": util.ToMapStr(&doc)} // 整树投影,点路径可寻址
    rec.SetID(doc.ID)
    rec.Metadata.Labels = util.MapStr{
        "datasource": doc.Source.ID, "connector_id": doc.Source.ConnectorID,
        "lang": doc.Lang, "type": doc.Type,
    }
    return rec, nil
}
func (DocumentCodec) Encode(rec *event.Event) ([]byte, error) {
    v, err := rec.Fields.GetValue("document")
    if err != nil { return nil, err }
    return util.ToJSONBytes(v), nil
}
```

设计决策：**整树投影而非热字段白名单**——`util.MapStr` 点路径（`document.content`）天然可寻址，grok/mutate/`when` 条件直接可用；Document 加新字段零映射维护成本（白名单必然漂移）。Attachment 同模式注册 `coco_attachment`（namespace `attachment`）。

**D1.3 `documentview` 助手**（新增，`coco/core/documentview/view.go`）：

```go
func Materialize(rec *event.Event) (*core.Document, error) // Fields["document"] → 强类型(一次解码)
func Project(doc *core.Document, rec *event.Event)          // 强类型 → Fields["document"](一次编码)
```

不做 `Event.Private` 缓存（通用处理器绕过助手直写 Fields 无法检测失效，正确性优先）。

**D1.4 处理器契约两层化**：领域处理器实现 `Process`（记录级，`CurrentRecord`）与可选 `ProcessBatch`（LLM 批量，段式执行中作为批段一次调用整批）。`"messages"` 类型断言从处理器中消失，仅存于传输处理器。

**D1.5 `process_documents` 改造为链宿主**：外层门禁（dedup/幂等/数据源解析）保留；内层改为"解码一次 → 记录作用域 → 跑子链 → 编码一次"，等价于内联 `for_each(codec: coco_document)`。子链成为 enterprise 处理器与 LLM 处理器自由混编处。

### D2 契约与作用域校验

**D2.1 Codec 注册 v2**（framework，`for_each_codec.go` 扩展）：

```go
type RecordCodec interface {
    Name() string
    Namespace() string  // "document"|"attachment"|""(通用,如 otel);D2.2 契约按此匹配
    Decode(data []byte) (*event.Event, error)
    Encode(rec *event.Event) ([]byte, error)
}
func RegisterRecordCodec(c RecordCodec) error          // 返回 error(替代 panic,支持运行期注册)
func RegisterRecordCodecOrReplace(c RecordCodec) error // 版本升级
func ListRecordCodecs() []CodecInfo                    // 发现
```

绑定语义：**注册动态、绑定编译期**（§4.3-1）。新增 `GET /pipeline/codecs` 发现端点。
兼容：未实现 `Namespace()` 的旧 codec 经适配器默认返回 `""`。

**D2.2 记录契约**（framework，`core/pipeline` 新增，风格对齐 `WithConfigMetadata`）：

```go
type Scope int // ScopeBatch(链根) | ScopeRecord(for_each 内)

type Granularity int // GranularityRecord(默认,仅逐条) | GranularityBatchCapable(两路皆可,批量优先) | GranularityBatchOnly(仅批量通道)

type RecordContract struct {
    Scope          Scope
    Namespace      string   // "document"/"attachment"/""=任意
    RequiredFields []string // 可选硬断言,如 "document.content"
    Granularity    Granularity // 处理粒度声明:batch_only 只能作宿主直接成员(嵌套位置编译错误),batch_capable 嵌套位置警告(退化逐条)
}

type ContractProvider interface { RecordContract() RecordContract }
```

**批量执行语义 = 段式执行（stage-wise），顺序即书写顺序**：编译按书写序切交替段（连续逐条处理器=逐条段，批量处理器=批段），逐段整批推进；批处理器看到前面逐条处理器的修改；早段 drop 排除出后续段、批段失败给批内活动记录打标、clone 流入后续段。现行 `for_each` 两阶段行为（批量成员先跑、看不到前面逐条处理器的修改）废弃——`BatchProcessor` 当前全生态零实现，零风险修正。`granularity` 放置校验保留（batch_only 嵌套位置编译错误），catalog 透出 `granularity` 供画布渲染徽章（段式下顺序即书写顺序，无需相位分区）。

Coco 处理器声明示例：

```go
func (p *Processor) RecordContract() pipeline.RecordContract {
    return pipeline.RecordContract{Scope: pipeline.ScopeRecord, Namespace: "document"}
}
```

**契约匹配命名空间而非 codec 名**（`coco_document_v2` 只要 namespace 仍是 `document`，处理器零改动）。

**D2.3 作用域感知编译**（framework）：

```go
func NewPipelineWithScope(cfg []*config.Config, s ScopeInfo) (*Processors, error)
// NewPipeline 保持原签名,默认 batch 作用域
// for_each 内部改用 NewPipelineWithScope(Record, codec.Namespace())
```

编译期错误示例：`processor [extract_entities] requires record namespace [document], for_each codec [otel] provides []`；嵌套 `for_each` 递归进入新作用域（与 `RecordContextKey` save/restore 语义一致）。

**D2.4 运行期兜底**：`pipeline.CheckContract(ctx, contract) error`，声明契约的处理器在 `Process` 开头调用，错误交 `for_each` 的 `on_failure: tag|fail` 处理——静默空转 → 显式 tag 失败 + 可观测。

**D2.5 编排期透出**：
- `GetProcessorCatalog()` 每项增加 `scope` 与 `record_contract`（未声明者标 `unconstrained`）；
- dry-run（D5）用同一编译器，返回结构化逐步错误（步序号 + 处理器名 + 原因），画布定位标红；
- 画布节点面板按所处作用域过滤（for_each 容器内 + 所选 codec 的 namespace → 仅显示兼容处理器，**置灰而非隐藏**，tooltip 说明原因）；
- 软校验：grok 等通用处理器的 `field: document.content` 配置引用，dry-run 时用样本数据跑一遍报告缺失字段路径。

**D2.6 行为变化声明**：启用校验后，存量管道中"非法但今日静默空转"的组合将从空转变为编译失败。提供 `PipelineConfigV2.strict_contract: false` 逃生门，存量管道按旧语义跑完再逐步收紧（需 changelog 明示）。

### D3 Lane 通道

**命名定稿**：Queue（队列，持久）/ Lane（通道，实时）——**用户可见词汇只有 lane 一个**（配置段落 `lane:`、处理器 `lane_consumer`/`lane_output`、引用 `lane:名`、Go 包 `core/lane`）。弃用"内存队列"与早期草稿的 "relay/中继" 叫法（一个概念一个词）。否决的候选：`channel`（与 Go chan 混淆）、`stream`（与 LogStream 冲突）、`bus`（暗示广播）、`buffer`（存储隐喻破坏"队列/通道"的管道隐喻对，且与 consumer 内部 buffer 等泛型用法冲突；"有界缓冲"仅用于**描述 lane 的内部机制**，不作配置名词）。

**为什么不能做成 queue 的内存后端**：`queue.Push(k *QueueConfig, v []byte)` 签名强制编解码，零拷贝价值在接口处即丢失。必须是独立抽象。

```go
// framework/core/lane/lane.go —— 新包,与 core/queue 平级
type Lane interface {
    Name() string
    Emit(rec *event.Event) error      // 按背压策略投递
    TryEmit(rec *event.Event) bool
    Subscribe(h Handler) (cancel func())
    Depth() int                        // 指标用(D5 StageStats 的 lane 维度)
    Close() error                      // 进程关闭 drain-or-drop,显式 at-most-once
}

type LaneConfig struct {
    Name      string `config:"name"`
    Namespace string `config:"namespace"` // 承载的记录族,lane_consumer 据此建作用域(同 D2 契约)
    Capacity  int    `config:"capacity"`  // 有界,默认 4096
    Overflow  string `config:"overflow"`  // block | drop | spill:<queue名>
}
```

- **记录即消息**：无信封、无 offset；`*event.Event` 指针直接传递，零序列化。
- **背压三策略，`block` 为默认**：`block`（生产者减速，背压如实传导——生产者是可安全减速的任务协程时永远选它）；`drop`（best-effort 遥测）；`spill:<queue>`（**仅限快慢路径跑等价处理**的快车道形态，溢写仅表示"此条改走慢的同款链路"——快路径含必需步骤时 spill 会绕过该步骤，属正确性 bug，且它掩盖背压信号可致磁盘队列无界增长）。生产者不可阻塞且数据需持久的诚实答案是 queue-first，不是 spill。
- **跨 lane 提交正确性（AckSet 提交门）**：`lane_output` 不是提交点、consumer 的 offset 只覆盖子链内 durable 写。三种姿势：durable-first（子链内先 `queue_output`，lane 仅旁路，默认）；**ack 回执**（`lane_output{ack:true}` 回执挂入 ctx AckSet，consumer 等 AckSet 落定才提交——成功 / 失败入 `dead_letter` 队列 / 显式 drop 均算落定，未决或超时不提交、由磁盘队列重放；跨 lane 达成 at-least-once，重复由幂等 sink 消化——lane 不存数据，保证借自上游 offset）；内联（需严格耦合时不解耦）。配套**边界无损不变式**：进入 `lane_consumer` 的每条记录必然落定于 `output_queue`/`dead_letter`/显式 drop 之一（详见 SPEC 附录"回执与提交门"）。
- **处理器对**：`lane_consumer`（订阅 lane，直接进入记录作用域，免 codec）/ `lane_output`（发记录进 lane），与 `consumer`/`queue_output` 对仗。
- **类型化入口**（配合 D4-L3）：codec 增加 `DecodeFromDocument(*core.Document) *event.Event` 变体，连接器进程内路径零解码进入链。

### D4 零拷贝与性能（L0→L4 阶梯，逐级独立落地）

| 级 | 内容 | 改动位置 | 效果 |
|---|---|---|---|
| **L0** | 记录模型（D1）：链内指针传递、原地修改，入口/出口各一次编解码 | coco | 全量序列化 ~10 → 3 次 |
| **L1** | `AcquireEvent/ReleaseEvent` 池化；编码缓冲池；jsoniter 替换 stdlib（已在 go.mod 间接依赖，benchmark 把关）；`for_each` 记录级并行（worker 池 + 每 goroutine context 克隆，输出乱序无害） | framework | 分配率降 50-80%；CPU 型链随核数扩展 |
| **L2** | **冷热分离**：codec 解码时冷子树（`document_chunk` 含 embedding）以 `json.RawMessage` 原字节寄生，`json.Marshal` 原样拼回不解析；embedding 处理器是唯一访问者（只解/只重编 chunks 子树）；门禁字段（ID/ContentHash/Source.ID）全在热区 | coco codec + embedding 处理器 | 最重字段全程 **0 次**序列化；有效解码体积 1MB → 几十 KB |
| **L3** | 类型化入口 + Lane（D3）：进程内符合条件的流 0~1 次序列化 | framework + coco 连接器 | 入口零解码 |
| **L4** | `indexing_merge` `_id` 提取改部分解析（gjson/jsonparser 式零分配路径查询）；队列 payload 二进制/base64-float codec 变体（按流选，不全局切） | framework | 边界编码 2-4×；merge 路径大幅改善 |

**量化预期**（1MB 文档、K=4、含 2 次 LLM 调用）：

| 配置 | 全量序列化次数 | 数据搬运成本/条 |
|---|---|---|
| 现状 | ~10 | ~70ms CPU + ~30MB 垃圾 |
| L0 | 3 | ~21ms |
| L0+L2 | 3 次小 + 1 次 chunks 子树 | ~5ms |
| L0+L1+L2 | 同上，分配走池 | ~3ms，GC 压力趋零 |
| +L3（适用流） | 0~1 | 接近纯处理耗时 |

诚实边界：LLM 链的单条延迟由 API 决定（秒级），本优化改善的是吞吐、CPU 成本、大文档尾延迟；LLM 链的最大杠杆是批量化（`ProcessBatch`）而非 JSON。Gateway 日志管道（无 LLM 高吞吐）的主菜是 L1 + 记录并行。

**不做**：COW map、跨队列边界池化 Event（生命周期止于 Encode）、LLM 链过度优化。

### D5 可观测与在线调试

**D5.1 执行事件模型**（framework，`core/pipeline/observer.go`）——统一的地基：

```go
type StepEvent struct {
    TraceID   string        // 记录级追踪 ID,入口生成,贯穿全链(过 queue/lane 均携带)
    Pipeline  string
    Step      int
    Processor string
    Scope     ScopeInfo     // D2 同一模型
    Before, After Snapshot  // 按需
    Duration  time.Duration
    Outcome   Outcome       // ok | error | dropped | skipped-contract
    Err       string
}

type ExecutionObserver interface { OnStep(e *StepEvent) }
// 埋点位置:Processors.Process 循环 + for_each 逐条循环;observers 为空时 fast path 零成本
```

三投影：dry-run = 全量捕获 Observer；运行指标 = 聚合 Observer；飞行记录 = 采样 Observer（环形缓冲，容量如 1024 步/管道，默认 1/N 采样，**错误记录永远全采**）。

**D5.2 framework `pipeline/dryrun` 包**（合并两套实现）：

```go
type Sample struct { Raw []byte; Codec string }  // 日志行 / Document JSON / 任意 codec 载荷

func Run(chain []map[string]interface{}, samples []Sample, opts Options) (*Report, error)

type Options struct {
    ScopeCheck  bool             // 契约编译校验(D2)
    SideEffects SideEffectPolicy // capture(默认,截获不投递) | passthrough(显式 opt-in)
    LLMPolicy   LLMPolicy        // real(默认,现状) | mock(录制回放,CI/脱敏)
}

type Report struct {
    Compiled    []StepDesc        // 编译 + 契约结论
    Steps       [][]StepResult    // 每样本 × 每步:{前/后快照, 字段级 Diff, Duration, Outcome}
    SideEffects []CapturedWrite   // 被截获的 queue/lane 写入(回答"真跑会写什么")
}
```

**修复 §3.6 隐患**：dry-run 上下文中 `queue_output`/`lane_output` 自动 capture。交付为 **framework 统一端点 `POST /pipeline/dry-run`，不做产品 wrapper**：Coco `/pipeline/studio/test`、Logpilot `/logpilot/streams/:id/dry-run` 及 `parsing/test` 等产品端点**直接删除**（清单见 §7），前端统一调用新端点；样本一律由客户端内联传入，产品的样本取数 API 属数据接口、不在 pipeline API 范畴。

**D5.3 标准指标集**（吸收 Logpilot Stage 模型，去 LogStream 绑定）：

```go
// 每处理器:in / out / error / drop / inflight / latency(直方图)
// 每阶段(StageStats,通用化自 PipelineStageStats):
type StageStats struct {
    Stage   string // 任意阶段名(采集/staging/富化/indexing/bulk/产品自定义)
    Carrier string // "queue:<name>" | "lane:<name>" | "inline"
    Depth, LagAmount int64   // queue: 深度/lag;lane: lane.Depth()
    InTotal, OutTotal, InRate, OutRate float64
    Errors, Drops int64
}
```

Logpilot 看板改喂标准模型（ES 存储与展示不动）；Coco/Gateway 从无到有。

**D5.4 在线操控**：

| 能力 | 设计 | 现状对照 |
|---|---|---|
| 注入 | `POST /pipeline/tasks/:id/_inject`（body=Sample+目标队列/lane） | Logpilot `_inject` 通用化 |
| 暂停 | `POST /pipeline/task/:id/_pause/_resume`（处理完当前批次后挂起，非硬停） | Context 状态机已有，只缺 API |
| 分流采样 | `tap` 处理器：按采样率把记录副本发调试队列/lane，主流程无感——**在真实流量上调试** | 三产品均无 |
| 轨迹查询 | `GET /pipeline/tasks/:id/flight?trace_id=xxx` / `?errors=1` | Logpilot `debug/chain` 通用化 |
| 日志关联 | `pipelinelog.Warnf(ctx, ...)` 自动携带 trace_id（增量约定，不迁移存量日志） | 无关联 |

### D6 编排 UI（共享组件包 `@infinilabs/pipeline-ui`）

- **单一实现**：新建 `ui-common/packages/Pipeline`（`@infinilabs/pipeline-ui`，pnpm monorepo，coco/web 已有 `@infinilabs/entity-ui` 消费先例），承载：画布（React Flow：源/处理器/汇节点、if/for_each 容器、when 徽章、JSON 双视图）、处理器面板（catalog 驱动 + 按 D2.5 契约过滤，置灰+原因）、schema 配置表单、dry-run 调试抽屉（调用统一端点、渲染统一 `Report`）、飞行记录时间线、阶段指标卡，以及配套 API client。**Coco 与 Logpilot 前端消费同一包，交互天然一致，编辑器本体零分叉。**
- **画布布局存储**：`PipelineConfigV2` 增加扩展字段存 react-flow graph JSON（仅布局，链的权威仍是 `Processors` 数组）。
- **前置必做**：Coco ~40 个处理器/连接器从 `RegisterProcessorPlugin` 换为 `RegisterProcessorPluginWithConfigMetadata(name, New, Config{})`——否则 catalog 无 `properties`，schema 表单渲染不出（enterprise 处理器即此模式，已验证可行）。
- **产品差异收窄为数据绑定**（props 注入）：链存哪（Coco `PipelineConfigV2` / Logpilot `LogStream.Spec`）、样本从哪来、外壳页面路由；Logpilot 特有面板（placement、链路拓扑、压力监控）留在其外壳层。

### D7 分布式调度与治理（多 Gateway）

**现状基线**（已核实）：framework `locker.Hold`（KV 租约锁，TTL 过期自动接管，`core/locker/locker.go:64`）+ PipeModule singleton 已做跨实例单例（`modules/pipeline/module.go:575`，锁 TTL = `max_running_in_ms`）；Logpilot configs-center（版本化托管配置、实例 `_sync` 拉取、孤儿清理）+ 实例体系/反向代理 + 带 `InstanceID` 的阶段指标。**执行引擎不动，分布式能力加在配置分发与状态回集两端。**

**D7.1 控制面/数据面分离**：控制面（产品承载）维护期望态 `PipelineDeployment`（pipeline + placement + rollout），经 managed-configs 通道下发；数据面（Gateway 实例）只认本地配置文件，跑 PipeModule，`_sync` 回报心跳与运行状态；控制面调和循环（desired vs reported，幂等重试），调度器自身用 locker 保证单例。

**D7.2 Placement DSL 与两种数据拓扑**：

```yaml
placement:
  selector: { labels: { role: log-gateway } }
  strategy: singleton            # singleton | all | sharded:N
  rollout: { canary: 1, batch: 2, max_unavailable: 1 }
```

| 策略 | 语义 | 数据拓扑 | 依赖 |
|---|---|---|---|
| `singleton` | 目标集一台跑，锁竞争，TTL 自动接管 | 数据随管道（本地磁盘队列） | 现有 locker |
| `all` | 每台各跑一份（本机 intake 场景） | 数据随管道 | 无 |
| `sharded:N` | N 台分片消费 | 管道随数据（kafka 共享队列 + 消费者组） | kafka queue 插件 |

**D7.3 协调四原语**：租约单例（已有，锁 TTL = 故障接管上界，文档化）；版本协商（实例 `_sync` 回报 `{pipeline_id: {generation, state, last_error}}`）；广播操控（`_pause/_resume/_stop` 带 generation 防脑裂）；排水/隔离（`_drain` 处理完当前批 + 提交 offset 后停，`_cordon` 不接新 assignment）。

**D7.4 集群监测**：`StageStats`/`StepEvent` 带 `instance_id` 上报（通用化 Logpilot 已有字段）；`_sync` 即心跳（超 N 周期未 sync 判离线）；控制面聚合端点 `cluster/overview`（管道×实例矩阵）、`cluster/stages`（跨实例指标）；锁接管事件 = 故障切换告警；trace_id 跨实例串联（D5 已随记录过队列）。

**D7.5 Fleet API**：见 SPEC §8（deployments CRUD/rollout/集群广播、instances drain/cordon/proxy、cluster 聚合视图）。

**不做**：不重造强一致调度器（期望态 DB + 幂等调和）；不做磁盘队列跨机复制（跨机共享仅 kafka 一条路，写入选型判据）；v1 不做自动重均衡（sharded 靠消费者组、singleton 靠锁竞争自然迁移）。

**D7.6 与现有配置中心（framework `modules/configs`）的配合**：配置中心是 **framework 级模块**（非 Logpilot 私有）——server 跑在管理端（`/instance/_register` 注册审批 + token 配对；`/configs/_sync` 心跳 + hash 差分下发 + secrets），client 跑在实例；`ManagedConfig` 行已支持三种定向（单实例 `InstanceID` / 全部 `AllInstancesID` / 分组 `Groups`）；行删除 → 实例删本地文件（协议保证）；Logpilot 的 configs-center API 只是它上面的治理视图（清单/孤儿清理）。**D7 不替代配置中心，而是意图层叠加在传输层之上**：`PipelineDeployment`（意图）→ reconciler 翻译 → `ManagedConfig` 行（定向+版本）→ `_sync` 差分 → 实例本地文件 → PipeModule fsnotify 热加载 → 运行状态经 `_sync` 回报 → reconciler 对账。需要的扩展仅三处：`_sync` 请求体增加运行状态回报（`{pipeline_id:{generation,state}}`，向后兼容）；placement.selector 的标签解析（v1 静态解析为 InstanceID/Groups 定向）；瞬态操控（pause/resume/drain）走实例代理而非配置行。

## 6. 关键接口定义汇总

| 接口/类型 | 仓库·包 | 状态 |
|---|---|---|
| `RecordCodec{Name/Namespace/Decode/Encode}` | framework `modules/pipeline` | 扩展（加 Namespace） |
| `RegisterRecordCodec(…we error)` / `ListRecordCodecs` | 同上 | 新增 |
| `RecordContract` / `ContractProvider` / `CheckContract` | framework `core/pipeline` | 新增 |
| `NewPipelineWithScope(cfg, ScopeInfo)` | framework `core/pipeline` | 新增（NewPipeline 保留） |
| `Lane` / `LaneConfig` / `lane_consumer` / `lane_output` | framework `core/lane` | 新增 |
| `StepEvent` / `ExecutionObserver` | framework `core/pipeline` | 新增 |
| `dryrun.Run/Sample/Report/Options` | framework `modules/pipeline/dryrun` | 新增 |
| `StageStats` 标准指标集 | framework `modules/pipeline` | 新增 |
| `tap` 处理器 | framework | 新增 |
| `DocumentCodec`（`coco_document`）/ `AttachmentCodec` | coco `modules/pipeline` | 新增 |
| `documentview.Materialize/Project` | coco `core/documentview` | 新增 |
| `DecodeFromDocument` 类型化入口 | codec 可选扩展 | 新增 |
| Coco 处理器 `WithConfigMetadata` + `RecordContract` 声明 | coco `plugins/processors` ×15、connectors | 改造 |
| `process_documents` 链宿主化 | coco | 改造 |
| `PipelineConfigV2.strict_contract` / 画布布局字段 | framework | 扩展 |

## 7. API 变更清单

**新增（全部增量，不动存量）**：

```
GET    /pipeline/codecs                       codec 发现
POST   /pipeline/dry-run                      统一 dry-run（唯一定义，无产品 wrapper）
POST   /pipeline/ai-generate                  AI 生成链（产品注册 ChainGenerator 实现）
POST   /pipeline/task/:id/_inject             样本注入（队列或 lane）
POST   /pipeline/task/:id/_pause | _resume    暂停/恢复（批处理完挂起）
GET    /pipeline/task/:id/flight              飞行记录查询（trace_id / errors 过滤）
GET    /pipeline/task/:id/stages              标准阶段指标
```

**行为变化（需 changelog）**：

- `GET /pipeline/processors` 响应每项增加 `scope`、`record_contract` 字段（加字段，向后兼容）；
- **删除产品 dry-run/AI 端点，收敛到 framework 统一端点**：Coco `/pipeline/studio/test`、`/pipeline/studio/ai-generate`；Logpilot `/logpilot/streams/:id/dry-run`、`/logpilot/parsing/test`、`/logpilot/parsing/ai-generate`、`debug/_inject`、`debug/chain` → 分别由 `POST /pipeline/dry-run`、`POST /pipeline/ai-generate`、`POST /pipeline/task/:id/_inject`、`GET /pipeline/task/:id/flight` 取代（**破坏性**：两产品前端同步切换）；
- dry-run 默认截获副作用（行为修正）。

## 8. 兼容性与迁移方案

### 8.1 兼容矩阵

| 对象 | 影响 | 结论 |
|---|---|---|
| 磁盘队列存量数据（三种 payload） | 队列格式零改动，codec 消费侧配置 | ✅ 无影响 |
| agent / gateway / logpilot 现有链 | codec 默认 otel 不变；registry 新增为纯加法 | ✅ 无影响（重编译即继承） |
| Gateway 自有 10 处理器 + 55 过滤器 | 记录无关 / 另一契约 | ✅ 零改动 |
| Coco 存量管道（messages 模式） | 处理器双模保留旧 `Process`；`strict_contract` 逃生门 | ✅ 按 `mode` 渐进 |
| 存量"非法但空转"的管道组合 | 启用契约校验后 → 编译失败 | ⚠️ 行为变化，`strict_contract: false` 过渡 |
| 前端 dry-run 调用方 | 响应结构统一 | ⚠️ 同步改造 |

### 8.2 迁移阶段

| 阶段 | 内容 | 破坏性 |
|---|---|---|
| P1 地基 | codec v2 接口 + `documentview` + `coco_document` codec + `ExecutionObserver` 埋点 | 零（纯增量） |
| P2 契约 | `RecordContract` + `NewPipelineWithScope` + catalog 透出 + `strict_contract` 开关（默认关） | 零（opt-in） |
| P3 双模迁移 | Coco 处理器加 `ProcessBatch`/记录契约；`process_documents` 支持 `mode: records`（默认旧路径）；新管道按 record 模式编写 | 零（存量不动） |
| P4 性能 | L1 池化/jsoniter/记录并行 → L2 冷热分离 → L4 `indexing_merge` 部分解析 | jsoniter 需 benchmark + 行为差异门禁 |
| P5 Lane | `core/lane` + `lane_consumer/out` + 连接器类型化入口 | 零（新平面） |
| P6 观测 | dryrun 包 + 统一端点（删产品 dry-run/AI 端点，修副作用隐患）+ 标准指标 + trace_id/飞行记录 + `_inject`/`_pause` | 产品端点删除（前端同步切） |
| P7 UI | Coco 处理器补 `WithConfigMetadata` → 共享组件包 `@infinilabs/pipeline-ui` → 两产品集成（调试抽屉/画布/面板全套） | 前端增量 + 端点切换 |
| P8 收尾 | 存量 `PipelineConfigV2` 一次性机械迁移（`X:{...}` → `for_each{codec: coco_document, processor:[X:{...}]}`）；删除 messages 旧路径；`strict_contract` 默认开 | ⚠️ 按产品节奏排期 |

### 8.3 滚动部署

- 队列内容不变 → 新旧二进制可混跑同一队列集群；
- 管道配置（ORM 存储）新字段旧二进制忽略 → 滚动升级安全；
- lane 仅进程内 → 无跨版本问题。

## 9. 性能预期

见 §D4 量化表。补充两个非拷贝维度：

- **记录级并行**：`for_each` CPU 型子链吞吐 ≈ 线性于 worker 数（记录间无共享状态，context 每 goroutine 克隆）；
- **LLM 批量**：`ProcessBatch` 把 N 条文档并一次 API 调用，是 LLM 链成本与吞吐的第一杠杆（与零拷贝正交叠加）。

验收基准（建议纳入 P4 门槛）：1MB 文档（250KB 向量）× K=4 链，记录模式较 messages 模式数据搬运 CPU 降 ≥80%；gateway 日志链（1KB 事件）`for_each` 并行后吞吐 ≥ 3× 单线程。

## 10. 风险与对策

| 风险 | 等级 | 对策 |
|---|---|---|
| 契约校验把存量"空转"变编译失败，升级即炸 | 高 | `strict_contract` 默认关 + 升级前扫描工具（列出不合规管道）+ 分批收紧 |
| jsoniter 与 stdlib 行为差异（大数、HTML 转义、map 排序） | 中 | benchmark 套件含行为对比门禁；可按管道回退 |
| mapstr 树往返的数值精度（float32 embedding 经 float64） | 中 | L2 冷热分离后向量走 RawMessage 不进 map；`RequiredFields` 软校验覆盖触达者 |
| 池化生命周期错误（double-release / 泄漏） | 中 | 仅框架内部池化（for_each 作用域），不暴露给处理器作者；race/泄漏测试 |
| lane at-most-once 丢数据被误用 | 中 | 选型判据写入文档与配置注释；`spill` 提供降级；UI 显式标注语义 |
| 记录级并行改变输出顺序 | 低 | 队列消费本无序；文档明示；需要顺序的链显式 `parallel: false` |
| dry-run 响应结构统一的前端破坏 | 低 | 同 PR 内前后端切换；版本化 API 可选 |
| enterprise 闭源依赖（私有 clone + license init） | 低 | 现状已如此，不新增依赖面 |
| 处理器名全局冲突 panic | 低 | 重名 panic 是既有防线；`category:name` 别名兜底；CI 注册冒烟测试 |
| 飞行记录采样带来的观测成本 | 低 | 默认 1/N + 环形上限 + 错误全采策略；可完全关闭（observers 空 fast path） |

## 11. 落地计划与工作量

| 阶段 | 主要仓库 | 规模 | 依赖 |
|---|---|---|---|
| P1 地基 | framework + coco | M | — |
| P2 契约 | framework + coco | M | P1 |
| P3 双模迁移 | coco | L（15 处理器 + 连接器） | P1/P2 |
| P4 性能 | framework + coco | M（L1/L4 小，L2 中） | P3（记录模式先行） |
| P5 Lane | framework + coco | M | P1（记录模型） |
| P6 观测 | framework + coco + logpilot | L（dryrun 包 + 统一端点/删产品端点 + 指标） | P2（契约复用） |
| P7 UI | ui-common + coco web + logpilot web | M-L（元数据补齐 + 共享组件包 + 两产品集成） | P2/P6 |
| P9 分布式 | framework + logpilot + ui-common | M（协调原语 + deployment/调和 + Fleet API + 集群 UI） | P6（指标/操控复用） |
| P8 收尾 | coco + framework | S-M（迁移脚本 + 删旧路径） | P3-P7 稳定后 |

关键路径：P1 → P2 → P3 → P8；P4/P5/P6/P7 可与 P3 并行分头推进。

## 12. 评审要点（开放问题清单）

以下为需要评审拍板的决策点，按影响排序：

1. **[决策] `strict_contract` 默认开启节奏**：P8 一次切 vs 按数据源分批切？是否需要独立扫描工具先行？（§8.2/§10）
2. **[决策] codec 归属位置**：`coco_document` codec 放 `coco/modules/pipeline`（现状倾向）还是更中立模块？若未来要"Coco 富化管道远程部署到 gateway/agent 执行"，codec 需被对端进程引用——是否预留？（§D1.2）
3. **[决策] `document.*` 投影的字段命名定稿**：整树投影意味着 ES 侧 source 形状由 Encode 决定（仍是 Document JSON，无破坏）；但链内条件/处理器引用的路径（`document.content`）一旦被用户写入管道配置即为公共契约，需冻结命名规则。（§D1.2）
4. **[决策] `for_each` 记录级并行默认开/关**：默认开（吞吐优先，顺序不保证）还是默认关（保守，按链 opt-in）？（§D4-L1）
5. **[决策] dry-run 的 LLM mock 策略**：录制回放的存储位置与脱敏规则；CI 是否强制 mock。（§D5.2）
6. **[决策] jsoniter 切换范围**：全 framework 还是仅 codec 热路径局部替换？（§D4-L1）
7. **[决策] StageStats 持久化**：标准模型在内存，各产品自行落 ES——是否需要 framework 内置可选持久化（避免三产品再各写一遍）？（§D5.3）
8. **[决策] Lane 是否进 P5 首批**：若无近期进程内零拷贝诉求，可延后，先做判据文档。（§D3）
9. **[确认] 共享包 `@infinilabs/pipeline-ui` 的组件边界**：画布 if/for_each 容器交互的完整集；哪些组件进共享包、哪些留产品外壳（placement/拓扑/压力监控）；包的发布节奏（源码引用 vs npm 发版）。
10. **[确认] 性能验收基准数字**（§9）：80%/3× 是否合适，基准环境如何指定。
11. **[决策] 控制面承载方**：deployment/reconciler 由 framework 全包，还是各产品自实现（console 是否纳入）？（§D7）
12. **[决策] `sharded` 策略是否进 v1**：依赖 kafka 队列后端的部署占比；v1 只做 singleton/all 是否足够。（§D7）

---

## 附录 A：术语表

| 术语 | 英文 | 定义 |
|---|---|---|
| 队列 | Queue | 持久化落盘的消息存储；bytes 载荷；at-least-once；offset/重放 |
| 通道 | Lane | 进程内有界记录通道；`*event.Event` 指针载荷（记录即消息）；at-most-once |
| 记录 | Record | 链内在途标准载体 `*event.Event`，`Fields` 为唯一权威 |
| 记录作用域 | Record Scope | `for_each`/`lane_consumer` 内的执行上下文，`CurrentRecord` 可用 |
| 命名空间 | Namespace | codec 产出的记录族标识（`document`/`attachment`/`""`），契约匹配依据 |
| 契约 | Record Contract | 处理器声明的作用域/命名空间/必需字段依赖 |
| 冷热分离 | Cold/Hot Split | 重子树（embedding）以 `json.RawMessage` 原字节寄生，访问者才解码 |
| 投影 | Projection | 领域对象 ↔ `Fields` 子树的映射（`documentview.Materialize/Project`） |
| 飞行记录 | Flight Recorder | 采样式执行事件环形缓冲，错误全采 |
| 执行事件 | StepEvent | 一次处理器调用的标准描述，dry-run/指标/日志的共同投影源 |

## 附录 B：现状证据索引（评审可复核）

| 事实 | 位置 |
|---|---|
| 处理器接口/注册表/契约 | `framework/core/pipeline/processor.go`、`register.go`、`record.go` |
| for_each 与 codec 注册表 | `framework/modules/pipeline/for_each.go`、`for_each_codec.go` |
| 统计仅 3 计数器 | `framework/core/pipeline/processor.go:277-281` |
| queue bytes 接口 | `framework/core/queue/simple_queue.go:37` |
| stdlib JSON | `framework/core/util/bytes.go:288` |
| Context 池化/无 Event 池/pause 未暴露 | `framework/core/pipeline/context.go:114-128` |
| indexing_merge 全量解析取 _id | `framework/plugins/elastic/indexing_merge/indexing_merge.go:275` |
| Coco messages 约定/门禁/子链宿主 | `coco/plugins/processors/process_documents/processor.go:76-228` |
| Coco 处理器裸注册（无元数据） | `coco/plugins/processors/*/processor.go`（grep `RegisterProcessorPlugin`） |
| 连接器进程内 RunPipelineSync | `coco/plugins/connectors/common/processor.go:102-112` |
| 两套 dry-run | `coco/modules/pipeline/dryrun.go`、`logpilot/plugins/logpilot/api_dryrun.go` |
| Logpilot Stage 指标/注入/链调试 | `logpilot/plugins/logpilot/{model_pipeline,pipeline_metrics,api_debug}.go` |
| Gateway 双契约与流管道实锤 | `gateway/main.go:49-89`、`gateway/config/stream-*-pipelines.yml` |
| 画布/表单组件 | `logpilot/web/src/modules/streams/processing-canvas.tsx`、`components/schema-form.tsx` |
| catalog 无作用域信息 | `framework/core/pipeline/register.go:330-352` |
