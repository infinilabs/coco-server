# DESIGN: LLM Wiki 完整方案（声明性知识 + 程序性技能 + 经验进化闭环）

> 状态: draft（v2，合入第二轮学习）
> 参考:
> - WikiSkill (Google Research, arXiv 2608.27454): 把 agent 经验编译进持久化知识库，技能与 wiki 共同进化
> - kfchou/wiki-skills: Karpathy "LLM Wiki" 提念的 Claude Code 实现（wiki-init/ingest/query/lint/update/audit/merge 七技能，schema+生成式 index+backlink+git 日志）
> - developersdigest《Wiki Skills: The Missing Graph Layer in Agent Context》: Agent Skills 规范缺图层——links/reach-for/loads 三字段提案
> - santhoshtr/wiki-skills: ⚠️ 同名不同物——Wikimedia 领域的 Agent Skills 包（Wikipedia/Wikidata/MediaWiki 工具技能），不是"LLM 维护的 wiki"；但其 SKILL.md + references/ 打包格式是事实标准
> - HuggingFace 论文页与社区复现: 门控价值与诚实记录的实证
> - Coco 现有 Wiki 模块: `modules/wiki/`、`core/wiki.go`、`web/src/pages/wiki/`

## 0. 第二轮学习的增量结论（v2 新增）

### 0.1 论文补遗（HF 页面摘要校正）

- **技能进化与模型规模互补**：越大的模型从进化技能中获益越多；带技能的小模型可以大幅胜过无技能的大模型。对 Coco 的含义：我们大量服务本地/小参数模型，**技能层对小模型的增益正是最需要的场景**
- **跨模型/跨助手迁移**：技能发现与技能执行是两种能力——A 助手进化的技能可让 B 助手受益（超出原设计的"跨 KB"，扩展到"跨 assistant"）
- 社区复现的两条实证（ashutoshsinghpr7/wikiskill、Stahl-G/wikiskill）：验证门控真实拦下过有害技能并回滚；有一轮诚实产出"无技能可学"（no_action）并记录而非硬造——**进化日志需要 no_action 作为一等结果**

### 0.2 图层缺失（developersdigest 文章，对 Skills 层设计影响最大）

Anthropic Agent Skills 的三级渐进披露（name/description → SKILL.md 正文 → 附带文件）解决了"按需加载"，但技能之间**没有边**——是平的卡片目录，库越大一级税（常驻 system prompt 的 name+description）越重。文章提案给技能页加三个可选 frontmatter 字段：

| 字段 | 语义 | 映射到 Coco |
|---|---|---|
| `links` | 指向其他技能的边表；**红链（未解析）是特性不是错误**=值得补写的知识缺口 | WikiSkill.Links（复用 LinkedPages 解析；红链产 knowledge_gap 提案） |
| `reach-for` | 何时"够得着"这个技能：触发短语、glob（文件模式）、形态——一级目录只列入口，靠它主动加载 | WikiSkill.ReachFor（触发短语并入现有 Applicability，新增 globs） |
| `loads` | 条件加载的第三层文件清单（路由语义：path + when） | WikiSkill.Loads → 现有 Attachments/Steps 引用扩展 |

收敛证据（文章引用）：OpenAI harness 把 AGENTS.md 从 800 行减到 ~100 行目录、正文进 docs/；Manus"文件系统是终极上下文，留指针、删内容"。**可度量的健康主张：图生效时，单任务平均加载上下文下降而正确技能仍被找到**——这正好并入 G5 影响追踪作为进化循环的验收指标。

### 0.3 santhoshtr/wiki-skills 的正名与可取处

它是 Wikimedia 领域的 agent 技能包（wikipedia-editor-research、wikidata-natural-query、MediaWiki 表结构、Gerrit 流程……每个 SKILL.md + references/ + assets/），与"LLM 维护的 wiki"无关。可取处两点：
1. **SKILL.md + references/ 是跨 agent 的事实打包标准**——Coco 的 skill 页应支持导出/导入该格式（社区复现也证明标准格式才能跨 agent 迁移）
2. 领域技能包是**内容**：P3 可做"技能包导入"，把外部打包技能装进 KB 技能层

### 0.4 对下文设计的修订

- §4.1 `WikiSkill` 增加 `Links []WikiLinkedPage`、`ReachFor { Phrases []string; Globs []string }`、`Loads []SkillLoad { Path, When }` 三组字段（§3 图中 Skills 层随之升级为"带边的技能图"）
- §4.3 注入策略细化为三级渐进披露：一级=技能索引页（名称+一句话+reach-for，常驻 prompt，控制一级税）；二级=命中的技能页正文；三级=loads 清单里的附件按需取
- §4.2 门控增加 `no_action` 进化日志结果；G5 验收指标补"单任务平均加载上下文不升而命中率不降"
- §4.4 前端补技能图健康视图（孤儿技能/红链/hub 度数）——lint 已有死链检测，天然覆盖
- P2 增加一条：skill 页导出为 SKILL.md 格式（含 references/ 打包）；P3 的跨 KB 共享扩为"跨 KB/跨助手/外部技能包导入"

## 1. 第一轮来源的核心结论

### 1.1 WikiSkill 论文（值得照搬的机制）

**三层知识架构**：

| 层 | 内容 | 关键性质 |
|---|---|---|
| Raw 层 (`raw/`) | agent 执行轨迹（推理、工具调用、输出、答案） | **不可变**，仅供维护者/提议者分析 |
| Wiki 层 (`wiki/`) | patterns/ 模式页（失败模式+成功策略+解法）、index.md、logs.md 进化日志、skill-impact.md 影响追踪 | **持续累积、永不回滚**；patch 式增量编辑 |
| Skills 层 (`skills/`) | SKILL.md（可执行程序性知识）+ PURPOSE.md（回溯到激励该技能的 wiki 模式） | **受门控约束**，可回滚 |

**四角色进化循环**（每轮迭代）：
1. **推理 agent**：携带当前技能集执行任务；**进化期间禁止读 wiki**（消融：读了反而让轨迹信息量下降，最终技能质量 63.7% → 60.9%，LiveMath 72.6% → 64.8%）
2. **Wiki 维护者**：采样轨迹（≤8 条：5 败 3 成），做根因分析，patch 式更新模式页 + 重写 index + 追加进化日志
3. **技能提议者**：ReAct 式 agent，先读 index + skill-impact.md（**含被拒提案全文，避免重蹈覆辙**），按需深挖模式页与原始轨迹；每轮只产出一个**原子提案**（新建或 patch 单个技能），动笔前必读 ≥4 条轨迹
4. **门控与回滚**：候选技能在验证集上评估，分数超过历史最优才接受，否则回滚技能；无论接受与否，**wiki 与 skill-impact 日志都保留**（由外层程序化写入，非 LLM）

**其他关键数字**：提议者有 wiki 访问权 +15.0 pts（48.7% → 63.7%）；技能跨模型迁移有效，甚至"别人进化的技能"可能胜过"自己进化的"（Qwen-9B 用 27B 的技能 70.2% vs 自进化 63.4%）——技能发现与技能执行是两种能力。

### 1.2 wiki-skills 仓库（工程纪律）

- **Schema 即契约**：SCHEMA.md 全局约定，页面 frontmatter（title/tags/aliases/created/updated/sources/confidence）是唯一事实源
- **index 永远生成、不可手编**；`[[backlink]]` 图由保存时解析
- **lint（页级校验：schema 合规、死链、孤儿页、必填 frontmatter）与 audit（全库质量：矛盾、过期源、孤儿）分离**
- **merge 保留历史**；git log 即操作日志
- 七个动词化技能：init / ingest / query / lint / update / audit / merge

## 2. Coco 现状盘点（对照差距）

已有的（相当扎实，部分领先于两个来源）：

| WikiSkill 概念 | Coco 对应物 | 状态 |
|---|---|---|
| Wiki 层（声明性知识） | WikiArticle（entity/concept/source/topic/map 五类、Sources(doc_id+locator) 引用契约、aliases、状态机、Maturity） | ✅ 已有 |
| 版本/回滚 | WikiVersion 不可变快照 + change_type | ✅ 已有（比 git 更结构化） |
| 反向链接 | 保存时解析 [[wikilink]] → LinkedPages、`_backlinks` API | ✅ 已有 |
| 原始材料层 | datasource 文档（KB 绑定 + 上传） | ✅ 已有（但只有"文档"，无"经验"） |
| 编译器 | MAP→REDUCE→PLAN→WRITE topic 页、consolidate 薄页、WikiCompileRules | ✅ 已有 |
| 治理 | WikiGovernanceProposal 13 类（stale/duplicate/conflict/orphan/knowledge_gap/article_refresh/entity_duplicate/knowledge_compile…） | ✅ 已有 |
| 定时任务 | freshness-check(1h) / governance-scan(24h) / auto-approve sweep(2m) | ✅ 已有 |
| 实体本体 | KB 级实体池、ontology schema 词表、merge/晋升/传播 | ✅ 已有 |
| agent 检索接入 | internal_search 的 knowledge tier（wiki_article/entity） | ✅ 已有 |
| MCP | 40+ wiki 端点挂 MCPTool | ✅ 已有 |

**缺失的（即本方案要补的）**：

| # | 缺口 | 说明 |
|---|---|---|
| G1 | **Skills 层（程序性知识）** | 无 page_type=skill/pattern；没有"怎么做"的结构化步骤页；无 SKILL/PURPOSE 分离 |
| G2 | **Raw 经验层** | 聊天/deep-research 的执行轨迹没有被结构化捕获为学习材料；from_chat 只是手动单篇落库 |
| G3 | **进化闭环** | 治理扫描是"维护驱动"（过期/重复），不是"经验驱动"（从失败中挖掘该学什么）；没有维护者→提议者→门控的循环 |
| G4 | **门控** | auto-approve 是规则扫；没有第二 LLM 评审"该提案是否真有贡献"、没有基于评估集的接受/回滚语义 |
| G5 | **影响追踪** | Maturity 是覆盖度；没有"该页/该技能实际提升了多少回答"的使用侧指标（论文的 skill-impact.md） |
| G6 | **进化日志** | 版本是页级的；没有 KB 级决策日志让提议者避免重复已被拒的提案 |
| G7 | **lint 实时校验** | 保存时无页级 schema/死链/引用完整性 lint（死链数据其实已有：LinkedPages 解析） |
| G8 | **探索/利用分离** | internal_search 永远带 wiki tier；论文证明"用于喂 wiki 的研究型运行"应禁读 wiki |

## 3. 目标架构

```
┌──────────────────────────────────────────────────────────────┐
│                    Coco Assistant Runtime                     │
│   chat / deep-think / deep-research                           │
│   • 生产推理: 注入 skills + wiki knowledge tier（利用）         │
│   • 进化运行: 注入 skills，屏蔽 wiki tier（探索）               │
└──────────────┬───────────────────────────────────────────────┘
               │ 执行轨迹（问、答、citations、工具调用、用户反馈）
               ▼
┌──────────────────────────────────────────────────────────────┐
│ ① Raw 经验层   WikiExperience（不可变，KB 隔离，opt-in）        │
│    session 级轨迹 + 分级反馈（thumbs/追问/"没用"信号）           │
└──────────────┬───────────────────────────────────────────────┘
               │ 维护者 agent（采样 ≤N 条：败多成少，根因分析）
               ▼
┌──────────────────────────────────────────────────────────────┐
│ ② Wiki 层（已有，扩展）                                        │
│    article: topic/concept/entity/source/map + 新增            │
│      page_type=pattern（模式页：问题+根因+解法，10-30 行）       │
│    + EvolutionLog（KB 级，追加式，含被拒提案 diff——永不回滚）    │
│    + lint on save（schema/死链/引用完整性 → 页内警告）           │
└──────────────┬───────────────────────────────────────────────┘
               │ 提议者 agent（读 index+evolution log，按需深挖，
               │  每轮一个原子提案）
               ▼
┌──────────────────────────────────────────────────────────────┐
│ ③ Skills 层（新增）                                            │
│    page_type=skill：适用条件 / 步骤 / 反例 / When-NOT-to-apply │
│    + provenance：关联到激励它的 pattern 页（PURPOSE 语义）       │
│    门控：LLM 评审 + 验证集回放评分 → 接受 / 回滚；               │
│    结果（diff+分数+结论）程序化写入 EvolutionLog 与影响表         │
└──────────────────────────────────────────────────────────────┘
```

## 4. 详细设计

### 4.1 数据模型（core/wiki.go 增量）

```go
// Raw 经验层
type WikiExperience struct {
    WikiKnowledgeBaseID string            // KB 隔离
    SessionID           string            // chat / deep-research session
    Kind                string            // chat | deepthink | deepresearch
    Question, Answer    string
    Citations           []WikiSourceReference
    ToolCalls           []ExperienceToolCall  // 检索词、命中等关键步骤（截断存储）
    Feedback            string            // positive | negative | none
    FeedbackNote        string
    Outcome             string            // resolved | abandoned | escalated
    Retention           time.Time         // 过期清理（不可变但可过期）
}

// Wiki 层扩展
// WikiArticle.page_type 增加: "pattern" | "skill"
// WikiArticle 增加:
//   Pattern   *WikiPattern   { Symptom, RootCause, Fix, Evidence []WikiSourceReference }
//   Skill     *WikiSkill     { Applicability, Steps []string, AntiPatterns []string,
//                              WhenNotToApply []string,
//                              ReachFor { Phrases []string, Globs []string },  // v2: 一级目录的主动加载条件
//                              Loads []SkillLoad { Path, When } }              // v2: 条件加载的三级附件路由
//   Provenance []WikiLink     // skill → 激励它的 pattern 页（PURPOSE 语义）
//   （skill 页的 [[技能名]] 链接经 LinkedPages 解析成技能图的边；红链=知识缺口，
//     v2: 走 knowledge_gap 提案而非仅 lint 警告）
// WikiArticle.Impact *WikiImpact { AnswerCitations int, PositiveRate float64,
//                                   LastUsedAt time }   // 程序化更新，非 LLM

// KB 级进化日志（永不回滚；写入方为门控程序，非 LLM）
type WikiEvolutionEntry struct {
    KBID        string
    Iteration   int
    Proposal    EvolutionProposal  // target article/skill, patch diff 摘要
    Verdict     string             // accepted | rejected | no_action  // v2: 诚实记录"无可学"
    Score       *EvalScore         // 验证回放得分（可空）
    ReviewerNote string            // 门控 LLM 评审理由
}
```

### 4.2 后端模块（modules/wiki/ 增量）

**经验捕获（G2）**：chat/deep-research 会话结束（或反馈发生）时写入 `WikiExperience`；KB 绑定 assistant 且 KB 开关 `experience_capture: opt-in` 才记；工具调用内容截断、敏感字段脱敏走现有 masking 配置。

**维护者 agent（G3 前半）**：新定时任务 `wiki-experience-distill`（默认 6h，KB 可配）+ 手动触发端点 `POST /wiki/kb/:id/_distill`。复用编译器 job 框架（单 worker + 进度轮询）。输入：上次蒸馏水位以来的经验样本（分层采样：负反馈优先、最多 N 条）+ 现有 pattern 页。输出：**全部走治理提案**（新提案类型 `pattern_create` / `pattern_update`，patch 操作集：append/replace/insert_after，与论文维护者一致）。同时维护生成式 pattern 索引页（`index` 语义：每行 `名称 — 问题+根因+解法一句话`，程序生成，禁止手编）。

**提议者 agent（G3 后半）**：`POST /wiki/kb/:id/_propose`（也挂 MCP）。ReAct 式：先读 pattern 索引 + EvolutionLog（**必读，含被拒提案**），按需检索 pattern 页与原始经验（复用 internal_search 的 knowledge tier 工具化）。产出**原子提案**：`skill_create` / `skill_patch`（patch 操作集同上），带 Provenance 指向模式页。一次只动一个技能。

**门控（G4）**：提案应用前两道闸：
1. **LLM 评审**（第二模型，非生成者）：对照"该提案是否与现有内容重复/矛盾、是否有经验证据支撑、是否真的解决 EvolutionLog 中未解决的问题"，输出 verdict + 理由 → 存入提案；
2. **验证回放**（可选，KB 配置评估集：一组带期望答案的查询）：应用技能前后各跑一次检索+回答评分，**分数下降则回滚技能（WikiVersion 已天然支持）， EvolutionLog 照写**。没有评估集的 KB 只走闸 1 + 人工审批。

社区复现的实证（v2）：验证门控真实拦下过有害技能；提议者某轮"无可学"时输出 no_action 并记录——门控与日志必须把 no_action 当一等结果，禁止硬造技能凑数。

**lint（G7）**：文章保存路径（已有 LinkedPages 解析处）加页级校验：page_type 必填字段、死链（LinkedPages 解析失败项）、引用完整性（pattern/skill 必须有 Evidence/Provenance）。结果作为页内 lint 警告字段（不打扰保存，不进治理队列），前端阅读/编辑页展示。

**影响追踪（G5）**：internal_search 命中 wiki_article/entity/skill 时写命中计数（防抖、按 session 去重）；chat 反馈回填到被引页面的 PositiveRate。定时任务据此更新 Impact；低影响 + 陈旧 → 产出 `article_refresh`/归档提案（复用治理）；高影响页 → consolidate 跳过保护。

**探索/利用分离（G8）**：assistant 调用参数增加 `explore_mode`；进化链路（蒸馏、提议者的资料收集运行）在发起检索时置为 true —— internal_search 排除 wiki tier。生产 chat 不变。

### 4.3 检索与注入（modules/assistant/）

- 生产推理：skills **不整篇注入 prompt**，按三级渐进披露（v2，对齐 Agent Skills 规范 + 图层修订）：
  - 一级：技能索引页（名称 + 一句话 + reach-for 触发条件）常驻 system prompt——控制一级税，只列入口不装正文；
  - 二级：intent/检索命中的技能页正文按需加载（reach-for 的短语/ globs 命中也触发）；
  - 三级：Loads 清单里的附件文件按 when 条件取用。
- 把 `page_type=skill` 提升为 knowledge tier 的高优先级源；prompt 段标注"程序性指导，按适用条件取用"
- 跨 KB/跨助手迁移（论文的 transfer 结论 + v2 扩展）：skill 页支持 tenant scope（复用 WikiOntologySchema 的 scope 机制），同租户其他 KB/其他 assistant 可只读引用——"A KB/A 助手进化的技能让 B 受益"；论文实证技能进化与模型规模互补、带技能的小模型可胜大模型，本地小模型场景收益最大
- **SKILL.md 互操作**（v2）：skill 页支持导出为 Agent Skills 标准打包（SKILL.md + references/），也支持导入外部技能包（如 santhoshtr/wiki-skills 这类领域包）

### 4.4 前端（web/src/pages/wiki/）

- article/[id] 渲染 pattern/skill 两种新页型（步骤列表、反例块、适用条件卡）；lint 警告条
- KB 概览加"进化"面板：EvolutionLog 时间线（提案 diff、接受/拒绝/no_action、评分）、影响 Top 页、经验捕获开关与水位
- 技能图健康视图（v2）：孤儿技能（无入边）、红链清单（值得补写的缺口）、hub 度数——健康主张可度量："单任务平均加载上下文下降而正确技能仍被命中"（并入 G5 验收）
- 治理队列支持新提案类型的专用卡片（pattern/skill 的 patch 预览走现有 diff 组件）
- 聊天侧 saveToWiki 扩展"存为 pattern/skill"选项（from_chat 已有落库通道，加目标类型）

### 4.5 MCP

`wiki_distill`、`wiki_propose`、`wiki_evolution_log` 三个新工具（对齐论文三个角色的可编程入口）。

## 5. 分期路线

| 阶段 | 内容 | 依赖 |
|---|---|---|
| **P0 地基**（已实现 ✅） | lint on save（G7）；Impact 命中计数（G5）；explore_mode 开关（G8）；EvolutionLog 类型与写入（G6，治理应用/驳回已入日志） | 无 |
| **P1 经验层** | WikiExperience 捕获 + KB 开关 + 过期策略（G2）；维护者定时蒸馏 → pattern 提案（G3 前半）；pattern 页型 + 生成式索引 + 前端（G1 一半） | P0 |
| **P2 技能层与门控** | skill 页型（含 v2 的 Links/ReachFor/Loads 三组字段）+ 提议者 agent（G1/G3 后半）；LLM 评审门控（G4，含 no_action 一等结果）；验证回放（可选）；skill_patch 原子提案；三级渐进披露注入；SKILL.md 导出 | P1 |
| **P3 进阶** | 验证集管理 UI；技能跨 KB/跨助手 tenant 级共享；外部技能包导入；MCP 三工具；影响驱动的归档/保护策略 | P2 |

## 6. 风险与开放问题

- **隐私**：经验轨迹含用户提问与内部检索行为——必须 opt-in、脱敏、可过期（Retention），并在 KB 设置页明示
- **成本**：蒸馏/提议都是 LLM 重活——水位增量处理 + 每 KB 每日预算上限（复用 assistant 配额机制）
- **评估集从哪来**：没有现成验证集的 KB 只走 LLM 评审门 + 人工审批，验证回放作为渐进能力；论文的"验证分数才能接受"在无评估集场景退化为"评审 + 人审"
- **wiki 永不回滚的代价**：pattern/日志会膨胀——论文自己也列为 limitation；影响追踪（G5）+ 归档提案是 Coco 的对应答案
- **负迁移**：跨 KB 共享技能可能有害（论文 SpreadSheet 案例：小模型技能让大模型 −32 pts）——tenant 共享默认只读试用、可按 KB 停用
