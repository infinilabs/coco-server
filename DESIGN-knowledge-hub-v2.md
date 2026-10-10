# Coco 知识库 v2 设计文档(WeKnora/RAGFlow 对照定稿)

> 定稿日期:2026-10-08 · 状态:待评审
> 依据:WORKPLAN-knowledge-hub.md 第五篇(WeKnora 总览)/补遗(知识库域源码级)/补遗二(解析专页)/第六篇(RAGFlow 预处理)/第七篇(知识编译)的全部机制级调研;所有"现状"均已核到 file:line
> 红线(继承,不变):PR 不引入 license 相关文件(go.mod/go.sum、config/generated.go、.public/static.go、widget dist、二进制、.claw/)——**即本设计一律零新 Go 依赖,原生 zip/XML 实现(先例:pptx.go 原生解析)**;AI 只产草稿/提议,发布必须人工;系统只推荐、绝不擅自删;单二进制(不引 Redis/Asynq/Neo4j/Python 边车)

## 0. 一页总览

**问题**:三个缺陷级事实(块向量在写从未被查=死重;编辑/API 直建文档静默绕过切块与嵌入;无重处理端点、嵌入模型无版本戳新旧混查)+ 两个结构短板(解析无结构单元、检索粒度=整文档)。

**方案**:十八个工作流(W0-W8、W10-W18,+远期 W9)+安全三件套(S1/S2/S3),分五批落地。目标态一句话:**解析产出结构化单元 → 切成带溯源位的块入独立块索引 → 多路 RRF 打块 → 组合精排 → 页/节级引用**;实体成为全站级知识入口(实体卡片+entity 检索路+编辑传播);每条知识(文章/实体/答案)携带块级来源与可视化引用;预览直达原始页并高亮命中(六格式落地+降级阶梯);数据源变更事件化实时联动知识库(信号实时、生效仍走人工闸门);搜索结果按指纹折叠重复(文档+图片)、可展开查看合并情况与重复内容;文档预处理与知识提取全过程可视化(加工时间线+产物对照原文);知识编辑支持图片无缝上传嵌入 markdown、页面附件统一管理;搜索结果配右侧知识面板(实体关系图谱/标签云/信息卡)与左侧聚合筛选;知识库=内置数据源(发布文章索引层与文档统一),AI 聊天知识分层——知识库/实体优先于原始数据源;**左侧菜单按六大类重组(首页/知识/数据/检索/智能/系统),settings 14 tab 分组收敛,新功能一律先归类再进导航**;文档生命周期可观测、可重入;组织面(虚拟目录/受控标签/FAQ 库)补齐;agent 面(确认制长期记忆/带令牌的 MCP 端点)开放;安全三件套垫底:凭据加密(S1)、出站 SSRF 收口(S2)、引擎原生安全整合(S3——字段级权限/字段脱敏/属性化文档权限,策略单源投影、双层执行)。

**抄机制、不抄架构**:全部借件均以"Easysearch 原生 + 既有管道/队列/治理队列"承载,不引入 WeKnora 的 pgvector/Asynq/Neo4j/docreader 侧车任何一件。

## 1. 目标与非目标

**目标**
1. 检索粒度从整文档降到块,引用从文档级升到页/节级(D2 wiki locator 的文档侧断点接通)
2. 文档全生命周期(建/编/重处理/删)与富化状态可观测、幂等、可重入
3. 解析质量分级提升:零模型启发式先行,外部视觉后端可插拔(默认关)
4. 知识组织补齐:虚拟目录、受控标签、FAQ 库类型
5. agent 面开放:确认制长期记忆、按端点令牌发布的 MCP 服务
6. 安全垫底:凭据静态加密(S1)、出站 SSRF 收口(S2)、**引擎原生安全整合(S3:字段级权限/字段脱敏/属性化文档权限——存储层强制+应用层兜底双层执行)**
7. **实体成为知识库一级入口**:全站实体卡片(凡实体出现处)、实体作为独立搜索结果类型、实体编辑向关联页面传播、实体证据链落到块级
8. **每条知识指明来源到源头块位置并可视化**:文章/实体/答案的引用带 locator+quote,点击穿透原文高亮;生成侧引用协议 fail-closed 防幻觉
9. **附件作为证据增补知识文档**(AI 只产更正草稿,人工采纳);**数据源变更事件化实时联动**知识库关联数据(提议实时入队,页面更新人工闸门)
10. **去重可视化**:文档与图片按指纹(sha256/simhash/phash)在搜索结果折叠展示,展开可见组员清单与合并证据;**折叠尊重评审**——标记非重复的成对内容永不折叠
11. **加工可视化**:文档预处理与知识提取(解析/切块/实体/标签/摘要/嵌入/指纹)以时间线展示过程(状态/耗时/尝试/降级原因),产物对照原文锚定展示(块着色/实体证据高亮/结构单元定位)
12. **编辑面闭环**:wiki 编辑器支持图片粘贴/拖拽/按钮三入口无缝上传并插入 markdown;页面级附件(内嵌图/文件/证据)有统一管理面板,含孤儿检测与引用一致性保护
13. **搜索知识面板与聚合筛选**:右侧面板展示实体关系图谱/标签云/各类信息卡;左侧补充数据源(含内置知识库)/类型/标签/实体类型/时间等聚合筛选
14. **知识库=内置数据源**:发布文章以文档同款 mapping 投影进文档索引(治理记录为事实源、文档索引为检索投影);AI 聊天知识分层——知识库/实体优先于原始数据源(检索权重+上下文组装顺序+rerank 来源权重三处落地)
15. **预览定位与高亮**:搜索结果/引用/证据点击进预览,直达文件原始页(页/节/sheet 行/时间轴)并高亮命中内容(块背景带+词高亮),六格式落地、四级降级阶梯
16. **导航信息架构重组**:左侧菜单六大类(首页/知识/数据/检索/智能/系统),存量 ~18 平铺页归位、settings 14 tab 分组收敛;新功能一律先归类再进导航(IA 即准入门槛)

**非目标**(见 §8 不做清单):多向量库抽象、对象存储体系、IM 渠道、沙箱/浏览器控制、会话 fork-rewind、RAGFlow 式 LLM 自动合并知识(合并决策必须人工)。

## 2. 设计原则

1. **抄机制不抄架构**——借的是算法/纪律/数据形态,不是部署形态
2. **引擎原生优先**——向量/BM25/RRF 尽量落 Easysearch(含引擎 processor),客户端路径只做降级兜底(P0/P1.5 既定三层规划不变)
3. **降级纪律**——任何新腿/新后端失败不影响主链路,Warning 头注明(语义路/改写路/rerank 同构)
4. **人工闸门**——记忆、知识编译、查重动作、冲突裁决全部走治理提议,AI 只产草稿
5. **评估护航**——任何检索侧改动前后必跑 D9 评估集;任何走查固化为 scenario DSL 进 CI

## 3. 现状基线

### 3.1 缺陷级事实(必须先修)

| # | 事实 | 位置 |
|---|---|---|
| 1 | 语义路只查 `ai_insights.embedding.embedding1024`(每文档一个向量);`document_embedding` 给每块写的向量**从未被任何查询消费**;块文本不在 BM25 字段,响应侧被 field_access 整体排除 | modules/document/service.go:27-29,46;plugins/processors/embedding/processor.go:198-235;field_access.go:18 |
| 2 | `PUT /document/:id` 内容变更不触发重切块/重嵌入(只 orm.Save+指纹重盖);`POST /document`、`POST /datasource/:id/_doc` 直建**完全绕过富化管道** | document.go:323-355;datasource.go:246-324 |
| 3 | 无 `_reprocess` 端点;嵌入模型无 per-doc 版本戳,换模型后新旧向量静默混查 | 全仓无对应;engine_capability.go 已报 with_vectors |

### 3.2 结构性短板

- 切块:定长 7000 rune(pipeline.tpl:53)、零重叠、ChunkRange{起止页}无消费方;表格/图片/正文同一种东西(第六篇病根判定:缺中间表示)
- 解析:Tika text/plain 单后端,无结构、无逐页分诊、无魔数纠偏;EPUB/XMind/MHTML/本地 HTML 全缺
- 组织:无目录树;extract_tags 自由生成无受控词表;检索参数无 tags 作用域;无 FAQ 类型
- 安全:ModelProvider.APIKey 明文落 ES(core/llm_provider.go:31);出站(连接器/scraper/MCP client)无 SSRF 校验

### 3.3 可复用资产(近零新基建)

五路 RRF 引擎(rrfFuseMulti,加路=一个名字+一个权重)/ D9 评估集+run_search_eval / D1 指纹(modules/common/fingerprint,orm 钩子+连接器源侧双盖章)/ 治理提议队列+AnchoredProposalID / wiki_article 页模型(Sources.locator)/ Pipeline Studio dry-run / 引擎 AI 管道(coco-embedding/coco-semantic-rrf 下发机制)/ SearchSettings 灰度开关先例(search_type)/ 队列链(indexing_documents→process_documents→…→ingest_documents)/ pptx.go 原生 zip/XML 解析先例 / **实体卡片标准契约**(@infinilabs/entity-ui 组件 EntityCard/EntityLabel + AvatarLabel 通用封装 + `/entity/card/:type/:id`、`/entity/label/_batch_get` API + `generic#entity:card/read` 权限——W10 接入此契约,不新建)。

## 4. 目标架构

```
连接器/WEbHOOK/API/编辑 ──► indexing_documents 队列
                                   │
                     process_documents(状态机:status 字段)
                                   │
  ┌─ 解析(W2 L0/L0.5→W7 L1)──► 结构化单元 IR {type, text, page, locator, context}
  ├─ 切块(W3):token/rune 预算+硬上限+overlap+面包屑;父子块(mom 隐藏行)
  ├─ 富化:摘要/标签(受控)/实体/指纹/生成式问题/嵌入(块级)
  └─ 双写:coco_document(整文档,BM25+ai_insights,兼容存量)+ coco_chunk(块,新)
                                   │
  检索:/query/_search(hybrid_rrf)─ text/semantic 腿打 coco_chunk(存量文档回落文档级)
     多路 RRF(text/semantic/wiki/graph/rewrite/entity)── 数学不变,加路=名字+权重
     ──► 组合精排(W3:0.6 模型+0.3 RRF+0.1 来源权重;MMR;邻块扩展)
     ──► 命中携带 {chunk_id, locator, quote≤300, breadcrumb} ──► 页/节级引用+高亮
     ──► 指纹折叠(content_hash/dedup_group_id)── 行内展开组员+合并证据(W12)
                                   │
  实体中枢(W10):实体卡片全站 5 展示位 / entity 检索路 / 编辑传播提议 / 证据链块级化
  联动(W11):文档变更 completion 事件 →(实体页/引用文章/交叉引用)刷新提议——信号实时,生效人工
  内置数据源(W16):wiki KB 发布文章投影进 coco_document(治理记录=事实源,文档索引=检索投影,索引层统一)
  搜索面(W15):右侧知识面板(实体卡/迷你图谱/标签云/KB 卡)+ 左侧聚合筛选;AI 分层:知识库/实体 > 原始数据源
  安全(S3):coco 策略单源 → 引擎投影(DLS/FLS/掩码)双层执行;探针不可用回退应用层
  agent 面:确认制记忆(治理提议→coco_memory)/ MCP 端点令牌(/mcp/:endpoint_id)
```

**coco_chunk 映射草案**(Easysearch,权限按 source 反规范化复用 BuildDatasourceFilter):

```
id, doc_id, source{type,name,id}, seq, chunk_type(text|table|image_ocr|faq|summary),
breadcrumb(text+keyword, 标题面包屑), text(text, BM25;title.pinyin 同款分析器可选),
quote(keyword, ≤300 原文摘录), locators(object: {pdf:{page,bbox}, sheet:{name,row_start,row_end}, slide:{n}, section:{title}, time:{start_ms,end_ms}}),
mom_id(keyword, 父块链接), available(boolean, mom 行 false 不进检索),
embedding{embedding1024 knn_dense_float_vector(与现文档索引同配置)},
model_id(keyword, 嵌入模型戳), content_hash(keyword), created/updated
```

## 5. 工作流设计

### W0. 缺陷修复批(先行,独立可发)

- **W0.1 编辑闭环**:`PUT /document/:id` 比较 content 指纹(fingerprint 包现成),内容实变 → 保存后推 `indexing_documents` 队列重跑管道(重切块/重嵌入/重摘要);仅改 tags/title/元数据不动管道(指纹不变即零成本,天然幂等)
- **W0.2 入口闭环**:`POST /document`、`POST /datasource/:id/_doc(/:doc_id)` 从 orm.Create 直落改为 create 后推同一队列;text_attachment_extraction 增加纯文本直通分支(无附件 → content 视为单页进切块)
- **验收**:编辑后块/向量按管道时延刷新;API 直建文档获得摘要/标签/块;单测+scenario DSL

### W1. 生命周期与重处理(D15)+ S1 凭据加密

- **状态机**:Document 增 `status`(indexing|completed|failed)+`error_message`;富化在途可观测——管道各节点耗时/错误收进运行记录(数据钩子见 W13 加工可视化,v1 最小形态 metadata.pipeline_stats[{node,ms,error}])
- **_reprocess**:`POST /document/:id/_reprocess`(单)+ `POST /datasource/:id/_reprocess`(批量走队列);语义=清 document_chunk/ai_insights/summary → 重入队 → **重戳 embedding_model**;content_hash 未变且 status=completed 时跳过(幂等,force=true 除外)
- **嵌入模型治理**:Document/chunk 落 `embedding_model` 戳;engine-capability 增按模型分布聚合;默认嵌入模型变更时设置页提示"N 篇文档在旧模型上——批量重处理?"
- **入库去重闸门**:process_documents 消费侧查 content_hash+source.id 已存在 → 跳过创建、刷 updated(D1.5 从事后扫描前移到入口;默认开,数据源可关)
- **S1 凭据加密**:`modules/common/secretbox`——AES-256-GCM,主钥环境变量(缺失=明文兼容模式+启动警告);密文 `enc:v1:` 前缀,读侧容忍历史明文(渐进迁移零停机);覆盖 ModelProvider.APIKey、datasource.connector.config 敏感键(api_key/token/secret/password 按键名识别)、MCP server 凭据;出参永不回显(在既有序列化脱敏之上加规则);可选后台回填任务
- **验收**:状态机单测;reprocess 幂等;同文件二次摄取单文档;加密往返/明文兼容/响应无密钥(grep 测试);**密钥丢失=凭据不可读需重录,无托管**(文档明示)

### W2. 解析分级 L0/L0.5(D13 前半,全零模型零依赖)

- **L0**(落在既有处理器):①Tika 切 **XHTML 路径**拿结构(div.page 内 h1-h6/表格 → 结构化单元,喂 W3);②**魔数纠偏**(OLE 魔数的".docx"→DOC,file_type_detection 加;WPS/改名容忍);③**Excel 结构化路**(原生 xlsx zip/XML 解析,先例 pptx.go:合并单元格先填充左上主值→每行 `列名: 值` 块+sheet/行区间 locator+首行表头开关;剔除 =DISPIMG/_xlfn.IMAGE);④保守页眉页脚剔除(每页首尾行+短+≥60% 文本页才删)+乱码页检测(char 分布异常→标记强制 OCR);⑤PDF outline 书签提取(原生最小 PDF 对象解析,**零新依赖**约束下 ~200 行;做不了则随 L1 后端做)
- **L0.5 格式补全**(Go 原生 stdlib):EPUB(zip+XHTML→markdown,TOC 序+DC 元数据)、XMind(content.json→缩进大纲)、MHTML(MIME 解析+Content-Location 图别名)、本地 HTML;音频 ingestion 走既有 enterprise ASR(时间轴 locator)
- **验收**:每格式 fixture 单测;scenario DSL 带样例文件;CI 校验 go.mod 零变更

### W3. 分块与块级检索(D10+D14+D4.5)——核心工作流

**写入侧(D10)**
- **中间表示先行**:解析器输出 `ChunkUnit{type(text|table|image|heading), text, page, locator, context_above/below}`;切块器消费 IR 而非字符串(旧数据源无 IR → 整页当单 text 单元,优雅回落)
- **预算与契约**:v1 用 rune 预算×语言系数(zh 1.7 / en 4.0,WeKnora 同款)**硬上限契约**(任何块 ≤ 预算;超预算单元先句子边界展开,仍超硬切;裸分隔符保留,切片拼回=原文逐字节——无损性单测锁);overlap 15-20%(前缀修边不超上限);保护 span(代码围栏/表格/LaTeX)不切断;BPE 表因零依赖红线不做,v2 再议
- **面包屑**:heading 单元路径 → chunk.breadcrumb,拼入嵌入输入(单块编辑单块重嵌)
- **表格/图片不进文本合并流**:表格单元遇之即 flush;大表切片**表头随片**;图片单元带上下文窗口
- **父子块(mom 形态)**:父=切分前自然单元(天然段落/小节尺寸,不另设父切分);mom 行 available=false 不进检索,子行带 mom_id;检索命中子块,Refine 返回 mom 文本作上下文

**检索侧(D14)**
- text/semantic 腿改打 coco_chunk;**存量文档回落**:同请求内文档级腿并行跑,未迁移文档(无块)由文档级腿覆盖,对外仍是一个 route,RRF 权重零改动
- 向量化双路:客户端 document_embedding 扩写块索引 + 引擎侧 `coco-chunk-embedding` ingest 管道(镜像 coco-embedding 下发机制,引擎优先客户端兜底)
- **引用升级**:命中携带 {chunk_id, locator, quote≤300, breadcrumb};wiki Sources.locator 的 page|section|clause 从此有文档侧着落;引用展示的预览落点与高亮交互独立成 **W17**
- **组合精排(D4.5)**:0.6×模型分+0.3×RRF 归一分+0.1×来源权重(数据源→权重配置表);MMR λ=0.7 去冗;邻块扩展(命中块 <350 字沿 seq 前后扩至 ~850);上轮引用注入(助手侧,与上轮引文 Jaccard≥0.15 才注入、≤3 条);无 reranker 降级纯 RRF(现状不变)
- **迁移**:不强制重切;数据源级 _reprocess 选迁;检索折叠升级项独立成 W12(本工作流不碰折叠)
- **验收(硬门槛)**:无损性(拼回==原文)/overlap 上限/评估集**不回归**(top-4/MRR 基线之上,块级用例新增)/权限过滤在块索引上经非管理员场景验证/引用 locator 在 fixture 上逐条正确

### W4. 知识组织与受控标签(D16)

- **虚拟目录**:Document 增 `folder_path`(keyword,规范 /a/b,深度 ≤16、段 ≤128 字符);树=聚合端点;移动/改名=path 前缀 rewrite(**不重解析**)
- **受控标签**:数据源级词表 `tags_vocab`;extract_tags 受控模式(prompt 附词表,输出必须是子集,未知词丢弃并计数);**永不新建、永不删手工标签**;检索参数 `?tags=`(text 腿 filter,与 disabled 过滤同构)
- **能力位**:`GET /datasource/:id/capabilities` 计算 {vector, wiki, graph, faq}——助手编辑器与 MCP search_documents schema 暴露,agent 按能力过滤
- **AI 描述(GeneratedProfile)**:数据源 `ai_description` 独立字段,语料聚合 hash 不变跳过模型调用,**永不覆盖人工描述**
- **验收**:词表约束单测;目录移动零重解析;能力位正确性

### W5. FAQ 知识库(D11)

- **形态**:FAQ 条目=普通文档 type=faq,metadata.faq{standard, similar[], negative[], answer, strategy};走标准管道+`faq_compile` 节 → chunk_type=faq 块:**索引内容=标准问+相似问**(question_answer 模式追加答案),**负例问题绝不入索引**(存文档侧供查询期过滤)
- **查询期**:text/semantic 腿限 chunk_type=faq 的混合召回 → 负例精确过滤(归一化精确命中即整条出局)→ 唯一条目不足且向量饱和则迭代倍增 TopK(≤5 轮,seed=3×TopK 封顶 500);FAQ 加权走 D4.5 来源权重或 rerank 专项 boost(不新增 RRF 路,路由面稳定)
- **直答**:top1 ≥ 阈值 → 响应标 exact=true 附标准答案,模型提示词允许直接采用
- **幂等**:问题归一化(复用 D1 归一化链)ContentHash → 同问 upsert 不重复;CSV/Excel 批量导入(复用 W2 xlsx 解析)
- **验收**:负例过滤/迭代扩展/exact 标记/CSV 导入 E2E

### W6. 长期记忆——确认制(D12)

- **形态**:治理提议 type=memory{kind: profile|preference|fact|task|interest, content, 上下文(会话/查询/答案摘录)};**pending 不确认绝不注入**——人工闸门同 D1/D7;确认→写 `coco_memory` 索引{user_id, kind, content, embedding1024, supersedes_id, status};同主题新推断确认前旧条目保持生效;拒绝留抑制记录
- **注入**:fact/task 查询期向量召回 top-3(用户隔离);profile/preference 常驻限长(~1500 字)进系统提示词组装
- **蒸馏**:会话结束异步(延迟 90s、每用户间隔 ≥300s);游标 checkpoint 落 kv 防重启丢消息;蒸馏模型=默认语言模型;输出 JSON 候选数组
- **agent 面**:search_memory MCP 工具(第 46 个,当前用户作用域);隐私:按用户隔离,提议负载最小化(评审者可见上下文是刻意权衡,文档明示)
- **验收**:pending 不注入/确认后注入/取代语义/限频/重启游标

### W7. MCP 端点令牌(P5)+ S2 SSRF + 解析后端 L1(D13 后半)

- **端点实体**:mcp_endpoint{name, endpoint_id, token_hash(明文仅示一次/可轮换), enabled, datasource_ids[](空=全部), tool_groups(read/ask/wiki/write 默认关), rate_limit_per_minute, last_used_at};路由 `/mcp/:endpoint_id` Bearer 鉴权,作用域=端点白名单 ∩ api.MCPTool 权限过滤;限速进程内滑动窗(单二进制无 Redis,多实例局限文档明示)
- **S2 出站收口**:`modules/common/netguard`——ValidateURL(拒绝内网/环回/link-local/云 metadata 169.254.169.254/危险端口;白名单模式可配默认关,DNS 解析前拒绝);自定义 CheckRedirect **逐跳重验**;挂载点:连接器抓取/scraper 工具/MCP client/L1 解析后端
- **L1 可插拔解析后端**:接口 `parser.Backend{Parse(ctx, file, cfg) → []ChunkUnit}`;注册表 builtin(Tika)+remote(url/timeout);per-数据源 parser_rules{文件类型→后端+类型化覆盖};**FirstParser 首胜链**(remote→Tika,首个非空即胜);**嵌套超时预算**(后端<单次调用<文档总限);mineru/docling 类远程视觉服务可选配,默认全关
- **验收**:端点令牌 E2E+作用域收窄;SSRF 单测(私网/metadata/重定向至私网全拒);L1 mock 后端降级链

### W8. 连接器护栏(并入 W6 工程质量项)

- **流式断点续传**:分页连接器按资源粒度游标(kv),每 50 项/30s checkpoint,崩溃续传
- **删除护栏**:本轮同步将丢失 ≥20 项且 ≥80% 存量 → 中止并产 sync_anomaly 治理提议(防凭证过期把全库"同步"没了)
- **退避纪律**:429/5xx 指数 2/4/8s×3 统一助手
- **验收**:护栏中止单测;checkpoint 续传单测

### W9. 知识编译器(D17,远期,独立设计见 WORKPLAN 第七篇)

四段流水线 MAP→REDUCE→PLAN→REFINE,产物全部改道治理提议(人工发布红线不变);依赖 W3 的块与 SourceChunkIDs;验收口径同 WORKPLAN(同批二次编译零 LLM、增量只产受影响页提议、证据链可点回原文)。**本设计不展开,排期待 W3 落地后**。

### W10. 实体中枢与引用可视化 — 新增(2026-10-08 需求合入)

**动因**:实体是知识库的一级入口;凡实体出现处展示实体卡片;搜索同时覆盖文章与实体;每条知识指明来源到源头块位置并可视化展示。

**实体卡片(全站五展示位)——复用既有标准化契约,零新卡片端点**
- **既有标准(已核实,不得另起炉灶)**:UI 组件库 `@infinilabs/entity-ui` 的 `EntityCard`(弹出卡,autoPlacement)/`EntityLabel`(标签);通用封装 `web/src/components/Resource/AvatarLabel.tsx`(showCard 开关、点开懒加载);标准 API 契约(web/src/service/api/entity.ts):`POST /entity/card/:type/:id`(卡片)、`POST /entity/card/user/:id`(用户卡)、`POST /entity/label/_batch_get`(批量标签,search 页/useResource 已在消费);权限键 `generic#entity:card/read`。Go 侧当前未注册这些路由——**本项工作=在 coco Go 侧实现既有契约并把 wiki 实体接入**
- 落法:契约路由实现按 EntityCard 组件消费的形状返回;wiki 实体注册为新 type(如 `wiki_entity`,id=实体 ID),字段映射 {name,type,aliases,summary,status,article 链接,top_relations,backlink/source 计数};`_batch_get` 支持该 type 批量解析(entity 检索路结果的行内胶囊直接用);服务端缓存 60s 租户级、实体更新即失效
- 五个展示位全部走 AvatarLabel(showCard)/_batch_get:①搜索结果行内胶囊(文档命中已带 entity_ids);②wiki 文章 wikilink 胶囊(O4 既有胶囊包 AvatarLabel,点击出卡);③文档详情页 entity_ids 卡片墙;④助手答案引用面板;⑤MCP 工具 get_entity_card(数据形状与卡片契约同构)——卡片是只读导航件,不承担写操作

**entity 检索路(RRF 第六路)**
- BM25 打 WikiEntity(name^20/aliases/type,combined copy_to 既有模式),仅 reviewed/published(与 graph 路同闸门);命中包装伪文档(Source=entity,直达实体页/卡片)
- 与 graph 路分工:graph=关系遍历(识别种子→邻居页);entity=实体本身词面/别名直接命中("查支付网关"不依赖关系存在)
- 参数 entity_weight 同型;搜索实验室加面板;结果 UI 分型卡片(文档/wiki 文章/实体);MCP search_documents schema 加 type 过滤

**实体证据链升级(块级来源)**
- extract_entities 落 evidence 列表:entity.sources 从 doc_id 平面列表升级为 [{doc_id, chunk_id, locator, quote}](依赖 W3 块索引;过渡期 doc_id 兜底)——"每个知识指明来源到块位置"对实体同样成立
- 实体页 Sources 区按文档分组、逐条点击穿透(同下方引用可视化规格)

**实体编辑传播(编辑合入各知识库页面)**
- 两个入口:①**提取侧合入**——管道 extract_entities 增量合并 entity:新证据并入 sources、新别名产提议,声明性字段(别名/来源)直接更新,实体页正文变更走提议;②**人工编辑侧**——实体管理页 merge/改名/类型变更/关系修改
- 传播:按 AnchoredEntityID 反查引用文章(实体页+wikilink 反链)→ 逐篇 `article_refresh` 治理提议(负载:变更 diff+受影响文章清单+建议编辑点);merge 场景额外产 wikilink 重定向建议(旧名胶囊→新名)
- 红线:全部提议态、人工在文章面应用;幂等锚 AnchoredProposalID(entity 锚+代数)
- 关联数据更新:实体/文章变更 → linked_pages/反链/toc 重算提议(W11 事件链统一分发)

**引用可视化规格(展示侧)**
- **行内引用徽标**:答案/文章内 [n] 或胶囊,hover 弹出来源卡(文档标题+面包屑+quote 原文高亮)
- **点击穿透**:来源卡 → 文档预览直达 locator(page/section/sheet 行区间)并高亮 quote;对齐只比字母数字,老格式回退文本搜索(补遗二纪律)
- **引用面板**:文档/文章详情页列全部 Sources,按文档分组、逐条带 locator 链接
- **生成侧协议**(WeKnora fail-closed 借件):答案生成前注册 cN 别名(块/wiki 页/实体),提示词要求自闭合引用标签、禁止造新别名,**流式解码未知别名直接删除**——结构性防幻觉引用
- **验收**:五展示位卡片渲染**全部走既有 EntityCard/AvatarLabel 标准组件与 /entity/card 契约**(零新卡片端点、零新卡片组件);entity 路评估集用例(实体名/别名命中);编辑传播提议幂等;穿透在 fixture 上逐条 locator 断言;未知引用别名被删(单测)

### W11. 附件增补与实时联动 — 新增(2026-10-08 需求合入)

**附件作为证据增补知识文档**
- 入口:文档详情页**拖拽区** → `POST /document/:id/attachments`(复用 /attachment/_upload + enrich_attachments 管道:Tika/OCR/切块);attachment 记 relation=amendment_evidence 挂 document.attachments
- 提炼(amendment 处理器/独立 stage):
  1. 附件按 W2 管道解析 → 块+locator
  2. LLM 对齐:附件主张 vs 当前文档内容 → 结构化 diff 三类{**更正**(冲突,双方证据)/ **补充**(新信息)/ **过期**(文档内容被附件推翻)},每条带附件块 locator 引用
  3. 产物=`document_amendment` 治理提议(diff 列表+逐条证据链+建议改写文本)——**绝不直接改文档**(红线)
  4. 人工裁决:采纳条目合并进文档 content → 触发 W0.1 编辑闭环(指纹变→重切块/重嵌入/重摘要,下游实体/文章联动自动跟随);驳回留抑制记录
- 与 D7 分工:D7=用户口头纠正,amendment=文件证据纠正——同一治理队列不同类型,裁决路由表同构
- **验收**:拖拽→提议端到端;采纳后文档+块刷新(时延断言);证据链点击穿透到附件 locator

**数据源变更实时联动**
- 事件链(既有钩子串联,零新组件):
  1. **变更感知**:webhook 推送优先(既有,秒级);轮询兜底(dispatcher interval;数据库类连接器 mysql/pg/mongo 等走 sync.strategy 增量)
  2. **文档新增/更新/删除**(W0.1/W0.2 闭环路径全覆盖)→ 管道完成(W1 status 翻转)→ **completion 事件**
  3. **completion 分发**(轻量事件器:进程内分发+ES 索引当 durable log,同锚末事件赢;trailing-edge debounce):
     - 实体面:entity.sources 增量合并 → 实体页新鲜度检查(governance/freshness 既有钩子)→ stale/refresh 提议
     - 文章面:按 Sources.doc_id 反查引用文章 → article_refresh 提议;**文档删除→失证据页只标 stale,绝不自动删**(D17 撤证据语义前置)
     - 交叉引用:linked_pages/反链/toc 重算提议
  4. 实体编辑传播(W10)同链入队
- **"实时"口径**:事件驱动——变更到提议入队=管道完成即触发(分钟级,LLM 阶段主导);页面内容更新仍走人工闸门。**实时的是信号,不是生效**
- 幂等:提议锚定(AnchoredProposalID,doc/anchor+generation);开放提议只刷计数
- **验收**:E2E 场景——文档更新→治理队列出现 article_refresh/stale 提议(时延断言);删除→失证据只标 stale 不自动删;webhook 与轮询双路径覆盖

### W12. 去重折叠与展开交互(文档+图片)— 新增(2026-10-08 需求合入)

**动因**:去重不该只活在查重报告里——搜索结果要折叠重复(文档+图片)、可查看合并情况、可展开看重复内容。已有底座:D1 四档查重+评审闭环(排除/删除/标记非重复/暂缓)、D1.5 指纹落库+**页内** hash 折叠(fingerprint_duplicates{count,ids});缺:图片去重、全局折叠、展开交互与合并证据展示。

**图片指纹(新,确定性零 LLM)**
- 64 位感知哈希 phash(原生 Go 零依赖:解码→缩 32×32 灰度→DCT→左上 8×8→中值阈值),与 D1 simhash 同哲学;二进制 sha256 做精确档
- 盖章点:附件管道(image 描述路径/generate_cover)与图片类文档(file_type_detection 判图后)——Attachment 增 phash 字段;图片文档落 metadata.image_phash;**连接器路径同样要盖**(沿用 D1.5 源侧盖章模式,不重蹈"orm 钩子绕过"覆辙)
- 分档:sha256 精确相同 / phash 海明 ≤8 近似;**两段式验证**:phash 只提名,再以缩略图差异像素比复核——防渐变背景/裁边截图误报(同 D1"simhash 只提名、bigram Jaccard 认边"纪律)

**查重报告扩展**
- dedup 扫描纳入图片对象(附件+图片文档),四档标签同构(完全相同/OCR 内容相同/高度相似/疑似变体);证据带缩略图对比
- 新动作"**确认分组**":操作员确认某组为真重复 → 组员盖 dedup_group_id(代表位=最新更新优先);与折叠的关系:hash 精确组自动折叠,**近似组须确认后才折叠**(误报不进搜索面),dismiss 的成对内容永不折叠

**检索折叠升级(全局化+富证据+展开)**
- 折叠键:content_hash(精确,自动)+dedup_group_id(确认的近似组);折叠尊重评审——DocumentDedupDismissal 命中对永不折
- **全局折叠**:优先引擎原生 `collapse`(content_hash+inner_hits 带组员)——先探针验证 Easysearch 支持(项目探针纪律),不支持回落现行页内折叠(不倒退)
- 折叠行富负载:fingerprint_duplicates 从 {count,ids} 扩为 {count, ids, tier, similarity%, members[{id,title,source,size,updated}]};图片折叠行=代表缩略图+"N 张相同/相似"+组员缩略图网格
- **展开交互**:折叠行胶囊("另有 N 个重复·完全相同/高度相似")→ 行内手风琴展开组员(标题/来源/大小/时间/相似度/直达预览)→ 底部"在查重报告中处理"(跳既有查重 tab)——**搜索面只看+跳转,不做删除/排除动作**(动作留在查重报告二次确认,红线同构)
- 生效面:/query/_search(app/widget/MCP 共用)+文档管理列表页;widget(ui-search)同步交互
- 与 W3 协调:折叠作用于融合后的文档级结果(块级命中折叠到文档),两工作流同碰 /query/_search 响应组装,实施时合批联调

**验收**:phash 稳定性单测(同图缩放/重压缩哈希不变、异图海明大)+两段式防误报;dismiss 永不折叠(场景断言);collapse 探针+回落路径;折叠行展开组员与证据完整;图片组缩略图网格;web+widget 双端走查

### W13. 文档加工可视化(过程+产物)— 新增(2026-10-08 需求合入)

**动因**:预处理与知识提取对用户是黑盒——解析出什么结构、切成什么样、抽了哪些实体/标签、嵌入是否成功,只有最终字段没有过程;出错(空摘要/缺向量/漏实体/切得稀碎)无从下手。要求:过程可看(时间线)、产物可看(对照原文锚定展示)。

**过程可视化(加工时间线)**
- 数据基座(W1 钩子的完整形态):管道运行记录 pipeline_runs,每节点 {node, processor, status(done|running|failed|skipped), started_at, took_ms, attempt, error, degrade_note(降级原因,与检索 Warning 纪律同源)};每文档保留最近 N 次运行(N 默认 5,含 _reprocess 各次);重负载快照(节点输入/输出样本)存 KV 带 TTL,**不进 Document 本体**
- 端点:`GET /document/:id/_timeline`(时间线+运行列表);在途文档靠 status 机轮询刷新(v1 轮询,SSE 后置)
- UI:文档详情页**"加工"面板**——垂直时间线/流程图,节点卡带状态徽标/耗时/尝试号/错误详情/降级说明;失败节点展开错误;时间线尾部挂"重新处理"入口(直连 W1 _reprocess)
- **与 Pipeline Studio 打通**:"用本文档试跑"——dry-run 与生产同一 DSL(既有机制),结果用同一时间线组件渲染;改配置→试跑→对照生产时间线,配置试验闭环

**产物可视化(左原文右产物,逐类锚定)**
- **解析产物(W2 IR)**:页面分诊徽标(text/scanned/强制 OCR)、结构单元清单(heading/table/image 带页码/bbox/行区间),点击滚动原文对应位置
- **切块产物(W3)**:块列表(面包屑+quote+块型),块区段在原文预览**着色分段**;父子块(mom)缩进树;每块向量状态(有/无/模型戳/维度——只显状态不显向量本体);切分依据标注(标题边界/句子边界/预算上限)——"为什么这么切"直接可见
- **实体产物(W10,知识提取核心可视化)**:实体卡片行,每卡带**证据位置**(实体从哪几块抽出,块 locator+quote),点证据跳原文高亮
- **标签/摘要(W4)**:标签 chips(受控命中/未知被弃分别标识)、summary/ai_insights 渲染
- **指纹(W12)**:content_hash/simhash/phash 值+所属折叠组链接(有则显示)
- **增补证据(W11)**:amendment 附件与提议的关联视图(有则显示)
- 权限:产物面板吃字段级权限(field_access 既有),无权字段不可见

**验收**:fixture 文档时间线全节点断言(状态/耗时/降级说明);在途→完成状态实时翻转;每类产物点击锚定原文位置正确(与 W3 引用 locator 断言同族);dry-run 与生产时间线同构渲染;字段权限隔离生效

### W14. 编辑器图片嵌入与页面附件管理 — 新增(2026-10-08 需求合入)

**动因**:wiki 文章编辑器目前只能贴纯文本/外链图片;图片与附件没有"上传→插入→管理"闭环;页面级附件散落正文无从统一管理。既有底座:core.Attachment(KV bucket 二进制+ES 元数据)、attachment:// 运行时重写(search.go 既有)、/attachment/_upload 上传管道。

**图片无缝上传插入**
- 编辑器三入口:①**粘贴**(截图直接 Ctrl+V)②**拖拽**入编辑区③工具栏上传按钮
- 流程:本地校验(类型按魔数非扩展名+大小上限默认 10MB)→ POST /attachment/_upload(带 scope)→ 返回 attachment://UUID → 光标/落点处插入 `![文件名](attachment://UUID)`;上传期间先插占位符,完成即替换,失败占位转错误提示
- **引用纪律**:markdown 源**只存规范引用 `attachment://UUID`,绝不存渲染 URL**——二进制可迁移、鉴权/签名留在服务时;渲染管线复用既有 attachment:// 重写机制,wiki 文章渲染路径接入
- 图片处理:上传即生成缩略图(既有 generate_cover 能力)供编辑器即时预览与面板网格;OCR/VLM 描述后置随 W2 图片管道(产物进 W13 加工面板)

**页面级附件统一管理**
- 数据模型:Attachment 增 owner scope{type(document|wiki_article|chat), id, relation(embedded|linked|evidence)}——聊天上传=用户作用域,文档/文章附件=资源作用域;W11 的 amendment_evidence 挂同一模型(先此定模型,W11 实施时直接用)
- 端点:GET/POST/DELETE `/wiki/article/:id/attachment(s)`(列表/追加/移除;移除=解除关联,无其他引用时可选删二进制)
- **管理面板**(编辑器侧栏+文章详情页同款):全部附件分组展示——正文内嵌图(缩略图网格)/页面文件(名/类型/大小/时间列表)/证据附件(W11);动作:插入正文(光标处)/预览/下载/移除
- **引用一致性保护**:面板实时比对"正文引用(解析 attachment://)vs 关联清单"——**孤儿附件**(上传未引用/正文删图后残留)打标可一键清理;移除已内嵌附件时警告正文引用数并要求确认(**绝不静默改正文**,红线同构)
- 权限:附件访问跟资源作用域——私有文章附件需登录+文章读权;公开文章附件用短 TTL 签名 URL;上传/管理=文章编辑权

**验收**:粘贴/拖拽/按钮三入口 E2E(截图→光标处出图);markdown 源只含规范引用(渲染 URL 不落库,断言);孤儿检测准确(插入后删正文引用场景);移除内嵌附件有引用警告与确认;私有文章附件未授权 403、签名过期失效;缩略图即时预览

### W15. 搜索知识面板与聚合筛选 — 新增(2026-10-08 需求合入)

**动因**:搜索结果目前只有列表;实体/知识库相关属性(关系图谱/标签云/各类信息卡)无处展示;筛选面只有 tags 聚合一团。要求:右侧知识面板+左侧聚合筛选,丰富结果展示。

**右侧知识面板(查询上下文驱动,组件全走标准契约)**
- 面板实体来源:①查询识别的实体(graph 路种子,既有)②结果集聚合 top 实体(entity_ids terms 聚合)——合并去重,上限 3 个
- 内容组件(全部复用既有标准件,零新组件体系):
  - **实体信息卡**:标准 EntityCard 契约(W10 接入的 /entity/card)——属性/别名/状态/直达文章
  - **迷你关系图谱**:entity_neighbors 一跳,复用 KbGraph 画布小号模式;节点点击出标准卡片、跳实体页
  - **标签云图**:结果集 tags 聚合(词频→字号),点击即加左侧筛选
  - **知识库信息卡**:命中 wiki 文章所属 KB(名称/文章数/相关文章 top3=实体反链),点击进 KB
- 面板可折叠;widget(ui-search)窄屏降级为结果上方横向卡带

**左侧聚合筛选**
- 聚合端点 `/query/_aggregations`:同一 hybrid_rrf 融合窗口内计算(权限过滤与检索完全一致),桶:数据源(**含内置知识库数据源**,W16)、类型(document|wiki_article|faq|entity)、标签(W4 受控词表)、实体类型、时间直方图
- 筛选落位:text 腿 filter(datasource ids/tags/type/时间——与 W4 tags 过滤同机制);wiki/entity 路对 type/datasource 天然可滤;**筛选与折叠(W12)共存**:折叠在筛选后执行
- UI:web 搜索页左侧栏分面;widget 保持精简(tags+数据源两团)

**验收**:面板组件全部走标准契约(EntityCard/KbGraph/聚合);筛选往返一致(选桶→结果⊂桶,清空复原);筛选后折叠仍正确;widget 降级布局;非管理员聚合不越权(权限过滤同检索断言)

### W16. 知识库内置数据源与索引统一 + AI 知识分层 — 新增(2026-10-08 需求合入)

**动因**:wiki 文章在独立 wiki_article 索引,与文档索引两套格式、两套权限。定位修正:**知识库=内置数据源,底层格式至少索引层统一**——统一后聚合/筛选/分块/折叠/加工可视化全走一套机制;AI 聊天要把实体+知识库当重要参考,**优先级高于原始数据源**(JitKnow"核心层直接读、边缘层兜底"的 D2 初衷在此架构闭环)。

**内置数据源模型(W16a,与 W3 合批定索引)**
- 每个 wiki KB 注册一条内置数据源记录(connector 型"内置",不可删只可停用);FAQ 库(W5,type=faq 文档)同机制——先决一致
- **发布即投影**:文章人工发布(status→published)→ 写 coco_document-v2 一行(**同款 mapping**:title/summary/content/tags/entity_ids/url/metadata.wiki_sources 溯源);下架/归档 → 删投影行。**wiki_article 索引仍是治理事实源**(版本/提议/审校状态),文档索引只是检索投影——单向投影,不双向同步
- **wiki 路切换数据源**:从打 wiki_article 索引改为打文档索引(type=wiki_article 过滤);路由与权重(wiki_weight)保持——**索引统一,路保留**(独立路正是优先级的控制手段);噪声闸门天然保持(仅 published 被投影)
- 迁移:存量 published 文章一次性脚本投影(前后计数断言);SearchSettings 灰度开关(与 search_type 纪律同款)可回退旧路
- 权限:内置数据源走数据源分享模型(BuildDatasourceFilter 统一);既有 wiki 文章检索权限过渡期双查兼容

**索引层统一的红利**
- wiki 文章进 W3 块索引(切分/嵌入/溯源 locator 与文档同管道)——聊天引用到文章段落级
- W15 聚合天然含知识库桶;W12 折叠对 wiki 文章生效;W13 加工面板可看文章切块产物;W16 与 D6 来源优先级共用一套数据源权重表

**AI 聊天知识分层(W16b,优先级:知识库/实体 > 原始数据源)**
- **检索层**:助手检索默认 hybrid_rrf;助手场景 wiki/graph/entity 路权重默认档位高于 text/semantic(可按助手覆盖)
- **精排层**:rerank 来源权重(W3 的 0.1×source_weight)默认映射——内置知识源 1.0 > 原始数据源按 D6 来源优先级;治理冲突裁决与检索排序**共用同一优先级表**
- **组装层**:上下文注入顺序 wiki 文章(含实体页)→ 实体卡片摘要 → 原始文档块;分层标注进提示词("知识库内容为审校过的权威参考");引用面板按分层排序
- 降级:无 KB/知识库为空时自然回落纯文档检索,行为不变
- **验收**:发布文章即时可检(text 腿+wiki 路双确认);投影与治理记录计数一致;助手上下文顺序断言(知识库先);rerank 来源权重生效(评估集用例:同分内容知识库位次靠前);权限映射不越权;存量迁移零丢失(计数断言)

### W17. 预览定位与高亮 — 新增(2026-10-08 需求合入)**动因**:搜索结果/引用/证据点击进预览后落在文档开头,命中位置要用户自己找。要求:预览**直达文件原始页并高亮**命中内容。

**深链契约(所有入口统一)**
- 命中携带高亮上下文(W3 检索侧数据):{best_chunk_id, locator, quote≤300, breadcrumb, terms(归一化查询词;改写路命中含改写词,D8)}
- 预览 URL 结构化参数:`/preview/document/:id?q=<terms>&chunk=<id>&loc=pdf:12 | section:<title> | sheet:S2:5-9 | time:81200-90400`;wiki 文章用 section anchor
- 消费入口:搜索结果行(web+widget ResultDetail)、答案引用点击(W10)、引用面板、amendment 证据(W11)、实体证据(W10/W13)——一处契约处处可用

**六格式落地策略**
1. **PDF 原件**:打开即跳页(#page=N/内嵌 viewer 页参数);有文本层则命中词高亮(viewer 内搜索定位);扫描件(OCR 路径)→ 跳页+OCR 文本侧栏定位
2. **Office 原件**(s3/local_fs 流式):原件模式页级定位能力受限→回落提取文本预览模式定位高亮,顶部保留"查看原件"入口
3. **文本/HTML/Markdown**(coco 渲染路径):chunk StartAt 偏移可用则精确滚动,否则 quote 模糊定位(对齐只比字母数字——补遗二纪律)
4. **wiki 文章**(W16 投影):section anchor 滚动+高亮
5. **Excel**:sheet+行区间定位(切 sheet+滚行/选区)
6. **音频**(enterprise ASR,W2 L0.5):时间轴定位(seek start_ms)

**四级降级阶梯(诚实降级)**:chunk 精确锚 → quote 模糊匹配 → terms 全文搜索定位 → 顶部+状态提示"未能定位";无 locator 的存量文档自动落到第三级(terms 搜索)仍可用

**高亮实现(零新依赖)**
- 文本:CSS Custom Highlight API 可用则用,不可用回落 DOM Range+`<mark>` 包裹;**两层高亮**:命中块背景带(quote 范围)+ 词高亮(terms 前三个,多词多色),图例说明
- 大文档性能:可视区优先高亮(RAF 分帧),全量高亮按需展开
- widget(ui-search)ResultDetail 同能力

**可观测**:定位成功率进 ops 概览(landing_rate 分档:精确锚/模糊/词搜/未定位)——降级诚实可度量

**验收**:六格式各 fixture 场景(页跳转/滚动/选区/seek 断言);降级阶梯各级触发正确(含存量文档回落);widget 同步;landing_rate 出数;多词高亮互不干扰;折叠组(W12)定位到代表位

### W18. 导航信息架构重组 — 新增(2026-10-08 需求合入)

**动因**:控制台现状是 ~18 个顶级页面全平铺 + settings 单页挂 14 个 tab(外观/服务器/引擎 AI/查重/连接器/Wiki…全塞一处);知识库 v2 的新功能面(实体中枢/FAQ/记忆/查重折叠/引擎安全)若无归类将继续平铺恶化。要求:按**几个大类**重组左侧菜单,功能向类内归置并细化。

**六大类目标架构(左菜单分组;顶导 search/chat/wiki 三应用切换不变——左菜单管控制台,顶管应用形态)**

1. **首页** — home
2. **知识** — 知识库(wiki 列表/KB 详情/文章/共享/治理队列)、本体 schema、实体管理(W10)、公司地图(D6);**新面落位**:FAQ 库(W5)、长期记忆管理(W6,确认/拒绝/导出)
3. **数据** — 数据源(含文档管理/附件)、连接器、Webhook、处理管道(Pipeline Studio;W13"用本文档试跑"入口在此);**新面落位**:内容查重从 settings tab 挪出为独立页(W12 查重报告+确认分组+折叠组管理)
4. **检索** — 检索实验室(多路调参/评估集 D9/改写面板 D8;W15 聚合预览在此试跑)、检索运营(策略分布/低召回榜/缺口回流;W17 landing_rate、W13 失败文档榜并入)
5. **智能** — AI 助手、模型供应商、MCP 服务(P5 端点令牌管理并入本页)、技能、集成(widget/渠道)
6. **系统** — 设置(分组收敛,见下)、用户、角色/权限、API Token

**settings 14 tab 收敛(页内二级侧栏分组,不再平铺)**
- 通用:外观、服务器、环境
- 检索:检索设置(默认 search_type)、引擎 AI(引擎安全 S3 面板加挂于此)
- 处理:文档处理(默认管道)
- 安全:数据安全(字段权限/脱敏)、凭据加密状态(S1:是否启用/密钥指纹,不显密钥)
- 知识:Wiki 设置(编译规则)

**落位原则与实施注意**
- 菜单=文件路由派生+权限过滤(auth routes)**机制不动**——重组是渲染层分组(route meta 加 category),api.RequirePermission 照旧;无权限的组自动隐藏
- 存量 URL 兼容:旧路径 redirect 到新分组内路径(书签/分享链接不死);elegant-router 文件路由不变,仅菜单分组与 settings 收敛
- i18n:六大类与新增菜单键补中英(route.* 命名空间)
- 细化原则:**两层为限**——大类→功能页,功能内的多形态(如 wiki 的 kb/article/governance)做页内 tab 不做三级菜单;settings 分组同理
- 新功能一律先答"归哪个类"再进导航(W10 实体→知识、W12 查重→数据、P5 端点→智能·MCP、S3→系统·检索引擎 AI 组)——IA 是新增功能的准入门槛

**验收**:六大类渲染+无权组隐藏(非管理员场景);旧 URL redirect 全量回归(scenario);settings 二级分组 14 tab 归位;中英菜单键齐全(tsc 棘轮);新功能落位表与实际导航一致

### S3. 引擎原生安全整合(字段级权限/字段脱敏/属性化文档权限)— 细化设计(2026-10-08 需求合入,同日按引擎实况细化)

**动因与现状**:coco 已有应用层三级收敛(索引/文档/字段,5c757cd7)+召回后动态脱敏(交模型前);但引擎始终以**单一特权身份**连接——策略在应用层执行,明文数据先到 coco 再过滤,缺存储层的纵深防御。Easysearch 内置安全可承接同源策略,把强制点下沉到引擎。

#### S3.0 引擎能力事实(2026-10-08 核实;来源:官方博文 JSON 实例 + 本机 bundle config/security + OpenSearch security 血统)

| 能力 | Easysearch 机制 | 状态 |
|---|---|---|
| 角色/用户模型 | `PUT _security/role/<name>`:`{cluster[], indices:[{names[], privileges[], field_mask[]}]}`;`PUT _security/user/<name>`:`{password, roles[]}`(bcrypt);role_mapping 支持 external_roles 后端角色映射 | ✅ 已证实(博文 JSON 实例) |
| 字段脱敏 | indices 块内 `field_mask`:`字段名::/正则/::替换模式`,多条正则**从左向右管道**;无正则形态=哈希脱敏(默认 **BLAKE2b**,可选 MD5/SHA-1/SHA-384/SHA-512)——确定性哈希,两处编译同输入同输出天然成立 | ✅ 已证实 |
| 身份模拟 | 请求头 **`security_run_as: <username>`**——特权连接身份模拟档位用户,**档位用户密码不出 coco、请求不换连接** | ✅ 已证实 |
| 配置文件模型 | config/security/{role,user,role_mapping,audit,config,whitelist}.yml;reserved 角色保护 | ✅ 已证实(本地 bundle) |
| 审计 | audit.yml 配置存在(事件细则随启用探察) | ✅ 存在性已证实 |
| DLS/FLS | 同 indices 块内(OpenSearch security 血统,预期 `dls` 查询串 / `fls` 字段列表,exclude 惯例 `~` 前缀) | ⏳ 探针确认项(S3.1,不阻塞架构) |

#### S3.1 适配层(SecurityAdapter,唯一引擎进出口)

```
Capabilities(ctx) -> {dls, fls, mask, runAs, audit}     // 金丝雀探针,60s 缓存(engine-capability 同款纪律)
ApplyRoles(ctx, []EngineRole) error                      // 幂等 PUT _security/role/coco_<tier>
ApplyUsers(ctx, []EngineUser) error                      // 幂等 PUT _security/user/coco_<tier>
FetchRoles(ctx) (map[name]EngineRole, error)             // 漂移对照取"已部署"
WithRunAs(ctx, tierUser string) RequestOption            // 注入 security_run_as 头
```
三态实现:disabled(默认,零开销)/ engine(启用)/ 探针失败自动降级 disabled。全部引擎交互走适配层,检索主链路对执行层级无感知。

#### S3.2 数据模型

- **SecuritySettings**(settings 表,与 SearchSettings 同款表驱动):`{enabled, credential_mode(run_as|pool), mask_rules[{field, action(hash|regex), algo, pattern, replacement}], dry_run, audit_pull_enabled}`;`dry_run` 是灰度开关——只投影+对照,读路径不 run-as
- **档位不落库**(纯函数,请求时推导);**投影记录** `coco_security_projection`(kv/索引):每档位 {signature, role body hash, deployed hash, synced_at}——漂移检测的"期望"基准
- **档位用户密码**:随机 32 字节,S1 `enc:v1:` 存储,永不回显,可轮换(rotate)

#### S3.3 档位归并(纯函数)

- 输入:用户有效权限集(自身角色 ∪ 组,减 deny);输出:**档位签名** = hash(排序(数据源可见集) + 排序(字段限制集) + 排序(脱敏规则集) + 用户属性集)
- 档位用户命名 `coco_tier_<签名前8位>`;常见集预播种(viewer/member/auditor/admin+内置库只读);新签名动态建档位用户,**上限 32 档**,超限告警提示归并(档位是引擎层粗筛,归并不损安全性——精确执行在应用层)
- **属性化 DLS**:用户属性(部门/密级/自定义 metadata)作为 DLS terms 进签名——属性变更→签名变→自动重投影

#### S3.4 策略编译器(coco → 引擎,编译示例)

```json
PUT _security/role/coco_tier_a1b2c3d4
{
  "cluster": [],
  "indices": [{
    "names": ["coco_document-v2", "coco_chunk", "coco_wiki_article"],
    "privileges": ["read"],
    "dls": "{\"bool\":{\"should\":[{\"terms\":{\"source.id\":[\"ds_wiki_kb1\",\"ds_hr\"]}},{\"terms\":{\"_system.owner_id\":[\"u_123\"]}}]}}",
    "fls":  ["~payload", "~document_chunk", "~ai_insights.embedding"],
    "field_mask": ["metadata.phone::/(\\d{3})\\d{4}(\\d{4})/$1****$2", "owner.userid"]
  }]
}
```
- **文档级→DLS**:数据源分享模型(BuildDatasourceFilter 的 same 语义)编译为 source.id terms + owner terms 的 bool 查询;**W16 内置知识库数据源与 W3 块索引(source 反规范化)同过滤覆盖**——统一索引的安全红利,一条 terms 管文档/文章/块
- **字段级→FLS**:全局排除(field_access 现有集)∪ 角色字段限制,`~` 前缀 exclude 语义(键名按 S3.0 探针结论)
- **脱敏两处编译**:同一规则编译到引擎 field_mask 与应用层动态脱敏(既有);BLAKE2b 确定性+同正则 ⇒ 同输入同输出断言可测

#### S3.5 请求路径

- **读路径**(检索/聚合 W15/文档读/MCP):dry_run=false 且探针通过 → 请求带 `security_run_as: coco_tier_x`;**应用层过滤器照旧执行**——双层取严:以两者交集为结果,差集告警(一次性去重)
- **写路径**(管道/merge/bulk/管理):特权身份,**永不 run-as**(权限是读侧概念)
- **MCP(W7)**:端点作用域先算 → 推档位 → run-as——端点令牌∩用户权限∩引擎投影三层收窄
- **失败语义**:run-as 被拒(档位用户缺/投影漂移)→ 自动重投影一次+重试;再失败 → 回落特权身份+应用层过滤+Warning 头注明"引擎层未生效"——**投影故障绝不阻塞检索**

#### S3.6 API 与设置页

- `GET /search/security-engine`:探针结果/enabled/dry_run/档位数/漂移计数/期望 vs 已部署 diff;`POST /search/security-engine/sync`(全量投影);`POST /search/security-engine/rotate`(轮换档位用户密码)
- 设置页"引擎安全" tab(与"引擎 AI" tab 同款骨架):总开关+dry_run 档/档位表(签名/权限摘要/覆盖用户数)/脱敏规则编辑器(正则即时试掩)/同步与漂移对照面板/审计聚合视图(coco orm 审计+引擎安全审计合并)
- 权限键 `search/security-engine`(admin);投影/轮换全审计

#### S3.7 漂移检测

- 期望 hash=编译器输出体 hash;已部署=FetchRoles 实际体 hash;**周期对照(30min)+变更触发(角色/分享/字段限制/脱敏规则变更 debounce 10s 重投影)**
- 漂移处理:自动重投影(≤3 次)→ 仍漂移 → 设置页告警+索引体检(P4)红标;检索照常(应用层兜底在位)
- **外部篡改**(引擎侧手改/误删 coco_* 角色)同样表现为漂移——重投影覆盖重建;档位用户密码轮换即斩断旧凭据
- 引擎凭据与档位密码:S1 加密、永不回显、不落日志(grep 断言)

#### S3.8 启用流程(灰度,四步)

1. 引擎侧启用安全(运维动作,coco 外;本机演示栈 security off 即此步未做)
2. coco 设置页开 S3 → 探针 → 预播种档位 → **dry_run**:只投影+对照,读路径不 run-as,观察 24h(零风险)
3. 切执行(run-as 生效)——观察期靠 Warning 头/审计比对双层结果差集,差集应趋零
4. 稳定后应用层脱敏降级为幂等校验(**保留不拆**——引擎层失效时的兜底)

#### S3.9 测试与验收

- 单测:档位签名归并确定性;编译器权限集→role JSON golden;脱敏两处编译同输入同输出(BLAKE2b+正则各);漂移 hash
- 集成(scenario DSL,security-on 栈):档位用户检索不可见数据源→total 断言+引擎审计证据;字段限制双层;探针关闭→回退零功能损失
- live 走查:设置页 sync/drift/rotate 全回路;非管理员视角实操
- 红线:档位密码零回显零日志;写路径永不 run-as;投影故障不阻塞检索(回退路径断言)

## 6. 优先级与落地批次

### 6.0 优先级总则(回答"是不是先做好做的")

**对一半,三条总则**:
1. **缺陷与安全无条件最先(P0)**——与难度无关:W0 是正确性 bug(编辑静默过期/入口绕过管道/向量静默过期),不做则后续一切建立在不一致数据上;S1 小而实,顺手垫底
2. **快赢确实先做(P1)**——成立前提三条:不依赖核心、用户可感知、与主线并行不占人力。P1 六项全满足:底座全是已落地资产(D1 指纹、实体卡标准契约、附件管道、D8 terms),缺的只是交互层
3. **核心主线不可因快赢让路(P2)**——W1→W2→W3(+W16a)是价值最高也最难的;"先做好做的"最大风险是核心被无限推迟——P1 全做完,检索粒度仍是整文档、引用仍是文档级。**快赢与主线是并行双轨,不是串行队列**

**第一个可演示里程碑 = P0+P1**:实体卡+entity 检索路+去重折叠展开+预览词搜高亮+加工时间线+编辑器图片嵌入——全部用户可感知,一次走查讲得完的完整故事;且每一项独立可发,不赌大盘。

### 6.1 优先级矩阵

| 项 | 级 | 价值 | 大小 | 依赖 | 备注 |
|---|---|---|---|---|---|
| W0 缺陷修复 | **P0** | 高(正确性) | S | 无 | bug 无条件第一 |
| S1 凭据加密 | **P0** | 高(安全) | S | 无 | 与 W0 并行,零索引变更 |
| W12 去重折叠展示 | **P1** | 中高 | M | 无(D1 已落) | 快赢:phash+折叠交互 |
| W13a 加工时间线 | **P1** | 中 | S | W1 钩子(轻) | 快赢:过程可视化 |
| W14 编辑器图片嵌入 | **P1** | 中高 | M | 无 | 快赢:附件基建现成 |
| W10 前半 实体中枢 | **P1** | 高 | M | 无(契约已标准化) | 快赢:卡片+entity 路+编辑传播 |
| W17 词搜定位模式 | **P1** | 中高 | S | 无 | 快赢:预览高亮先行(精确定位随 W3 升级) |
| W18 存量菜单重组 | **P1** | 中高 | S-M | 无 | 快赢:纯渲染层分组+redirect,六大类先立住;新功能随各自批次归位(IA 即准入门槛) |
| W1 生命周期/重处理 | **P2** | 高 | M | W0 | **主线前置**:状态钩子/模型戳/_reprocess |
| W2 解析 L0/L0.5 | **P2** | 高 | M | 无 | **主线前置**:IR+格式补全(本身零模型零依赖) |
| W3 分块+块级检索+精排 | **P2** | **最高** | L | W1+W2 | **核心中的核心**,不可因快赢推迟 |
| W16a 内置数据源/索引统一 | **P2** | 高 | M-L | 与 W3 合批 | 索引模型决策必须随 W3 先行 |
| W13b 产物可视化 | P3 | 中 | M | W3 | 块/IR 产物数据 |
| W17 精确定位 | P3 | 高 | M | W3 locator | 六格式+降级阶梯 |
| W4 组织/受控标签 | P3 | 中 | M | 无强依赖 | 可与 P2 并行插队 |
| W5 FAQ 库 | P3 | 中 | M | W3(faq_compile) | |
| W11 附件增补+实时联动 | P3 | 高 | M-L | W0.1/W1/W2 | 联动是 W9 前置子集 |
| W15 搜索面板+聚合 | P3 | 中高 | M | W10 前半/W16a | 面板组件依赖卡片 |
| W16b AI 知识分层 | P3 | 高 | S | W16a/W3 权重 | |
| W6 长期记忆 | P4 | 中 | M | 无强依赖 | 独立大件 |
| W7 MCP 端点+S2+L1 | P4 | 中高 | M | 无强依赖 | S2 校验器本体小,可随 S1 提前 |
| W8 连接器护栏 | P4 | 中 | S | 无 | 随时可插队的小件 |
| S3 引擎安全整合 | P4 | 高 | L | W16(DLS 面) | **dry_run 灰度可随批 2 早开**(零风险观察) |
| W9 知识编译器 | P5 | 高 | L | W3+W11 | 远期 |

### 6.2 批次表(双轨并行)

```
批 0(立即):W0 缺陷修复[P0] + S1 凭据加密[P0]                ← 修正确性与安全,无索引变更,先行可发
批 1(立即,双轨):
  快赢轨:W12 去重折叠[P1] + W13a 时间线[P1] + W14 编辑器图片[P1] + W10 前半实体中枢[P1] + W17 词搜定位[P1] + W18 存量菜单重组[P1,六大类先立住,新功能随各批次归位]
  主线轨:W1 生命周期[P2] → W2 解析 L0/L0.5[P2]
批 2(核心):W3 分块+块级检索+组合精排[P2] ‖ W16a 索引统一[P2,与 W3 联调] ‖ S3 dry_run 灰度可开[P4 提前量,零风险]
批 3:P3 依赖核心的后半——W13b 产物可视化 + W17 精确定位 + W4 组织/标签 + W5 FAQ + W11 附件增补/实时联动 + W15 搜索面板/聚合 + W16b AI 分层
批 4:P4 独立大件——W6 记忆 + W7 MCP 端点/S2/L1 + W8 连接器护栏 + S3 切执行
批 5:P5——W9 知识编译器(视排期)
```

每批收口:单测+scenario DSL 进 CI → 本机 live 走查(演示栈 9200/9003)→ WORKPLAN 落地记录;检索侧改动(D4.5/W3/W5)前后各跑一遍评估集留档。**里程碑口径:P0+P1 完成=第一个可演示版本;批 2 完成=检索粒度质变(块级+引用定位);批 3 完成=体验面完整。**

## 7. 风险与缓解

| 风险 | 缓解 |
|---|---|
| 块索引迁移成本/双索引一致性 | 不强制重切;文档级腿并行兜底;_reprocess 按数据源选迁;折叠与权限过滤回归场景锁 CI |
| 评估集回归(块化后排序变化) | 硬门槛:top-4/MRR 不低于基线;块级新用例单独建,不与基线混算 |
| 块索引权限泄露(source 反规范化失准) | 权限过滤器复用 BuildDatasourceFilter 同一实现;非管理员场景在 scenario DSL 固化 |
| 解析改 XHTML 路径引入存量格式回归 | L0 逐格式灰度(数据源级开关);fixture 覆盖 PDF/DOCX/PPTX/XLSX/MD 五类 |
| 加密主钥丢失 | 文档明示:凭据不可读需重录,无托管;启动时校验密钥长度并警告 |
| MCP 端点滥用 | 令牌哈希+限速+工具组默认只读;write 组显式开启 |
| 记忆隐私 | 用户隔离+确认制+负载最小化;评审可见范围与 correction 提议对齐 |
| SSRF 白名单运维负担 | 白名单模式默认关;默认仅拦内网/metadata/危险端口 |
| 附件增补 LLM 对齐误判(错误"更正") | 每条带双方证据链人工裁决;绝不自动改;驳回留抑制记录 |
| 实时联动提议风暴(高频变更源) | trailing-edge debounce+锚定幂等+开放提议只刷计数;可按数据源关联动 |
| 实体卡片缓存失效不及时 | 60s 短缓存+实体更新主动失效;卡片只读导航,陈旧无正确性风险 |
| phash 近似误报(渐变背景/裁边截图) | 两段式验证(缩略图差异像素比复核)+海明阈值保守+近似组须人工确认才折叠 |
| 引擎 collapse 不支持/行为不符预期 | 探针先行(项目探针纪律);回落现行页内折叠,功能不倒退 |
| 运行快照存储膨胀(每文档×每节点×每次运行) | 只存摘要引用+采样快照,重负载 KV 带 TTL;每文档仅保留最近 N 次运行;timeline 端点分页 |
| 附件越权访问/存储滥用 | 附件访问跟资源作用域(私有=登录+读权,公开=短 TTL 签名);上传魔数校验+大小上限;孤儿清理回收空间 |
| 投影不一致(发布成功但检索投影失败) | 投影走既有队列带重试;投影状态标记入索引体检(P4)可见;灰度开关可回退旧 wiki 路 |
| 权限迁移期 wiki 检索越权/过严 | 旧权限键过渡期双查;非管理员场景断言;内置数据源可见性默认=原 wiki 检索权限 |
| 档位归并导致引擎层粒度粗于 coco | 档位只做引擎层粗筛,精确执行保留在应用层;双层取严,粒度损失不构成安全损失 |
| 引擎策略与 coco 策略漂移 | 期望 vs 已部署对照+漂移标记(引擎 AI 面板同款);变更 debounce 重投影;不一致取严并告警 |
| DLS/FLS 探针未过(键名/能力与预期不符) | 适配层三态设计:降级 disabled 零功能损失;探针结论回写 S3.0 事实表;文档级权限可先以索引集合授权+field_mask 过渡,run-as/脱敏独立先行 |
| 扫描件/Office 原件页内高亮受限 | 降级阶梯(跳页+OCR 侧栏/文本预览模式定位)+状态提示诚实;landing_rate 可观测 |
| 大文档全量高亮卡顿 | 可视区优先(RAF 分帧)+按需展开;terms 截前三个 |

## 8. 明确不做(防 cargo-cult)

对象存储多后端与配额(附件 KV+源 URL 够用)/ 向量库抽象与不可变绑定(引擎原生是护城河)/ docreader Python 边车与多引擎注册表(L1 抽象已覆盖)/ Redis+Asynq(进程内队列配单二进制)/ Neo4j(图谱在 ES)/ WeKnora 式 LLM 自动合并知识(合并决策必须人工)/ IM 渠道、沙箱、真浏览器控制、会话 fork-rewind、Langfuse OTLP、桌面 Lite(产品级决策,另议)/ BPE 内嵌表(零依赖红线,v2 再议)。

## 9. 总验收口径

1. CI 8/8 + scenario DSL(每批新增场景,含权限/降级/幂等)
2. 评估集基线不回归;块级/FAQ/改写增益用例可量化展示趋势
3. 全链降级诚实:任何新腿/后端失败,Warning 头注明,主链路不断
4. 红线审计:零自动删、AI 产物全部 draft/提议态、PR 不含 license 文件、go.mod 零变更
5. live 走查记录进 WORKPLAN(演示栈全链,含非管理员视角)
6. 实体面:五展示位卡片渲染、entity 检索路评估用例(实体名/别名命中)、实体编辑传播提议幂等
7. 引用面:点击穿透在 fixture 上逐条 locator 断言;答案生成侧未知引用别名被流式解码删除
8. 联动面:附件增补全链(拖拽→amendment 提议→人工采纳→块刷新);数据源变更→治理提议入队时延断言;失证据只标 stale 零自动删
9. 去重面:phash 稳定性与两段式防误报单测;dismiss 成对永不折叠;折叠行展开组员+证据完整(tier/相似度/来源/时间);图片组缩略图网格;web+widget 双端交互走查
10. 加工面:时间线全节点断言(状态/耗时/尝试/降级说明);产物点击锚定原文位置逐条正确;dry-run 与生产同构渲染;快照存储不进 Document 本体(体积断言)
11. 编辑面:三入口上传插入 E2E;markdown 源仅含 attachment:// 规范引用;孤儿检测与移除引用警告;附件作用域权限(未授权 403/签名过期失效)
12. 搜索面:知识面板组件全走标准契约;聚合筛选往返一致;非管理员聚合不越权
13. 内置源:发布即时可检(text 腿+wiki 路双确认);投影与治理记录计数一致、存量迁移零丢失;AI 上下文分层顺序断言;rerank 来源权重评估用例(同分内容知识库位次靠前)
14. 安全面(S3.9 细则):引擎层 DLS/FLS/脱敏生效有审计证据;双层结果一致或取严;dry_run 灰度全程可用;探针关闭回退应用层零功能损失;脱敏规则两处编译同输入同输出;档位视角评估用例(越权即回归);档位密码/引擎凭据零回显零日志(grep 断言);写路径永不 run-as
15. 预览面:六格式定位高亮 fixture 断言;降级阶梯各级触发正确;widget 同步;landing_rate 分档出数;存量文档回落词搜仍可定位
16. 导航面:六大类渲染+无权组隐藏;旧 URL redirect 全量回归;settings 二级分组 14 tab 归位;新功能落位与导航一致
