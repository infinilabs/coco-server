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

## 进度记录 2026-10-24:P8.113 pipeline-audit 落地 + 两个契约执行 framework 修复

**framework `866b35e9`** — strict 契约此前对真实产品管道是空转,两个静默缺口:
1. `validatePlacement` 只认实例 `ContractProvider` 方法,而 coco 15 处理器全部经 `RegisterProcessorContract` 注册 → registry 兜底补上,放置检查终于覆盖产品处理器。
2. for_each/lane_consumer 子链编译不透传 `Strict` → 新增 `ScopeAwareProcessor` 接口,编译期把外层 ScopeInfo 交给宿主重编译子链(if/case 本就透传)。
单测:strict 下 for_each 内 batch-scoped 处理器报错(实例声明+registry 声明两变体)、non-strict 仍放行、ValidatePipelineConfig warnings 可见(即审计工具的信号源)。

**coco `8817f8b1`** — `cmd/pipeline-audit`(P8 #113):复用控制面自己的校验闸(ValidatePipelineConfig)+ 全量处理器目录,分级 compliant / needs-migration(strict 关着才编过 = 翻转即断) / non-compliant;另标 process_documents 仍走 messages 约定(#116 候选)。ES + 文件双源,--json,exit 0/1/2 可进 CI。
CLI 环境适配:复制模块 Setup 的 dag/echo/for_each 注册、内存 KV 满足构造期检查、fd 级 stdout 静默(seelog 异步 flush + import 期 env 日志都会污染报告;PIPELINE_AUDIT_VERBOSE=1 保留)。

**首个真实审计结论**(本机存量):`enrich_documents`、`enrich_attachments` 均为 needs-migration —— record 处理器落在 batch 根,正是 #114 的 `for_each{codec: coco_document}` 迁移形态。**P8 默认翻转当前阻塞:2 条**。

**P8 后续**:#114 迁移工具 → 存量迁移 → #115/#116/#117 删旧+翻转(前置:P3-P7 稳定一个发布周期)。P4 性能与 logpilot 侧未动。

### 增量 2026-10-24:P8.114 pipeline-migrate 落地 (coco `40b69727`)

- **命名空间分段是关键**:第一版单一 `for_each{codec: coco_document}` 全链包装在附件管道上被自己的验证闸拦下(attachment 处理器要求 attachment 命名空间)→ 改为按声明命名空间切最大连续段、各段配对 codec(coco_document/coco_attachment),无约束步骤继承邻段命名空间。两存量管道迁移后均零错误零告警。
- 迁移结果必须过 ValidatePipelineConfig 双零(错误+告警)才算数,否则拒写;batch-scoped 步骤/未知命名空间 → 拒绝转人工。默认干跑;`--write` 全文档写回 ES(其他字段保留);`--only` 定向;文件输入只干跑。
- `cmd/internal/pipescan` 共享包(目录装配/ES+文件装载/校验闸)统一 audit 与 migrate;audit 瘦身后实时输出不变。
- E2E:丢弃型探针管道(根上 record 链)→ `--only --write` → ES 中形状为 for_each 包装、labels/enabled 保留 → 复审 compliant → 探针删除;真实管道全程未动。
- 单测:document 包装/attachment codec/交替分段/batch 拒绝/legacy 形态告警,全绿。

**当前 P8 状态**:audit + migrate 工具就绪;存量 2 条管道审计为 needs-migration、干跑迁移双绿 —— 待运行时等效验证(#115-117 删旧+翻转仍受"稳定一个发布周期"前置约束)。

## P4 进度记录 (2026-10-24)

### 已落地（framework 1fea4bea + coco c38a6c7a）
- **#67 记录事件池**: `framework/core/event/pool.go` — AcquireEvent/ReleaseEvent; map 原地清空(条目上限128,超限放弃回收)、原子拒绝双重 release、非池化事件 no-op(兼容 codec/clone 自建事件); `Event.pooled` 私有回指字段(Private 已被 drop marker 占用)。池计数器 PoolStats 支撑泄漏检测。
- **#68 接线**: for_each 在 active 集初始化后 defer 统一 release(覆盖正常/取消/失败/丢弃全部出口,clones 一并); coco 两 codec Decode 改 AcquireEvent + 复用池化 map 壳。
- **#75 基准套件**: framework(`for_each_bench_test.go`: 1MB 文档 K=4 池化/非池化、1KB CPU 链 parallel 1 vs 4 批32条) + coco(`modules/pipeline/for_each_bench_test.go`: 真实 DocumentCodec 1MB embedding 占主导负载 records vs legacy 全量 typed 往返、冷热分离反事实对照)。
- **-race 门槛修复**(既有 bug,非本次引入): debug 日志把可变 `*Processors` 交给 seelog 惰性异步格式化器 → 测试改 p.sub.List 时数据竞争; 三处调用点改为先物化 `procs.String()`。

### 实测数字 (Apple M2, 8核)
- 并行门槛: CPU 链(256遍fnv×K4) parallel:1 41.6ms → parallel:4 10.9ms = **3.8×**(≥3× 达标)。
- 1MB embedding 主导文档 K=4: 冷热分离 7.96ms/**221 allocs** vs 无分离 14.4ms/99062 allocs = **45% CPU、99.8% 分配削减**;"≥80% CPU" 在纯 embedding 字节占比更高的真实负载成立,内容占比高时为 27-45%(诚实记录)。
- **D-6 jsoniter 决策被实测推翻**: jsoniter 通用 interface{} 编解码对 float 数组逐元素分配(1MB document_chunk 子树 122k allocs vs stdlib 19,字节完全一致);codec 双向回退 stdlib,行为门禁改写为 codec 往返(大数/HTML转义/nil/嵌套 fixture 不变)。#70 documentview/dryrun 的 jsoniter 替换因此**不再执行**(同为 interface{} 树,预期同退化)。
- #73 已由 #71 冷分离满足: records 模式 decode 一次,document_chunk 原始字节不解析,链上处理器只读热字段。

### 状态与遗留
- **#67/#68/#69/#70/#71/#73/#74(:275 主路径)/#75 完成**; #74 :381/:447 同类点待复核。
- **#72(embedding chunks typed 访问器)延后**: embedding 处理器仍是 messages 路径(全量 typed 解码+重序列化),记录路径访问器在 embedding 切到 records 模式(P8 翻转)后才有意义; 冷分离已保证 chunks 字节在记录链上零解析。
- 池化收益现状: Event 壳+外层 map 复用,单记录省 ~3 次分配; inner tree 仍全新(原始 JSON→interface{} 本质),进一步收益需 typed view 访问器路线(与 #72 同理)。

### 补遗 (framework e1845cc1)
- 首次带池化重启 coco 时启动即崩: `Event.pooled *pooledEvent` + `pooledEvent` 内嵌 `Event` 构成递归类型, `GetTagsByTagName`(无环保护)反射无限递归栈溢出。回指字段改 `interface{}`(反射不钻 interface 字段)断环; `-race` 全绿后重启验证 200。教训: 核心结构体禁止自引用类型的类型化回指。

## P6.2 进度 (2026-10-24, framework fd47c673 + coco 128e4484)

- **#91 完成**: framework `POST /pipeline/ai-generate` + `ChainGenerator` 注册表(空名默认唯一注册、多注册拒绝、未注册 501、生成失败 502);验证/回放归调用方的 /pipeline/dry-run。
- **#92 完成**: coco ai_generate 改造为 ChainGenerator("coco")注册;**studio 两端点已删(实测 404)**;dryrun.go 整文件删除(-475 行)。生成 prompt 切 record 约定(document.* 寻址)。
- **#94 完成(coco 侧)**: pipeline.ts 切统一端点,dry-run Report 映射回 steps/finals 形状(编译失败上抛为测试错误);AI 生成后显式跟进 dry-run 替代 auto_test。typecheck 598=基线、locale 通过。
- **活体验证**: ai-generate 200 且生成链用 document.lang(record 约定生效)→ dry-run 编译+回放 ok;旧端点 404。
- **遗留**: #93 logpilot 侧 5 个端点删除+ChainGenerator 注册(logpilot 仓);#94 logpilot 前端;P7 ui-common Pipeline 包共享。

### #93 logpilot 侧收尾 (logpilot 6cf4fcb, 2026-10-24)
- 路由层早已删除(`streams/:id/dry-run`、`debug/_inject`、`parsing/test`),本次清掉遗留死处理器 `DryRunStream` + `InjectDebugLog` 及其专用 OTLP 导出助手(-199 行);探针/buildNamedProcessor/流样本/DebugChain 保留(module 测试与产品专有体检在用)。
- **`parsing/ai-generate` 有意保留产品侧**(偏离计划原文,已双处注释): 其输入是原始日志行 + Pattern 聚类 + 知识包 + placement 目录(gateway/agent),与 ChainGenerator 的"解析后 JSON 文档"契约不匹配,强行统一会丢能力。`debug/chain` 同理(流基础设施体检,非管道执行追踪)。
- **P6.2 至此关闭**: 统一端点为 dry-run/AI 生成唯一定义这一验收在"文档形生成"域成立;日志行域的产品端点保留属记录在案的设计决定。

### P7 进度 (2026-10-24, ui-common 7e69e72 + coco b970fd8e + logpilot db2cef4)
- **#105 完成**: `@infinilabs/pipeline-ui` 新增 `pipeline-api.ts` — 统一端点类型化 client(Report/StepResult/StepDesc/CapturedWrite/AIGenerateRequest/CodecInfo 类型镜像 Go 源)+ `createPipelineClient(fetchers)` 工厂,产品注入自己的 request 实例,包保持零 API 依赖原则;根级 `api.ts` 垫片兼容 classic node 解析(exports 子路径 tsc "node" 模式不识别)。
- **#107 完成**: coco/web 与 logpilot/web 均 `file:` 源码引用接入(D-9,不发 npm)。
- **coco 实际消费**: studio dry-run 调用改走 `createPipelineClient`(axios 适配边界注入),Report/StepResult 用包类型替换本地 any — typecheck 598=基线。
- **P7 剩余**: #102 画布迁移(processing-canvas.tsx 2131 行 → 共享包)、#103 SchemaForm、#104 DryRunDrawer/AiGeneratePanel/FlightTimeline、#108-110 coco 画布集成(含 #109 canvas_layout 字段)、#111 logpilot 三视图切换。注意: 包内既有组件(chain-editor 等)引用 logpilot 的 `@/components/ui/*` 别名,迁入共享前需去别名化(props 注入或本地原语),否则 coco 消费根包会挂类型(本轮以 ./api 子路径规避)。

### P7 增量 2 (2026-10-24, ui-common 57e56fd)
- **#103 完成**: SchemaForm 从 logpilot schema-form.tsx 迁入共享包,产品缝隙改注入: `t`(i18n,恒等默认)+ `patternPicker`(logpilot 注入 GrokPatternPicker,缺省退纯 textarea)。
- **全包去别名化完成**: 本地原语(Button/Badge/Input/Label/Select/Textarea,兼容 shadcn variant/size 面)替换 `@/components/ui/*`;顺带修掉包内既有 bug(类型声明未导出、lanes-panel 局部 state 遮蔽 props)。包自带 tsconfig,`tsc --noEmit` 零错误(logpilot 工具链编译,类型依赖符号链接——ui-common 根 pnpm 11 与本机 node 不兼容,待环境修)。
- coco 门禁 598 不变(仍只消费 ./api 子路径)。**P7 剩余**: #102 画布(2131 行)、#104 DryRunDrawer/AiGeneratePanel/FlightTimeline、#108-111 产品侧切换。

### P7 增量 3 (2026-10-24, ui-common ce8b7a4)
- **#104 主体完成**: `DryRunDrawer`(渲染统一 Report:编译判定/样本切换/逐步 outcome+耗时+`+/-/~` 字段 diff 键/已捕获副作用脚注)+ `AiGeneratePanel`(完全哑交互壳:要求输入、generate/refine 模式、busy/错误面;产品经 pipeline-api 绑定真实调用并自行串联 dry-run 校验)。均 props 驱动零产品依赖,包 tsc 零错误。
- FlightTimeline 不单列: runtime-detail 的 PipelineRuntime 已覆盖 stages+flight 渲染。
- **P7 剩余**: #102 画布迁移(processing-canvas.tsx 2131 行,最大块)、#108-111 两产品视图切换(coco studio 可先消费 DryRunDrawer/AiGeneratePanel 替换内联渲染)。

### P7 增量 4 (2026-10-24, framework 437d3c13)
- **#109 完成**: `PipelineConfigV2` 新增 `canvas_layout` 字段(map[string]interface{},omitempty,ES 不索引)承载编排画布节点坐标;round-trip 测试覆盖 ucfg 解包 + JSON 序列化 + legacy 配置零变化。
- **踩坑记录(键名约定)**: ucfg 对 map 解包会把**纯数字键**当数组下位标静默丢弃——`"0": {x,y}` 解包后 CanvasLayout 为空 map,而 `"n0"`/`"node-0"` 完好(与 Labels 同形状对照实验定位)。因此键约定为 `n<index>`,字段注释与测试双处写明;前端写 layout 时必须遵守。
- 执行侧永不读取该字段(纯呈现态);#108 coco 画布集成与 #102 画布迁移时按此约定读写。
- **P7 剩余**: #102 画布迁移(2131 行,最大块)、#108-111 产品侧切换(logpilot 先行,有 tailwind)。

### P7 增量 5 (2026-10-24, ui-common 18e890b + logpilot e5388cb)
- **#102 完成**: 2131 行画布从 logpilot modules/streams 迁入 `@infinilabs/pipeline-ui`(processing-canvas.tsx)。产品缝隙全注入: `t`(context 下发, 组件内 `const __t = useT()` 与原调用点同名零改动)、`fetchProcessorCatalog`(目录端点)、`patternPicker`(透传 SchemaForm); UI 原语走 ./primitives。
- **去重**: 画布本地 DryRunReport/DryRunStepResult 与 pipeline-api 同源(framework dryrun JSON), 改为 re-export + 别名, 消费方类型不变。identityTranslate 升级为 logpilot t 语义(仅替换 vars 中存在的 {name} 占位符); Button 补 forwardRef(节点库弹层锚定)。
- logpilot 原文件改薄包装(-2094 行, 注入 t/GrokPatternPicker/目录 fetcher + 类型转发), **五个消费方 import 路径零改动**; 包 tsc 零错误、logpilot `tsc -b` + `vite build` 通过。pnpm file: 快照需 `pnpm install` 刷新(新增文件不进旧快照)。
- **遗留**: 画布内位置记忆仍是会话态 posMap, 未接 #109 canvas_layout 持久化(需 UI 键约定 n<index> + 调用方存取, 留给 #108 产品侧集成)。

### P7 增量 6 (2026-10-24, ui-common 9348593 + logpilot 9bda611 + coco 3f4d47c6)
- **预编译 CSS 方案落地(无 tailwind 产品接入的关键)**: 包内 `tailwind.config.mjs` + `src/theme.css`(preflight 关、:root shadcn 色板默认)+ `scripts/build-css.sh`(借 logpilot/web 的 tailwind v3 CLI)产出 `dist/pipeline-ui.css`(22.6KB, 随包提交), package.json 导出 `./styles.css`。**非 tailwind 产品(coco/antd)import 一个 CSS 文件即可获得全部工具类+色板, 无需引入 tailwind; tailwind 产品(logpilot)不 import, 继续自家 JIT**。教训: build-css.sh 里相对路径要用 ../../../logpilot(仓根兄弟目录)。
- **#111 完成(logpilot 9bda611)**: 删除 web/src/components/schema-form.tsx, ProcessorCatalog 类型改 re-export 自共享包; 五消费方 import 不变。
- **#110 完成(coco 3f4d47c6)**: studio PipelineEditor 挂共享 `DryRunDrawer`(统一 Report 渲染: 样本切换/逐步 outcome/diff 键)+ `AiGeneratePanel`(哑交互壳, 绑统一 /pipeline/ai-generate, 生成后显式跟进 dry-run); 本地 StepSnapshot/StepDebugger/fieldDiff 全删; testPipelineChain 返回 report 原始载荷喂 drawer。@xyflow/react ^12 进 coco 依赖(为 #108 备料)。typecheck 598=基线。

### P7 增量 7 (2026-10-24, ui-common cf5ec31 + coco 06455a0a) — #108 完成
- **包侧**: CanvasProps 增 `layout`/`onLayoutChange` 受控布局 — posMap 以 layout 播种(prop 解构名 layoutSeed 避开既有蛇形布局 memo), useEffect 在 posMap 变化时回调全量坐标; 键为内容指纹(天然含字母/不可见字符, 符合 #109 非纯数字键约定)。
- **coco 侧**: 自由编排编辑器视图切换扩为 卡片/画布/JSON; 画布挂 ProcessingCanvas(目录分组 useMemo 适配 {processors:{cat:{name:{properties,category}}}} 形状、dry-run run overlay 喂 report/samples/sampleIdx、`@infinilabs/pipeline-ui/styles.css`); canvasLayout 状态初始自 initial.canvas_layout, 保存时非空才写入 canvas_layout 字段; framework REST DecodeJSON 直入 PipelineConfigV2 已实测透传。locale 补 canvasView/canvasSource/canvasSink。
- **浏览器验收**: 用户 dev(9527)高级模式模板→自由编排, 画布视图渲染正常 — 点阵背景/来源(蓝)/存储(紫)锚点圆角卡片+连线/重置布局+图例徽章/缩放控件/小地图, 无裸样式或 CSS 缺失(DOM 复核 2 锚点+1 边)。全页截图曾被视口裁切误导(节点在折叠线下方), scrollIntoView 后确认。
- **#112 验收(域内达成)**: 两产品的链编排/试运行/调试交互现已同源共享包 — logpilot(画布+SchemaForm+目录, e5388cb/9bda611)与 coco(画布+DryRunDrawer+AiGeneratePanel, 06455a0a)均消费 @infinilabs/pipeline-ui, 统一端点(pipeline-api)驱动。**P7 关闭**。
- 遗留(记录在案不阻塞): 包内 lanes-panel/inject-dialog/pipeline-runtime 迁移版丢 i18n(en-US 会回退), 未迁; ingestion 面板两产品暂各自维护。

### P7 增量 8 (2026-10-24, coco 1bf99bce) — #112 端到端验收 + 两处修复
- **验收全链路(浏览器实测)**: 编辑页自由编排 → 画布(预编译 CSS 渲染正常/目录弹层带 ✓ 兼容标记、consumer/confluence 等作用域外项无 ✓) → 画布加 add_locale 节点 → 运行测试 → 共享 DryRunDrawer 内联面板「Dry-run编译通过 · add_locale 0ms」 → 拖拽节点 → 保存 → API/ES `canvas_layout={"add_locale\u0000{}":{x:313.7,y:52.5}}`(指纹键非纯数字) → 重载编辑页 → 节点 transform 精确恢复 `translate(313.718px, 52.479px)`。测试管道(canvas_e2e_accept/probe_layout2)已删净。
- **修复 A — 内嵌引擎测量急救(coco 1bf99bce)**: React Flow 以「ResizeObserver 初始回调」为节点测量门(未测量即 visibility:hidden)。**ZCode 内置浏览器(IAB)的 RO 不派发 observe() 的初始回调**(后续 resize 正常)——节点永久隐藏、不可拖、坐标不持久, 且无任何控制台报错。PipelineEditor 加一次性特性检测(探针 div observe+400ms 窗口), 中招则在挂载/重建后对节点做 0.5px 宽度扰动两帧触发真实回调完成测量; 真实 Chrome/Safari 检测通过零开销。**教训: 在嵌入式浏览器里验收 canvas 类组件, 先测引擎的 ResizeObserver 语义再怀疑业务代码。**
- **修复 B — 旧二进制陷阱**: API roundtrip(canvas_layout POST→GET 丢字段)根因是**运行中的 coco 服务器二进制早于 framework 437d3c13**(struct 无 CanvasLayout, json 解码静默忽略未知字段)。重建(go build, go.mod replace ../framework 生效)+ 标准重启流程(kill -9 + sleep 15 + -config coco.yml)后服务端 roundtrip 即通; ES 旧索引 mapping 冲突(processor enabled 参数)仅为启动告警, 动态映射接管新字段写入无碍。**教训: 排查字段丢失先确认运行二进制的构建时间 vs 字段引入时间。**
- 附带: @xyflow/react 钉精确 12.11.3(= logpilot 解析版本, 跨产品一致; 12.12.0 与 12.11.3 在 IAB 下同症状, 版本非根因)。typecheck 598=基线、locale 对齐。
- 会话基建备注: 重启后 /tmp/coco-token 失效, 走 POST /account/login {"login":...} 重取; IAB evaluate 传输通道间歇性「Unexpected token '}'」(同代码重试即过), 多步操作宜拆最小 cell。

### P7 增量 9 (2026-10-25, logpilot d9acf6d + ui-common eafa034 + coco e5013c9c) — logpilot 画布真实引擎验收 + 急救收敛重试
- **logpilot 白屏根因(承接增量 8 后续)**: dev 页面空白(根节点 0 子元素、无报错), 模块 BFS 扫描定位 `@infinilabs_pipeline-ui.js` 504 — **vite 依赖优化器卡死**(包源码变更后 .vite/deps 预打包不更新)。重启 dev server 恢复; 后续源码同步需 `vite --force`(普通重启校验缓存哈希通过即不重建, metadata src 指向新文件但产物仍是旧的)。
- **logpilot 画布塌缩 2px(真产品 bug, 本增量主修复)**: 画布卡片 `h-[380px]` 只剩边框 2px — 根因两层: ① logpilot 的 tailwind content 只扫自身 src, 包内工具类不生成, 而**薄包装漏 import 包预编译 CSS**(coco 侧增量 7 有 import, logpilot 侧没有); ② pnpm file: 依赖是**安装时快照**, logpilot 的快照(22:31)早于 dist/pipeline-ui.css 产生(22:37), exports ./styles.css 指向的文件根本不在 node_modules 里。修复: 包装层 import `@infinilabs/pipeline-ui/styles.css`(logpilot d9acf6d) + 快照手动同步 dist/src(正式流程: 包 dist 重建后消费方 `pnpm install --force`)。
- **测量急救升级为有界重试(ui-common eafa034, coco e5013c9c 同步)**: 单发宽度扰动会被节点数组结构性重建竞争掉(schema 异步加载/选中态重建换入未测量的新对象, logpilot 实测 3 可见 2 锚点隐藏, 手动扰动后全可见)。改为 6 轮有界重试(首轮 50ms 后续 500ms), 每轮只扰动仍 hidden 的节点, 全部收敛提前退出; 真实引擎 RO 特性检测通过则零开销。
- **IAB 帧循环冻结(环境级发现)**: 会话后段 IAB 全部标签页 rAF 永不触发(visibilityState 谎报 visible), RO 连后续 resize 回调也不派发 — RF 测量/任何 rAF 依赖逻辑在该环境下全部假死, 页面内无法自救。**教训: IAB 里"节点不可见/拖不动"先测 rAF 是否触发(document.visibilityState 不可信); 帧冻结是环境缺陷不是产品 bug。**
- **真引擎兜底验收(headless Chrome 裸 CDP)**: 无 playwright/puppeteer, 用 `/Applications/Google Chrome.app --headless=new --remote-debugging-port` + node22 内置 WebSocket 手写 CDP 驱动: 页内 fetch /account/login 200(Set-Cookie Path=/ HttpOnly) → about:blank 中转后进流管道 tab(AuthProvider 随新文档重建, hash-only navigate 不重建 SPA 文档!) → **rAF fired / 卡片 664x380 / 5/5 节点 visible / CDP Input 真实拖拽 node0 translate(0,0)→(207.9,31.2)**。logpilot 共享画布集成至此真正验收通过(此前仅 DOM 级)。
- 工程小坑: `pkill -f <模式>` 会自匹配包含该模式文本的当前 shell 命令行(清理动作要与正事分命令执行); HttpOnly cookie 在 document.cookie 恒为空, 判会话要走请求; evaluate 传含反斜杠正则经传输层会被吃, 解析用 split。
- **P7 最终态**: 两产品画布/试运行/调试同源共享包且各自真实验收(coco IAB 内、logpilot headless Chrome); P8(#113-118)维持"稳定一个发布周期"门控不动。

### P8 增量 10 (2026-10-25, framework 3806459f + coco 1e8a0c34) — #118 文档三件套完成
- **framework 侧**: ① `references/lane.md` — lane 参考文档(容量/溢写三策略及隐性成本、动态 lane、lane_consumer、统一 output、跨 lane 提交正确性三姿势), **queue-vs-lane 选型判据** 从设计稿附录蒸馏为正式文档(at-most-once 边界一句话 + 场景信号表 + 安全性三依据); ② `references/glossary.md` — 消息/批/记录/作用域/契约/通道/控制面全术语表, README 加精简版术语表并链接; ③ references 索引收录两页。文档字段名全部对照实现核过(Stats 无 pending_acks/depth_pct——那是 SPEC 愿景未落地, 按代码写; ack_timeout_ms 在 queue consumer 侧非 lane_consumer)。
- **coco 侧**: `references/path_contract.md` — **document.*/attachment.\* 路径契约冻结公告**(冻结路径表、三条冻结规则: 整树投影增量演进/热路径名与 labels 键永不改名/删除改名走 major+codec 版本+1、实操指引), 开发中版本 release-notes 置顶公告。D-3 评审项就此落锤。
- **修复 docs 构建(两处环境级, 均为存量问题)**: ① framework release-notes #415 条目**逐字包含 hugo 内部占位符前缀** `HAHAHUGOSHORTCODE-`, 0.139 的占位符替换器将其误解析为未闭合 shortcode token——"illegal state in content; shortcode token missing end delim", 逐行二分定位(注意 grep -c ERROR 会被 "ERROR deprecated" 污染, 二分要用 hugo 退出码/特征串); 改写该行修复。② 本机 vendored theme(gitignore, 不随仓走)对 hugo 0.139 两处不兼容: `.Site.IsMultiLingual`(0.124 弃用→0.139 升格硬错误) 与 `_internal/google_analytics_async.html`(已移除模板)——theme 本地修补, 两站构建全绿。
- #118 验收对照: 术语表 ✓(README+references)、选型判据 ✓(lane.md)、契约冻结公告 ✓(path_contract.md + release notes)。**P8 剩余: #115/#116/#117 删旧+翻转, 仍受"稳定一个发布周期"前置; 迁移工具的存量运行时等效验证待做。**

### P8 增量 11 (2026-10-25, framework 9f17a4a3 + coco bf9999a2) — #114 运行时等效验证闭环
- **framework — for_each 干跑收养**: dryrun 骨架给链绑定一条已解码记录但不给消息批, for_each 的批查找落空后**静默 no-op** —— 所有含 for_each 的链干跑一律"无效果"(画布干跑同病, #112 E2E 的"add_locale 0ms"实为空转)。修复: 干跑态+无批+有绑定记录时, for_each 收养该记录为零解码类型化入口(与 lane_consumer 运行时同款), 子链真实执行; 收养记录强制串行(确定性 diff)、跳过编码回写与池释放(记录归骨架所有)。Process 重构为 runOnRecords(records, msgs) 共享批路径与收养路径; 非干跑的无批跳过行为原样保留。新增 pipeline.InDryRun 探针。单测: 收养后子链真实改写并恢复绑定记录; 非干跑 no-op。
- **coco 单测 (bf9999a2)**: equivalence_test.go — 两条真实 needs-migration 管道 fixture(镜像 coco_pipelineconfigv2 存量) → migrateChain → dryrun.Run 双形态同样本: legacy 按序执行 fixture 步骤(首错即停, 在哪步错取决于环境 LLM 配置), for_each 宿主收养不失败, 终态相等(时间类键归一化)。CI 可复跑。
- **活体 A/B(决定性证据)**: 重建重启 coco(带新 framework), 真实文档(README.md, local_fs 数据源——detection/metadata/text_attachment 门控真实放行) + 真实 LLM, 两形态各一跑(legacy 233.8s / migrated 215.9s):
  - legacy: 9 步全 ok, 真实写入(detection +5 字段, cover +1, summarization +1, extract_tags +1, embedding +2);
  - migrated: 单步 for_each 收养执行, +9 字段, **终态 document 16 键集合完全一致**;
  - 逐字段深比(LLM 产物字段按存在性/类型归一): **唯一差异 document.cover 的一次性附件 ID**(两跑各自生成的封面附件, 生成机制等价); metadata.mime_type/content_type(markdown)/content_category(doc) 逐值相等, status=completed/processed 相等。**等效结论成立。**
- **发现待办 — 干跑副作用泄漏**: generate_document_cover 直写附件库(不经 output 捕获路径), 干跑真实落了 2 条 coco_attachment-v2(已删净)。P6.1 的副作用治理覆盖 queue/lane 写, 处理器内附件直写是漏网之鱼 —— 后续给附件存储写路径加干跑护栏(增量候选)。
- 工程教训: coco 测试二进制 stdout 被吞(audit 工具链 seelog fd 级静宁), -v/-json 均不可见, 判定靠退出码+临时 marker 文件; zsh 中 `echo ===X===` 触发 =word 展开会炸命令行。
- **P8 状态**: #113 audit + #114 migrate + 等效验证(单测+活体)全部闭环; 翻转前置链条(audit 扫描→迁移→等效验证)就绪, 剩 #115/#116/#117 受"稳定一个发布周期"门控。

### P8 增量 12 (2026-10-25, framework 539e914d + coco fa104131) — 干跑附件泄漏修复 (增量 11 待办闭环)
- **根因双层**: ① 处理器侧 —— generate_cover(文档/附件两变体)与 fileproc/chunk 的附件落库全部经由单一卡口 `attachment.UploadToBlobStore`, 干跑时无护栏真实写 ES 元数据行 + kv blob(P8 等效 A/B 实测落了 2 条); ② 框架侧 —— 干跑状态原存于 pipeline.Context 的参数表, 而处理器回调(utils.EachDocumentRecord)拿到的是从 `ctx.Context` 派生的裸 context, 深层(orm/附件层)永远看不见干跑标记 —— 第一版护栏断言 `*pipeline.Context` 因此落空(教训: 跨层状态要走 Go context 链, 参数表只活在本层)。
- **framework 539e914d**: EnableDryRun 改为在 `ctx.Context` 上挂 `context.WithValue` 状态, InDryRun/DryRunCapture 统一经 Value 链读取, 新增 InDryRunCtx/DryRunCaptureCtx 供持裸 context 的深层使用; AcquireContext 每次新建结构体, 无池化泄漏; 旧 kv 键零残留读者。DryRunWrite.Kind 注释放宽(attachment)。
- **coco fa104131**: UploadToBlobStore 卡口干跑捕获模式 → 不落盘、副作用捕获(Kind=attachment, id+字节数+mime 预览)、仍返回 ID(下游 document.cover 等字段照常可写); passthrough 干跑与非管道调用方(HTTP 上传 API、wiki)不受影响。单测覆盖派生 context 捕获 + 裸 context 旁路(无后端时触达真实持久化段即证护栏未误伤)。
- **活体验证**: 重建重启后干跑 generate_document_cover —— side_effects 出现 `attachment/..._cover.png (34042 bytes, image/png)` 捕获项, document.cover 正常赋值, ES 对该 ID 404(未落盘); 期间误写的 1 条泄漏附件已删净。**干跑副作用治理补全: queue/lane(P6.1) + attachment(本增量)。**
- P8 状态不变: #115-117 仍门控; 干跑现在对两形态 A/B 也更干净(不再有一次性附件 ID 差异, cover 字段捕获后两跑 ID 仍各自随机但均不入库)。

### P8 增量 13 (2026-10-23, ui-common 4d6d699) — 增量 7 遗留收口: 包内三组件 i18n 补齐
- **共享接缝 `src/i18n.ts`**: TContext/useT/identityTranslate 从画布私有提升为包级模块(画布改 import, 行为不变); lanes-panel / inject-dialog / runtime-detail 三组件统一换 t 注入形态 — `t?: Translate` prop + Provider 包装导出组件 + 内层组件 `const __t = useT()`, 与画布/SchemaForm 完全同构。
- **键与 logpilot 本地版逐字一致**(对照 modules/ingestion/ 本地副本核过): inject-dialog 全部 9 键含 `注入 {v0} 条 → {v1}`/`注入 {length} 条`/`目标队列 (可选 — 留空用管线默认)`; runtime-detail 共享键 `运行时 · {pipeline}`/`刷新`/`次`/`错`/`通道:`(注意本地版是半角冒号)/`暂无指标`, 独有键(近期失败/查看此记录的日志)按中文源串即键。**产品后续切包版本时现有词条目录直接命中, 零新增翻译。**
- index.ts 补 props 类型导出(LanesPanelProps/InjectDialogProps/PipelineRuntimeProps); runtime-detail 顺手清掉迁移残留的死 proxy 函数(带 /logpilot/ 产品路径, 不该留在共享包)。包 tsc 零错误(源码包, dist 仅 CSS 无涉, 不需重建)。
- **现状注明**: 当前两产品仍消费各自本地副本(未传 t), 本提交为 drop-in 适配铺路——en-US 场景下包版本不再硬编码中文回退。ingestion 面板两产品各自维护的既定结论不变。
- P8 状态不变: #115/#116/#117 仍受"稳定一个发布周期"门控, 前置链条(audit→migrate→等效验证→副作用治理)全部就绪。

### P9 增量 14 (2026-10-23, framework c02da194) — P9 现状盘点 + #131 ORM 管道热生效
- **P9 现状盘点(重要发现)**: P9 任务表(10-10 撰写)与仓内实态差距大 —— 10-15 已有一批存量提交实现了 #123(deployment 模型+placement targeting)/#124(reconciler, 并入 deployment.go 非 reconciler.go)/#126(placement 翻译器, singleton/all)/Fleet API 骨架(deployments CRUD+status+cluster/overview 路由)与 #119 的 _sync 运行时报告(RuntimeSnapshot→configs_client)。该批提交早于本 WORKPLAN 的增量记录体系(10-24 起), 此前未盘点入账。**仍开放**: #120(广播操控 generation 防脑裂)/#121(_drain/_cordon)/#122(锁接管事件)/#127(instance_id 进指标)/#128(logpilot StreamSpec 对接)/#129(ClusterOverview UI)/#130(多实例验收)。
- **#132 复核(已实现变体, 不再动)**: process_documents 的 `compile_cache_ttl_ms` TTL 编译缓存即 #132 的落地形态(代码注释自证), 偏离设计稿的"按 version/updated 版本感知"——TTL 档同时省掉 orm.GetV2 与 NewPipeline, 版本感知档每消息仍需 GetV2 读版本, TTL 是更彻底的削减、代价是 TTL+变更时延内的旧语义。默认 0=关闭(逐消息编译, 与现状一致), 验收口径"缓存命中零 GetV2/NewPipeline"仅在开启时成立。该处理器有并行会话 WIP(processor_w1_test.go), 本增量不触碰。
- **#131 完成(framework c02da194)**: /pipelines PUT/POST/DELETE 此前只落库, 本地运行任务抱旧编译链直到重启。三个 CRUD handler 现触发 hotReloadPipeline —— 复用 fsnotify 同款重建语义: stopAndWaitForRelease(排干在飞批, ≤1min)→deleteTask→createPipelineTask(新配置); 删除态传 nil 只停不建。**严格按任务 ID 匹配**: deployment 渲染配置保留 ORM pipeline ID故 API 编辑直达运行实例; 本地 YAML 任务(自有 ID)与 transient 任务不受扰。语义未变(Equals+enabled)跳过重建。重建走异步(不阻塞 API 响应)+互斥串行(防两次快速编辑的 stop/delete/create 竞态被 LoadOrStore 吞掉收敛到旧配置)。新增 generations 计数(1=启动版)随 RuntimeSnapshot 上报, 满足验收"generation 递增可见"。单测覆盖: no-op/换链+generation/删除/未知 ID/transient 五路(hotreload_test.go)。
- **工程教训**: 包内测试注册的处理器替身必须 `test_` 前缀 —— TestEmptyConfigAudit 会遍历进程级注册表审计"空配置接受度", 非 test_ 前缀的替身会被当成产品目录项打 FAIL(替身约定写在审计测试注释里); Pause() 用 WaitGroup 阻塞, 完结任务在测试里安全驻留不空转。
- P9 状态: #131 闭环; #132 按 TTL 变体记录关闭; 其余开放项(#120-122/#127-130)与 P8 #115-117(发布周期门控)待后续。

### P9 增量 15 (2026-10-23, framework 3d75258c) — P9.1 协调原语三件套 (#120/#121/#122)
- **#120 控制指令 generation 守卫**: `_pause/_resume/_start/_stop`(含新 `_drain`)接受可选 `?generation=N` —— 指令仅当本地任务正运行该配置版本时生效: 还停在旧 generation 的实例(热重载未落地)绝不执行为新版本发出的指令, 反之亦然; 不匹配答 409 并同时携带 current/instruction 两代。缺省参数保持今日无条件执行(零破坏)。**广播传输本体仍属 #125(未落)**, 本增量交付的是实例侧契约 —— 控制面逐实例定向(或未来广播)时携带 generation 即得防脑裂。
- **#121 drain(优雅停) + cordon(封闸)**: ① core Context 新增 `Drain()/IsDraining()` —— 与 CancelTask(批中.abort, 触发重投)相对, drain 语义=停止 FETCH: consumer 在批边界检查该标志, 当前批处理完、offset 提交(CLEAN_BUFFER 路径)后 worker 返回; 检查点四处(探测器循环×2、批读取边界、READ_DOCS 回跳条件——漏最后一处会 CLEAN_BUFFER 死循环)。② 端点: `POST /pipeline/task/:id/_drain`(任务优雅停, 停后驻留 STOPPED 可 _start 复活, 带 settle 等待与超时)与 `POST /pipeline/_drain`(实例级全量, 先统一标记再一次共享限时等待, 返回逐任务终态)。③ `Restart()` 同时清 drain 标志, 否则 drain 后重启的 consumer 立刻又停。④ cordon: `POST /pipeline/_cordon/_uncordon` + `GET /pipeline/_cordon`; 封闸实例上运行任务照常跑完, 但一切启动路径(auto_start/_start/_resume, 以及热重建重建出的任务)在任务循环 STARTING 分支驻 STOPPED; 显式 _start/_resume 答 409。**cordon 状态目前实例本地**——到达控制面 placement 侧需 Instance 模型/心跳同步扩展(与 #128 集成时一起做, 已记录)。
- **#122 锁接管事件**: singleton 重取锁前经 `probeLockTakeover`(GetAllocateInfo + recover 防无 kv 环境)探测"租约已过期且属他实例"的锁, Hold 成功后发 `lock-takeover` StepEvent(新增 Outcome, 非 ok 类被 flight ring 恒留, errors 过滤视图含之, UI 未知 outcome 走灰色兜底类)+ warn 日志。测试用内存 KVStore 注册(kv.Register 全局 handler, 一次 Once 守护)直写 `dis_locker` 桶的 `clientID/unix` 行覆盖四分支(无锁/过期他持/新鲜他持/过期自持)。
- **验证**: 新增 control_test.go 七测(参数解析/drain 停+复活/全量 drain/封闸停车+解封/接管探测四分支/事件入环), 三包(modules/pipeline、plugins/queue/consumer、core/pipeline)`-race` 全绿, 全套 ×3 稳定。曾有一次 TestCordon 区未复现失败(后续 9 连绿), 疑环境抖动, 留意复发。
- P9 状态: #120(实例侧)/#121/#122 闭环; 剩 #120 广播传输(#125 内)、#127(instance_id 进指标)、#128(logpilot StreamSpec 对接, 含 cordon 上报扩展)、#129(ClusterOverview UI)、#130(多实例验收)。P8 #115-117 仍发布周期门控。

### P9 增量 16 (2026-10-23, framework 265b8671 + 68cb052a) — #127 instance_id 归因 + cluster/stages 聚合
- **#127 完成 (265b8671)**: 三层归因 —— ① StepEvent 增 InstanceID, emitStep 卡点统一盖章(控制面事件 recordPipelineEvent 同步盖章); ② StageStats 每行带 instance_id(processor 行与 lane 行都盖本地节点); ③ 持久化行 PipelineStageStat 增 instance_id keyword mapping —— 把 logpilot InstanceEPS 已有的字段通用化进框架。JSON 增量字段, 老消费者(含包 UI StageRow)无感。
- **cluster/stages 聚合 (68cb052a, #125 缺口/#130 验收地基)**: `GET /pipeline/cluster/stages[?pipeline=]` 从持久化快照聚合 —— 每实例每 stage 只取**最新 tick**(旧 tick 被取代不累加), 计数跨实例求和, 延迟取调用次数加权均值, PerInstance 明细随行下发(聚合=各实例明细之和, #130 验收口径可直接对账)。行序无关(函数内按 Ts 降序排); 纯函数 aggregateStageRows 单测覆盖: 双实例求和/旧 tick 取代/乱序输入/加权均值/明细与聚合对账。
- 验证: 三包 -race 全绿(modules/pipeline × 多轮、core/pipeline)。#127 验收口径(StepEvent/StageStats 携带 instance_id)✓; cluster/stages 端点就绪, 多实例活体验证归 #130。
- P9 剩余: #125 缺口(_rollout/集群广播/instances proxy)、#128(logpilot StreamSpec 对接 + cordon 上报)、#129(ClusterOverview UI, 可消费 cluster/overview+cluster/stages 两端点)、#130(多实例验收)。P8 #115-117 仍发布周期门控。

### P9 增量 17 (2026-10-23, ui-common f2f2311) — #129 ClusterOverview 入共享包
- 组件 `cluster-overview.tsx`: 舰队视图双区块 —— ① 部署矩阵: 管线(可点击回调)/部署名/策略/vN/ready÷desired(三色)/逐实例状态 chip(running 绿/stopped 灰/failed 红/unknown 黄, title 带 vN+last_seen+失败原因), "暂无实例上报"空态显式可见(缺席≠健康); ② 集群阶段指标(可选, fetchStages 省略则只渲染矩阵): 聚合行 + 行尾各实例最新一拍明细("inst-a: 300 · inst-b: 100"), 调用加权延迟, tooltip 写明"集群合计应等于各实例之和"的对账口径。
- 数据全 props 注入(包零 API 依赖原则): fetchOverview 绑 GET /pipeline/cluster/overview、fetchStages 绑 GET /pipeline/cluster/stages(增量 16 两端点); 类型逐字段镜像 Go(DeploymentStatus/InstanceReport/ClusterStageStat); t 注入与包内其余组件同构; refreshMs 自动刷新(默认 10s, 0 关)。tsc 零错误。
- **现状注明**: 两产品尚无消费方(包侧交付完毕, 产品按需绑定 fetcher 接入); rollout 进度区块待 #125 _rollout 端点落地后补。#129 的"矩阵"要件 ✓, "rollout 进度"记为后续。
- P9 剩余: #125 缺口(_rollout/集群广播/instances proxy)、#128(logpilot StreamSpec 对接 + cordon 上报)、#130(多实例验收)。P8 #115-117 发布周期门控。

## Increment 18（2026-10-25）— 美心知识库 Demo 整备（数据运营 + 1 个管线 bug 修复）

用户指令：review 下美心的知识库，查漏补缺，是否需要额外的素材，需要就采集，反正要确保 Demo 亮眼。

### 诊断（KB db6d7nn3edbulgmnpfu0「美心」）
- 3 个数据源（maximscakes2 21 文档 / maximscakes 8 / 手工上传 3），富集链从未完整跑通：maximscakes2 的 10 个文档卡在 indexing 28+ 分钟、零报错
- 实体图谱空置：仅 3 个 10-12 的测试实体（测试实体A/重复测试X/美心）
- 素材缺口：月饼（旗舰产品）零内容、无中文品牌叙事

### 修复（代码，ce14e92e 已推送）
- `reprocessDatasourceDocs` 不清 `Processed`：consumer 的 alreadyEnriched 门（Processed + 同 hash）静默吞掉全部 force 重排消息 → 整个数据源永久停在 indexing。单文档端点早有同款修复与注释，批量端点漏了。补 `doc.Processed = false` + 回归测试（含 sqlite 检索对 term 过滤宽松的注释说明）
- 期间发现：修复后的首次批量调用（服务器启动后 ~1 分钟）save 未落盘，重跑即正常——启动瞬态，未复现，留观

### 运营动作（全部 API/ES 完成）
- 删 7 个垃圾文档（.DS_Store/测试残留）；retire 3 个垃圾实体（batch_status，设计 §4.1 实体不物理删除）
- 采集上传 3 个中文叙事素材（/tmp/meixin-material/*.md → POST /wiki/kb/:id/_files/upload，.md 直入 content）：月饼流心奶黄与礼盒、集团品牌故事（1956 起）、西饼招牌蛋糕产品线
- force 重排跑通全部 32 文档（32 completed）
- 跑两轮 _compile（63/64 MAP 批缓存命中；LLM 供应商内容过滤拦掉 maxims_en_wikipedia 一批 + 每轮 1/4 topic compose 失败——供应商长提示词不稳定，非主题相关）；proposals 被并行会话的自动应用环消费，产出 4 新页面
- 4 个 draft 走完 draft→reviewed→published
- TOC 重组：原为 250+ 节点单层平铺（含 stores.csv/Geocoding error 等垃圾）。重建为 7 个主题文件夹（品牌与集团 65 / 月饼与节庆 24 / 蛋糕与产品线 49 / 礼券与会员 27 / 分店网络 87 / 专题与概念 40 / 原始素材与维护页 15）。关键认知：reconcileToc 会把树上缺席的文章按 orphan 回填到根——想"删"节点必须改为"收纳"进尾文件夹
- 49 个 proposed 实体批量发布

### 终态（Demo 面）
- 文档：32/32 completed（3 数据源）
- 文章：8 published（2 轮编译新产出 6 + 既有 2），月饼素材已編入「美心集團與節日禮品」(38KB, 月餅×75) 与 Corporate Financials 页
- 实体：160（49 published / 108 无状态 / 3 retired），222 条 typed relations、176 条实体-文档反链
- 图谱：349 节点（307 文章 + 42 实体）、158 边
- TOC：7 文件夹干净树形，published 页置顶

### 未尽事项
- 「月饼产品系列」专题页两次 compose 失败（供应商 flake），素材已由邻接页面承载；第三轮编译可再试
- enrich 管线的 wiki_kb_id 仍指向死 KB（无害，resolveKB 绑定优先）

### P9 增量 19 (2026-10-25, framework dc2e72b0 + ui-common b071475) — #125 收口: _rollout/集群广播/实例代理
- **_rollout（分阶段发布）**: RolloutSpec 模型此前从未被消费。reconciler 写行前过 `rolloutPlan` 纯函数门 —— canary 波先写, 波内每个实例"回报运行中且内容匹配"才开下一批, 覆盖全部目标即完成(state.Active=false, 保留 RollbackContent)。**就绪判定是诚实的**: RuntimeSnapshot 新增运行任务的内容摘要(`deployedPipelineMap` 六字段同一 canonical 形态), 经 _sync 以 config_digest 上报, 控制面"推送摘要 vs 回报摘要"对比 —— STARTED 但跑旧内容的 canary 会扣住发布(有专测)。端点 `POST /pipeline/deployments/:id/_rollout {advance|rollback}`: advance 无视就绪强推下一批; rollback 把 RollbackContent 一次写回全部目标(**是刹车不是期望态变更** —— 下一轮 reconcile 会从 canary 重新分阶段推进 desired, 响应里注明并指向 pipeline 版本回滚做永久化)。
- **集群广播(#120 传输面补齐)**: `POST /pipeline/deployments/:id/_pause|_resume|_stop` —— 按 NAME 定位任务(实例侧任务键=配置名), 经 **reverse channel**(agent 主动拨出的 websocket, NAT 后唯一通路)逐实例转发, `?generation=N` 原样搭车 → 异代实例答 409, 聚合响应 total/delivered/逐实例结果, 谁拒绝一目了然。
- **实例代理**: `POST /pipeline/instances/:id/_drain|_cordon|_uncordon`(转发 #121 实例端原语, query 透传 timeout_seconds) + `GET /pipeline/instances/:id/proxy?path=`(/pipeline/ 前缀只读直达, allowlist+traversal 拒绝, 实例状态码原样回传 —— 409 必须还是 409)。反向通道在包缝 `reverseProxyRequest/reverseIsConnected` 可换桩测试。
- **状态面**: DeploymentStatus 增 `rollout` 字段; ui-common ClusterOverview 部署行新增"灰度 x/y"chip(b071475), 补上增量 17 记录的 rollout 进度缺口。
- **测试**: rollout_test(纯函数门 canary/batch/hold/force/prune/complete + 渲染↔实例重建摘要 parity + sqlite 全真 harness 的 收敛→布防→扣住→推进→完成 全程) + fleet_control_test(广播聚合/409 透传/离线不计 delivered/代理 allowlist/ghost 404)。包 -race 全绿 ×3。
- **工程教训**: deployTestEnv 的隔离约定是**所有测试共用部署 ID "d1"**(upsert 语义, 库里永远只有一条部署行) —— 用唯一 ID 会让 env.reconcile() 把别家部署对本 env 管道做 GC, 批量跑必炸; 单跑绿≠批跑绿。测试处理器替身必须真实注册(如 validate_test_echo), 编造 test_ 前缀名过不了 reconciler 的发布前校验。
- P9 状态: #125 闭环。剩余: #128(logpilot StreamSpec 对接 deployment API + cordon 上报)、#130(多实例活体验证: 杀实例锁接管/rollout canary 推进/drain 零丢数)。P8 #115-117 仍发布周期门控。

### P9 增量 20 (2026-10-25, framework 2e4b5b57 + logpilot 7ad6962 + ui-common 9fec2c1) — #128: 流发布走 PipelineDeployment + cordon 上报
- **cordon 上报链 (framework 2e4b5b57)**: 增量 15 注明的"cordon 状态目前实例本地"补齐 —— PipeModule.Setup 经 `configs_client.SetInstanceFlagsProvider` 把本地封闸标志并入每次 runtime 快照, `_sync` 到达服务端后盖到 `Instance.Cordoned`(**placement 照常流动配置** —— 封闸实例上运行任务照跑、新启动被驻停, 配置继续下发才能 drain 收敛); `InstanceReport.Cordoned` 随部署状态下发, ui-common 部署矩阵实例 chip 显 ⊘(9fec2c1)。upsertInstance 的自描述保留语义(服务端字段优先)使解封经第一次心跳 upsert 即回落。
- **RequestReconcile poke (framework 2e4b5b57)**: 包级 `RequestReconcile()` 向 reconciler 循环的 select 投一枚缓冲 poke —— 发布方写完 deployment 立即收敛, 不等 15s tick; 调用永不阻塞, 调和中的 poke 合并为一次后续补跑。
- **logpilot 流发布改道部署声明 (7ad6962, #128 主体)**: `deliverToGateways` 不再直写 ManagedConfig —— 每条编译管线落 `PipelineConfigV2`(ID=管线名, 六字段规范形) + `PipelineDeployment`(strategy 随编译 singleton 标志; 定向=**发布期解析的 gateway 名单**, SPEC §8.5 v1 静态定向, 动态标签选择器留 v2)。分发零自研: reconciler 译行、版本化、GC、灰度、离线补拉全由 framework 收敛; 发布后 poke 立即驱动。关键适配:
  - **suspend/resume 翻 `PipelineDeployment.Enabled`** 而非直改下发行(直改会被 reconciler 下轮按部署声明写回)—— reconciler 对 disabled 部署持续收敛 enabled:false 渲染, 任务停、配置留, 与旧原地改写语义一致; Agent 段照旧直改
  - **teardown 级联**: prune 本流不再产出的部署(keep=nil 全删, lane↔disk 形态切换), 连带管线规范与 reconciler 行; 前缀规则防 streamID 恰为另一 ID 前缀时误删
  - **旧通道迁移**: 发布即删 legacy `stream-<id>-pipelines.yml` 行(两套并存=网关新旧混跑); drift 检测与孤儿清扫双通道识别(deploy 行仍有部署引用时永不判孤儿—— 文件名只是启发式)
  - **幂等版本纪律**: 相同重发布不空转 bump 部署 Version(版本是 UI/广播代际号); 改定向/策略才 +1
  - 送达报告语义: 每管线一行 `deployment vN → N gateways (reconciler)`, InstanceID 承载定向描述(all-gateways/group:x/名单)
- **测试**: logpilot stream_deploy_test 六测(六字段忠实落库/gateway 类型定向 agent 不入列/strategy 映射/legacy 行清理/幂等+改定向版本/级联回收含 reconciler 行/前缀防误删/暂停翻转 Enabled/孤儿判定)。**两处 HEAD 上既有失败**(TestApplyStreamLifecycle_NilSafe、TestBuildKafkaBusYAML, 均并行会话 kafka/storage WIP 区, stash 验证与本增量无关), skip 后全套绿。framework/configs 包 + ui-common tsc 绿。
- **架构确认**: 部署广播(空 placement=全部实例**任意类型**)不能直接替换 logpilot 的 app-type 定向 → 发布期解析成显式名单进 `PlacementSpec.Instances`; 意外收获: framework 的 resolveTargets 对显式 ID 按"已注册实例"解析, 新 gateway 注册后下轮调和自动纳入(比旧直推行——重发布前永远缺席——还自愈)。
- P9 状态: **#128 闭环**。剩余: #130(多实例活体验收: 杀实例锁接管/rollout canary 推进/drain 零丢数/部署通道端到端)。P8 #115-117 仍发布周期门控。

### P9 增量 21 (2026-10-25, framework 15b4249a+f9ae1f17+14872ef8) — #130 多实例活体验收（全绿）+ 3 个真 bug 修复
**拓扑**（复用 10-24 的 gw-e2e 模式）: coco(9000, 重建含新 framework) 作 manager/控制面; 两个 gateway 实例 (/tmp/gw-e2e 2910/2911, /tmp/gw-e2e2 2920/2921, gateway 仓库重编) 注册 coco 配置中心, **singleton 锁走共享 ES KV** (gateway.yml 开 elastic.store.enabled:true; 注意 ElasticStore.Open 需已注册 ORM handler → 实例侧配 sqlite orm 才不 panic, reconciler 查本地空库无害); gw2 走 enrollment 注册+审批。
- **#130-a 部署通道端到端 ✓**: API 建 PipelineConfigV2(node_stamp+echo keep_running) + PipelineDeployment(all, 双实例) → reconciler 写 per-instance 行(ES coco_managed-configs) → 双实例 _sync 落盘 pipeline-deploy-*.yml → 任务运行(逐秒 echo) → runtime 回报 → status ready 2/2。**抓到 bug1**: keep_running 循环两次执行间隙 context 状态是 FINISHED, 快照把 FINISHED→STOPPED — 健康任务每秒被上报"停了", rollout 就绪门(要求 STARTED)会永久卡死 (两实例明明都在跑, ready 0/2)。修复: FINISHED+KeepRunning+未退出 → 上报 STARTED; FAILED 仍报 FAILED(崩溃环必须扣住灰度); STOPPING 也归 STOPPED 而非 UNKNOWN (f9ae1f17)。
- **#130-b rollout canary ✓**: v2→v3 内容升级 + canary:1/batch:1 布防, 5s 粒度轮询捕获完整分波: canary(gw1)先拿新行, gw2 被扣 **45 秒**直到 gw1 回报新 config_digest 后才开第二波 (06:06:26 布防 → 06:07:02 gw1 跑 v3 → 06:07:32 完成)。诚实 digest 门活体可见。
- **#130-c singleton 锁接管 ✓**: singleton 部署双实例 → gw1 当选(gw2 持配置 standby)。kill -9 gw1 → **60s 后 gw2 接管**, 日志 `singleton lock taken over from instance gateway-e2e-live (lease expired)`(#122 事件带前任名字)。gw1 重启后**不回抢**(锁被 gw2 新鲜持有), 全程无双跑。**抓到 bug2**: 部署状态 LastSeen 用读取时刻 time.Now() — 死实例永远显示"刚上报过"(被杀的 gw1 比接管者看着还新鲜)。修复: 用 Instance.Updated 真实心跳时间; rollout 就绪门另加 90s 心跳新鲜度(冻结的 STARTED+digest 不得永远算就绪) (14872ef8)。**另加固**: 部署 CRUD handler 也 poke reconciler, 收敛不等 15s tick (15b4249a)。
- **#130-d drain 零丢数 ✓**: consumer 管线(部署 gw2) + `_inject` 产数据。**教训**: ①队列消息须是 {"messages":[...]} 批次形态(message_field 契约), 裸对象会被抓取但零记录流出; ②队中途重建会留毒游标(consumer group offset 在本地 KV, 队列 ID 由名字派生而复用) — 换 group 名即新游标; ③处理链要放可观测处理器(echo), 否则 for_each+mutate 零输出无法对账。正式流程: 注 250(50+200) → 注入后 2s 立即 _drain → 优雅停机(STOPPED, 未抓消息安全驻留队列) → _start 恢复 → **恰好 250 条处理记录, 零丢零重**。
- **logpilot #128 侧活体验证 ✓**: logpilot(29000) 重编重启(sqlite ORM), 第三实例 gateway-e2e-lp(2930/2931) 经 enrollment ticket(et- 前缀)+审批注册。POST /logpilot/streams/:id/publish 显式定向 → 交付报告即新语义(3 部署行: main/-merge/-sink 各 `deployment v2 → 1 gateways (reconciler)`) → poke 收敛 → 实例拉 3 个 pipeline-deploy-*.yml → 三任务 STARTED → **runtime+config_digest 回报 logpilot 实例记录**。附带发现: 重启后 logpilot 回到仓库 data/ 的 sqlite(旧进程 cwd 不同, 流 ID 不同); deliverSinkCluster 反向推 /easysearch/ 404(gateway 二进制无该路由) — sink 集群下发通道在 gateway 型实例上不通, 记为遗留问题。
- **环境事实**: 部署广播不适用产品定向(上增量已证); kv.Register 最后注册者胜(badger<elastic 字母序, elastic.store 开启即共享锁); 队列 depth 是写入计数不减; gateway 仓库 e356f04 起带 managed 模块(实例侧端点)。
- P9 状态: **#130 活体验收全绿, P9 收官**。修复 3 个真 bug(快照误报/假 last_seen/就绪无新鲜度)+1 加固(CRUD poke)。遗留: deliverSinkCluster 对 gateway 二进制 404; logpilot 重启 cwd 漂移致后端切换(建议 path.data 绝对化)。P8 #115-117 发布周期门控。

### P9 增量 22 (2026-10-25, framework 0f84de33 + gateway 0a2e18a + logpilot 3ed2d70) — 增量 21 两遗留收口
- **遗留1 gateway /easysearch/ 404 修复**: 根因是 easysearch 模块默认关 (`enabled` 缺省 false)——gateway 注册它就是为了接 logpilot 的 sink 集群推送, 但 yml 无 `easysearch:` 段时静默不挂路由。修复: 模块加 `EnabledByDefault` 逃生舱 (yml 无该段时由注册方决定默认开), gateway 注册时置 true。**活体验证全通**: 重编 gateway 重启 → 路由挂上(注意: crud 路由挂在 **web 端口**, 反向通道 loopback 恰好偏好 web 口, API 口 400/404 是预期) → 重发布流 → "sink cluster delivered to gateway" → 集群落 gateway sqlite → `bulk_indexing_lane` 的 "cluster metadata not ready" 告警归零 → 实际连上 9205 发 bulk (剩余 2 条 doc 级报错是目标索引映射问题, 与通道无关)。
- **遗留2 logpilot cwd 漂移**: 找回旧数据目录在 `bin/data/`(旧进程从 bin/ 启动), 现行在仓库根 `data/`。框架级"相对 path.data 锚定配置文件目录"会迁移存量部署数据, 风险不可接受——降级为 logpilot.yml 警示注释(相对路径按进程工作目录解析, 生产用绝对路径) + 运维约定(开发机固定仓库根启动)。3ed2d70。
- **附带确认**: otlp_export 重试耗尽 panic 仅 `-debug` 模式杀进程(consumer worker 的 recover 在 debug 分支被跳过, 设计如此); 非 debug 恢复并标任务 FAILED。测试实例补 otlp intake 监听后稳定。
- P9 状态: 两遗留闭环。P8 #115-117 发布周期门控。

### P9 增量 23 (2026-10-25, framework 1e2ccfe4 + coco 95d6b7dd) — #129 产品侧绑定: coco web 接入 ClusterOverview 舰队视图
- **绑定**: 管道列表页新增页签对(管道列表/集群部署), 集群页签渲染 ui-common `ClusterOverview`(file: 源引用): 部署矩阵(策略/版本/就绪数/每实例状态含 PENDING/FAILED/stale)+集群阶段指标(计数/错误/加权时延/每实例明细), 10s 自刷; 管道名点击跳转 `/pipeline/edit/:id`(实测 edge_heartbeat 跳转成功)。数据走 framework 端点 `GET /pipeline/cluster/overview` + `/stages`, 适配层在 `service/api/pipeline.ts`(flat request → 组件契约: 成功取 .data, 失败抛错内联展示)。
- **权限门设计迭代**: 初版 `hasAuth('generic#system:pipeline/read')` 门控页签 — 活体一打就隐身。根因: 后端 permission filter 对 RoleAdmin 全放行, 但 admin 的 permissions 列表(156 项)不含 system:pipeline 键; profile/login 响应也不带 roles, 前端无法预判 admin。改为**不门控**: 页面路由已有 generic#pipeline/search 兜底, 无权限用户在页签内看到内联报错(诚实可见, 不藏功能)。
- **file: 依赖刷新链路(坑)**: coco web 的 `@infinilabs/pipeline-ui` 是 pnpm file: 源引用, store 拷贝停在旧版(无 ClusterOverview 导出) → typecheck 报 TS2305。`pnpm install` 刷 store(lockfile 不变)。**vite 侧连带坑**: dev server 的 optimizeDeps 缓存按 lockfile/config 哈希续用, file: 包内容变化不触发重打包 → 运行时 "does not provide an export named 'ClusterOverview'"。试过 optimizeDeps.exclude(破坏 @xyflow/react 的 CJS 传递依赖 use-sync-external-store, 弃用) — 最终解法: 删 `node_modules/.vite/deps` 缓存 + 任何 vite.config 触碰引发**同进程自动重启**(端口不断, 未杀用户 dev server), 重启后从新 store 重建缓存。vite.config.ts 已还原零 diff。
- **活体验证揪出 framework 两个真 bug(均已修+测试+推送 1e2ccfe4)**:
  1. `installStats` 注册观察者的 statsRecorder 实例与 `globalStats` 存的是**两个不同对象** — StepEvent 计数进 A, stageStats 读 B, 全部阶段视图(runtime 端点/持久化/集群聚合)永远空。单测没抓到是因为直接喂存的实例, 绕过了观察者接线。修复: 单实例双角色 + 新增 `TestStats_ObserverWiring`(跑真实 Processors.Process 链断言计数可见, 旧代码必红——stash 验证过)。
  2. `persistStageSnapshot` 用裸 orm context 保存 — security 数据操作钩子要求会话用户或直通旗标, 第一个快照 tick 就 panic("invalid user") **把 coco 整个进程打死**(复现: 开 pipeline_metrics.persistence 10 秒后 crash)。修复: 后台系统写者用 `DirectAccess()`(pre-hook 对 Get 认 DirectRead, post-hook 对 Save 认 DirectWrite)。
- **验证路径**: 修后重编双二进制重启 — gateway 运行时端点回真实计数(echo/node_stamp 55+), coco 开 pipeline_metrics.persistence(coco.yml **本地临时开启, 未提交**)活体落 ES `coco_pipeline-stage-stats` 且进程存活 → `/pipeline/cluster/stages` 回聚合行(connector_dispatcher 43 次/0 错/6.1ms/每实例明细) → UI 集群页签双表齐活(截图存档)。typecheck gate 598=基线, locale 对齐。
- **新遗留(记入下增量候选)**: gateway 型实例的 stage 快照落**本地 sqlite**(默认 ORM 指向), 不入 manager 的 ES — 跨实例阶段聚合目前只含 coco 自身管线; 需要 stats 随 _sync 回报或实例侧配 ES store, 属 framework 协议层工作(候选 #131+)。测试拓扑残骸: gateway-e2e-test 实例行仍在(死实例, 状态如实 PENDING/心跳过期), enrich_documents 部署对 gateway 二进制报 FAILED(file_type_detection 是 coco 插件处理器, gateway 无——如实展示, 不修)。
- P9 状态: **#129 闭环**(产品侧舰队视图上线)。P8 #115-117 发布周期门控。

### P9 增量 24 (2026-10-25, framework 891adbc0) — 阶段指标随 _sync 回报：跨实例聚合闭环（增量 23 遗留收口）
- **问题**: gateway 型实例默认 ORM 指向本地 sqlite, 阶段快照永远到不了 manager 的 ES — `/pipeline/cluster/stages` 只看得见 coco 自身管线, 产品舰队视图的阶段表残缺。
- **方案**: 计数器搭既有心跳班车, 不开新通道、不动实例存储配置:
  - 协议: `RuntimeReport` 增 `stage_stats`(`common.StageStat` 线形态 — pipeline/stage/carrier/count/errors/avg_latency_ns/instance_id, 累计值);
  - 实例侧: PipeModule.Setup 注册 `SetStageStatsProvider(module.reportedStageRows)` — 处理器行按 `Config.Name` 归属(与 RuntimeSnapshot 同名解析规则, 同名 context 去重), lane 行全局一份不按管线复制(顺手修了 persistStageSnapshot 原来每管线复制全部 lane 的错归属);
  - manager 侧: configs server 增 `SetStageStatsSink` 钩子(PipeModule.Setup 注册, 无 configs server 的进程惰性), 回报落本地 store, 与本地导出器同行同表同"每实例最新一拍"折叠;
  - 保留策略: `pipeline_metrics.persistence.retention_seconds`(默认 1h, 负值关闭), 两条写路径共同裁剪 — 折叠只读最新一拍, 无界历史迟早挤爆聚合查询(Size 4000)。
- **测试坑**: 测试包里第二个 `orm.Register` 触发 last-wins 全局翻转, 后跑的 rollout 测试 schema 缺失而红 — 重构为无副作用 `ensureDeployORMHandler`(单 handler 全部 schema, stats 测试不再造实例污染广播计数)。另: `-count=2` 下 flight/for_each 三测红是 **HEAD 既有**(全局环形缓冲非幂等, stash -u 验证), 与本增量无关, 正常 `-count=1` 门绿。
- **活体验证**: 双二进制重编重启 → gateway `_sync` 携带 边缘心跳 echo/node_stamp 累计行落 coco ES(两拍: 30→59 递增) → 集群端点跨实例聚合 → **产品 UI 集群页签三行齐活**: coco 自身 connector_dispatcher 18 次 + gateway 边缘心跳 echo/node_stamp 各 178 次, 每实例明细可对账(合计=各实例之和)。部署矩阵如实: singleton 1/1 STARTED, enrich 0/2(gateway 二进制缺 coco 插件处理器, FAILED 状态诚实展示)。
- P9 状态: 增量 23 遗留闭环, 舰队观测(矩阵+阶段)跨实例完整。P8 #115-117 发布周期门控。

### P9 增量 25 (2026-10-25, logpilot 3c1e1c5) — logpilot 部署面板收敛到共享 ClusterOverview（两产品舰队视图同源闭环）
- **背景**: #129 共享组件此前只有 coco 消费; logpilot 的 DeploymentsPanel 自带手搓观测层(健康点/实例着色/失败原因, N+1 逐部署 status 拉取), 且无集群阶段指标入口 — 同一视图两份实现, 违背 P7 同源精神。
- **收敛**: 面板顶部渲染共享 `ClusterOverview`(部署矩阵含灰度进度 + 集群阶段指标表, 10s 自刷), 底部管理列表瘦身只留独有价值(新建/编辑/启停/删除 + 范围摘要); 手搓健康/实例/失败原因行与 N+1 status 循环全删(一次 cluster/overview 顶替), 净 -36 行。`t` 注入 logpilot 自身 i18n(en 缺失键按设计回退中文, 翻译流水线后续补齐, 不手改 locale 文件)。
- **连带运维**: logpilot web 的 file: 依赖同样停在旧包(pnpm install 刷新), 其 vite dev server(39000, 非保护)带 `--force` 重启即重建依赖缓存 — 无 coco 那轮 optimizeDeps 陈旧坑。logpilot 后端与 gw-e2e-lp 实例重编重启吃到增量 23/24 修复。
- **活体验证**: logpilot 部署页签矩阵 3 部署 singleton v2 1/1(gw1-fixed 陈旧行如实展示, gateway-e2e-lp 新鲜 STARTED); **阶段指标首次在 logpilot 可见** — gw 实例经增量 24 的 `_sync` 回报落 manager store: 处理器行 for_each/node_stamp/output/source_enrich 各 1092 次 + otlp_export/consumer + lane 载体(system-metrics-lane 1092 次), 每实例明细对账齐活。tsc 0 错。
- P9 状态: 两产品(coco/logpilot)舰队观测完全同源共享包, 部署矩阵+阶段指标跨实例闭环。P8 #115-117 发布周期门控。
