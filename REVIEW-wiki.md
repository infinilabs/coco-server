# Wiki 功能 Review 清单

日期: 2026-10-23 · 分支: feat/knowledge-hub · 基线: a9233342

## 一、总体状态

| 维度 | 状态 |
|---|---|
| 后端规模 | modules/wiki 60 文件，45 自定义路由 + 9 套 CRUD（kb/article/entity/bookmark/like/comment/notification/workspace/version 等） |
| 定时任务 | 编译 Agent（WIKI_COMPILE_AGENT_INTERVAL）、经验蒸馏（WIKI_DISTILL_INTERVAL）、新鲜度巡检（WIKI_FRESHNESS_INTERVAL） |
| 前端规模 | 8 个路由页面 + KB 详情 8 页签；45 路由中 38 个有 UI 消费方，7 个无 UI |
| Go 测试 | modules/wiki 除 1 个既有失败（TestWikiFlow_EntityLifecycle）外全部通过 |
| 数据一致性 | ES 各索引计数此前核验一致（articles/entities/versions/evolution/governance 全部可解析） |
| 权限 | 45/45 路由 RequireLogin；42/45 带 RequirePermission，3 个 login-only（见 §10） |
| i18n | zh/en 双语齐全，内部术语已清理（adcef5b9），新面板类型标签全部本地化且未知值回退原值 |

状态图例： ✅ 可用 · ⚠️ 有问题 · 🔲 后端有、无 UI

## 二、功能清单（按域）

### 1. 知识库管理
- ✅ KB CRUD / 列表 / 搜索
- ✅ 可见性 public/private/team（设置表单）
- ✅ 数据源绑定；助手绑定（经验捕获链路 KbSettings.assistant_id）
- ✅ 成员 / 分享（Shares 组件，owner/editor 语义）
- ✅ 自动审批开关（omitempty 修复后可开可关，往返验证过）
- ✅ 经验捕获开关（同上）
- ✅ 企业目录布局 toc_layout（部门顶层目录 + 维度槽位）
- ✅ TOC 读取/更新 + 三形态自愈（孤儿 append、空白重建、悬空 prune；≤1000 篇 guard；文章计数同步校准）

### 2. 文章
- ✅ CRUD / 状态流转（draft→reviewed→published→archived，审核门在 PUT status）
- ✅ 版本历史 + 详情 + diff（管线写入自动快照，REFINE 重写也可回滚）
- ✅ 结构化 Markdown / 双链 [[type:name]] / 引用 chips
- ✅ AI 生成（SSE 五阶段：圈语料→规划→大纲→撰写→落库）
- ✅ AI 编辑（选中片段/全文）
- ✅ 整页重写 REFINE（a9233342 本轮接入，无 LLM 时降级提示）
- ✅ 从聊天保存文章（SaveToWikiModal → _from_chat）
- ✅ Lint 提示条（thin_content / no_sources）
- ✅ 材料（_materials）与附件（_attachments）面板
- ✅ relink：未解析链接批量创建待审实体并重绑
- ✅ 评论 / 点赞 / 收藏

### 3. 实体
- ✅ CRUD / 分页搜索 / 批量状态；状态标签本地化（published→已发布，本轮）
- ✅ 实体合并（_merge：别名吸收、关系并集、引用重定向）
- ✅ 邻居 / 反链；实体悬浮卡（EntityHoverCard）
- ✅ KB 作用域隔离；锚定实体 ID（kbID+name）
- ✅ 本体 schema（get/put + 默认种子 6 类型）+ 本体词表页面（/ontology）
- ✅ 图谱页签（KbGraph，支持 ?entity= 深链聚焦）
- 🔲 entity/lookup 无 UI（候选：编辑器 wikilink 自动补全）
- ⚠️ TestWikiFlow_EntityLifecycle 失败（详见 §三.1）

### 4. 治理
- ✅ 提案队列（open/resolved/dismissed/all + KB 筛选）
- ✅ 单条应用 / 驳回；批量应用（带进度轮询）
- ✅ 聊天侧纠错提交（_correction）
- ✅ 自动审批 sweep（actor=auto-approve，进化日志可审计）
- ✅ 知识缺口：评估集 miss / 零命中 ≥3 次 → gap 提案进队列

### 5. LLM-wiki 闭环（P1/P2）
- ✅ 经验捕获：chat vote / correction / gap，KB 级 opt-in
- ✅ 经验记录列表 UI + 手动「蒸馏为提案」（a9233342 本轮）
- ✅ 进化日志 UI（append-only，6 类标签本地化；109 条实测渲染）
- ✅ 编译 Agent：定时 + 手动触发 + 报告弹窗（轮次/补丁/结论）
- ✅ 整页 REFINE 重写（与编译产物同管线）
- 🔲 新鲜度 sweep 完全黑盒（文档变更→页面再生→通知，无任何 UI 可见性）
- 🔲 技能闭环：_propose（技能提议）/ _skill_md（导出）/ _import_skill（导入）均无 UI
- 🔲 影响记录（RecordKnowledgeHits / impact policy）内部统计，无展示

### 6. 检索集成
- ✅ 混合搜索（RRF）为默认；wiki 分支 + 图谱分支 + 查询改写；无 LLM 自动降级关键词
- ✅ Wiki 投影开关 + 「运行投影回填」按钮（a9233342 本轮）
- ✅ 块级检索分支（chunk_route）
- ✅ WikiBadge：搜索结果标识策展内容（widget 侧）
- ✅ 检索实验室（分支权重/融合公式/试跑，文案已清理）
- ✅ 检索运营（零命中/低召回榜/策略分布/改写统计 + 演示级告警）
- ✅ 评估集（金标题用例，跑分趋势，miss 自动提交 gap 提案）

### 7. 文件 / 原材料
- ✅ KB 文件列表 / 上传（KbFileManager）
- ✅ 编译控制台（触发 / 进度 / 日志 / 取消，断线重连轮询）
- 🔲 consolidate（重复文章整理）无 UI

### 8. 通知
- ✅ 通知列表 / 标记已读（WikiShell 铃铛 + notifications 页）

### 9. 安全 / 权限
- ✅ 45/45 RequireLogin
- ✅ 42/45 RequirePermission，前端 hasAuth 门控与后端一致（新按钮逐一对照）
- ⚠️ 3 个 login-only 路由：experience/_vote（设计使然：不暴露捕获 KB 存在性）、company-map（读）、governance/_correction（聊天侧任意登录用户提交）——前两个合理，_correction 建议复核是否需要限流/权限

### 10. i18n / 文案
- ✅ zh/en 双语；术语清理完毕（腿→分支、(D8)、内部端点名、raw 枚举）
- ✅ 新增进化/经验/Agent/重写/回填文案双语同步

## 三、已知问题

1. ✅ **TestWikiFlow_EntityLifecycle 失败**（flow_test.go:289，既有，已在干净 HEAD 复证）：文章双链 `[[campaign:Autumn Promo]]` 创建后 store 实体 Relations 为空。测试直接 orm.Create 实体且未带 kb_id，与后来"实体按 KB 作用域"的改动脱节——要么测试补 kb_id，要么 entity_propagation 的跨库解析需要复核（涉及 B3 mentions 边传播）。
2. **coco.yml 本地改动**（override_exists_mapping: false→true）未定夺提交或回退。
3. ✅ 测试日志噪音：liaison 实体撤回扫描报 `no such table: wiki-entity-graph`（WRN，测试环境建表时序问题，不致失败）。
4. ✅ 预存 typecheck 噪音：widgets 工作区 51 个（双 @types/react 冲突）+ app 层 auto-import 相关报错；本轮改动零新增。

## 四、待办事项

**P0（正确性）**
1. ✅ 修复 TestWikiFlow_EntityLifecycle：先给测试实体补 kb_id 复跑；若仍空 Relations 则排查 entity_propagation 的 KB 过滤逻辑。
2. 定夺 coco.yml 的 override_exists_mapping 改动（提交或回退）。

**P1（闭环补全，后端已就绪只差 UI）**
3. ✅ 新鲜度可观测性：KB 概览或进化页签展示 freshness 统计/水位（changedDocs/regenerated/notified），至少把 sweep 结果写入进化日志或治理提案。
4. ✅ 技能闭环 UI：文章工具栏「导出 SKILL.md」、KB 设置「导入技能」、手动触发技能提议按钮（结果已进进化日志）。
5. ✅ company-map 公司地图视图（图谱页签内或独立页）。
6. ✅ consolidate 入口：文章列表批量「整理重复页」（带确认与结果报告）。
7. ✅ entity_lookup 接入编辑器：wikilink 输入自动补全目标实体。

**P2（打磨）**
8. ✅ 进化日志行内跳转：detail 里的 article_id/proposal 链接化，点击直达文章/治理详情（治理侧带 focus 高亮定位）。
9. ✅ 经验记录展示 retention/过期状态；蒸馏定时器（WIKI_DISTILL_INTERVAL）当前值与最近运行时间可见（经验面板 Alert + 过期列标红）。
10. ✅ 编译 Agent 报告 notes 结构化（列表化而非分号拼接）。
11. ✅ 文章页展示 impact 命中统计：核实已有（文章页头部 impact pill，answer_hits）。
12. ✅ 复核 governance/_correction 的权限与限流：login-only 属设计（匿名纠错更易被刷）；补每用户 20 次/小时滑动窗口限流 + 标题按 rune 截断 + 测试。
13. ✅ en/zh locale key 对齐校验脚本 `pnpm locale:check`（web/scripts/check-locales.mjs），并修复全部 165 处历史漂移，当前零漂移。
14. ✅ 清理预存 typecheck 噪音：归因完毕——607 vs 基线 598 的 +9 全部来自依赖漂移（widgets/ui-search 自带 @types/react 双解析等），本轮 wiki 提交 set-diff 证实 0 新增；已偿还 9 处真实错误（ignoreError 类型声明 + i18n count 传 number），gate 回到 598=基线。widgets 双 @types/react 的根治（约 30 处）属独立重构项，未纳入本轮。
