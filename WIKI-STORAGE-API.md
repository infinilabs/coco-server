# Wiki 后端存储与 API 清单

日期: 2026-10-23 · 基线: a9233342 · 实测环境: localhost:9000 (ES 集群 prod, index_prefix `coco_`, 索引带 `-v2` 版本后缀)

## 一、存储层

### 1. Elasticsearch（主存储，ORM 经 framework `orm` 模块）

索引命名规则: `coco_` 前缀（coco.yml `orm.index_prefix`）+ 显式注册名 + `-v2` 后缀
（注册见 `modules/coco.go` Setup + `modules/common/schema.go` 的 `SCHEMA_VER`）。
**注意**: 仅 12 个 wiki 类型被显式注册（kebab + -v2）；6 个较新类型未注册，
走 framework 兜底命名 = 小写结构体名（无连字符、无后缀），见下表。（标 ✓ 为实测确认）

**Wiki 专属索引（18 个）**

| 索引 | 结构体 | 内容 | 实测计数 |
|---|---|---|---|
| coco_wiki-kb-v2 | WikiKnowledgeBase | 知识库（含 auto_approve/experience_capture/toc_layout 设置） | 2 ✓ |
| coco_wiki-article-v2 | WikiArticle | 文章正文（content 不索引，版本历史另存） | 502 ✓ |
| coco_wiki-toc-v2 | WikiToc | 每库一棵目录树（nodes 数组） | — |
| coco_wiki-version-v2 | WikiVersion | 版本快照（管线写入自动留痕，diff/回滚依据） | — |
| coco_wiki-entity-v2 | WikiEntity | 实体（KB 作用域、锚定 ID、状态机） | 279 ✓ |
| coco_wiki-ontology-schema-v2 | WikiOntologySchema | 本体 schema + 编译规则（D6，默认种子 6 类型） | — |
| coco_wiki-governance-v2 | WikiGovernanceProposal | 治理提案队列 | — |
| coco_wikievolutionentry | WikiEvolutionEntry | 进化日志（append-only 决策记录）※未注册 | 109 ✓(UI) |
| coco_wikiexperience | WikiExperience | 经验记录（chat/correction/gap，带 retention）※未注册 | — |
| coco_wikipattern | WikiPattern | 蒸馏产出的模式页（P2）※未注册 | — |
| coco_wikiskill | WikiSkill | 技能（SKILL.md 体系，P2）※未注册 | — |
| coco_wikiimpact | WikiImpact | 页面被引用统计 ※未注册 | — |
| coco_wikiarticleattachment | WikiArticleAttachment | 文章附件关联 ※未注册 | — |
| coco_wiki-bookmark-v2 | WikiBookmark | 收藏 | 1 ✓ |
| coco_wiki-like-v2 | WikiLike | 点赞 | 1 ✓ |
| coco_wiki-comment-v2 | WikiComment | 评论 | 0 ✓ |
| coco_wiki-notification-v2 | WikiNotification | 通知 | 293 ✓ |
| coco_wiki-workspace-v2 | WikiWorkspace | 工作区 | — |

**共享索引（文档模块，wiki 只写投影行）**

| 索引 | 用途 |
|---|---|
| coco_document-v2（193 ✓） | Wiki 投影: 已发布文章镜像为 `type=wiki_article` 行（source.type=wiki），走统一噪声门; wiki_article 索引仍是治理事实源 |
| coco_knowledge-chunk-v2 | 块级检索分支: 重处理时把文档 chunk 镜像为独立行（chunk_type mom/child） |

**Mapping 特性**（elastic_mapping 标签驱动）: title/name 走 keyword + text + pinyin（pinyin_analyzer）三字段并 copy_to combined_fulltext；content/relations/citations 等大对象 `enabled:false`（不索引只存储）；override_exists_template/mapping: true（本地配置）。

### 2. KV 存储（framework `kv`，bbolt 本地库）

| Bucket | 用途 |
|---|---|
| wiki-freshness | 新鲜度巡检水位（上次扫描高水位时间） |
| wiki-distill | 每库蒸馏水位（增量采样） |
| /wiki/governance/batchapply | 批量应用进度（可轮询） |

### 3. Blob 存储

KB 文件上传（_files/upload）经 `attachment.UploadToBlobStore` 落对象存储 + Attachment 记录 + 同名建 Document（重传原地更新不堆双），下载走 `/attachment/:id`。

### 4. 内存/调度（无持久化状态）

三个定时任务: 编译 Agent（WIKI_COMPILE_AGENT_INTERVAL）、蒸馏（WIKI_DISTILL_INTERVAL）、新鲜度巡检（WIKI_FRESHNESS_INTERVAL），产出全部落到上表 ES 索引。

## 二、API 清单（共 99 个端点 = 9 套 CRUD × 6 + 45 自定义）

### 1. CRUD 六件套（framework 统一生成: POST / · GET|POST _search · GET/:id · PUT/:id · DELETE/:id）

| Prefix | 资源 | 权限模型 |
|---|---|---|
| /wiki/workspace | 工作区 | 资源级 simple permission |
| /wiki/kb | 知识库 | 资源级; visibility=team 走 share 模块（SharingResource） |
| /wiki/article | 文章 | 资源级 |
| /wiki/entity | 实体 | 资源级 |
| /wiki/governance | 治理提案 | 资源级 |
| /wiki/bookmark /wiki/like /wiki/comment /wiki/notification | 互动 | 资源级 |

### 2. 自定义端点（45 个，按域分组）

**目录/版本/状态**
- GET /wiki/kb/:id/toc — 目录（含三形态自愈: 孤儿补挂/空白重建/悬空剪除）
- PUT /wiki/kb/:id/toc — 目录更新
- GET /wiki/article/:id/versions · GET .../versions/:version — 版本列表/详情
- PUT /wiki/article/:id/status — 状态流转（审核门）

**LLM-wiki 闭环**
- POST /wiki/kb/:id/_agent_compile — 运行编译 Agent（同步, 返回 iterations/patches/no_action/notes）
- POST /wiki/kb/:id/_compile · GET 同路径 · POST .../_cancel — 编译任务（异步 job + 进度 + 取消）
- POST /wiki/kb/:id/_distill — 手动蒸馏（sampled/proposals）
- GET /wiki/kb/:id/_evolution_log — 进化日志
- POST|GET /wiki/kb/:id/_experience — 经验捕获/列表
- POST /wiki/experience/_vote — 聊天侧投票捕获（login-only, 不暴露捕获 KB 存在性）
- POST /wiki/kb/:id/_propose · GET /wiki/article/:id/_skill_md · POST /wiki/kb/:id/_import_skill — 技能提议/SKILL.md 导出/导入（P2）

**实体**
- POST /wiki/entity/ — 创建（lenient 可选）
- PUT /wiki/entity/_batch_status — 批量状态
- POST /wiki/entity/_merge — 合并
- GET /wiki/entity/lookup · GET /wiki/entity/:id/neighbors · GET /wiki/entity/:id/backlinks — 查找/邻居/反链

**生成/编辑**
- POST /wiki/kb/:id/ai/generate — AI 生成（SSE 五阶段）
- POST /wiki/article/:id/ai/edit — AI 编辑（SSE）
- POST /wiki/article/:id/_rewrite — 整页 REFINE 重写
- POST /wiki/article/_from_chat — 从聊天建文章

**治理**
- POST /wiki/governance/_correction — 纠错提交（login-only）
- PUT /wiki/governance/:id/status · PUT /wiki/governance/_batch_status — 状态/批量状态
- POST /wiki/governance/:id/_apply · POST /wiki/governance/_batch_apply · GET /wiki/governance/_batch_apply — 应用/批量（KV 进度）

**本体**
- GET|PUT /wiki/ontology/schema — schema 读写（PUT 携带 entity_types + rules）

**材料/附件/文件**
- GET /wiki/article/:id/_materials · GET|PUT /wiki/article/:id/_attachments — 材料/附件
- POST /wiki/article/:id/_relink — 未解析链接→待审实体并重绑
- GET /wiki/kb/:id/_files · POST /wiki/kb/:id/_files/upload — 文件列表/上传

**图谱/地图**
- GET /wiki/kb/:id/_graph — KB 图谱
- GET /wiki/company-map — 公司地图（login-only）

**检索支撑**
- POST /wiki/kb/:id/_consolidate — 重复文章整理
- POST /wiki/_backfill_projection — 投影回填（total/projected/skipped）

### 3. 横切面

- **权限**: 45/45 RequireLogin; 42/45 RequirePermission（login-only 3 个: _vote/company-map/_correction）; CRUD 六件套按资源动作鉴权
- **MCP**: 18 个端点同时注册为 MCP 工具（run_compile_agent、distill_wiki_experience、wiki_evolution_log、record_wiki_experience 等），Agent 可直接调用
- **前端消费**: 45 自定义端点中 38 个有 UI 消费方（本分支 a9233342 补齐 6 个后）
