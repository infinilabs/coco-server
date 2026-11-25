# 统一 Pipeline — API / 配置 / 数据结构 / DSL 规范

| | |
|---|---|
| **配套文档** | [DESIGN-unified-pipeline.md](./DESIGN-unified-pipeline.md) · [WORKPLAN-unified-pipeline.md](./WORKPLAN-unified-pipeline.md) |
| **标记约定** | 【现有】已存在且不变 ·【扩展】已存在，加字段向后兼容 ·【新增】本方案新引入 |
| **版本** | Draft v1.0 — 与 P1-P8 落地同步演进 |

---

## 1. 总览：四层关系

```
DSL(链定义语法,§5)
   │ 写入
   ▼
配置载体(§3): YAML(coco.yml/gateway config) ─或─ ORM 文档(PipelineConfigV2,经 API CRUD)
   │ 由 PipeModule 加载/热重载
   ▼
运行时数据结构(§4): Processor 链 + Context + Event/Record + Queue/Lane
   ▲ 管理面
   │
REST API(§2): 任务控制 / 管道 CRUD / 发现 / 观测操控 / Studio
```

---

## 2. REST API 完整清单

所有端点挂 framework `PipeModule`（三产品共享），权限走 `security.PermissionSystemPipeline*` 或 `generic:pipeline:*`。

### 2.1 任务控制【现有】

| 方法 | 路径 | 用途 |
|---|---|---|
| GET | `/pipeline/tasks/` | 运行中任务列表 |
| POST | `/pipeline/tasks/` | 创建任务 |
| POST | `/pipeline/tasks/_search` | 任务搜索 |
| GET / DELETE | `/pipeline/task/:id` | 任务详情 / 删除 |
| POST | `/pipeline/task/:id/_start` | 启动 |
| POST | `/pipeline/task/:id/_stop` | 停止 |
| GET | `/pipelines/_running` | 运行中管道 |

### 2.2 管道文档 CRUD（ORM 持久化）【现有】

| 方法 | 路径 | 用途 |
|---|---|---|
| POST | `/pipelines/_search` | 搜索 |
| POST | `/pipelines/` | 创建 |
| GET / PUT / DELETE | `/pipelines/:id` | 读 / 改 / 删 |
| POST | `/pipelines/:id/_start` / `_stop` | 启停 |

**请求/响应体**：`PipelineConfigV2`（见 §3.2）。

### 2.3 发现类

| 方法 | 路径 | 用途 | 状态 |
|---|---|---|---|
| GET | `/pipeline/processors`（`?grouped=1`） | 处理器目录（schema + 契约） |【扩展】 |
| GET | `/pipeline/codecs` | codec 目录 |【新增】 |
| GET | `/pipeline/lanes` | lane 清单 + 实时指标（depth/in·out rate/drops/spills/emit_blocks/pending_acks/ack_timeouts/owner/consumer 任务） |【新增】 |
| GET | `/pipeline/lanes/:name` | 单 lane 详情（配置/指标/消费者/回执状态） |【新增】 |

**`GET /pipeline/processors` 响应【扩展后】**：

```json
{
  "ai": {
    "extract_entities": {
      "properties": { "message_field": {"type":"string","default":"messages"}, "...": "..." },
      "category": "ai",
      "scope": "record",                                  // 【新增】batch | record | unconstrained
      "record_contract": {                                 // 【新增】unconstrained 时为 null
        "namespace": "document",
        "required_fields": ["document.content"],
        "granularity": "batch_capable"                     // 【新增】record | batch_capable | batch_only
      }
    }
  }
}
```

**`GET /pipeline/codecs` 响应【新增】**：

```json
{ "codecs": [
  { "name": "otel",         "namespace": "",           "version": 1, "description": "OTLP/agent LogEvent 信封" },
  { "name": "coco_document","namespace": "document",   "version": 1, "description": "Coco Document 投影" },
  { "name": "coco_attachment","namespace": "attachment","version": 1, "description": "Coco Attachment 投影" }
]}
```

### 2.4 观测与在线操控【新增】

> 单任务操作统一用 `/pipeline/task/:id/...` 前缀（与现有 `_start/_stop` 一致）。

| 方法 | 路径 | 用途 |
|---|---|---|
| POST | `/pipeline/task/:id/_inject` | 向任务输入注入样本 |
| POST | `/pipeline/task/:id/_pause` | 暂停（当前批次完成后挂起） |
| POST | `/pipeline/task/:id/_resume` | 恢复 |
| GET | `/pipeline/task/:id/flight?trace_id=xxx` | 单记录全链轨迹 |
| GET | `/pipeline/task/:id/flight?errors=1&limit=50` | 最近失败步骤 |
| GET | `/pipeline/task/:id/flight?window=7d` | 带 window 查**持久化索引**（不带查内存环形缓冲，同一端点同一格式） |
| POST | `/pipeline/metrics/_search` | 持久化指标搜索（`pipeline_stage_stats` 快照，ORM 透传） |
| GET | `/pipeline/task/:id/stages` | 标准阶段指标 |

**`_inject` 请求**（Sample 模型见 §4.4）：

```json
{
  "queue": "indexing_documents",              // 或 "lane": "doc_enrichment"——结构化目标(恰好一个)
  "samples": [
    { "raw": "{\"id\":\"doc-1\",\"title\":\"...\",\"content\":\"...\"}", "codec": "coco_document" }
  ]
}
```

**`flight` 响应**（StepEvent 数组，见 §4.4）：

```json
{ "events": [
  { "trace_id":"tr-8f2a", "pipeline":"process_documents", "step":3, "processor":"extract_entities",
    "scope":{"kind":"record","namespace":"document"},
    "duration_ms":1240, "outcome":"ok" },
  { "trace_id":"tr-8f2b", "step":4, "processor":"embedding", "outcome":"error", "err":"provider timeout" }
]}
```

**`stages` 响应**（StageStats 数组，见 §4.4）：

```json
{ "stages": [
  { "stage":"enrichment", "carrier":"queue:indexing_documents", "depth":120, "lag_amount":0,
    "in_total":45000, "out_total":44800, "in_rate":51.2, "out_rate":50.8, "errors":3, "drops":12 },
  { "stage":"enrichment", "carrier":"lane:doc_enrichment", "depth":8,
    "in_total":900, "out_total":900, "in_rate":10.1, "out_rate":10.1, "errors":0, "drops":0 }
]}
```

### 2.5 统一 Dry-Run 与 AI 生成【新增·framework 层，无产品 wrapper】

| 方法 | 路径 | 用途 |
|---|---|---|
| POST | `/pipeline/dry-run` | dry-run（**唯一定义处**，替代一切产品级 dry-run） |
| POST | `/pipeline/ai-generate` | AI 生成处理器链（产品注册生成器实现） |

**`POST /pipeline/dry-run` 请求**：

```json
{
  "processor": [ /* 链 DSL，见 §5 */ ],
  "samples": [ { "raw": "{...document json...}", "codec": "coco_document" } ],
  "options": {
    "scope_check": true,
    "side_effects": "capture",      // capture | passthrough
    "llm_policy": "real"             // real | mock
  }
}
```

**响应 `Report`**：

```json
{
  "compiled": [
    { "step":1, "processor":"for_each", "ok":true },
    { "step":2, "processor":"grok", "ok":true,
      "note":"contract: unconstrained, namespace=document 可用" }
  ],
  "steps": [ [ /* 每样本一层数组，每步一个 StepResult */ ] ],
  "side_effects": [
    { "kind":"queue", "target":"documents_processed", "records":1, "preview":"{...首条截获写入...}" }
  ]
}
```

`StepResult` 单元素：

```json
{
  "step": 2, "processor": "grok",
  "before": { "document.content": "1.2.3.4 - - [10/Oct/2026] ..." },
  "after":  { "document.content": "...", "document.client_ip": "1.2.3.4" },
  "diff":   { "added": {"document.client_ip":"1.2.3.4"}, "removed": {}, "changed": {} },
  "duration_ms": 3,
  "outcome": "ok"                    // ok | error | dropped | skipped-contract
}
```

**`POST /pipeline/ai-generate` 请求/响应**：

```json
// 请求
{
  "intent": "给文档提取实体并打标签",
  "namespace": "document",            // 目标记录族；默认以 for_each{codec} 包裹生成链
  "hints": { "samples": [ { "raw": "...", "codec": "coco_document" } ] }  // 可选
}
// 响应
{ "processor": [ /* 链 DSL */ ], "generator": "coco-llm" }
```

framework 定义 `ChainGenerator` 接口，产品在 init 注册实现（Coco 用 `modules/llm`；Logpilot 注册自己的）；未注册时返回 501。生成结果**不直接落库**——前端拿到链后立即调 `/pipeline/dry-run` 校验，再由用户确认保存。

**样本约定**：样本一律由客户端内联传入。产品的样本来源（如 Logpilot 流历史取数）属数据接口，由前端自行取回后填入 `samples`，不进入 pipeline API 契约。

### 2.6 产品端点收敛（删除/迁移清单）

| 产品端点 | 处置 |
|---|---|
| `POST /pipeline/studio/test`（coco） | **删除** → `POST /pipeline/dry-run` |
| `POST /pipeline/studio/ai-generate`（coco） | **删除** → `POST /pipeline/ai-generate` |
| `POST /logpilot/streams/:id/dry-run` | **删除** → `POST /pipeline/dry-run`（前端内联传 `spec.processing.processors` + 样本） |
| `POST /logpilot/parsing/test` | **删除** → `POST /pipeline/dry-run`（单处理器即单步链） |
| `POST /logpilot/parsing/ai-generate` | **删除** → `POST /pipeline/ai-generate` |
| `POST /logpilot/streams/:id/debug/_inject` | **删除** → `POST /pipeline/task/:id/_inject` |
| `GET /logpilot/streams/:id/debug/chain` | **删除** → `GET /pipeline/task/:id/flight` |
| `GET /logpilot/pipeline-metrics`、`/pipeline-summary` | 保留（数据源改喂标准 StageStats，响应不变，看板不动） |
| `GET/PUT /logpilot/streams/:id/spec` 及版本/发布 | 保留（流模型属 Logpilot 数据层，不动） |

---

## 3. 配置文件与格式

### 3.1 YAML 管道配置【扩展】

入口：`coco.yml` / gateway config 目录的顶层 `pipeline:` 段（fsnotify 热重载）。完整字段：

```yaml
pipeline:
  - name: process_documents            # 管道名(== ORM 文档 ID)
    enabled: true
    auto_start: true
    keep_running: true                  # 失败重试
    singleton: true                     # 分布式单例(locker)
    retry_delay_in_ms: 5000
    max_running_in_ms: 60000            # 单例锁 TTL
    strict_contract: false              # 【新增】契约校验开关(P8 前默认 false)
    processor:                          # 链 DSL(§5)
      - consumer:
          message_field: messages
          queue_selector: { keys: [indexing_documents] }
          consumer: { group: enriched_documents, fetch_max_messages: 10 }
          auto_commit_offset: true
          processor:                    # ⚠ 持久化动作必须在 consumer 子链内:
                                      #    offset 仅在子链成功后推进提交(consumer.go
                                      #    at-least-once 语义),放外面=提交先于落盘的丢数窗口
            - for_each:                 # batch→record 边界
                codec: coco_document    # 【扩展】新增 codec 取值
                parallel: 4             # 【新增】记录级并行(默认 1=串行,待 D-4)
                on_failure: tag         # ignore | tag | fail
                failure_tag: _processing_failed
                processor:
                  - grok:
                      field: document.content
                      patterns: ["..."]
                  - extract_entities: {}
            - output:                   # 在 consumer 子链内:先落盘,后提交 offset
                message_field: messages
                queue: documents_processed
```

**路径成本注记**：上例是**持久化路径**（durable path）——输入来自磁盘队列，`for_each` 两侧各付一次编解码（L2 冷热分离后仅热字段，向量子树原字节通过），链内零拷贝；适用含 LLM 等不可重算步骤的链。若整链均为可重算的轻步骤（且上游可重采），应改用 **lane 快路径**（`lane_consumer` 直达记录作用域，零编解码，见附录"Lane 使用场景速查"）。两路径的选择判据与 at-most-once 边界见该附录。

### 3.2 `PipelineConfigV2` ORM 文档【扩展】

`PUT/POST /pipelines/` 的 JSON 体（framework `core/pipeline/config.go`）：

```json
{
  "id": "default-document-enrichment",
  "name": "default-document-enrichment",
  "enabled": true,
  "singleton": false,
  "auto_start": true,
  "keep_running": true,
  "retry_delay_in_ms": 5000,
  "max_running_in_ms": 60000,
  "strict_contract": false,             // 【新增】
  "labels": { "owner": "coco" },
  "transient": false,
  "processor": [ /* 链 DSL,同 §3.1 */ ],
  "canvas_layout": {                    // 【新增】画布布局(react-flow graph JSON,仅布局非权威)
    "nodes": [ { "id": "n1", "position": {"x":100,"y":40}, "data": {"stepIndex": 0} } ],
    "edges": [ { "id": "e1", "source": "n1", "target": "n2" } ]
  }
}
```

旧字段全部保留；旧读写方忽略新字段 → 滚动升级安全。

### 3.3 Lane lane 配置【新增】

**词汇约定**：用户可见词汇只有 **lane（通道）**一个——配置段落 `lane:`、处理器 `lane_consumer`。对仗：入口 `consumer` ↔ `lane_consumer`（宿主），出口统一为 `output`（步骤）。**目标引用一律用 one-of 结构化字段**（`queue: <名>` / `lane: <名>`，恰好一个——output/tap/_inject 同款规则）；展示串（StageStats carrier 的 `queue:xxx`）仅只读输出用。

```yaml
lane:
  - name: doc_enrichment
    namespace: document          # 承载记录族;lane_consumer 据此做契约编译
    capacity: 4096               # 有界
    overflow: block              # block | drop | spill:indexing_documents
```

**动态 lane（动态管道场景）**：lane 注册表为运行期 map，除配置文件外支持运行期平面——`lane.Create(cfg)` / `lane.Delete(name)`（排空后删）/ `GetOrCreate`（首次 Emit 惰性建）。动态管道编译器按需生成（名字派生自流 ID，如 `stream-<id>-fast`），带 `owner` 字段做级联清理（流删除→级联，孤儿检测复用 configs-center 模式）。**隔离规则：一条动态管道一条 lane，绝不共享**（一条 lane 一个消费任务一条链，共享=语义错误）；背压因此天然隔离——慢的流只堵自己。内存三护栏：惰性分配（空闲 lane ≈0，实现用可增长有界队列而非固定 chan）、动态 lane 小默认容量（256~512）、全局预算 `total_budget`（全体动态 lane 在途总量触顶按各自 overflow 背压）。生命周期：停上游任务（在途回执落定/超时）→ 停下游（排空）→ 删 lane → 孤儿检测兜底。**动态管道默认仍走磁盘队列，lane 为 opt-in**（spec 声明或编译器对确定性链建议）。

### 3.4 关键处理器配置 schema 速查

| 处理器 | 作用域 | 关键配置 |
|---|---|---|
| `consumer` | batch | `queue_selector{keys}`、`consumer{group,fetch_max_messages}`、`auto_commit_offset`、`message_field`、`processor`(子链)；【新增糖】`codec` + `output_lane`——磁盘→内存桥接三明治（consumer+for_each+output{lane:…}）压成一块：fetch→decode→emit，offset 由 AckSet 门控 |
| `for_each` | batch→record | `message_field`(默认 messages)、`codec`(默认 otel)、`on_failure`、`failure_tag`、`parallel`【新增】、`processor`(子链) |

| `lane_consumer`【新增】 | →record(链宿主) | `lane`、`batch{max,flush_ms}`(攒批)、`on_failure`(tag 默认 / dead_letter:\<queue\> / ignore)、`failure_tag`、`ack_timeout_ms`(ack 模式回执超时→上游不提交→重放)、`output_queue`(可选,持久桥)、`output_codec`(可选,缺省三级解析：**记录 `_codec` 提示 > lane.namespace 主 codec > 默认 otel**)、`processor`(子链)。**执行契约与 for_each 完全一致**（段式执行：顺序即书写顺序，drop/clone/失败标签同款），仅两处差异：入口零解码（记录由 Emit 方构造）；出口仅当配 `output_queue` 时编码——`queue.Message.Data = codec.Encode(record)`，与该 codec 的全部既有消费者天然兼容（如 `coco_document` 产出与今日 `indexing_documents` 内容同构，现有链零改动）。**`_codec` 提示由宿主打**（for_each 解码循环/类型化入口盖 `metadata.labels._codec`），编解码往返对称。**保真规则**：命名空间 codec 只序列化自身子树（`document.*`），子树外的根级字段为链内瞬态数据不落队列；dry-run 的编码边界步骤可见此差异。**边界无损不变式**：每条进入的记录必然落定于 output_queue / dead_letter / 显式 drop 之一（详见附录"回执与提交门"） |
| `output`【新增·统一出口】 | batch+record（按目标条件约束，见语义矩阵附录提案） | 目标 one-of 字段：`queue: <名>` / `lane: <名>`，恰好一个；`ack`(lane 目标可选)。**子链内的普通步骤（非宿主）**：把当前记录 `Emit` 进目标 lane 后立即返回、本链继续；lane 目标：记录的后续处理发生在**订阅该 lane 的任务**的 `lane_consumer`（异步交接，发送方不等待；`overflow: block` 时 Emit 阻塞=背压传回本链）。`ack: true` 时 Emit 返回回执并挂入 ctx 的 **AckSet（提交门）**——上游 consumer 在 AckSet 全部落定（成功/失败入 DLQ/显式 drop）后才提交 offset，未决或超时则不提交、由磁盘队列重放（跨 lane 达成 at-least-once，重复由幂等 sink 消化；lane 自身不存数据，投递保证借自上游 offset）。dry-run 时可截获 |
| `tap`【新增】 | record | `sample_ratio`(0~1)、目标 one-of 字段（`queue: <名>` / `lane: <名>`，恰好一个） |
| `indexing_merge` | batch | `elasticsearch`、`index_name`、`input_queue`、`output_queue{name}`、`write_op_type`、`key_field`/`key_fields` |
| `bulk_indexing` | batch | `elasticsearch`、`queue_selector{keys}` |
| `if/then/else` | 任意 | `if:`(条件)、`then:`/`else:`(子链) |
| `dag` | batch | 见 framework `dag_processor` |

Coco 记录处理器（scope=record，namespace=document；配置 schema 由 `WithConfigMetadata` 自动生成，代表性的）：

| 处理器 | 关键配置 |
|---|---|
| `extract_entities` | `message_field`、`model_provider` |
| `document_embedding` | `model_provider`、输出维度约束 `RequiredEmbeddingDimension=1024` |
| `document_summarization` | `model_provider`、prompt 模板 |
| `extract_tags` | `model_provider`、`output_queue` |
| `process_documents` | `message_field`、`output_queue`、`mode: messages\|records`【新增】 |

---

## 4. Go 数据结构（按包分组）

### 4.1 `framework/core/pipeline` — 引擎

```go
// 【现有】处理器契约
type ProcessorBase interface { Name() string }
type Processor interface { ProcessorBase; Process(s *Context) error }
type BatchProcessor interface { Processor; ProcessBatch(ctx *Context, records []*event.Event) error }

// 【现有】链编译(P2 起新增作用域感知变体)
func NewPipeline(cfg []*config.Config) (*Processors, error)
func NewPipelineWithScope(cfg []*config.Config, s ScopeInfo) (*Processors, error)   // 【新增】

// 【新增】作用域与契约
type Scope int                                  // ScopeBatch | ScopeRecord
type ScopeInfo struct { Kind Scope; Namespace string }
type RecordContract struct {
    Scope          Scope
    Namespace      string        // "document"|"attachment"|""=任意
    RequiredFields []string      // 如 "document.content"
}
type ContractProvider interface { RecordContract() RecordContract }
func CheckContract(ctx *Context, c RecordContract) error   // 运行期兜底

// 【新增】观测
type Outcome string   // "ok"|"error"|"dropped"|"skipped-contract"
type Snapshot struct { Fields util.MapStr; Dropped bool }
type StepEvent struct {
    TraceID string; Pipeline string; Step int; Processor string
    Scope ScopeInfo; Before, After Snapshot
    Duration time.Duration; Outcome Outcome; Err string
}
type ExecutionObserver interface { OnStep(e *StepEvent) }

// 【现有】记录约定(P1 增 trace 助手)
const RecordContextKey param.ParaKey = "record"
func CurrentRecord(ctx *Context) (*event.Event, bool)
func TraceID(rec *event.Event) string                    // 【新增】
func SetTraceID(rec *event.Event, id string)             // 【新增】

// 【现有】管道配置 ORM 对象(扩展字段见 §3.2)
type PipelineConfigV2 struct { /* Name/Enabled/Singleton/AutoStart/KeepRunning/
    RetryDelayInMs/MaxRunningInMs/Processors/Labels/Transient +
    StrictContract *bool + CanvasLayout util.MapStr */ }
```

### 4.2 `framework/modules/pipeline` — 运行宿主与 codec

```go
// 【扩展】codec 契约
type RecordCodec interface {
    Name() string
    Namespace() string                                   // 【新增】""=通用
    Decode(data []byte) (*event.Event, error)
    Encode(rec *event.Event) ([]byte, error)
}
func RegisterRecordCodec(c RecordCodec) error            // 【扩展】返回 error
func RegisterRecordCodecOrReplace(c RecordCodec) error   // 【新增】
func ListRecordCodecs() []CodecInfo                      // 【新增】
type CodecInfo struct { Name, Namespace, Version, Description string }

// 【新增】dryrun 包(framework/modules/pipeline/dryrun)
type Sample struct { Raw []byte; Codec string }
type SideEffectPolicy string   // "capture"|"passthrough"
type LLMPolicy string          // "real"|"mock"
type Options struct { ScopeCheck bool; SideEffects SideEffectPolicy; LLMPolicy LLMPolicy }
type StepResult struct { Step int; Processor string; Before, After util.MapStr
    Diff FieldDiff; DurationMs int64; Outcome string; Err string }
type FieldDiff struct { Added, Removed, Changed util.MapStr }
type CapturedWrite struct { Kind, Target string; Records int; Preview string }
type Report struct { Compiled []StepDesc; Steps [][]StepResult; SideEffects []CapturedWrite }
func Run(chain []map[string]interface{}, samples []Sample, opts Options) (*Report, error)

// 【新增】标准指标
type StageStats struct {
    Stage string                    // "queue:<name>"|"lane:<name>"|"inline"
    Carrier string
    Depth, LagAmount int64
    InTotal, OutTotal, InRate, OutRate float64
    Errors, Drops int64
}
```

### 4.3 `framework/core/lane` —【新增】

```go
type Handler func(rec *event.Event)
type Lane interface {
    Name() string
    Emit(rec *event.Event) error        // 按背压策略
    TryEmit(rec *event.Event) bool
    Subscribe(h Handler) (cancel func())
    Depth() int
    Close() error
}
type LaneConfig struct {
    Name      string `config:"name"`
    Namespace string `config:"namespace"`
    Capacity  int    `config:"capacity"`
    Overflow  string `config:"overflow"`   // block|drop|spill:<queue>
}
func GetLane(name string) (Lane, error)
func Emit(lane string, rec *event.Event) error
```

### 4.4 `framework/core/event` 与 `framework/core/queue` —【现有，仅列关键】

```go
// event.Event —— 链内唯一在途记录(记录即消息)
type Event struct {
    Agent      *AgentMeta
    QueueName  string        `json:"-"`
    Timestamp  time.Time     `json:"timestamp,omitempty"`
    Metadata   EventMetadata `json:"metadata"`       // Labels/Category/Name/Version/Datatype
    Fields     util.MapStr   `json:"payload"`        // 唯一权威;document.* 子树在此
    Meta       util.MapStr   `json:"-"`
    Private    interface{}   `json:"-"`
}
// Metadata.Labels 惯例键: _trace_id(§2.4)、datasource、connector_id、lang、type

// queue.Message —— 持久传输信封(bytes-only)
type Message struct {
    Timestamp  int64
    Offset     Offset
    NextOffset Offset
    Size       int
    Data       []byte         // Document JSON | otel 信封 | fasthttp 字节,由 codec 解释
}
```

### 4.5 Coco 侧 —【新增】

```go
// coco/modules/pipeline —— codec
type DocumentCodec struct{}   // Name()="coco_document"; Namespace()="document"
// Decode: Document JSON → Event{Fields:{"document": <整树>}, Labels, _trace_id}
// Encode: Fields["document"] → Document JSON
// DecodeFromDocument(doc *core.Document) *event.Event   // P5 类型化入口(零解码)
type AttachmentCodec struct{} // Name()="coco_attachment"; Namespace()="attachment"

// coco/core/documentview —— 领域视图
func Materialize(rec *event.Event) (*core.Document, error)
func Project(doc *core.Document, rec *event.Event)

// 处理器改造后形态(示例 embedding)
func (p *Processor) RecordContract() pipeline.RecordContract {
    return pipeline.RecordContract{Scope: pipeline.ScopeRecord, Namespace: "document"}
}
func (p *Processor) Process(ctx *pipeline.Context) error          // 单记录
func (p *Processor) ProcessBatch(ctx *pipeline.Context, rs []*event.Event) error // LLM 批量
```

---

## 5. DSL 规范

### 5.1 链文法

```
pipeline         := entry+                          # 有序数组,顺序即执行序
entry            := processor_entry | conditional_entry
processor_entry  := { NAME ":" config }             # 每项恰好一个键 = 处理器名
                   | { "category:NAME" ":" config } # 域限定别名(与 bare 等价)
conditional_entry:= { "if" ":" condition, "then" ":" pipeline, "else" ":" pipeline? }
config           := { 处理器自有字段..., "when" ":" condition? }   # when 可包裹任意处理器
NAME             := 注册名(bare) — consumer | for_each | output | grok | mutate
                   | extract_entities | document_embedding | lane_consumer | tap | ...
```

嵌套宿主：`for_each.processor`、`if.then/else`、`consumer.processor`、`dag` 内部均为完整 `pipeline`（递归）。

**求值与批量语义（重要）**：

- **条件求值是双模的**（`core/pipeline/conditionals.go:200`）：record 作用域内（for_each 子链中）`when`/`if` 对 `CurrentRecord` 的字段求值（`document.lang` 等记录路径）；无记录绑定时（链根）对 pipeline 参数求值（`conditions.ValuesMap(ctx)`，如 consumer 写入的上下文）。记录字段**遮蔽**同名参数。
- **处理粒度是声明式的**（不靠运行时检测）：契约新增 `granularity` 三档——`record`（默认，仅逐条有意义）、`batch_capable`（批量+单条两路皆有效，批量优先）、`batch_only`（仅批量通道有意义，如跨记录聚合）。实现 `BatchProcessor` 即默认承担"单条兜底"契约。
- **执行语义 = 段式执行（stage-wise），顺序即书写顺序**：编译时按书写序把子链切成交替段——连续逐条处理器合成逐条段（整批逐条过）、每个批量处理器自成一个批段（一次调用整批）——运行时逐段推进。批处理器**看到其前面逐条处理器的修改**（所见即所得）。配套语义：早段 drop 的记录从后续所有段（含批段）排除；批段失败给批内全部活动记录打 failure_tag；逐条段产生的 clone 加入活动集流入后续段。**注**：现行 framework `for_each` 的两阶段行为（批量成员无论位置先跑）在本方案中废弃——`BatchProcessor` 当前全生态零实现，无存量依赖，属零风险语义修正（P2 落地窗口）。
- **放置校验**：`batch_only` 只能作为 `for_each.processor`/`lane_consumer.processor` 的**直接成员**，置于 `if.then/else` 等嵌套位置为**编译错误**（该处批量通道不可达，只会静默退化）；`batch_capable` 置于嵌套位置为**警告**（退化为逐条 `Process`，需自带有效单条路径）。实现 `BatchProcessor` 即默认承担"单条兜底"契约。
- **粒度错配矩阵**（正确性全部安全）：单条×逐条=正常；单条×batch_capable=len=1 退化批（语义正确）；单条×batch_only=编译错误；批量×逐条=逐条各跑（无批量收益）；批量×batch_capable=最优；批量×batch_only=正常。
- **编排 UI**：catalog 每项新增 `granularity` 字段（节点渲染"批量"徽章）；段式执行下顺序即书写顺序，画布无需特殊分区——批量节点以徽章 + 连线上的段标记（`批(×1)/条(×N)`）呈现执行形态；`batch_only` 拖入嵌套位置即时红叉、`batch_capable` 拖入嵌套位置黄色降级提示（与编译校验同源）；dry-run 每步显示实际调用形态（`batch(40)`/`record`），多样本走批量路、单样本走单条路。
- `dag` 处理器是**单处理器内的作业并行构造**（`Pipeline` 串行 / `Spawns` 并行 / `Any` 竞速 / `Join` 汇合，`Parse(dsl)` 支持 DSL 文法），并行分支经共享 Context 参数传递中间结果——它是 fan-out/join 的作业图，不是数据流 DAG 引擎；逐记录的数据分叉用 `if`/`when` + `clone`，跨记录的批处理用 `ProcessBatch`。

### 5.2 条件文法（framework `core/conditions`，map 形式，键为字段路径）

```yaml
when:
  equals:  { document.lang: "en" }
  # 或组合:
  and: [ { equals: { document.type: "pdf" } }, { not: { exists: { field: document.summary } } } ]
```

可用条件：`equals`、`contains`、`range`、`exists`、`and`、`or`、`not` 等。**字段路径语法在 record 作用域内为事件路径**（`document.content`、`tags`、`metadata.labels.datasource`）。

### 5.3 作用域与契约规则（编译期）

| 位置 | 作用域 | Namespace |
|---|---|---|
| 链根 | batch | — |
| `for_each{codec: X}` 子链 | record | `X.Namespace()` |
| `lane_consumer{lane: L}` 后继 | record | `L.Namespace` |
| 嵌套 `for_each` | 内层覆盖外层 | 内层 codec |

校验：处理器声明 `RecordContract` 且 (Scope 不符) 或 (Namespace 非空且 ≠ 当前) → 编译失败，错误含步序号/处理器名/两端命名空间。`unconstrained` 处理器任意位置放行。`strict_contract: false` 时降级为警告。

### 5.4 完整示例

**示例 1 — Coco 文档富化（record 模式，混排 enterprise + LLM 处理器）**：

```yaml
pipeline:
  - name: document-enrichment
    auto_start: true
    keep_running: true
    processor:
      - consumer:
          queue_selector: { keys: [indexing_documents] }
          consumer: { group: enriched_documents, fetch_max_messages: 10 }
          processor:
            - for_each:
                codec: coco_document
                on_failure: tag
                processor:
                  - redact:                       # enterprise:脱敏,无契约约束
                      fields: [ document.content ]
                      rules: [phone, id_card]
                  - extract_entities: {}           # coco:契约 namespace=document ✅
                  - when: { equals: { document.lang: [ "en", "zh" ] } }
                    extract_tags: {}
                  - document_embedding: {}         # BatchProcessor:批段一次调用整批(段式执行)
            - output:
                queue: documents_processed
      - indexing_merge:
          elasticsearch: default
          index_name: documents
          input_queue: documents_processed
          output_queue: { name: bulk_documents }
      - bulk_indexing:
          elasticsearch: default
          queue_selector: { keys: [bulk_documents] }
```

**示例 2 — 事件分支路由**：

```yaml
processor:
  - for_each:
      codec: otel
      processor:
        - if:
            equals: { metadata.labels.source: "nginx" }
          then:
            - grok: { field: message, patterns: ["%{IPORHOST:client_ip} %{WORD:method} %{URIPATHPARAM:uri}"] }
            - geoip: { field: client_ip }
          else:
            - json: { field: message, target_field: payload }
        - drop_fields: { fields: [ident, referrer] }
```

**示例 3 — Lane 进程内直通（零序列化，宿主形态）**：

```yaml
lane:
  - name: doc_enrichment
    namespace: document
    capacity: 4096
    overflow: block                 # 默认:阻塞背压(见附录 Lane 速查)

pipeline:
  - name: fast-enrich
    processor:
      - lane_consumer:                   # 链宿主,执行契约同 for_each
          lane: doc_enrichment      # 记录指针原样到达,免 codec
          batch: { max: 40, flush_ms: 100 }   # 攒批:BatchProcessor 批量通道生效
          output_queue: indexing_documents    # 持久桥:幸存记录编码一次入队
          processor:
            - extract_tags: {}      # BatchProcessor:40 条/批,1 次 LLM 调用
```

**示例 4 — 磁盘 → 内存桥接（consumer 糖形态，推荐）**：

```yaml
lane:
  - name: mem_processing
    namespace: ""                    # 或 document(按 payload 族)
    capacity: 4096
    overflow: block                  # 满了阻塞 emit → consumer 减速 → 背压传回磁盘消费

pipeline:
  - name: disk-to-lane               # 阶段1:磁盘 → 内存,这个任务只做这一件事
    auto_start: true
    keep_running: true
    processor:
      - consumer:
          queue_selector:
            keys: [ ingest ]         # 从哪条磁盘队列读
          consumer:
            group: disk-to-lane      # 消费组(offset 记在组名下)
            fetch_max_messages: 40   # 每轮拉多少条 = 每轮递多少条
          codec: otel                # bytes → 记录(按 payload 格式选:otel/coco_document/…)
          output_lane: mem_processing  # 递进这条内存通道,任务到此结束
          ack_timeout_ms: 30000      # 回执超时:未落定 → 不 commit → 下轮重放(防丢)

  - name: fast-enrich                # 阶段2:见示例 3(内存 → 工作流 → 落盘)
    processor:
      - lane_consumer:
          lane: mem_processing
          batch: { max: 40, flush_ms: 100 }
          output_queue: indexing_documents
          processor: [ extract_tags: {} ]
```

执行时序（每轮）：`fetch 40 条 bytes → codec.Decode ×40 → Emit ×40 进通道 → 等 AckSet（阶段2处理完成/落盘 → commit offset；超时未落定 → 不 commit → 重放）`。

**等价展开形态**（糖展开后的三明治，无需手写）：

```yaml
      - consumer:
          queue_selector: { keys: [ingest] }
          consumer: { group: disk-to-lane, fetch_max_messages: 40 }
          processor:
            - for_each:
                codec: otel
                processor:
                  - output: { lane: mem_processing, ack: true }
```

---

## 6. 命名空间与路径公共契约【需冻结，评审项 D-3】

一旦用户将路径写入管道配置/条件，即成为对外契约。v1 冻结范围：

| 路径 | 含义 | 生产者 |
|---|---|---|
| `document.*` | `core.Document` 整树投影（`title`/`content`/`lang`/`tags`/`summary`/`source.id`/`content_hash`/`status`/...） | `coco_document` codec |
| `document.document_chunk` | 冷子树：链内为 `json.RawMessage` 原字节（契约上视作不透明，仅 embedding 处理器读写） | 同上 |
| `attachment.*` | `core.Attachment` 投影 | `coco_attachment` codec |
| `metadata.labels._trace_id` | 记录追踪 ID | 入口(consumer/lane_consumer/连接器) |
| `metadata.labels.datasource` / `connector_id` / `lang` / `type` | 路由元数据 | `coco_document` codec |
| `tags` | failure_tag 落点（`on_failure: tag` 时合并写入） | for_each |

规则：Document 新增字段自动以 `document.<json字段名>` 可见（整树投影），但**热路径字段名与 metadata.labels 键一经发布不改名**；`document.*` 子树的删除/改名走 major 版本（`coco_document` codec 版本 +1）。

---

## 7. UI 统一：共享组件包 `@infinilabs/pipeline-ui`【新增】

**位置**：`ui-common/packages/Pipeline`（pnpm monorepo，与 `@infinilabs/entity-ui` 等并列；coco/web 已有 `@infinilabs/*` 消费先例）。

**原则**：编辑与调试的交互只有一份实现，Coco 与 Logpilot 消费同一包，交互天然一致；产品差异收窄为数据绑定与外壳页面。

| 组件 | 职责 | 对接 API |
|---|---|---|
| `PipelineCanvas` | React Flow 画布：源/处理器/汇节点、`if`/`for_each` 容器、`when` 徽章、JSON 双视图、布局存取回调（props 注入） | — |
| `ProcessorPalette` | 节点面板：catalog 驱动，按 `scope`/`record_contract` 过滤，不兼容项置灰 + 原因 tooltip | `GET /pipeline/processors` |
| `SchemaForm` | 处理器配置表单（catalog `properties` 驱动） | 同上 |
| `DryRunDrawer` | dry-run 调试抽屉：逐步快照 / 字段 diff / 副作用预览 | `POST /pipeline/dry-run` |
| `AiGeneratePanel` | AI 生成链（意图输入 → 生成 → 自动触发 dry-run 校验） | `POST /pipeline/ai-generate` |
| `FlightTimeline` | 飞行记录时间线（按 trace_id / 错误过滤） | `GET /pipeline/task/:id/flight` |
| `StageStatsPanel` | 阶段指标卡（queue 深度/lag、lane 深度、速率、错误） | `GET /pipeline/task/:id/stages` |
| `LaneOverviewPanel` | lane 总览：深度条（<60% 绿/<85% 黄/红）、in/out 迷你趋势、drops/spills 徽章、单 lane 下钻飞行记录 | `GET /pipeline/lanes` |
| `InjectDialog` | 样本注入对话框 | `POST /pipeline/task/:id/_inject` |
| `pipeline-api`（TS client） | 上述端点的类型化封装（`Report`/`StepResult`/`StageStats` 等类型随包发布） | 全部 |

**产品绑定（props 注入，编辑器本体零分叉）**：

| 绑定项 | Coco | Logpilot |
|---|---|---|
| 链持久化 | `PipelineConfigV2` CRUD（`/pipelines/*`） | `LogStream.Spec`（`/logpilot/streams/:id/spec`） |
| 画布布局存储 | `PipelineConfigV2.canvas_layout` | `spec.processing.graph` |
| 样本来源 | 文档样本（用户粘贴/历史文档） | 流历史取数 API |
| 外壳自有面板 | — | placement、链路拓扑、压力监控（不进共享包） |

---

## 8. 分布式调度与治理【新增·多 Gateway】

### 8.1 部署期望态：`PipelineDeployment`（控制面对象）

```json
{
  "id": "dep-stream-nginx",
  "pipeline_id": "stream-nginx-parse",
  "version": 7,
  "placement": {
    "selector": { "labels": { "role": "log-gateway" } },
    "strategy": "singleton",
    "rollout": { "canary": 1, "batch": 2, "max_unavailable": 1 }
  },
  "enabled": true
}
```

- `strategy`（**v1 仅前两种**，D-12 已决）：`singleton`（目标集一台跑，锁竞争 + TTL 接管）| `all`（每台各跑，本机 intake 场景）；`sharded:N`（N 台分片消费 kafka 共享队列）**设计保留、暂不支持**（生产无 kafka 后端）；
- `rollout`: 版本化配置天然支持回退；canary → 批次推进，`max_unavailable` 为同时不可用上限；
- 实例回报（`_sync` 内容扩展）：`{ pipeline_id: { "generation": 7, "state": "running|stopped|draining|failed", "last_error": "" } }`——控制面据此生成"期望 vs 实际"矩阵。

### 8.2 Fleet API

| 方法 | 路径 | 用途 |
|---|---|---|
| POST | `/pipeline/deployments/` | 创建部署（pipeline + placement） |
| GET | `/pipeline/deployments/:id` | 期望态 + 每实例 assignment 实际态 |
| PUT | `/pipeline/deployments/:id` | 更新（触发 rollout） |
| POST | `/pipeline/deployments/:id/_rollout` | 手动推进 / 回滚 |
| POST | `/pipeline/deployments/:id/_pause` / `_resume` / `_stop` | 集群级广播（带 generation 防脑裂） |
| POST | `/pipeline/instances/:id/_drain` / `_cordon` / `_uncordon` | 实例排水 / 隔离 / 恢复 |
| GET | `/pipeline/instances/:id/proxy?path=...` | 实例本地 API 直达（单实例调试，flight/inject 等） |
| GET | `/pipeline/cluster/overview` | 管道 × 实例矩阵（谁在跑/版本/状态） |
| GET | `/pipeline/cluster/stages?pipeline_id=` | 跨实例阶段指标聚合 + 明细下钻 |

### 8.3 YAML 等价形式（实例侧最终收到的仍是普通 managed config）

```yaml
# 控制面把 PipelineDeployment 翻译为 N 份标准 pipeline: 配置经 configs-center 下发,
# 实例侧无感知"部署"概念——分布式只在控制面和回集层。
pipeline:
  - name: stream-nginx-parse
    singleton: true          # singleton 策略的落地即现有锁语义
    ...
```

### 8.4 监测与告警约定

- `StageStats`/`StepEvent` 全部携带 `instance_id`；`_sync` 即心跳，超 N 周期未 sync 判离线；
- **锁接管事件 = 故障切换告警**（心跳丢失 + 接管同时发生）；
- trace_id 跨实例串联：控制面按 trace_id 查各实例 flight 拼接全链轨迹。

### 8.5 与配置中心协议的映射（framework `modules/configs`，零替代）

```
PipelineDeployment (意图层,本方案)                      实例侧
   │ reconciler 翻译(placement → 定向)                  ▲
   ▼                                                    │ 本地文件 + fsnotify
ManagedConfig 行 (传输层,现有配置中心) ──/_sync 差分──► PipeModule
   InstanceID | AllInstancesID | Groups 三种定向         │
   Version 递增 / 行删除=实例删本地文件                   │ 运行状态回报(扩展)
   ◄──────────────── _sync: 心跳 upsert + hash diff + {pipeline:{generation,state}} ────┘
```

| Deployment 意图 | 配置中心落地 |
|---|---|
| `placement.strategy: singleton` + selector | 选中实例集写定向行（每实例一行或 Groups），行内 `singleton: true`，锁竞争决定谁跑 |
| `placement.strategy: all` | `AllInstancesID` 一行（或选中集 Groups 一行） |
| `placement.strategy: sharded` | 选中集定向行 + kafka 队列后端配置随行下发 |
| rollout canary→批次 | 先写 1 台定向行 → 确认其 `_sync` 回报 generation 达标 → 批次扩写 |
| 回滚 | 写回旧 Version 行内容，实例 hash diff 自动拉取 |
| 排水后下线 | 先经实例代理 `_drain`（等 offset 提交）→ 删行（实例下次 `_sync` 删本地文件） |
| 删除部署 | 级联删行（复用 orphan 清理模式兜底） |
| 瞬态操控（pause/resume/stop） | **不进配置**——走实例代理 + generation 防脑裂 |

`_sync` 回报扩展为向后兼容增量：`ConfigSyncRequest` 增加可选 `runtime` 字段，旧实例不报 → reconciler 记 `unknown`；selector 标签解析 v1 为静态（部署时解析为定向名单），动态标签选择器留 v2。

---

## 附录：提案 — 统一 `output` 步骤（D-13，待评审定稿，未替换现有内容）

```yaml
- output: { queue: documents_processed }      # 唯一出口步骤;目标=恰好一个目标字段
- output: { lane: doc_sink }
- output: { queue: bulk_x, codec: coco_document }   # 可选 codec 覆盖三级解析
# 未来: - output: { kafka: topic-x }  ← 新目标类型=新字段名,schema 可见
```

**语义矩阵**（位置 × 目标，编译器静态可判）：

| 位置 \ 目标 | `queue: <名>` 字段 | `lane: <名>` 字段 |
|---|---|---|
| batch 作用域 | 消息字节直推（= `queue_output`） | ❌ 编译错误，指向 `consumer.codec+output_lane` 糖 |
| record 作用域 | 编码当前记录入队（三级 codec 解析，= `lane_consumer.output_queue` 规则） | 指针 Emit（= `lane_output`，ack 可选） |

要点：目标用 **one-of 结构化字段**（恰好一个，多写/漏写在 config.Unpack 即报错；字段名进 catalog schema，画布/补全可提示；新目标类型=新字段，扩展对静态检查可见）——弃用打包字符串 `to: type:name`（拼写错误不可枚举、校验推迟、工具黑盒）。写入面（output/tap/_inject）全部结构化；展示面（StageStats carrier 的 `queue:xxx` 展示串）保留字符串（只读、紧凑）。`queue_output`/`lane_output` 保留为别名（存量零破坏）；`lane_consumer.output_queue` 降格为糖（= 子链尾自动追加 output 步骤，显式步骤支持 `if` 条件输出）；入口分立（宿主异构）、出口统一（步骤同构）为刻意设计。

## 附录：Lane 使用场景速查

**边界一句话**：`Emit` 返回成功即生产者责任终点（at-most-once）。一段 lane 流程安全 ⇔ 满足其一：① 上游可重算（重采集/上游重投/可丢弃）；② 上游队列 offset 未提交覆盖全程（崩溃可重放）；③ 本就是 best-effort（调试/采样）。

| 场景 | 形态 | 溢写策略 | 安全依据 |
|---|---|---|---|
| 连接器采集轻预处理（脱敏/类型识别/规整）→ 落盘 | `BatchCollect` Emit(类型化入口) → `lane_consumer` 宿主（`batch` 攒批 + `output_queue` 持久桥，子链跑确定性步骤） | **`block`（默认）**：生产者可安全减速（持久源+任务协程），背压沿 lane→队列→LLM 一路传回 | ① 重采集即恢复 |
| Webhook 实时处理（GitLab 等自带重投） | handler Emit → `lane_consumer` 宿主（`output_queue` 落盘需持久事件） | `block`（反压为 HTTP 429/慢响应，上游重投）或 `drop` | ① 上游重投 |
| tap 观测采样 → 调试索引 | 主链 `tap{sample_ratio: 0.01, lane: debug_samples}` → `lane_consumer` 宿主（`output_queue` 指向调试索引队列） | drop | ③ 观测可丢 |
| 快车道+等价慢车道（spill 唯一站得住的形态） | lane 溢写目标队列的消费链跑**同一条链**——spill 仅表示"此条改走慢的同款路径"，不跳过任何处理步骤 | `spill:queue` | 路径等价 |
| 持久点之后的进程内 fan-out | OTLP: `queue.Push(staging)`+ACK 之后，consumer 链内 `output{lane:…}` → N 个 `lane_consumer` 分发任务 | block | ② offset 未提交覆盖（需配合提交点在最终 durable 写之后） |
| **反例**：跨进程传递、ACK 前的旁路、LLM 富化 | — | — | 违反全部三条 → 必须走队列 |

溢写策略的业务立场：**`block` = 默认**（背压如实传导，生产者减速，数据零风险——生产者是可安全减速的任务协程时永远选它）；`drop` = 可丢弃遥测；`spill:queue` = 仅限**快慢路径处理等价**的快车道形态。**spill 的两笔隐性成本**：① 若快路径包含必需步骤（如脱敏），溢写记录绕过它 = 正确性 bug；② 掩盖背压信号，下游是最终瓶颈时磁盘队列无界上涨直至盘满。生产者不可阻塞且数据需持久的诚实答案是 queue-first（先落盘再 ACK），不是 spill。

**spill 的加载时机与三条推论**：溢出记录**无回灌、永不回到 lane**——落盘后由慢路径 consumer 的常规轮询周期取走、走等价链完成；不存在"lane 空了读回内存"的机制（那是两级缓冲，属另一个组件；需要它的场景直接用磁盘队列）。推论：① **路径终身制**——`TryEmit` 失败即原子分岔，一条记录要么快路径要么慢路径完成，无重复消费；② **崩溃不对称**——溢出记录落盘即持久（重启后续传），lane 内记录崩溃即丢（at-most-once），突发中溢出尾部反而更安全；③ **顺序分叉**——溢出期间双路径并行排空，完成顺序不保证；强顺序需求用 `block`（lane FIFO）。

**回执与提交门（跨 lane 的 commit 正确性）**：两条铁律——`output{lane:…}` 不是提交点；consumer 的 offset 只覆盖自己子链内的 durable 写。三种正确姿势按保证强度选：

| 姿势 | 形态 | 保证 |
|---|---|---|
| durable-first（默认） | 子链内先 `output{queue:…}`（commit 覆盖它），lane 分支只做可选附加加工（tap/fan-out），崩了从队列重放 | 零成本正确 |
| **ack 回执**（端到端） | `output{lane:…, ack: true}` 回执挂入 ctx **AckSet**；consumer 提交前等 AckSet 落定（成功→`output_queue` / 失败→`dead_letter:<queue>` / 显式 drop 均算落定），未决或超 `ack_timeout_ms` 不提交→磁盘队列重放 | **跨 lane at-least-once**：永不丢、可能重复（幂等 sink 消化，ES `_id` 写入天然幂等）；lane 不存数据，保证借自上游 offset |
| 内联 | 需严格"处理成功才提交"时不解耦：`for_each` 直跑，durable 写在子链内 | 最强、最简单 |

**边界无损不变式**：进入 `lane_consumer` 的每条记录必然落定于 `output_queue`（成功）/ `dead_letter`（失败，可修复重放）/ 显式 drop（`on_failure: ignore`，仅 best-effort）三处之一；`on_failure` 默认 `tag`（打标继续走 `output_queue`）。失败不静默：`StepEvent{outcome:error}` 入飞行记录、`errors`/`dead_letters` 计数入 StageStats、trace_id 贯穿。崩溃窗口枚举：发射方崩（未 commit→重放）/ 接收方崩（回执未落定→超时不提交→重放）/ 回执表随进程丢失（全部退化为未落定→重放）——任何一侧崩溃的结局都是重放而非丢失。

**offset 同步细则（queue ↔ lane 的唯一耦合点）**：lane 零状态、从不"同步"——唯一事实源是磁盘队列 offset，规则一条：**回执全部落定才提交，未提交即可重放**。细则：① 每张回执记录来源 offset（slice, offset）；② 提交粒度 v1 = **整批 all-or-nothing**（批内全部落定才 commit 批末 offset；一条卡住整批不提交、整批重放，重复由幂等吸收——磁盘 offset 是高位水位不可跳提交）；可选优化为批内**连续前缀提交**（提交到最后一个连续落定的 offset），留作性能项不进 v1；③ **超时集成复用现有语义**：AckSet 超时 = 子链失败 = consumer 现有失败/重试路径（offset 不推进、按 retry_delay 重拉同批），不新增状态机；④ `on_failure` 三档（tag/dead_letter/ignore）均算落定——配置合理的批每条必然落定、整批照常提交，"卡住"仅发生于下游崩溃；⑤ clone 不带回执（挂原始记录），重放自然重建；⑥ 代价：上游提交等下游落定，吞吐耦合（每 slice 同时一批在途，多 `num_of_slices` 线性放大）——与 durable-first 的取舍点。

## 附录：现有 ↔ 目标对照速查

| 维度 | 现状 | 目标（本文档） |
|---|---|---|
| 链内载体 | coco=`[]queue.Message` / 其他=`*event.Event` | 统一 `*event.Event`（`Fields` 唯一权威） |
| codec | 仅 `otel`，注册 panic | +`coco_document`/`coco_attachment`，error 注册 + 发现 API + Namespace |
| 契约 | 无（错放=静默空转） | `RecordContract` 编译期校验 + 运行期兜底 + catalog 透出 |
| dry-run | 两套独立实现 | framework 统一端点 `POST /pipeline/dry-run`（产品端点删除）+ 共享组件 `DryRunDrawer` |
| AI 生成 | coco studio / logpilot parsing 各一套 | 统一 `POST /pipeline/ai-generate` + `ChainGenerator` 产品注册 |
| 指标 | 3 计数器 + logpilot 私有 | `StepEvent` 三投影 + `StageStats` 标准 |
| 操控 | 仅启停 | +inject/pause/resume/flight/tap |
| 进程内传递 | 无 | `lane`（记录即消息，at-most-once） |
| 配置 | PipelineConfigV2 | +`strict_contract`/`canvas_layout`/`parallel`/`mode`/`lane:` 段 |
