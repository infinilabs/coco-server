# DESIGN: 编译 Agent（EINO）— 从一次性 MapReduce 到持续迭代的知识编译循环

> 状态: draft → 实施中
> 决策: 选项 A —— 引入 CloudWeGo EINO（LogPilot 同款，`eino v0.9.12` + `eino-ext openai`），
> 只对"知识编译"子系统先换；assistant 的 deep think/research 维持 langchaingo。
> 前置阅读: DESIGN-llm-wiki.md（三层知识架构与治理闸门）

## 1. 为什么换

一次性 MAP→REDUCE→PLAN→WRITE 的结构性局限（实测暴露）：

| 问题 | 机制原因 |
|---|---|
| 标题与内容错位（"API Authentication Tokens" 装着 Yuque 内容） | PLAN 一次性命名，更新优先合并继续往错标题堆节 |
| 摘要缺失/跑偏 | WRITE 只渲染缓存语句，从不回看成品 |
| 证据面窄（成熟度 1.0 只引 1 源） | 单遍读取，无"再回去拿内容"环节 |
| 无自检 | 写完即止，没有"引用是否支撑、内容是否答题"的评审 |

这些全部是 Agent 循环（评估→定向取材→写→自检→再取材）的教科书场景。

## 2. 架构

```
定时(30m, env 可关) / 手动 POST /wiki/kb/:id/_agent_compile (MCP: run_compile_agent)
        │ 每 KB 顺序执行，每轮 ≤ compileAgentMaxIters(3) 次迭代，单迭代 ≤ MaxStep(8) 步
        ▼
┌─ EINO ReAct Agent（每轮新建，KB 范围闭包工具）────────────────────────┐
│ 模型: coco 默认语言模型（OpenAI 兼容 provider → eino-ext openai）      │
│ 工具:                                                                │
│   search_documents(query,size) — KB 绑定数据源 BM25                  │
│   read_document(doc_id)        — 全文截断                            │
│   list_topic_pages()           — 专题页清单(标题+摘要+更新时间)        │
│   read_topic_page(title)       — 现有页全文+大纲                     │
│   list_entities()              — 本体实体清单                        │
│   read_evolution_log()         — 近期决策(含被拒)                    │
│   propose_page_patch(title,rationale,content) — 登记 compile_patch   │
│     提案（开放中同题去重；页不存在时提案改名/新建）                      │
└──────────────────────────────────────────────────────────────────────┘
        │ 产物：compile_patch 治理提案（人工 / 自动审批闸门不变）
        ▼
   治理队列 → apply（整页替换 + AI 版本快照 + lint + 投影）
        │ 结果与 no_action 全部写入进化日志（kind=compile_agent）
        ▼
   影响计数 → 下一轮 agent 通过 read_evolution_log / Impact 看到效果
```

## 3. 关键决策

- **每轮只动一页（原子迭代）**：与论文提议者一致——评审面小、可回滚、可解释
- **产物走治理提案而非直写**：闸门（人工/自动审批）与影响追踪复用现有机制
- **进化日志必读**：工具化暴露给 agent，避免重复提交被拒的修改
- **no_action 是一等结果**：agent 判定无需改动时如实记录，禁止硬造
- **模型边界**：v1 仅支持 OpenAI 兼容 provider（coco 默认语言模型解析）；非兼容
  provider 回退一次性编译管线。EINO 引入边界 = 编译 Agent，不动 assistant
- **预算**：每轮 ≤3 迭代 × ≤8 步；调度间隔 30m（`WIKI_COMPILE_AGENT_INTERVAL`，
  `off` 停用）。后续按 KB 加每日预算

## 4. 与一次性编译的关系

MapReduce 编译（`_compile`）保留：冷启动/全量重建仍用它。Agent 是其上的**持续
维护层**——语料变更后由 freshness/实体传播/定时触发增量修缮。两者产出同一治理
提案通道，页面状态机/版本/lint/投影全部共用。
