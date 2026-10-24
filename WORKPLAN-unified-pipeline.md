# 统一 Pipeline 工作计划与影响范围清单

| | |
|---|---|
| **配套设计文档** | [DESIGN-unified-pipeline.md](./DESIGN-unified-pipeline.md) |
| **日期** | 2026-10-10 |
| **阶段划分** | P1-P8，与设计文档 §8.2/§11 对应；每阶段独立可交付、可停止 |
| **路径基准** | `framework/` = `infini.sh/framework`，`coco/` = `infini.sh/coco`，`logpilot/` = `infini.sh/logpilot`，`gateway/` = `infini.sh/gateway`（同级目录） |

## 规模总览

| 仓库 | 新增文件 | 修改文件 | 说明 |
|---|---|---|---|
| framework | ~20 | ~13 | 引擎、契约、lane、观测、dryrun 包、统一端点、分布式 deployment/reconciler |
| coco（Go） | ~8 | ~45 | 15 处理器 + 25 连接器 + 1 webhook + 模块；studio 端点删除 |
| **ui-common** | 1 包（~7 组件 + TS client） | 0 | `@infinilabs/pipeline-ui` 共享包（P7 核心） |
| coco（web） | 薄壳若干 | ~4 | 消费共享包 + 页面集成 |
| logpilot（Go） | 0 | ~7 | dry-run/parsing/debug 端点删除 + AI 生成注册 ChainGenerator + 指标改喂 + StreamSpec 对接 deployment |
| logpilot（web） | 0 | ~4 | 切共享组件 + 统一端点（删本地实现） |
| gateway | 0 | **0** | 仅随 framework 重编译 |

阶段依赖：`P1 → P2 → P3 → P8` 为关键路径；`P4 / P5 / P6 / P7` 在 P1（P2）之后可并行；`P9`（分布式）依赖 P6，可与 P7/P8 并行。

## 改动与验收一览（每阶段：改什么 / 怎么算完成）

| 阶段 | 改动要点（新增/修改/删除） | 验收门禁（可测断言） |
|---|---|---|
| P1 | +codec v2 接口与注册、+document_codec/attachment_codec、+documentview、+observer.go 埋点（framework ~6 新文件；coco ~4） | codec 重名返回 error；Document↔Event 往返逐字段等价（含 float32 向量精度、空值、超长数组）；observer 空时零额外分配（benchmark）；存量管道输出逐字节不变（回归） |
| P2 | +contract.go（Scope/Namespace/Granularity）、NewPipelineWithScope、for_each 子链切 scope 编译、catalog 透出、`/pipeline/codecs`、`PipelineConfigV2.strict_contract` | 契约矩阵测试全绿（batch↔record×namespace×嵌套×granularity 六格）；不合规组合编译失败且错误含步序号+双端命名空间；unconstrained 处理器任意位置放行；`strict_contract:false` 时仅告警 |
| P3 | 15 处理器双模改造+WithConfigMetadata+契约声明、26 连接器/webhook 换注册、process_documents `mode:records`、+utils 适配（coco ~45 文件） | 每处理器双模等价测试（同输入两模式输出 JSON 等价）；`GET /pipeline/processors` 41 项全部含 properties+scope+granularity；`coco.yml` 存量管道行为回归不变；catalog 重名 panic 防线冒烟 |
| P4 | +event 池、for_each 池化+parallel、jsoniter 切换、codec 冷热分离、embedding chunks 访问器、indexing_merge 部分解析、+基准套件 | 1MB 文档 K=4 链数据搬运 CPU 降 ≥80%；1KB 事件并行吞吐 ≥3×；`-race` + 池泄漏检测干净；并行模式输出集合等价（乱序允许）；jsoniter 行为对比门禁（大数/HTML 转义/nil） |
| P5 | +core/lane（Lane/Create/Delete/惰性/预算）、+lane 模块、+output.go 统一出口、+AckSet、consumer 桥接糖、+lanes 端点（framework ~10 文件） | 背压三策略/spill 落盘回读/并发 Emit 单测；AckSet：杀下游→超时不提交→重放重连（集成测试）；offset 六细则各一测；`/pipeline/lanes` 指标与压测实测一致；动态 lane 创建/级联删除/孤儿清理；output one-of 校验（0/2 目标 Unpack 报错） |
| P6 | +dryrun 包、统一端点、**删**产品 dry-run/AI 端点（coco 2 个、logpilot 5 个）、+metrics/flight 持久化、+tap、+操控端点 | dry-run 对含 output 的链零真实投递（queue+lane 专项）；被删端点 404；ChainGenerator 未注册 501；`flight?trace_id=` 还原单记录全链；`?window=7d` 查到持久化错误事件；暂停语义=当前批次完成后挂起 |
| P7 | ui-common +Pipeline 包（~7 组件+TS client）、coco/web 集成、logpilot/web 切换删本地实现 | 画布编排混排链（consumer→for_each{grok→extract_entities→embedding}→output{queue}）dry-run 全绿；batch_only 拖入嵌套位置即时红叉（与编译错误同源）；两产品同链交互一致（共享包单源验证） |
| P8 | +pipeline-audit/+pipeline-migrate 工具、**删** 15 处理器 messages 旧路径、默认值翻转（mode:records、strict_contract:true） | audit 工具对全部存量管道给出合规结论；迁移后样本 dry-run 全绿且语义等价；旧式管道编译失败（预期）；strict_contract:true 下不合规管道启动即拒 |
| P9 | +deployment/reconciler/placement、Fleet API、_sync 回报扩展、drain/cordon、锁接管事件、UI ClusterOverview | 杀实例→锁 TTL 内接管+告警事件（集成测试）；rollout canary→批次推进/回滚；drain 后 offset 已提交零丢数；`cluster/stages` 聚合=各实例明细之和 |

---

## P1 地基：Codec v2 + 记录模型 + 观测埋点

**目标**：纯增量，不改任何现有行为。framework 埋观测钩子；coco 具备 `coco_document` codec 与 documentview。

### P1.1 Codec 注册 v2（framework）

| # | 任务 | 文件 | 性质 |
|---|---|---|---|
| 1 | `RecordCodec` 接口加 `Namespace() string`；`otelCodec` 返回 `""` | `framework/modules/pipeline/for_each_codec.go` | 修改 |
| 2 | `RegisterRecordCodec` 返回 `error`（保留 panic 版本供 init 期用）；新增 `RegisterRecordCodecOrReplace`、`ListRecordCodecs`、`CodecInfo` | 同上 | 修改 |
| 3 | 接口变更兼容性检查：全仓 grep `RecordCodec` 实现者（当前仅 framework 内 otel，无外部实现） | 全仓 | 验证 |

**验收**：单测覆盖重名 error、replace 版本递增、List 完整性；现有 for_each 行为不变（回归）。

### P1.2 `coco_document` codec（coco）

| # | 任务 | 文件 | 性质 |
|---|---|---|---|
| 4 | `DocumentCodec`：Name/Namespace/Decode/Encode，整树投影 `document.*`，元数据进 Labels——落位**独立轻量 Go module `infini.sh/coco-codecs`**（D-2 已决：中立模块，依赖面=framework+领域类型，不引 coco 的 llm/common；gateway/agent 未来可 blank-import 获得远程执行能力） | `coco-codecs/document_codec.go`（新 module） | 新增 |
| 5 | init 注册 + blank import 接线 | `coco/modules/pipeline/init.go` | 修改 |
| 6 | `AttachmentCodec`（`coco_attachment`，namespace `attachment`） | `coco-codecs/attachment_codec.go` | 新增 |
| 7 | Roundtrip 无损测试：含 embedding 精度（float32 经 mapstr）、全字段、空文档 | `coco-codecs/document_codec_test.go` | 新增 |

### P1.3 documentview 助手（coco）

| # | 任务 | 文件 | 性质 |
|---|---|---|---|
| 8 | `Materialize(rec) (*core.Document, error)` / `Project(doc, rec)` | `coco/core/documentview/view.go` | 新增 |
| 9 | 往返等价测试 + `document.content` 等点路径读写测试 | `coco/core/documentview/view_test.go` | 新增 |

### P1.4 ExecutionObserver 埋点（framework）

| # | 任务 | 文件 | 性质 |
|---|---|---|---|
| 10 | `StepEvent`/`ExecutionObserver`/`Outcome`/`Snapshot` 类型 + observer 注册/存储 | `framework/core/pipeline/observer.go` | 新增 |
| 11 | `Processors` 持 observers；`Process` 循环埋点（observers 空时 fast path） | `framework/core/pipeline/processor.go` | 修改 |
| 12 | `for_each` 逐条循环埋点（含 record 作用域信息） | `framework/modules/pipeline/for_each.go` | 修改 |
| 13 | trace_id 助手（生成/写入 `Event.Metadata.Labels["_trace_id"]`/读取） | `framework/core/pipeline/record.go` | 修改 |

**验收**：observer 空 + 非空两路径基准测试（空路径零额外分配）；埋点后现有管道输出逐字节不变。

---

## P2 契约与作用域校验

**目标**：编译期能拒绝"放错作用域/命名空间"的处理器组合，catalog 透出契约，默认不启用（`strict_contract` 开关，默认 false）。

### P2.1 契约类型与编译器（framework）

| # | 任务 | 文件 | 性质 |
|---|---|---|---|
| 14 | `Scope`/`ScopeInfo`/`RecordContract`/`ContractProvider`/`CheckContract` | `framework/core/pipeline/contract.go` | 新增 |
| 15 | `NewPipelineWithScope(cfg, ScopeInfo)`：作用域跟踪 + 契约校验 + 精确错误信息；`NewPipeline` 保留原签名（默认 batch、不校验 unconstrained） | `framework/core/pipeline/processor.go` | 修改 |
| 16 | 嵌套作用域：内层 for_each 覆盖外层（与 `RecordContextKey` save/restore 一致） | 同上 + `framework/modules/pipeline/for_each.go` | 修改 |
| 17 | for_each 子链编译切换到 `NewPipelineWithScope` | `framework/modules/pipeline/for_each.go` | 修改 |
| 18 | `PipelineConfigV2` 加 `StrictContract *bool` | `framework/core/pipeline/config.go` | 修改 |
| 18a | 契约新增 `Granularity` 三档（record/batch_capable/batch_only）+ **段式执行（stage-wise）executor**：for_each/lane_consumer 子链按书写序切交替段（逐条段/批段），顺序即书写顺序、批处理器可见前面逐条处理器的修改、drop 排除/批段失败全批打标/clone 流入后续段——**废弃现行两阶段行为**（BatchProcessor 全生态零实现，零风险窗口）+ 放置校验（batch_only 嵌套位置编译错误） | `framework/core/pipeline/contract.go`、`framework/modules/pipeline/for_each.go`（executor 重构）、`lane_consumer.go` | 修改 |
| 19 | 契约编译测试矩阵：batch↔record 错配、namespace 错配、嵌套、unconstrained 放行、strict 开关、**granularity 放置矩阵 + 段式执行语义测试（书写序=执行序、批段可见前段修改、drop/clone 跨段传播、批段失败打标）** | `framework/core/pipeline/contract_test.go`、`framework/modules/pipeline/for_each_stage_test.go` | 新增 |

### P2.2 catalog 与发现 API（framework）

| # | 任务 | 文件 | 性质 |
|---|---|---|---|
| 20 | 注册表记录处理器契约（构造时探测 `ContractProvider`）；`GetProcessorCatalog()` 每项输出 `scope`/`record_contract`（未声明 → `unconstrained`） | `framework/core/pipeline/register.go` | 修改 |
| 21 | catalog API 序列化新字段 | `framework/modules/pipeline/processors_api.go` | 修改 |
| 22 | `GET /pipeline/codecs` 端点（ListRecordCodecs） | `framework/modules/pipeline/module.go` | 修改 |
| 23 | catalog 响应结构测试 | `framework/modules/pipeline/processors_api_test.go` | 修改/新增 |

**验收**：不合规组合在编译期得到带步序号/处理器名/双端描述的错误；存量管道全部编译通过（CI 全量管道配置回归）。

---

## P3 Coco 处理器双模迁移

**目标**：每个处理器同时支持旧 messages 模式与记录模式；`process_documents` 支持 `mode: records`。存量行为零变化。

### P3.1 处理器改造（15 个注册点，13 个目录）

统一改造模式（每个文件三件事：注册换 `RegisterProcessorPluginWithConfigMetadata`、实现 `RecordContract()`、`Process` 保留旧路径 + 新增 `ProcessBatch` 记录路径）：

| # | 处理器（注册名） | 文件 | 行 |
|---|---|---|---|
| 24 | `process_documents`（mode 开关 + 链宿主化，见 P3.2） | `coco/plugins/processors/process_documents/processor.go` | :49 |
| 25 | `document_embedding`（ProcessBatch 批量，LLM 批量化） | `coco/plugins/processors/embedding/processor.go` | :41 |
| 26 | `document_summarization` | `coco/plugins/processors/summary/processor.go` | :60 |
| 27 | `extract_entities` | `coco/plugins/processors/extract_entities/processor.go` | :73 |
| 28 | `extract_tags` | `coco/plugins/processors/extract_tags/processor.go` | :55 |
| 29 | `file_metadata` | `coco/plugins/processors/file_metadata/processor.go` | :36 |
| 30 | `file_type_detection` | `coco/plugins/processors/file_type_detection/processor.go` | :40 |
| 31 | `face_extraction` | `coco/plugins/processors/face_extraction/processor.go` | :36 |
| 32 | `process_attachments` | `coco/plugins/processors/process_attachments/processor.go` | :31 |
| 33 | `generate_document_cover` | `coco/plugins/processors/generate_cover/document_processor.go` | :39 |
| 34 | `generate_attachment_cover` | `coco/plugins/processors/generate_cover/attachment_processor.go` | :31 |
| 35 | `document_text_attachment_extraction` | `coco/plugins/processors/text_attachment_extraction/document_processor.go` | :39 |
| 36 | `attachment_text_extraction` | `coco/plugins/processors/text_attachment_extraction/attachment_processor.go` | :35 |
| 37 | `connector_dispatcher` | `coco/plugins/processors/dispatcher/dispatcher.go` | :37 |
| 38 | 共享助手抽取（messages↔records 适配、双模工具） | `coco/plugins/processors/utils.go` | 修改 |

每处理器配套双模一致性测试（同输入，两种模式输出 JSON 等价）：`coco/plugins/processors/<name>/processor_dualmode_test.go`（新增 ×13）。

### P3.2 `process_documents` 链宿主化

| # | 任务 | 文件 | 性质 |
|---|---|---|---|
| 39 | `mode: records` 开关：门禁读热字段；解码一次→记录作用域→子链→编码一次（复用 P1 codec） | `coco/plugins/processors/process_documents/processor.go` | 修改 |
| 40 | `applyProcessingOutcome`/chunk 镜像等出口逻辑适配记录模式 | 同上 + 同目录辅助文件 | 修改 |

### P3.3 连接器与 webhook 元数据补齐（26 个注册点）

契约声明：连接器/webhook 处理器为 **batch 作用域**（链根驱动）。仅换注册函数 + 声明契约，不改逻辑：

| # | 连接器 | 文件 | 行 |
|---|---|---|---|
| 41 | neo4j | `coco/plugins/connectors/neo4j/plugin.go` | :31 |
| 42 | postgresql | `coco/plugins/connectors/postgresql/plugin.go` | :23 |
| 43 | s3 | `coco/plugins/connectors/s3/plugin.go` | :32 |
| 44 | box | `coco/plugins/connectors/box/plugin.go` | :44 |
| 45 | gitlab | `coco/plugins/connectors/gitlab/plugin.go` | :22 |
| 46 | hugo_site | `coco/plugins/connectors/hugo_site/processor.go` | :24 |
| 47 | local_fs | `coco/plugins/connectors/local_fs/plugin.go` | :38 |
| 48 | feishu（ConnectorFeishu） | `coco/plugins/connectors/feishu/plugin.go` | :164 |
| 49 | feishu（ConnectorLark） | 同上 | :165 |
| 50 | oracle | `coco/plugins/connectors/oracle/plugin.go` | :25 |
| 51 | rss | `coco/plugins/connectors/rss/plugin.go` | :104 |
| 52 | mongodb | `coco/plugins/connectors/mongodb/plugin.go` | :19 |
| 53 | notion | `coco/plugins/connectors/notion/plugin.go` | :33 |
| 54 | gitea | `coco/plugins/connectors/gitea/plugin.go` | :26 |
| 55 | github | `coco/plugins/connectors/github/plugin.go` | :23 |
| 56 | jira | `coco/plugins/connectors/jira/plugin.go` | :25 |
| 57 | mysql | `coco/plugins/connectors/mysql/plugin.go` | :23 |
| 58 | confluence | `coco/plugins/connectors/confluence/plugin.go` | :43 |
| 59 | network_drive | `coco/plugins/connectors/network_drive/plugin.go` | :51 |
| 60 | dropbox | `coco/plugins/connectors/dropbox/plugin.go` | :41 |
| 61 | salesforce | `coco/plugins/connectors/salesforce/plugin.go` | :31 |
| 62 | yuque | `coco/plugins/connectors/yuque/plugin.go` | :42 |
| 63 | mssql | `coco/plugins/connectors/mssql/plugin.go` | :23 |
| 64 | milvus | `coco/plugins/connectors/milvus/plugin.go` | :22 |
| 65 | google_drive | `coco/plugins/connectors/google_drive/processor.go` | :27 |
| 66 | gitlab webhook | `coco/plugins/webhooks/gitlab/incoming/incoming_event_processor.go` | :65 |

连接器基类（`Init`/`GetBasicInfo`/`BatchCollect`）不动：`coco/plugins/connectors/common/processor.go`（P5 才改类型化入口）。

**验收**：`GET /pipeline/processors`（coco 实例）返回全部 41 项含 properties + scope；catalog 冒烟测试防重名；双模测试全绿；`coco.yml` 现有管道（messages 模式）行为回归不变。

---

## P4 性能：池化 / 冷热分离 / 并行 / 部分解析

**目标**：记录模式链数据搬运 CPU 降 ≥80%（1MB 文档基准）；`for_each` CPU 型链并行吞吐 ≥3×。

| # | 任务 | 文件 | 性质 |
|---|---|---|---|
| 67 | `AcquireEvent/ReleaseEvent` 池（Fields clear 复用） | `framework/core/event/pool.go` | 新增 |
| 68 | for_each 解码 acquire / 编码后 release 接线 | `framework/modules/pipeline/for_each.go` | 修改 |
| 69 | for_each 记录级并行：worker 池 + per-goroutine context 克隆 + `parallel` 配置项（**默认 false**，待评审 D-4） | 同上 | 修改 |
| 70 | jsoniter **仅热路径局部替换**（D-6 已决）：codec Encode/Decode、documentview、dryrun 快照三处直用 jsoniter；全局 `util.ToJSONBytes` 保持 stdlib；行为对比门禁（大数/HTML 转义/map 序/nil）只对热路径跑 | `framework/modules/pipeline/for_each_codec.go`、`coco-codecs/*`、`framework/modules/pipeline/dryrun/*` | 修改 |
| 71 | 冷热分离：`document_chunk` 子树以 `json.RawMessage` 寄生 Decode/Encode |  `coco-codecs/document_codec.go` | 修改 |
| 72 | embedding 处理器：chunks 子树 typed 访问器（唯一读者/写者，只重编该子树） | `coco/plugins/processors/embedding/processor.go` | 修改 |
| 73 | process_documents 门禁读热字段（不经冷子树） | `coco/plugins/processors/process_documents/processor.go` | 修改 |
| 74 | `indexing_merge` `_id` 提取改零分配部分解析 | `framework/plugins/elastic/indexing_merge/indexing_merge.go`（:275 附近，另 :381/:447 同类） | 修改 |
| 75 | 基准套件：1MB 文档 K=4 链 messages vs records；1KB 事件并行扩展性；池化前后分配 profile | `framework/modules/pipeline/*_bench_test.go`、`coco/modules/pipeline/*_bench_test.go` | 新增 |

**验收**：§9 基准达标；池化无 double-release/泄漏（`-race` + 泄漏检测）；并行模式输出集合等价（乱序允许）。

---

## P5 Lane 通道

**目标**：进程内零序列化通道上线；连接器进程内富化路径零解码进入链。

| # | 任务 | 文件 | 性质 |
|---|---|---|---|
| 76 | `Lane` 接口 / `LaneConfig` / 注册表 | `framework/core/lane/lane.go` | 新增 |
| 77 | 默认实现：有界 chan + 背压三策略（block/drop/spill:\<queue\>） | `framework/core/lane/lane.go` | 新增 |
| 78 | lane 生命周期**并入 PipeModule**（D-8 已决，不设独立模块）：配置加载/热重载/关闭 drain-or-drop/Depth 指标+ **动态平面**：运行期 `Create/Delete/GetOrCreate`、`owner` 所有权级联清理（流删除→级联，孤儿检测复用 configs-center 模式）、惰性分配（可增长有界队列，空闲 lane≈0）、动态 lane 小默认容量（256~512）、全局预算 `total_budget` | `framework/modules/lane/module.go`、`framework/core/lane/lane.go` | 新增 |
| 78b | lane 观测：标准指标采集（in/out 计数与速率、drops/spills/emit_blocks/pending_acks/ack_timeouts 计数器）+ `GET /pipeline/lanes[/name]` 端点（清单含 owner/consumer 任务/depth_pct） | `framework/modules/lane/module.go`（指标）、`module.go`（路由） | 新增 |
| 78a | 动态管道编译器对接：CompileStream 按需生成 per-stream lane（`stream-<id>-fast`，一条管道一条 lane 绝不共享）+ 生命周期时序（停上游→停下游→删 lane）；spec 级 opt-in（`processing.in_process`） | `logpilot/plugins/logpilot/stream_compile.go` | 修改 |
| 79 | `lane_consumer` 处理器（**链宿主**：订阅 lane 攒批 `batch{max,flush_ms}` → 段式执行（stage-wise），顺序即书写顺序，执行契约与 for_each 一致；`output_queue` 持久桥=幸存记录编码入队；按 LaneConfig.Namespace 契约编译） | `framework/modules/lane/lane_consumer.go` | 新增 |
| 80 | 统一 `output` 处理器（one-of 目标字段；lane 目标 Emit 后即返回本链继续、`ack: true` 时回执挂入 ctx **AckSet**；queue 目标按作用域直推/编码入队；`queue_output`/`lane_output` 注册为兼容别名，**文档与编排 UI 一律产出 `output` 形式**）+ consumer AckSet 提交门（等回执落定才 commit，未决/超时（`ack_timeout_ms`，consumer 侧配置）不提交→重放）+ lane 回执 resolve（**记录处理完成即放行**——配置 `output_queue`/`dead_letter` 时完成即已落盘；未配置时链尾即完成，操作者保证链尾效果） | `framework/modules/pipeline/output.go`、`framework/core/pipeline/ackset.go`、`framework/plugins/queue/consumer/consumer.go` | 新增+修改 |
| 81 | ~~main.go 接线~~ **无需接线**（D-8 已决：并入 PipeModule，三产品已注册该模块，lane 能力随重编译自动生效） | — | 取消 |
| 81a | `consumer` 桥接糖：`codec` + `output_lane` 配置（fetch→decode→emit 一块完成，磁盘→内存最常见拓扑免三明治写法；offset 由 AckSet 门控） | `framework/plugins/queue/consumer/consumer.go` | 修改 |
| 82 | codec 类型化入口变体：`DecodeFromDocument(*core.Document) *event.Event` |  `coco-codecs/document_codec.go` | 修改 |
| 83 | `BatchCollect` 进程内路径：EnrichmentPipeline 记录模式 + 类型化入口 | `coco/plugins/connectors/common/processor.go`（:102-112 一带） | 修改 |
| 84 | lane 单测：背压三策略/spill 落盘回读/关闭语义/并发 Emit | `framework/core/lane/*_test.go` | 新增 |

**验收**：spill 洪峰测试（lane 满 → 写队列 → 消费回读完整）；崩溃注入下 at-most-once 语义符合文档；选型判据写入包注释。

---

## P6 可观测与在线调试

**目标**：dry-run 合一（修副作用隐患）、标准指标、飞行记录、注入/暂停 API。

### P6.1 framework dryrun 包

| # | 任务 | 文件 | 性质 |
|---|---|---|---|
| 85 | 包骨架：`Sample`/`Options`/`Report`/`StepResult`/`CapturedWrite` | `framework/modules/pipeline/dryrun/dryrun.go` | 新增 |
| 86 | 执行：全量捕获 Observer 安装 + 逐步回放（复用 P1 埋点） | `framework/modules/pipeline/dryrun/run.go` | 新增 |
| 87 | 副作用截获：dry-run 上下文中 queue/lane 写入转 capture | `framework/modules/pipeline/dryrun/sideeffect.go` + `framework/modules/pipeline/output.go` | 新增+修改 |
| 88 | 字段级 Diff（added/removed/changed 路径） | `framework/modules/pipeline/dryrun/diff.go` | 新增 |
| 89 | LLM mock：录制/回放策略 | `framework/modules/pipeline/dryrun/llmpolicy.go`（存储方案待评审 D-5） | 新增 |
| 90 | dryrun 单测（含契约校验联动） | `framework/modules/pipeline/dryrun/*_test.go` | 新增 |

### P6.2 统一端点与产品端点删除（不保留 wrapper）

| # | 任务 | 文件 | 性质 |
|---|---|---|---|
| 91 | framework 挂统一端点 `POST /pipeline/dry-run`、`POST /pipeline/ai-generate`（含 `ChainGenerator` 注册接口） | `framework/modules/pipeline/module.go`（路由）+ `dryrun/api.go`、`ai_generate.go` | 修改+新增 |
| 92 | Coco 端点删除：`/pipeline/studio/test`、`/pipeline/studio/ai-generate`；`ai_generate.go` 逻辑改造为 `ChainGenerator` 实现并 init 注册（`modules/llm` 后端）；`dryrun.go` 删除 | `coco/modules/pipeline/{init,dryrun,ai_generate}.go` | 删减+改造 |
| 93 | Logpilot 端点删除：`streams/:id/dry-run`、`parsing/test`、`parsing/ai-generate`、`debug/_inject`、`debug/chain`；AI 生成逻辑注册为 `ChainGenerator` | `logpilot/plugins/logpilot/{api_dryrun,api_parsing,api_debug}.go` | 删减+改造 |
| 94 | 前端同步切换（与 P7.3 同 PR）：两产品调试 UI 改调统一端点 | 见 P7 | 联动 |

**验收**：统一端点为 dry-run/AI 生成唯一定义；上表产品端点 404；`ChainGenerator` 未注册时 ai-generate 返回 501。

### P6.3 指标与操控

| # | 任务 | 文件 | 性质 |
|---|---|---|---|
| 95 | `StageStats` 标准模型 + 聚合 Observer（queue 深度/lag + lane Depth + 处理器六指标） | `framework/modules/pipeline/metrics.go` | 新增 |
| 96 | Logpilot 指标改喂标准模型（ES 存储与看板不动） | `logpilot/plugins/logpilot/pipeline_metrics.go`、`model_pipeline.go` | 修改 |
| 97 | 飞行记录器：环形缓冲采样 Observer（1/N + 错误全采） | `framework/modules/pipeline/flight.go` | 新增 |
| 98 | 端点：`_inject` / `_pause` / `_resume` / `flight` / `stages` | `framework/modules/pipeline/module.go`（路由）+ `tasks.go` | 修改 |
| 99 | consumer 入口生成 trace_id（不存在时） | `framework/plugins/queue/consumer/consumer.go` | 修改 |
| 100 | `tap` 处理器（采样分流到调试队列/lane） | `framework/modules/pipeline/tap.go` | 新增 |
| 101 | `pipelinelog` 助手（自动携带 trace_id） | `framework/core/pipeline/log.go` | 新增 |
| 101a | **指标持久化导出器**（关闭 D-7）：`pipeline_metrics.persistence{enabled,interval,retention}` 周期快照 StageStats（含 lane carrier、instance_id）→ ORM 索引 `pipeline_stage_stats` + `POST /pipeline/metrics/_search` + retention 清理 | `framework/modules/pipeline/metrics_persist.go` | 新增 |
| 101b | **飞行记录持久化**：错误 StepEvent 全量落 `pipeline_flight_errors` 索引；全量追踪 opt-in 时间窗（`trace_persist: 30m` 到点自动关）；flight 端点 `?window=` 跨界查询 | `framework/modules/pipeline/flight.go` 扩展 | 修改 |

**验收**：dry-run 对含 `output_queue` 的链不再真实投递（修复 §3.6 隐患，专项测试）；`flight?trace_id=` 能还原单记录全链轨迹；暂停语义为"当前批次完成后挂起"（测试）。

---

## P7 UI：共享组件包与两产品集成

**前置**：P3.3 元数据已补齐（否则 schema 表单无数据）；P6.2 统一端点已就绪。
**原则**：编辑与调试交互只有一份实现（共享包），产品做数据绑定与外壳。

### P7.1 共享组件包（ui-common）

| # | 任务 | 文件 | 性质 |
|---|---|---|---|
| 102 | 新建 `packages/Pipeline`（`@infinilabs/pipeline-ui`）：`PipelineCanvas`（React Flow：节点/连线/自动布局/if 与 for_each 容器/when 徽章/JSON 双视图）+ `ProcessorPalette`（catalog 驱动、契约过滤、置灰+原因）——源代码从 `logpilot/web/src/modules/streams/processing-canvas.tsx` 迁移重构 | `ui-common/packages/Pipeline/` | 新增 |
| 103 | `SchemaForm`（catalog properties 驱动，源自 `logpilot/web/src/components/schema-form.tsx`） | 同上 | 新增 |
| 104 | `DryRunDrawer`（渲染统一 `Report`：快照/diff/副作用）+ `AiGeneratePanel`（生成后自动 dry-run 校验）+ `FlightTimeline` + `StageStatsPanel` + `InjectDialog` | 同上 | 新增 |
| 105 | `pipeline-api` TS client（统一端点类型化封装，`Report`/`StepResult`/`StageStats` 类型随包发布） | 同上 | 新增 |
| 106 | 数据绑定抽象：链读写/布局存储/样本来源以 props 注入（产品适配层接口定义） | 同上 | 含于 102-105 |
| 107 | 共享包接入方式：**源码引用**（D-9 已决）——coco/web 与 logpilot/web 以 `file:`/workspace 协议指向 `ui-common/packages/Pipeline`，不发 npm；后续有跨团队消费需求再启发版 | `coco/web/package.json`、`logpilot/web/package.json` | 配置 |

### P7.2 Coco 集成

| # | 任务 | 文件 | 性质 |
|---|---|---|---|
| 108 | 引入 `@infinilabs/pipeline-ui`；pipeline edit/new 页接画布（保留表单模式切换）；链读写绑定 `PipelineConfigV2` CRUD | `coco/web/src/pages/pipeline/{edit,new}/`、`components/` | 修改 |
| 109 | 画布布局存储（`canvas_layout` 字段读写） | `framework/core/pipeline/config.go`（字段）+ orm 映射 + `coco/web` | 修改 |
| 110 | 调试抽屉挂载（DryRunDrawer/AiGeneratePanel/FlightTimeline/StageStatsPanel/InjectDialog，绑定统一端点） | `coco/web/src/pages/pipeline/components/`（薄壳） | 新增（薄壳） |

### P7.3 Logpilot 迁移

| # | 任务 | 文件 | 性质 |
|---|---|---|---|
| 111 | `spec-editor`/`parsing-studio`/`debug-panel` 切换共享组件与统一端点（删除本地 canvas/schema-form/dry-run 实现，链读写仍绑定 `LogStream.Spec`） | `logpilot/web/src/modules/streams/{spec-editor,parsing-studio,debug-panel}.tsx`、`web/src/components/{schema-form.tsx}` | 修改+删减 |
| 112 | 验收：两产品同一链的编排/dry-run/调试交互一致（共享包单源） | — | 验收 |

**验收**：画布可编排"consumer → for_each(codec: coco_document){ grok → extract_entities → embedding } → output{queue:…}"并 dry-run 全绿；不兼容节点在错误作用域内置灰。

---

## P8 收尾：迁移与删旧

**前置**：P3-P7 稳定运行至少一个发布周期。

| # | 任务 | 文件 | 性质 |
|---|---|---|---|
| 113 | 存量管道扫描工具（列出不合规组合，输出 P8 计划） | `coco/cmd/pipeline-audit/` | 新增 |
| 114 | 一次性迁移工具：`X:{...}` → `for_each{codec: coco_document, processor:[X]}` | `coco/cmd/pipeline-migrate/` | 新增 |
| 115 | 删除 15 个处理器的 messages 旧路径（P3 双模的旧分支） | P3.1 全部文件 | 修改 |
| 116 | `process_documents` 默认 `mode: records` | `coco/plugins/processors/process_documents/processor.go` | 修改 |
| 117 | `strict_contract` 默认 true——**一次切换**（D-1 已决）：audit 扫描→迁移→单次翻转，不分批；changelog 明示行为变化 | `framework/core/pipeline/config.go` | 修改 |
| 118 | 文档：术语表入 framework README、选型判据（queue vs lane）、`document.*` 路径契约冻结公告 | `framework/`、`coco/docs/` | 新增/修改 |

**验收**：`pipeline-audit` 列出全部存量管道的合规结论（合规/需迁移/不合规+原因）；迁移工具对样本管道执行后 dry-run 全绿且语义等价（双模等价测试复跑通过）；全部 Coco 处理器 messages 路径删除后 `coco.yml` 旧式管道样例编译失败（预期）；`strict_contract: true` 下不合规管道启动即拒绝；文档三件套齐备。

---

## P9 分布式调度与治理（多 Gateway）

**前置**：P6 指标/操控已就绪；configs-center 通道可用（Logpilot 已有）。
**原则**：执行引擎不动；分布式能力 = 配置分发端（placement → N 份 managed config）+ 状态回集端（`_sync` 扩展 + 集群聚合）。控制面由产品承载（coco/logpilot/console），framework 提供原语与端点。

### P9.1 协调原语（framework）

| # | 任务 | 文件 | 性质 |
|---|---|---|---|
| 119 | 实例 `_sync` 回报扩展：运行中管道 `{generation, state, last_error}`（心跳即同步） | `framework/modules/pipeline/module.go`（状态上报钩子）+ logpilot configs-center 对接 | 修改 |
| 120 | 广播操控带 generation（`_pause/_resume/_stop` 防脑裂：旧版本实例忽略新指令） | `framework/modules/pipeline/tasks.go` | 修改 |
| 121 | `_drain`（处理完当前批 + 提交 offset 后停）/ `_cordon` 实例端点 | 同上 | 修改 |
| 122 | 锁接管事件（lock takeover → StepEvent/告警通道） | `framework/modules/pipeline/module.go`（singleton 分支） | 修改 |

### P9.2 控制面：Deployment 与调和（framework 原语 + 产品承载）

| # | 任务 | 文件 | 性质 |
|---|---|---|---|
| 123 | `PipelineDeployment`/`Placement`/`Assignment` 模型（ORM 对象） | `framework/modules/pipeline/deployment.go` | 新增 |
| 124 | 调和循环（desired vs reported，幂等重试；调度器自身 locker 单例）——**framework 全包控制面**（D-11 已决）：deployment/reconciler/Fleet API 全部实现在 framework，产品只做 UI 与实例注册对接 | `framework/modules/pipeline/reconciler.go` | 新增 |
| 125 | Fleet API：deployments CRUD/`_rollout`/集群广播、instances drain/cordon/proxy、cluster overview/stages | `framework/modules/pipeline/deployment_api.go`、`module.go`（路由） | 新增 |
| 126 | placement→managed config 翻译器——**v1 仅 singleton/all**（D-12 已决：生产尚无 kafka 队列后端，sharded 保留设计、任务冻结至 kafka 引入后） | `framework/modules/pipeline/placement.go` | 新增 |
| 127 | StageStats/StepEvent 携带 instance_id（通用化 Logpilot 已有字段） | `framework/modules/pipeline/metrics.go`、`flight.go` | 修改 |

### P9.3 集成与验收

| # | 任务 | 文件 | 性质 |
|---|---|---|---|
| 128 | Logpilot：StreamSpec.Placement 对接 framework PipelineDeployment（流发布走 deployment API 而非直推 config；产品零自研调度） | `logpilot/plugins/logpilot/stream_compile.go`、`api_configs_center.go` | 修改 |
| 129 | UI：cluster overview 矩阵/rollout 进度入共享包（`ClusterOverview` 组件，SPEC §8） | `ui-common/packages/Pipeline/` | 新增 |
| 130 | 验收：杀实例 → 锁 TTL 内接管且产生告警事件；rollout canary→批次推进/回滚；drain 后 offset 已提交零丢数 | 多实例集成测试 | 验收 |

**验收补充**：singleton 接管时间 ≤ 锁 TTL（`max_running_in_ms`）；`cluster/stages` 聚合值 = 各实例明细之和。

### P9.4 热生效补齐（现状缺口，独立于 D7 也可先行）

> 现状核实：`updatePipelineHandler` 仅落库（`WaitForRefresh`），不触碰运行中任务；`startTask` 只恢复既有内存 Context（`Restart/Resume`），不重读配置。运行中任务的配置变更仅文件通道（fsnotify → 重建任务）可热生效。

| # | 任务 | 文件 | 性质 |
|---|---|---|---|
| 131 | ORM 管道更新热生效：update/delete handler 检测到"运行中任务对应的配置变化"→ 触发任务重建（复用 fsnotify 同款重建路径，drain 当前批后换编译产物） | `framework/modules/pipeline/pipeline.go`（update/delete handler）+ `module.go`（重建钩子） | 修改 |
| 132 | `process_documents` 编译缓存：按 ORM 文档 version/updated 做版本感知缓存，未变化复用编译产物（消除逐消息 GetV2+NewPipeline） | `coco/plugins/processors/process_documents/processor.go` | 修改 |

**验收**：PUT `/pipelines/:id` 后运行中任务在当前批次完成后切换新链（generation 递增可见）；缓存命中时零 orm.GetV2/零 NewPipeline 调用（单测+计数器）。

---

## Gateway 影响说明

**零代码改动**。`gateway/main.go:49-89` 已具备全部前提（framework plugins + enterprise 导入 + PipeModule 注册）。P5 任务 81 涉及 gateway/main.go 一行模块注册（若 lane 采用独立模块方案）。升级方式：随 framework 重编译发布，回归 `gateway/config/stream-*-pipelines.yml` 所列管道。

## 配置文件影响

| 文件 | 影响 |
|---|---|
| `coco/coco.yml`（:176-250 `pipeline:` 段） | P8 前不动；P8 后示例更新为 record 模式 |
| `logpilot/logpilot*.yml`（`pipeline.enabled: false`） | 不动（动态管道管理不变） |
| `gateway/config/stream-*.yml` | 不动（managed-configs 推送格式不变） |

## 测试与门禁总表

| 类别 | 内容 | 阶段 |
|---|---|---|
| 回归 | 全部存量管道配置编译+运行不变（三产品 CI 各自管道样例） | P1-P3 每阶段 |
| 契约矩阵 | batch/record × namespace × 嵌套 × strict 开关 | P2 |
| 双模等价 | 13 处理器同输入两模式输出 JSON 等价 | P3 |
| Roundtrip | Document↔Event 无损（含 float32 精度、空值、超长数组） | P1 |
| 性能基准 | §9 两项门槛 + jsoniter 行为对比 | P4 |
| 并发 | lane 并发 Emit、池化 `-race`、并行记录等价 | P4/P5 |
| 副作用 | dry-run 零真实投递（queue + lane） | P6 |
| 冒烟 | catalog 完整性/重名 panic 防线/41 注册点全覆盖 | P3/P7 |

## 评审待决项（影响任务排期）

| 编号 | 决策 | 关联任务 |
|---|---|---|
| D-1 | ~~strict_contract 收紧节奏~~ **已决：一次切换**（audit→迁移→单次翻转） | #117 |
| D-2 | ~~codec 归属位置~~ ~~独立轻量 module `infini.sh/coco-codecs`~~ **已反转：收编回 coco `core/codecs` 包**（远程执行未落地，独立 module 过早，见 DESIGN 决策日志 #15） | #4/#6 |
| D-3 | `document.*` 路径命名冻结规则 | #4/#118 |
| D-4 | `for_each.parallel` 默认值 | #69 |
| D-5 | LLM mock 录制存储与脱敏规则 | #89 |
| D-6 | ~~jsoniter 范围~~ **已决：仅热路径局部替换**（codec/documentview/dryrun；全局 util 保持 stdlib） | #70 |
| D-7 | ~~StageStats 是否内置持久化~~ **已决（任务 101a）：framework 内置可选持久化**（导出器 + `_search` 端点 + retention；一次实现三产品受益） | #95/#101a |
| D-8 | ~~lane 模块归属~~ **已决：并入 PipeModule**（无需三产品接线） | #78/#81 |
| D-9 | ~~共享包发布方式~~ **已决：源码引用**（file:/workspace 协议，不发 npm） | #102/#107 |
| D-10 | 性能基准环境与数字确认 | #75 |
| D-11 | ~~控制面承载方~~ **已决：framework 全包**（产品只做 UI/实例对接） | #124/#128 |
| D-12 | ~~sharded 进 v1~~ **已决：暂不支持**（生产无 kafka 后端；设计保留，任务冻结） | #126 |
| D-13 | **统一 `output` 步骤（已采纳）**：`output{queue: <名> / lane: <名>}` 统一出口，目标用 **one-of 结构化字段**（恰好一个，Unpack 即校验；弃用打包字符串 `to: type:name`——拼写错误不可枚举、工具黑盒）；语义矩阵（batch 域：queue=字节直推/lane=编译错误指向 consumer 糖；record 域：queue=三级 codec 编码入队/lane=指针 Emit）；`queue_output`/`lane_output` 保留别名零破坏；`lane_consumer.output_queue` 降格为糖（= 子链尾自动追加 output 步骤）；**入口分立出口统一**（宿主复杂异构 vs 步骤同构）；未来 `kafka:`/`http:` 目标字段即新目标类型,零改命名。**已采纳（2026-10-10）**：别名保留注册但不入文档，编排 UI/模板一律产出 `output`。详见 SPEC 提案节 | #80/#81a |
| D-10 | 性能基准环境与数字确认 | #75 |
