# 设计:主题页编译(Topic-Centric Compilation)

> 状态:提案,待确认后实施
> 背景:当前 AI 编译产出的知识库「每篇文章只有只言片语」,左侧菜单全是碎片页。
> 目标:每个入口是一个**丰富、多维度、结构化**的主体页面;新素材**优先更新和 enrich 既有页面**,而不是新增零散页。

## 一、同行前沿研究结论

### 1. Microsoft GraphRAG — 页面 = 社区,不是实体
- 从语料抽实体/关系图,用 Leiden 算法把实体**聚类成社区**(community),多层级。
- 每个**社区生成一份结构化报告**(不是每个实体一页):`name` + `summary`(执行摘要)+ `rating`(0-10 重要性 + 解释)+ `findings[]`(每条 finding = summary + 完整 explanation,有证据支撑)。
- 增量索引:新文档到达时**增量更新**图和社区,不整库重建。
- 对我们的启示:**页面粒度 = 主题(实体的聚类),实体只是页内素材**;报告有固定维度结构;更新优先于新建。

### 2. 斯坦福 STORM — 大纲先行,再写正文
- 写长文前先完成「研究 → 参考文献 → **大纲**」,按大纲逐节写作;直接让 LLM 生成整文,组织质量显著更差(经维基百科编辑盲评验证)。
- 多视角提问:不同 persona 对同一主题提出不同维度的问题,保证大纲**维度覆盖面**。
- 对我们的启示:**outline 是一等公民**,先规划章节再填内容;维度多样性靠多视角驱动。

### 3. 斯坦福 RAPTOR — 递归聚类 + 分层摘要
- chunk 聚类 → LLM 摘要 → 摘要再聚类 → 再摘要,自底向上构成树;高层节点是多个底层的**聚合抽象**。
- 对我们的启示:素材聚合是**逐层**的——陈述 → 小节 → 章节 → 页面,而不是陈述 → 页面一步到位(我们现在就是一步到位,所以薄)。

### 4. NotebookLM(现 Gemini Notebook)/ Deep Research 类产品
- 产品形态:素材进来后不生成「一页一实体」,而是生成**少量结构化长产物**(报告、思维导图、FAQ),素材永远挂在 notebook 上被引用。
- 对我们的启示:**入口要少而厚**,素材是页面背后的证据层(引用/附件),不是页面本身。

**共同范式**:实体进图谱,主题进菜单,大纲定结构,更新优先于新建。

## 二、当前实现的病根(代码定位)

| 病根 | 位置 | 后果 |
|---|---|---|
| 页面粒度 = 单个实体/概念 | `compiler.go:240-330`(`ReduceExtracts` 按 `(kind,name)` 聚合,`PlanCompile` 每 key 一页) | 一个实体只有 1-3 条陈述也开一篇文章 → 菜单全是碎片 |
| 无章节结构 | `core.WikiArticle` 只有 `Content` 一整段 markdown | REFINE 只能把两三条陈述扩写,「各个维度都有」无从谈起 |
| 陈述→页面一步到位 | `fileCompileProposal`(`compiler.go:933`)直接拿 key.Statements 当全部证据 | 缺少 RAPTOR 式逐层聚合,信息密度上不去 |
| 新素材默认产出新提议 | `PlanCompile` 的 update 分支只按 normKey 精确匹配既有文章 | 同一主题的新素材开出新页,而不是 enrich 既有页 |
| 实体页是常态而非例外 | entity/concept 都生成页面 | 实体本该是图谱节点 + 页内卡片(悬停卡片已具备) |

## 三、重设计

### 3.0 内容模型:页面与本体平级

一个 Wiki 由两类平级内容构成:

```
Wiki
├── 页面(主题页)——菜单入口,面向阅读,少而厚,多维度结构化
└── 本体(实体/概念/关系)——独立的一等公民:实体 tab 管理、图谱呈现、
    页内悬停卡片引用;可由编译引入,也可人工新增(已有能力)
```

- 页面**引用**本体(wikilink、关键实体卡片),但不拥有本体;删除页面不影响实体。
- 本体**不自动生成页面**。当某个实体/概念素材足够丰富,或用户在实体管理里手动钉选「为此实体建页」时,才为它生成一篇**同样走大纲结构的完整主题页**(该实体就是主题)。
- 编译器同时产出两者,互不混淆:实体抽取 → 本体池(WikiEntity,已有);主题聚类 → 页面。当前实现的病根正是把两者焊接在一起(实体 key 直接变文章页)。

### 3.1 页面模型:主题页(Subject Page)

页面 = **主题**(一组高内聚实体的聚类),结构化 markdown,多维度:

```go
type WikiSection struct {
    ID        string   `json:"id"`         // 稳定 id,enrich 时复用
    Heading   string   `json:"heading"`    // 章节标题
    Body      string   `json:"body"`       // 该节正文(带引用标记)
    DocIDs    []string `json:"doc_ids"`    // 该节证据来源
    UpdatedAt int64    `json:"updated_at"`
}

// WikiArticle 新增:
Outline   []WikiSection `json:"outline,omitempty"`  // 结构化章节(渲染为 markdown 存 Content,双写兼容)
KeyEntities []string   `json:"key_entities,omitempty"` // 页内核心实体(卡片化展示)
Maturity  struct{       // 成熟度,TOC 徽标 + 治理排序用
    Sources  int     `json:"sources"`   // 引用文档数
    Sections int     `json:"sections"`  // 已填章节 / 规划章节
    Score    float64 `json:"score"`     // 覆盖率评分 0-1
} `json:"maturity,omitempty"`
```

大纲维度模板(按 KB 类型自适应,LLM 从模板 + 素材中选题):
`概述 / 背景与上下文 / 核心机制(或:分维度专题 ×N) / 关键实体与关系 / 时间线 / 数据与事实 / 争议与开放问题 / 参考来源`

### 3.2 管线:MAP → REDUCE → **CLUSTER → OUTLINE → FILL** → REFINE

```
MAP       不变(2k 批次抽陈述,指纹缓存复用)
REDUCE    不变(实体 key 聚合 + 去重)
CLUSTER   新增:实体共现聚类 → 主题(见 3.3)
OUTLINE   新增:每主题 LLM 生成章节大纲(多维度模板 + 素材驱动);
          已有主题页时,大纲 = 既有 Outline + 新素材建议的增量章节
FILL      新增:陈述按章节路由(关键词/向量相似 + 兜底 LLM),
          每节从 routed statements 起草,保留引用 → RAPTOR 式逐层聚合
REFINE    改造:整页一次润色(现有 compiledProse 升级为按页非按 key)
```

实体/概念默认产出本体条目(WikiEntity,抽取管线已有),**不再自动生成文章页**;仅当某实体陈述量超过阈值(如 ≥ ConceptMinSources 的 2 倍)且 LLM 判定可独立成篇,或用户在实体管理里钉选「为此实体建页」时,才为该实体生成专页——专页同样走大纲结构,该实体即主题。

### 3.3 CLUSTER:实体共现聚主题

1. 从 REDUCE 结果 + 文档归属构建实体共现图(同一条陈述/同一文档共现 = 边,权重 = 共现次数)——现有 relations 数据可直接并入。
2. 贪心模块度聚类(Leiden 的简化版,无需引第三方库):按边权合并,主题数上限 = `min(20, max(6, 文档数/3))`,过小主题并入最近邻。
3. 每个聚类让 LLM 命名 + 圈定范围(title + 3-5 个 alias + 一句话 scope),产出 TopicRegistry。
4. 缓存:聚类结果按内容指纹缓存,增量素材只重算受影响的主题。

### 3.4 Update-first Planner(核心语义变更)

`PlanCompile` v2 对每个 TopicRegistry 条目:

1. **匹配既有页**:标题/别名精确命中 → 向量相似度(Title+Overview embedding)→ LLM judge(「新素材是否属于该页主题范围」)。
2. 命中 → **UPDATE 提议**:新陈述路由进既有章节,产出**章节级 diff**(新增节 / 扩写节 / 补充引用),Maturity 重算。
3. 未命中且素材充足 → CREATE 提议(照常进治理队列)。
4. **薄页吸收**:既有文章 Maturity.Score < 0.3 且主题落入某主题页范围 → `consolidate` 提议:并入主题页对应章节,原页 retired 并 301 式重定向(wikilink 解析时跟随),引用折叠为节证据。

治理队列呈现变化:一条提议 = 一个主题页的一次**章节级变更**(diff 预览),而不是 N 条「某实体某陈述」碎片提议。治理工作量天然下降一个量级。

### 3.5 入口与 UX

- 左侧菜单 = 页面(主题页,少而厚,带成熟度徽标: sources 数 / 章节覆盖率);本体入口维持现状:实体 tab、图谱 tab,与页面平级。
- 实体管理(WikiEntityManager)新增「为此实体建页」动作:把单个实体升格为一篇主题页(走同一大纲管线),页面与实体通过 wikilink 双向关联。
- 主题页内:章节目录锚点、关键实体卡片行、每节末尾引用折叠区、附件区(已有)。
- 编译控制台(job)沿用,新增 CLUSTER/OUTLINE/FILL 阶段计数。

### 3.6 兼容与迁移

- `Outline` 与 `Content` 双写:Content 由 sections 渲染生成,搜索/链接/版本机制零改动。
- 存量文章:首次重编译时,`PlanCompile` v2 的匹配器会把存量页当作 update 目标;薄页走 consolidate 收编。提供一次性「重组」治理提议(类型沿用 tree_organize 扩展)让用户过目后批量执行。
- 依赖:REFINE/OUTLINE/FILL 需要配置默认语言模型(当前实例丢失该配置,需先在设置里恢复)。

## 四、实施分期

| 期 | 内容 | 规模 |
|---|---|---|
| P1 | 数据模型(WikiSection/Outline/Maturity)+ 渲染器 + API/前端展示章节结构 | 小 |
| P2 | CLUSTER 阶段(共现聚类 + TopicRegistry)+ OUTLINE/FILL + REFINE 改造 | 中(核心) |
| P3 | Update-first Planner + 章节级 diff 提议 + consolidate 薄页收编 + 治理/TOC UX | 中 |
| P4 | 增量缓存细化(聚类/大纲级指纹)+ 存量重组批量提议 | 小 |

P2 落地即可看到「少而厚」的页面;P3 落地后新素材默认 enrich 既有页。

## 参考

- GraphRAG: https://microsoft.github.io/graphrag/ · https://github.com/microsoft/graphrag
- STORM: Shao et al., NAACL 2024 · https://arxiv.org/abs/2402.14207 · https://storm.genie.stanford.edu
- RAPTOR: Sarthi et al., ACL 2024 Findings · https://arxiv.org/abs/2401.18059
