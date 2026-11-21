# 统一 Pipeline 重构 — 状态与交付文档

> 2026-10-15 | 全部阶段完成，六仓提交，20 测试包全绿

## 架构总览

```
              跨进程/需持久                 进程内实时                  链内
          ┌────────────────┐        ┌────────────────┐      ┌────────────────┐
 概念     │  Queue 队列     │        │  Lane 通道      │      │  Context 作用域 │
 载体     │  bytes         │ codec  │ *event.Event   │ 指针 │ *event.Event   │
          │  Message.Data  │◄──────►│ 指针(记录即消息) │◄────►│ (原地修改)      │
 处理器   │ consumer       │        │ lane_consumer  │      │ for_each 内的   │
          │ output(queue)  │        │ output(lane)   │      │ 全体处理器       │
          └────────────────┘        └────────────────┘      └────────────────┘
 领域层:  Document/Attachment = codec 投影 (document.* / attachment.*)
 观测:    dry-run / metrics / logs = StepEvent 三投影 (一次埋点)
 分布式:  Deployment → reconcile → managed config → instance sync → runtime report
```

## 各阶段交付

### P1 记录模型
- `core/pipeline/codec.go` — RecordCodec 注册表（Namespace / error 注册 / OrReplace / 发现）
- `coco/core/codecs` — coco 领域 codec 包（coco_document / coco_attachment；原独立模块 `infini.sh/coco-codecs` 已收编回 coco core，D-2 撤销）
- `coco/core/documentview` — Materialize/Project（Fields ↔ Document）
- `core/pipeline/observer.go` — ExecutionObserver + StepEvent（一处埋点三投影）

### P2 契约与作用域
- `core/pipeline/contract.go` — RecordContract{Scope, Namespace, RequiredFields, Granularity}
- `NewPipelineWithScope` — 编译期放置校验
- 段式执行 — 书写顺序 = 执行顺序（废弃两阶段）
- Catalog 透出 scope + record_contract + granularity

### P3 双模迁移
- 15 个处理器 + 26 个连接器 + 1 webhook 全部转换
- `process_documents` mode:records + TTL 编译缓存
- 每处理器带双模测试（跳过路径等价）

### P4 性能
- L2 冷热分离（document_chunk RawMessage，向量零解析）
- for_each.parallel（记录级并行，~5×）
- processHistory O(n²) 修复（串行 6.5×）
- indexing_merge 部分解析（jsonparser 零分配 _id 提取）
- jsoniter 热路径（行为兼容门禁，两个 nil 分歧守卫）

### P5 Lane 通道
- `core/lane` — 有界通道 / block-drop-spill 背压 / 动态 Create-Delete
- AckSet 提交门 — 跨 lane at-least-once（offset 借自上游队列）
- lane_consumer 宿主 — 攒批 → 段式子链 → output_queue 桥
- 统一 output — one-of 目标字段（queue/lane + ack）
- tap 处理器 — 旁路采样复制

### P6 观测与调试
- `modules/pipeline/dryrun` — 统一 dry-run（快照/字段 diff/副作用截获）
- `POST /pipeline/dry-run` — 统一端点（唯一定义，产品端点删除）
- Flight 飞行记录 — 环形 1024 / 错误全采 / OK 1/100
- StageStats 聚合 + `/stages` 端点
- `/pipeline/lanes` / `/pipeline/codecs` 发现端点
- `_pause` / `_resume` / `_inject` 操控端点
- `pipelinelog` — trace_id 前缀日志关联
- PipelineVersion — 自动快照 / 回滚 / transient 跳过
- PipelineStageStat — 指标持久化

### P7 共享 UI
- `ui-common/packages/Pipeline` — @infinilabs/pipeline-ui
- 4 组件（ChainEditor / Runtime / LanesPanel / InjectDialog）
- 全部 props 驱动，零 API 依赖

### P9 分布式
- PipelineDeployment 模型 + Fleet API（7 端点）
- 调和循环 → configs-center 推送（managed config 行）
- 实例定向（PlacementSpec.Instances）
- 实例运行时回报（_sync Runtime 字段）
- DeploymentStatus + cluster overview

### Logpilot 集成
- DeploymentsPanel — 部署管理 UI（创建/启停/删除/健康）
- PipelineDetail — 三 tab 详情（指标轨迹/配置链/版本历史+回滚）
- LanesPanel — 通道深度可视化
- InjectDialog — 样本注入
- trace_id 日志关联（flight → 日志过滤）
- 统一 dry-run 切换 + 样本 API 产品化 + 旧端点删除
- 处理器面板带中文描述（双数据源：实例目录 + 文档层）

## API 端点清单

### 统一端点（framework，三产品共享）
```
GET  /pipeline/codecs                        codec 发现
GET  /pipeline/lanes                         通道观测
GET  /pipeline/processors                    处理器目录（+scope/granularity）
POST /pipeline/dry-run                       统一 dry-run
POST /pipeline/ai-generate                   AI 生成链（ChainGenerator）

GET  /pipeline/task/:id/stages               阶段指标
GET  /pipeline/task/:id/flight               飞行记录
POST /pipeline/task/:id/_pause               暂停（批间挂起）
POST /pipeline/task/:id/_resume              恢复
POST /pipeline/task/:id/_inject              样本注入

GET  /pipelines/:id/versions                 版本历史
GET  /pipelines/:id/versions/:v              单版本详情
POST /pipelines/:id/versions/:v/rollback     回滚

POST /pipeline/deployments/                  创建部署
GET  /pipeline/deployments/                  部署列表
GET  /pipeline/deployments/:id               部署详情
PUT  /pipeline/deployments/:id               更新
DELETE /pipeline/deployments/:id             删除
GET  /pipeline/deployments/:id/status        部署健康
GET  /pipeline/cluster/overview              集群总览
```

### Logpilot 产品端点
```
GET  /logpilot/processors                    处理器目录（+中文描述）
GET  /logpilot/streams/:id/samples           流样本（产品域）
```

## 五仓提交状态

| 仓库 | 分支 | 提交数 | 状态 |
|---|---|---|---|
| framework | refactoring-pipeline-oct | 2890+ | ✅ clean |
| coco | feat/knowledge-hub | 929+ | ✅ clean |
| logpilot | main | 174+ | ✅ clean |
| gateway | main | — | ✅ clean |
| ui-common | main | — | ✅ clean |

## 测试状态

| 仓库 | 测试包 | 通过 | 失败 |
|---|---|---|---|
| framework | 7 | 7 | 0 |
| coco | 13 | 13 | 0 |
| **合计** | **20** | **20** | **0** |

## 关键决策记录（14 条）

| # | 原方案 | 最终方案 | 原因 |
|---|---|---|---|
| 1 | dry-run 产品端点薄壳 | framework 统一端点 | 不需要 wrapper |
| 2 | 画布移植到 coco | @infinilabs/pipeline-ui 共享包 | 单一实现 |
| 3 | 内存队列 | Lane 通道 | 名字承诺契约 |
| 4 | relay 双词 | lane 单词 | 一个概念一个词 |
| 5 | spill 默认 | block 默认 | spill 绕过必需步骤 |
| 6 | 批量两阶段 | 段式执行 | 批处理器看不到前段修改 |
| 7 | relay_in 平铺 | 宿主形态 + output_queue | 平铺=无批量 |
| 8 | queue_output 在 consumer 外 | 必须在子链内 | offset 语义 |
| 9 | lane 无提交语义 | AckSet 提交门 | 保证借自 offset |
| 10 | lane 静态配置 | 动态平面 + 隔离 | 动态管道需求 |
| 11 | 无 granularity | 三档声明 | 配置/UI 清晰 |
| 12 | 新造调度器 | 叠加 configs-center | 协议全现成 |
| 13 | 双出口处理器 | 统一 output + one-of | 语法收敛 |
| 14 | tee Unix 行话 | tap 观测标准词 | 契约进名字 |

## 后续路线

| 项 | 优先级 | 说明 |
|---|---|---|
| Coco UI 接入 | 高 | 消费 @infinilabs/pipeline-ui |
| P8 收尾 | 中 | 稳定周期后：audit/migrate/删旧路径/默认值翻转 |
| P9 深化 | 中 | 分批 rollout / 标签选择器 |
| 101b | 低 | flight 错误落盘 + ?window= |
