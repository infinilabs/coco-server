# Coco AI 知识中枢(feat/knowledge-hub / PR #705)工作计划

> **定稿设计见 [DESIGN-knowledge-hub-v2.md](DESIGN-knowledge-hub-v2.md)**(2026-10-08,第五/六/七篇全部调研的合稿:工作流 W0-W9、批次、验收口径;本文件继续作为调研过程与落地记录日志)
> **批 0/W0 落地(2026-10-08,未提交待评审)**:编辑闭环(PUT /document 预读旧档按指纹比对,内容实变才重入 indexing_documents,元数据编辑零成本)+ 入口闭环(POST /document 与 /datasource/:id/_doc 两变体,带内容即入队)+ 纯文本直通切块(document_text_attachment_extraction 对非 file 文档按 content 重切,改内容绝不留旧块)+ 共享入队助手 modules/common/indexing_queue.go;单测:document handler 级(独立 sqlite+hook 替换)+ 处理器直测;gofmt/构建绿。
> **studio chunk 路面板+参数落地(2026-10-09,已提交 2b1bca0a)**:search_studio 补 chunk_weight 参数(RRF 配置映射)+chunk 路面板(直接调 chunkIndexRoute);**空结果如实报"no chunk rows for this query's scope — leg off or no projection yet"**(面板在灰度关/无投影时不静默消失);routes 数组泛化让前端零改动自动多出该面板(D2 红利第四次兑现)。W3 检索链的最后一面补齐:生产 hybrid_rrf(第七路)+ studio(调参面板)+ D9 评估集(护航)三面齐备
> **W3 方案 c 检索接线落地(2026-10-09,已提交 a619c39a)**:块索引成为 RRF **第七路**——chunkIndexRoute 与文档级 text 腿**并行**(两路同喂融合,chunk 命中带 mom/breadcrumb/quote 引用链);灰度开关 `search_settings.chunk_route`(默认关,_reprocess 投影足量文档后再开);`chunk_weight` 参数同 `<route>_weight` 约定;canonical route list 自动纳入——**studio 面板与 breakdown 零改动多出 chunk 路**(D2 泛化红利的第三次兑现);chunk-only 文档经该路独立浮出;关闭时零行为变化。测试三组:canonical names/融合数学(chunk#1+text#1=2/61,chunk-only=1/62)。**W3 至此完全闭环**:写入侧(结构切块+投影)→检索侧(方案 a 精排+方案 c 独立路)→组合精排→引用链;块的完整生命周期(生成/检索/精排/引用)端到端就绪
> **W3 方案 c 块独立索引落地(2026-10-09,已提交 6b51766d)**:[chunk_index.go](modules/document/chunk_index.go)——①**模型**:KnowledgeChunk{doc_id/source/seq/chunk_type(text|table|faq|mom)/breadcrumb(BM25)/text(BM25)/quote≤300(引用展示,keyword 不索引)/locators(页区间 v1)/MomID/Available(mom=false 不进检索)/embedding(与 Document 嵌套同款 1024 LSH)/model_id/content_hash};②**投影**:ProjectDocumentChunks——清旧行→连续同面包屑块分组→每组一 mom 行(节全文=上下文,不可检索)+子行(可检索,MomID 链);确定性行 ID `<doc>_<kind>_<seq>`;表格检测(竖线前缀);管道完成钩(process_documents 成功且有块即投影,ProjectDocumentChunksFn 可换 seam);③**检索面**:chunkIndexRoute——BM25 打 available 子行(breadcrumb^4/text^2),按 doc_id 分组取最优块→伪文档命中(mom_id/quote/breadcrumb/locator 进 metadata 供引用);空结果回 nil 信令调用方回落文档级。测试四组:行 ID 确定性/表格检测/locator/命中包装(quote 截断/mom 链/model 戳)。**W3 至此方案 a+c 双形态齐备**:方案 a(客户端最优块精排)已交付,方案 c(独立索引)模型+写入+检索面就绪,接线 hybrid_rrf text 腿切换是下一步(数据先积累)
> **W9 知识编译器 v1 落地(2026-10-09,已提交 b0c3b5db)**:[compiler.go](modules/wiki/compiler.go)——POST /wiki/kb/:id/_compile(异步跑,提议入治理队列)。①**MAP**:文档内容按 2k rune 批(行边界切)过默认 LLM 抽实体/概念+主张(≤8/批,JSON 数组);**指纹缓存**(内容+模型身份 → 内存 map)——未变批零 LLM;②**REDUCE**(纯代码零 LLM):normKey=kind+全空白剥离归一名;同键主张按文本哈希去重、别名跨源并集、**证据数降序**(编译容量给语料真正在谈的);③**PLAN**:对既有实体页 reconcile——精确名或别名命中→UPDATE(带 existing_id),否则 CREATE;embedding 候选匹配待页量 justify;④**REFINE v1=证据草稿**(主张+来源链+摘录)即提议负载——LLM 写作 pass 待编译频率 justify(管道形状已备);⑤锚定 ID compile:<kb>:<normKey>+代数,幂等重跑;提议上限 50/次(截断如实 warn)。测试五组:解析(围栏/非法 kind 丢弃)、REDUCE(键合并/主张去重/别名并集/序无关)、normKey(空白折叠/kind 分离)、PLAN(alias 命中→update/无命中→create/精确命中)、批切(预算/无损重组)。**fixture 教训三次入册**:REDUCE 的"payment gateway"与"支付网关"不同语言不同 normKey(fixture 错非实现错);PLAN 只对 entity 页 reconcile(concept 页被过滤,fixture 用了 concept 型);批切末尾换行差异用前缀断言。**设计全项至此 18/18 主体落地**(W18 前端菜单因并行占用为唯一例外,已有设计)
> **entity card 接入框架标准扩展点(2026-10-09 修正版,未提交待评审)**:medcl 指路后重写——框架自有 `core/entity_card` 包**已实现** /entity/card/:type/:id 与 /entity/label/_batch_get 路由 + EntityProvider 扩展口(GenEntityInfo/GenEntityLabel),enterprise 插件的 UserEntityProvider(user/team 型)即此模式;我前一手写的两条同名路由是**重复造轮子,已删**。现 wiki 实体以标准 provider 注册([entity_card.go](modules/wiki/entity_card.go)):type=wiki_entity,卡片=EntityInfo{title=名/subtitle=类型/tags=别名/url=文章页/details.table=自由属性行/properties=状态行};**缺档降级 id-当-title 空卡**(getLabelCardInfo 的 mustGetProviders 对未注册型 panic——只注册 wiki_entity,别的型不归 coco 管);GenEntityLabel 批量 {id,title,icon},未知 id 降级无空洞。测试四组(标准卡全字段/缺档壳/label 补洞/接口绊线)——**第五次撞 AppConfig-kv panic**(companyMapConfigFn 指针构造 swap)。**教训入册:动前端消费的 API 前先查 framework/core 与 plugins/enterprise 有没有现成扩展点——标准口永远优于手写路由**(user/team 型卡片由 enterprise provider 服务,coco 不碰)
> **entity card 契约初版(已废弃,教训保留)**:曾按 @infinilabs/entity-ui d.ts 形状手写两条路由——形状对齐思路正确(组件 EntityCardData 与框架 EntityInfo 字段一一对应),但落点错了;组件包读取法仍记档:形状源=web/node_modules/.pnpm/@infinilabs+entity-ui@*/dist/index.d.ts
> **W17 预览定位高亮(terms-only 模式)落地(2026-10-09,未提交待评审)**:①**预览页**([highlight.ts](web/src/pages/preview/document/highlight.ts)):URL `?q=` 携带检索词→内容渲染落定后 400ms,DOM TreeWalker 遍历文本节点(跳过 script/style/mark),**最长词优先**找最早命中,splitText+`<mark>` 包裹(黄色半透明/继承色),上限 400 mark;首命中 scrollIntoView 平滑滚到视口中心;contentRef 挂内容容器,依赖 [loading/error/q/contentBlobUrl/data] 重跑。②**widget 打开链**:ListItem 外链 onClick 经 withHighlightQuery——**仅 /#/preview/ 内链**追加 ?q=(读页面 URL 的 query 参数——搜索页 enableQueryParams 镜像了活查询,免 DOM 探测;第一版写 input[data-search-input] 探针但输入框无该标记,改为 URL 源更稳);外链原样。③精确锚定位(chunk/page 跳转)随 semantic_chunk 负载就绪,数据侧已通(quote/neighbors/pages),本批只做词搜模式(W17 的回落级,永不依赖块索引)。门禁:主工程 tsc 598≤598 + widget vite build 绿(15.3MB)
> **W17 预览定位高亮(terms-only 模式)落地(2026-10-09,未提交待评审)**:①**预览页**([highlight.ts](web/src/pages/preview/document/highlight.ts)):URL `?q=` 携带检索词→内容渲染落定后 400ms,DOM TreeWalker 遍历文本节点(跳过 script/style/mark),**最长词优先**找最早命中,splitText+`<mark>` 包裹(黄色半透明/继承色),上限 400 mark;首命中 scrollIntoView 平滑滚到视口中心;contentRef 挂内容容器,依赖 [loading/error/q/contentBlobUrl/data] 重跑。②**widget 打开链**:ListItem 外链 onClick 经 withHighlightQuery——**仅 /#/preview/ 内链**追加 ?q=(读页面 URL 的 query 参数——搜索页 enableQueryParams 镜像了活查询,免 DOM 探测;第一版写 input[data-search-input] 探针但输入框无该标记,改为 URL 源更稳);外链原样。③精确锚定位(chunk/page 跳转)随 semantic_chunk 负载就绪,数据侧已通(quote/neighbors/pages),本批只做词搜模式(W17 的回落级,永不依赖块索引)。门禁:主工程 tsc 598≤598 + widget vite build 绿(15.3MB)
> **W11 附件增补落地(2026-10-09,未提交待评审)**:POST /document/:doc_id/_amend(multipart file,20MB 上限)([amendment.go](modules/document/amendment.go))——①**证据抽取**:Tika(EffectiveTikaEndpoint 配置/默认回落)抽文本,**本地重实现 tika 调用**(import fileproc 会拉连接器栈成环——第二次撞环,教训固化:processors 不进 modules);失败回落原始字节;②**LLM 对齐**:默认语言模型,系统提示词定义三类 diff(correct=证据反驳现行/supplement=新信息/obsolete=现行过期)+规则(≤8 条最强优先/证据逐字引用);输入=当前文档 12k char 截断+附件 12k 截断;③**产物=document_amendment 治理提议**(锚定 ID doc:<id> 代数递增,evidence 带 items 全量+附件摘录+cascade 标记)——**模型绝不写文档**;采纳走既有文档 PUT(编辑闭环自动重切/重嵌/重摘要,下游实体/文章联动跟随);与 D7 分工:D7=口头纠正,amendment=文件证据;抽取失败 422/模型不可用 503/两文档一致返回"no differences"(空 diff 是合法裁决)。测试:解析(裸数组/围栏+闲聊/非法动作丢弃/claim 空丢弃/12 条截 8/无数组报错)+空数组合法。**前端拖拽区后置**(预览页加工面板旁,数据链已通)。W11 至此全件完成(实时联动前批+附件增补本批)
> **W15 侧栏融合计数接线(2026-10-09,未提交待评审)**:搜索页 onAggregation 升级——原有聚合(经 /query/_search 的 aggs body)**只描述单召回腿**(W15 调研结论),现并发调 /query/_aggregations(同一 query/search_type/fuzziness)把三个主面(source.id/type/tags)的桶**替换为融合真值**({value,count}→{key,doc_count} 映射进 widget 期望的 ES 形状);content_category(图片/文档标签页)保留自己的聚合不动;融合调用失败静默回落腿内计数(增强不阻塞)。widget 分面面板零改动(声明的六字段照旧),筛选交互(filter=field:any(...) 链路)不变——**用户看到的面还是那个面,数字从"单腿视角"变成"融合视角"**。tsc 598≤598 维持。W15 至此端点+消费闭环;右侧知识面板(entity 卡/迷你图谱)仍后置(依赖 /entity/card 契约)
> **W7 MCP 端点令牌落地(2026-10-09,未提交待评审)**:[modules/mcpep](modules/mcpep/endpoint.go)——①**模型**:MCPEndpoint{name/enabled/token_hash(SHA-256)/datasource_ids(空=全部)/tool_groups(空=全部,write 显式)/rate_limit_per_minute(默认 60)/last_used_at};②**鉴权走框架既有钩子** api.RegisterMCPToolAuthorizer(零框架改动):请求带 `Authorization: Bearer mcp_...` → 必须解析到 enabled 端点且过限速;**无 mcp_ 前缀 Bearer 或无 Authorization → 走原路径不动**(控制台会话鉴权零影响);**端点查询 fail-closed**(store 故障 recover→拒,pipeline 永不 panic);③**明文仅示一次**:create/rotate 返回 plaintext+提示语,库中只有哈希;rotate 即杀旧 token+清限速窗;list 响应剥 token_hash;④**限速**:进程内滑动窗(sync.Map→窗口切片),默认 60/min;last_used_at 写入 30s 去抖(异步);⑤CRUD:/mcp_endpoint/ 五端点(list/create/rotate/toggle/delete,权限 mcp/endpoint.*)。**v1 偏差记录**(vs 设计的 /mcp/:endpoint_id 子路径):框架单流式服务器挂 BasePath,子路径需框架改造——v1 用 Bearer 主体区分端点(URL 不变),作用域字段已持久化、逐工具收窄随后续版本;外部 agent 用法:`Authorization: Bearer mcp_xxx` 打 /mcp。测试五组:token 往返/无碰撞、bearer 解析(大小写/外族方案透传)、限速窗(第 4 拒)、默认限速、authorize(无 Bearer 走原路径/不可解析 token 拒)。**教训**:seelog 无 Fatal——随机源不可用返回空 token 由 handler 503(不 issue 可预测 token);ORM 未注册的测试环境 lookup panic→fail-closed recover
> **W15 聚合端点落地(2026-10-09,未提交待评审)**:GET /query/_aggregations([facets.go](modules/document/facets.go))——**复用 queryWithRRF 同一融合管线**(请求克隆改 from=0)对融合头 200 命中做桶聚合:数据源(名称回退 ID,**内置知识库源天然在列**)/类型(空回落 document,wiki_article/entity 各自成桶)/标签(多值)/实体类型(entity 命中的 metadata.entity_type)/月度直方图(提升字段 Updated,空跳过);桶按计数降序同数按值序,每面截 15;响应 {total(引擎真总数), window(聚合实际样本数,诚实暴露截断)}。**设计要点**:聚合在融合结果上客户端侧做——各路服务端聚合描述的是单路不是融合面,权限一致性由复用管线免费获得;query 与 q 双参数兼容。测试四组:排序截断/数据源+类型桶(空类型回落)/标签多值/月度(无时间戳跳过)。**教训**:fixture 自相矛盾("doc" vs "" 期望同桶)——断言失败先查 fixture 再查实现;提升字段 Updated 经嵌入结构体直达
> **记忆管理前端落地(2026-10-09,未提交待评审)**:settings 新 tab"我的记忆"([MemorySettings.tsx](web/src/pages/settings/modules/MemorySettings.tsx))——状态 Segmented 筛选(待确认/已生效/已拒绝/全部);列表五类彩色 Tag/内容/状态 Tag(pending 橙/confirmed 绿/rejected 红)/更新时间;**pending 行内确认/拒绝**(Modal 二次确认,文案如实说明后果:确认=进入助手上下文;拒绝=永不注入留审计);删除 Popconfirm;权限挂 coco#memory/update(read 不足时只读列表)。W6 至此全链闭环:蒸馏产 pending→这里人工裁决→confirmed 才进 Recall/MCP。i18n 中英 24 键;并行会话对 settings/index.tsx 的改动经查仅 6 行(同型 tab 追加),追加零冲突;tsc 597≤598 维持
> **已完成功能的管理前端落地(2026-10-09,未提交待评审)**:①**FAQ 管理页**(FaqManagement.tsx):数据源详情新 tab("FAQ 条目",与文件管理同 view 闸)——列表(标准问/答案/相似问 Tag 组/负例红 Tag 组)+新建 Modal(四字段,相似/负例按行拆)+删除(Popconfirm)+"**试搜**"卡(实时打 /query/_faq:exact 绿标直答/排序命中列表/无命中含负例过滤提示);重复创建如实提示"问题集已存在,跳过"(幂等可见)。②**折叠展开**(widget NormalList):命中行下薄副行"N 个重复副本(完全相同)——展开/收起",展开列成员(标题+来源),**只看不动**——删除/排除动作仍在查重报告(W12 红线:搜索面无破坏性动作);widget i18n 中英。③避让确认:data-source detail/search 页/widget NormalList 当时均空闲;settings/assistant/路由生成文件被并行占用不动。门禁:主工程 tsc 597≤598,**widget vite build 全绿**(15.3MB gzip 4.1MB);抓一个真错:lucide-react 无 RedoAlt 导出(版本差异,tsc 查不出 esm 导出名,只有 rollup 抓到)——换 RotateCcw。**仍无 UI 的已完成后端**:W15 面板、W17 高亮、/entity/card 五展示位
> **W6 长期记忆(确认制)落地(2026-10-09,未提交待评审)**:新模块 [modules/memory](modules/memory/memory.go)(独立包避开并行会话占用的 assistant/api)——①**模型**:MemoryRecord{kind 五类/status pending→confirmed|rejected/user_id/content/supersede_of/context(评审证据,enabled:false 不进提示词)};**确认闸门**:一切记录生而 pending,confirm/reject 仅属主(管理员也不可跨户读——记忆是个人上下文非共享知识);②**端点**:/memory/_mine(过滤 kind/status)、POST /(手动建,pending)、_confirm/_reject(pending 态一次性裁决,已裁决幂等返回 note)、DELETE;③**蒸馏** POST /memory/_distill:默认语言模型+独立系统提示词(五类定义/独立成句/用户语言/≤5 条/瞬态跳过),输出剥围栏找 JSON 数组,**每条落 pending**;**限频每用户 5 分钟**(sync.Map 时间戳);④**召回契约**:Recall(ctx,user,query,limit)——**仅 confirmed**:profile/preference 常驻必出,fact/task 关键词重叠召回(向量召回待记忆量上来再值——v1 注释明说);⑤**MCP 工具 search_memory**(第 46 个):挂 /memory/_search,描述言明"只有 confirmed 在此,pending 等属主评审"。测试:蒸馏输出解析(裸数组/围栏+闲聊/8 条截 5/无数组报错)/五类校验/关键词重叠(中英/空查询零召回)。**设计取舍记录**:注入点(助手系统提示词组装)暂未接——assistant 模块被并行会话占用,Recall 即契约,接线一笔即可
> **W16b AI 知识分层(组装侧)落地(2026-10-09,未提交待评审)**:FormatDocumentForReplyReferences(助手 RAG 引用→LLM 提示词的唯一咽喉)升级——①**分层分区**:isCuratedReference(Source.ID=wiki/wiki_kb_* 或 type=wiki_article/entity)先出、原始文档后出,层内序稳定;②**逐档 Layer 标注**(curated/source)+有 curated 时提示词首部声明"CURATED references are reviewed knowledge — authoritative tier when they conflict with raw sources"(无 curated 不注,零噪声);③精排层来源权重(W3 组合分 0.1×sourceWeight)已在前笔落地——三处落地已有其二。**后置**:检索层权重档位需 assistant 检索路先 RRF 化(InitialDocumentBriefSearch 现走纯 keyword,QueryDocuments 不带 RRF——切到 queryWithRRF+权重档位是"默认搜索模式接线"级改动,独立排期)。测试三组:分区稳定序/层标注顺序+权威声明/无 curated 零声明。**教训**:MaskContent 触 AppConfig→kv,测试环境 panic——包级 maskContentFn 可换(与 appConfigFn 同族模式,tools 包首个实例)
> **S3 引擎安全 dry_run 阶段落地(2026-10-09,未提交待评审)**:[modules/common/engineguard](modules/common/engineguard/engineguard.go)——①**适配层**:直连引擎 _security REST(引擎配置 Endpoint+BasicAuth——framework 字段是 BasicAuth 结构与 ucfg.SecretString,第一版写错两次后核对 domain.go);Probe 双探针(_security/role/admin 可读=security_api;**带 security_run_as 头的匿名检索期 401/403=run_as 受理**——200 说明头被无视);ApplyTierRole/ApplyTierUser(PUT)/FetchTierRole(404→nil 语义,漂移读回)。②**档位签名纯函数**:五面(排序后数据源集+owner+排序 FLS 集+掩码键+排序属性集)MD5;同面共档、序无关;TierUserName=coco_tier_前 8 位。③**策略编译器**:role JSON——DLS=source.id terms∨owner term∨_system.attrs terms 的 bool.should 串;FLS=`~` 前缀排除;field_mask 哈希形(裸字段)与正则形(字段::模式::替换)。④**端点**:GET /search/security-engine(探针+档位普查+enforcement: app-layer 标注)/POST /search/security-engine/sync(**只投影不切 run-as**——DESIGN §S3.8 第 2 步,切执行是独立灰度);权限挂 search/ops。测试七组:签名确定性/序无关/异面异档/role 形状(DLS 双 should+FLS+mask)/属性档/假引擎探针(run-as 401 判真)/应用-读回往返+404 语义。**假引擎教训**:switch 路由分支"长路径优先"想当然,通用 role 匹配截胡了 /_search——改前缀判断;真实路由表同样该防
> **S2 SSRF 收口本体落地(2026-10-09,未提交待评审)**:[modules/common/netguard](modules/common/netguard/netguard.go)——出站 URL 双模式守卫:①**deny-list(默认)**:非 http(s) 方案/环回/私网/链路本地/未指定/IPv6 ULA(fc|fd 前缀)/云 metadata(AWS/GCP/阿里)/危险端口(22/23/3306/5432/6379/9200/11211)全拒;**主机名走解析器路径**(localhost 这类名字解析后逐地址检查——名字不能绕);解析失败不是 SSRF 判定(让 fetch 自己报错)。②**whitelist(SSRF_WHITELIST=on)**:只放行 SSRF_WHITELIST_EXTRA 列出的主机,**名字比对在 DNS 之前**(WeKnora 语义);空名单拒一切——fail-closed 刻意。③首挂出站点:L1 远程解析后端(我方新增的 operator 可配出站);**遗留接线清单**:scraper 工具/MCP client fetch/连接器 HTTP(每点需 per-redirect 重验的 client 包装,非本包职责——包头已注明)。测试 15+ 情形(各私网段/metadata 名与 IP/端口/方案/白名单大小写/空名单/解析路径)。**护栏反噬自身测试**:httptest 绑 127.0.0.1 被正确拒绝——测试显式白名单环回(allowLoopbackForTest),这正是护栏在工作的证明
> **预览页加工面板落地(2026-10-09,未提交待评审)**:文档预览页右下角浮钮(嵌入/widget 模式隐藏)→ 560px 抽屉 [ProcessingPanel](web/src/pages/preview/document/components/ProcessingPanel.tsx)——①生命周期头(状态标签 indexing 蓝/completed 绿/failed 红/存量灰+嵌入模型+失败原因 Alert);②**运行记录**(最近 5 次:ok/fail 标签+管道名+耗时+时间+错误,含 passthrough 原因);③**块清单表**(序号/面包屑/页区间 pN-M/excerpt/vectorized ✓—,pageSize10,标题带总数)。i18n 中英 14 键(page.preview.processing.*);直接 request() 调用(_timeline/_chunks)带 headers 透传(app-integration-id 兼容)。**教训**:`ignoreError` 不在 request 的类型上(fetchEntityUser 的宽松类型才收它),错误吞掉改 `.catch(() => null)`——tsc 棘轮从 598 回落 596,不占并行会话余量。W13 至此前后端齐:运营页看全局(接入任务/同步状态),预览页看单档(过程+产物)
> **任务管理前端面落地(2026-10-09,未提交待评审)**:检索运营页新增两卡——①**接入任务**:生命周期四统计(处理中/已完成/失败/存量未盖章,失败计数红色示警)+失败清单表(文档/数据源/失败原因/更新时间,pageSize5)+**批量重试按钮**(Modal 确认→retryFailedDocs→结果 toast"已重试 N 篇,M 篇未能重试"→自动刷新);②**同步状态**:每数据源一行(名称/类型/启用/间隔/上次调度/增量游标/**最近运行状态标签**——running 蓝/superseded 橙"未记终态"/stale 红"疑似僵死"+Tooltip/从未运行灰)+进度文本(N 批/M 篇)。i18n 中英 28 键;**tsc 棘轮 596≤598 维持**(i18n 键与页面同批,API client 前笔已入)。注意:page.ts 两文件同时载有并行会话的键改动,后续提交该文件会扫入——分组提交时留意
> **数据接入专项:块清单端点(2026-10-09,未提交待评审)**:GET /document/:doc_id/_chunks?from&size——文档的解析产物(块)逐条可检视:index/面包屑/页区间/excerpt≤500 rune/**vectorized 位**(有无数向量);分页+maxSearchPageSize 上限;404 处理。W13b 产物可视化的读面就位(文档详情页"加工"面板的数据源:_timeline 看过程、_chunks 看产物)。测试:双块(长文本截断/向量位真假)/缺档 404
> **数据接入专项:同步运行日志+崩溃证据落地(2026-10-09,未提交待评审)**:[modules/common/syncrun.go](modules/common/syncrun.go)(**先写错位置后搬家**:connectors/common→数据源模块→文档模块成环,日志类型必须住中立层 modules/common——两侧都已依赖)。**Singleton 管道推出的完成语义**:同源管道 Singleton=true 不并发,新运行启动时残留 running 记录必是"死时未记终态"(进程崩/管道杀)——MarkSyncRunStarted 把它标 **superseded** 并保留 batches/documents 计数作**死亡现场证据**(dispatcher 起跑处 warn 一条"previous run ended unrecorded after N batches/M documents");BatchCollect 逐文档 RecordSyncBatch 进度(所有连接器共享咽喉,零连接器改动);状态端点只读加 **stale 标记**(running 且 30min 无更新,证据原样保留到下次 supersede,GET 不改写)。/_sync_status 的 run 子对象 {run_id/state/started/updated/batches/documents/stale} 就位。测试:stale 三情形(新鲜/陈旧/superseded 永不 stale)。**教训**:import 环要在选型时先画依赖图——共享基建住"最低公共层"(modules/common),别住调用方之一的包里
> **数据接入/解析专项:L1 可插拔解析后端落地(2026-10-09,未提交待评审)**:[parse_backend.go](plugins/processors/text_attachment_extraction/parse_backend.go)——远程视觉解析服务(mineru/docling 类)可选配,**FirstParser 首胜语义**:配置扩展名先走远程,非空答案即胜;任何失败(网络/超时/非 200/空响应)落回内置 Tika 路——可选加速器绝不做摄取单点。①**协议从简**:POST 文件字节+filename 参数,回 `{"pages":[...]}` 或纯文本(单页);响应上限 64MB;空页剔除、全空视为失败。②**管道级配置**(remote_parse_url/timeout 默认 300s/exts)——数据源各有管道即天然 per-datasource 差异化,零新数据源字段;SSRF 校验随 S2 落地时一并挂。③接线在魔数纠偏后、扩展名 switch 外包一层(if won → 直取 pages)。测试七组:JSON 多页/纯文本回落/502 报错/空响应判负(首胜竞速语义)/扩展名门控三情形/失败回退/大小写扩展名。**教训**:参数名 url 遮蔽 net/url 包(第二次撞命名遮蔽,path 之后)
> **数据接入/解析/任务管理专项落地(2026-10-09,medcl 定向优先,未提交待评审)**:**任务管理三端点+退避护栏**——①**处理总览** GET /search/ops/processing(?datasource=):status 四桶分布(indexing/completed/failed/**legacy**)——legacy=总数−已盖章桶和(**空 status 因 omitempty 根本不落字段,聚合永远看不到,算术余数是后端无关正解**);失败清单 top50(标题/数据源/错误原因/直达 _reprocess 链接);②**批量重试** POST /document/_retry_failed({datasource?}):failed→清产物→标 indexing→重入队(与 _reprocess 同语义,failed 永免指纹闸门),响应 retried/still_failed/truncated 如实;③**同步状态** GET /datasource/_sync_status:每数据源 enabled/sync 配置/**dispatcher 上次 tick**(/datasource/lastAccessTime kv)+**连接器增量游标**(/datasource/increment/lastModifiedTime kv)原样透出——接入侧第一次有"每个源上次拉取何时"的面;④**W8 退避护栏**(connectors/common/backoff.go):IsRetryableStatus(429/5xx)、BackoffDelay 2/4/8s 封顶、RetryWithBackoff(attempt 3,context 取消优先,耗尽如实报"giving up after N attempts")。测试:总览四桶+legacy 算术+失败清单、批量重试(独立数据源隔离——第四次撞共享库)+错误清除、退避五组(调度表/可重试码/成功/耗尽/取消不调 fn)。**教训**:DirectWrite 豁免不含读权限,读写都要直通的 handler 必须 DirectReadAccess()+Set(DirectWrite)双挂
> **批 3/W4 组织与受控标签落地(2026-10-09,未提交待评审)**:①**虚拟目录**(modules/document/folders.go):Document 增 folder_path(keyword);NormalizeFolderPath 规范化(首斜杠补/尾去/深度 ≤16/段 ≤128/**空段与 ./.. 全拒**——测试先写反再修正);树=terms 聚合端点 GET /datasource/:id/folders(根常驻);POST /document/_move_folder 移动=字段更新**绝不重解析**(批 100);②**tags 检索参数** ?tags=(逗号分隔):keyword/semantic 单型腿(search.go builder 创建点)与 hybrid 各腿(rrfRoute builder)统一 applyTagsFilter——UI 分面与召回过滤同源;③**受控标签**:DataSource 增 tags_vocab;extract_tags 受控模式——词表存在时 prompt 改"只从词表选"(MUST choose ONLY/no new tags 双强调),输出 filterTagsByVocab 子集过滤(大小写不敏感,**未知词丢弃计数,永不入词表**;nil 词表自短路=自由模式——测试抓出助手函数不自洽后修);词表按 doc.Source.ID 每文档解析加载(读失败回落自由模式);手工标签永不触碰(处理器只写 doc.Tags 一次,无删除路径);④**能力位** GET /datasource/:id/capabilities:documents 计数/vector(exists 向量场)/faq(type 计数)/wiki(KB DatasourceIDs 绑定)四信号,助手编辑器与 MCP schema 消费。**v1 后置**:文件夹改名(前缀 rewrite 循环)、GeneratedProfile(需 LLM 调用,归 W13b 一并)、graph 能力位(实体-数据源关联需先落 W10 证据链块级化)。测试:路径规范化 10 情形/词表过滤(保留/丢弃/nil)/受控 prompt 双强调
> **批 3/W5 FAQ 库落地(2026-10-09,未提交待评审)**:FAQ 条目=普通文档 type=faq([faq.go](modules/document/faq.go)):①**编译纪律**——索引内容=标准问+相似问(compileFAQContent),**负例问题与答案绝不入索引**(答案是被返回的不是被检索的),结构化负载存 metadata.faq{standard/similar/negative/answer};②**问题键归一化**:D1 链(NFKC 全半角/大小号)+ **去全部空白**(FAQ 层加强——中文问句杂空格常见,实测 fingerprint.Normalize 不折叠内部空格);③**创建端点** POST /document/faq(单条或批量+datasource 作用域):content_hash 预盖,同问集合同源**幂等跳过**(写时查重,不堆孪生行);④**检索端点** GET /query/_faq:BM25(title^20/pinyin/combined)限 type=faq+可选数据源过滤→**负例精确命中整条出局**(问的正是条目声明"不是"的)→命中循环中**任意措辞(含相似问)归一化精确命中即 exact=true 附标准答案直接返回**(模型可直接采用);无 exact 则返回排序命中列表。测试四组:编译排除负例与答案/归一化(全半角+大小写+空白)/负例命中/文档负载往返。**v1 后置**:迭代倍增 TopK(需语义向量,向量饱和判定)、CSV/Excel 批量导入(复用 W2 xlsx)、MCP faq_search 工具。**教训**:DecodeHits 只出 _source 不带分数,要分数走 Payload 原始解析(与 wiki_projection 路同法)
> **批 3/W11 实时联动落地(2026-10-09,未提交待评审)**:"信号实时、生效人工"闭环。①**钩子注册表**(modules/common/document_liaison.go):FireDocumentProcessed/FireDocumentDeleted 两事件,处理器同步+best-effort+**逐个 recover 隔离**(联络器公告性,panic 绝不断索引/删除流)——模块解耦(processors/document 不 import wiki,wiki 在自己 init 注册)。②**指纹守卫是关键工程决策**:dispatcher 每轮同步全量推文档不改内容也重跑管道,事件不比对必然提议泛滥;而 process_documents 在 merge **之前**运行,库里还是旧文档——onDocumentProcessed 比对新旧 ContentHash,**同指纹静默**(纯 resync 零提议),异指纹才扫引用文章。③**wiki 联络器**(modules/wiki/liaison.go):内容实变→引用文章(Sources.doc_id 内存扫描,上限 50)逐篇 article_refresh 提议(evidence.cascade=document-change+新指纹);删除→stale 提议(cascade=document-deleted)——**失证据只标记绝不删**(文章原样保留,断言锁定);两路都吃 fileProposal 的 open-twin 折叠。④**接线三处**:process_documents pushEnriched 盖章后触发;deleteDoc/batchDeleteDoc(逐 id);数据源级联删除(DeleteByQuery 前先取 id 清单)。测试三组:实变→1 提议+二次触发折叠仍 1;同指纹 resync 零提议(独立 doc+article 防共享库撞断言——隔离教训第三次);删除→stale+文章本体完好。**W11 余项**:附件增补(amendment 提议流)、事件 debounce(高频源)、completion 分发进 ES durable log
> **批 2/W16a 落地(2026-10-09,内置数据源投影,未提交待评审)**:①**投影模型**(modules/wiki/projection.go):published 文章单向镜像进 coco_document——确定性投影 ID `wikiproj_<articleID>`(幂等 upsert、删除免查)、type=wiki_article、**source.id=wiki_kb_<kbID>+KB 名**(虚拟内置数据源引用,真建 DataSource 记录后置——聚合面已够用)、URL 直达文章页、metadata 带 wiki_sources/page_type/confidence/source_count/article_id;**噪声闸门前移到投影时**(low 置信与低于源数下限的概念页根本不投影,读侧免费继承闸门);状态机闭环:发布→投影、下架/归档→删投影、已发布文章内容编辑→重投影、文章删除→删投影(状态端点+crud PostUpdate/PostDelete 三钩子)。**SyncArticleProjection 带 recover**:投影子系统故障(测试环境无 kv 等)只警告不拖挂文章写——best-effort 契约写进代码。②**回填端点** POST /wiki/_backfill_projection:存量 published 一次性投影,幂等(已有跳过不重写),响应 {total/projected/skipped}。③**灰度开关** search_settings.wiki_projection(默认关):开→wiki 路从打 wiki_article 索引切到文档索引(type=wiki_article,投影行本就是合规文档零包装);关→现状不变。**v1 偏差记录**:内置数据源暂为虚拟 source 引用而非真 DataSource 行(权限/聚合够用,真记录待 W15/W16b 需要分享模型时补);wiki 文章尚不切块嵌入(投影行 content 整段,W16b 进 W3 块索引)。测试三组:发布投影全字段/下架归档删投影/幂等;低置信永不投影;回填幂等(二跑 0 新)。**sqlite 语义教训**:Upsert 缺行报错(改查-建/更);GetV2 缺行返回错误而非 (false,nil)(助手容错)
> **批 2/W3 第三片落地(2026-10-09,D4.5 组合精排+MMR+邻块扩展,未提交待评审)**:①**组合分**:rerankFusedHits 从纯模型分排序改为 **0.6×模型分(min-max 归一)+0.3×RRF 归一分(融合分数窗口内归一,排序前捕获)+0.1×来源权重**——来源权重映射:内置 wiki=1.0、D6 source_priority 按位线性衰减(1.0→0.5)、未列=0.5 基线(推荐不否决);rerankScoreChange 增 composite_score(studio 对照卡可见);生产与 studio 两调用点接 loadCompileRules().SourcePriority。②**MMR λ0.7**:贪心选序 0.7×组合分−0.3×对已选集的最大 Jaccard(标题+摘要 token 集)——近似重复沉底、多样命中上浮,只重排精排窗不动尾部。③**邻块扩展**:semanticChunkVec 携全量块文本,refine 时最优块 <350 rune 触发前后邻块边缘拼接、总预算 ≤850(块本带 overlap 前缀,边缘自然缝合),负载 neighbors{before,after}——W17 预览定位的上下文就绪。测试:来源权重五情形(wiki 首位/列表衰减/未列基线)、同分同模型时优先级来源胜出+composite 如实上报、MMR 近重复沉底多样上浮、邻块扩展(触发/预算/边界只拉单侧)。**回归修正**:TestRerankTiesKeepRRFOrder 的旧 fixture 顺序与分数矛盾(首位低分)——组合分的 RRF 分量如实暴露了它,fixture 改为真实 RRF 形态(首位最高分);教训:新分量是旧 fixture 不真实性的探针
> **批 2/W3 第二片落地(2026-10-09,客户端语义腿块级精排 D14 方案 a,未提交待评审)**:clientSemanticRecall 升级为**两阶段**——阶段一照旧(文档级 ai_insights 向量余弦粗排,无块向量文档仍可参战),阶段二新增:头部 top-50(体量纪律:块向量 10 块×1024 维/文档,精排全窗每次查询要搬 MB 级;50 名开外对 RRF 排名无感)拉 document_chunk 嵌套块向量(**块向量从死重变成真实检索粒度**——补遗缺陷 1 的最小修复),逐文档取**最优块余弦**重打分(cosine+1 同尺度,全表稳定重排,NaN 尾保序),命中挂 `semantic_chunk{index,breadcrumb,pages{start,end},quote≤300}` 引用负载(W10 引用穿透/W17 预览定位的消费数据就位);无块向量命中保持文档级分数不动(降级纪律);块向量拉取失败仅降级为文档级粗排(debug 日志);note 如实注明 "best-chunk cosine on N docs"。fetchDocumentChunkVectorsFn 可注入测试。测试五组:最优块按余弦不按序/精排重排头部/无块向量零侵扰/上限只精排头 50/quote 截断。**W3 余项**:引擎侧块级(coco-chunk-embedding 管道)、块独立索引(方案 c,mom 形态)、text 腿块级
> **批 2/W3 第一片落地(2026-10-09,结构感知切块 D10 写入侧,未提交待评审)**:[structured_chunk.go](plugins/processors/fileproc/structured_chunk.go)——XHTML 结构路(W2)的标题标记在此被消费:①**标题分节**:行首 `#`-`######` 切节,块绝不跨标题边界;**面包屑**逐级累积("年报 > 财务 > 营收"),更深标题替换浅尾;②**硬上限契约**:任何块 ≤chunk_size(rune),优先行边界切(尾 1/4 窗口内有行边界则贴行切),中行硬切兜底;③**有界 overlap**:下一块携带前块尾部 15% 预算(裁到行首,不中行起);④**孤标题不发块**(标题行紧跟标题行=无可检索内容,记录开节标题行 flush 时判同跳过);⑤无标记输入回落单根节(仍按预算+overlap 开窗,优于旧腰斩);⑥页区间随节跟踪;⑦对抗配置(overlap≥cap)强制推进不死循环。DocumentChunk 增 **breadcrumb**(text,向后兼容);extractTextAndAttachment 与 chunkContentOnly(W0)双路切换,SplitPagesToChunks 保留兼容。测试八组:面包屑逐级/不跨节/硬上限+overlap 前缀/无损重组(**行对齐切不携带 overlap,中行硬切才重复 1 rune——重组规则按"首 rune 等于前块尾 rune 才剥"**)/页区间/无结构回落/孤标题/对抗配置。W0 老测试按新契约更新(25 rune@10 → 3 块 10/10/7)。**教训**:调试靠打印实证不靠脑内演算(7 块之谜是断言行号漂移);测试断言要随契约更新并写明新契约。**W3 余项(下一片)**:块独立索引 coco_chunk+检索腿切换+组合精排(D4.5)+溯源位
> **批 1/W2 收尾+W10 编辑传播落地(2026-10-09,未提交待评审)**:①**XHTML 结构路**:htmlToText 的结构标记逻辑抽成 selectionToStructuredText(共享),appendPage(Tika 页组装)从裸 .Text() 切到结构化渲染——**h1-h6 以 markdown 标题级、表格行整行存活进块文本**(此前全丢,正是"大表格打成位置汤"的病根);这是 W3 面包屑最便宜的信号源,EPUB/MHTML/HTML 路同享。②**PDF outline 按 L1 后置**:xref stream(PDF 1.5+)无新依赖解析不可靠,行使设计"做不了则随 L1 后端做"条款。③**W10 实体编辑传播**(entity_propagation.go):新增提议类型 article_refresh;两段式挂在实体 crud——PrepareUpdate(recordEntityEditForPropagation:存旧档对比 name/aliases/type/status/relations 五面,**仅实变记录**,store 未覆盖前唯一窗口)→ PostUpdate(保存成功才消费,propagateEntityEditAfterSave)——避免保存前发提议的伪报;受影响文章=实体自己的文章+LinkedPages 反链(客户端扫,字段未索引)+上限 20(枢纽实体不淹队列);复用 fileProposal 的 open-twin 折叠(同一文章被两个实体编辑触发只留一条 open 提议);evidence 带 cascade=entity-edit+新旧名+变更面;governanceTypeLabel 增"refresh after entity edit"。测试:两文章各得一提议+二次编辑折叠仍 2 open+琐碎编辑零提议(断言按 entity_id 限定——**测试隔离教训:共享 sqlite 库里全局"无 X"断言会被前一个测试的数据炸**)。gofmt/构建绿
> **批 1/W12 后端半落地(2026-10-08,未提交待评审)**:①**phash 指纹**(modules/common/fingerprint/phash.go,零新依赖,x/image 既有):box 均值重采样 32×32 灰度→2-D DCT→8×8 低频块(DC 除外)中值阈值 64 位;海明 ≤8 为近重候选带(两段式验证纪律同 simhash);x/image/webp+tiff 注册补编解码。②**盖章两处**:Attachment 增 image_phash(long 有符号,simhash 同理),UploadToBlobStore 唯一创建咽喉按 mime=image/* 就地盖章(字节只在此处在手);图片文档在 extractTextAndAttachment 图片分支盖 metadata.image_phash。③**折叠富负载**:fingerprint_duplicates 从 {count,ids} 扩为 +tier("exact")/similarity(100)/members[{id,title,source,size,updated}]——members 全取自页内命中,零额外查询。④**dismiss 成对永不折叠**:foldDuplicateHits 收 dismissed 集(60s TTL 缓存 dismissedPairsCached,加载失败回 nil=按 hash 折叠的旧行为);第三份同 hash 副本仍折入代表位(拒绝按对不按 hash)。⑤**collapse 探针**:engine-capability 报告增 server_side_collapse(60s TTL,SearchWithRawQueryDSL 原始探测 content_hash collapse)——只如实报告能力,折叠保持客户端(切换是刻意决策)。测试:phash(重采样稳定性——**逐位相等不是 phash 契约**,量化翻边界位,纯缩放在 ≤4;反向梯度 >20;字节/图像一致;非图零值)、折叠富负载、dismiss 按对不折+第三份仍折。**W12 余项**:前端手风琴+缩略图网格(web)、近似组确认动作(dedup_group_id)、存量回填
> **批 1/W10 前半落地(2026-10-08,entity 检索路,未提交待评审)**:RRF 第六路 `entity`(entity_route.go,wiki 路模板照搬):BM25 打 WikiEntity(name^20/name.pinyin^8/aliases^6/subtype^2/combined_fulltext),仅 reviewed/published(与 graph 路同闸门,proposed=草稿不进检索);权限复用 wiki 文章检索权(实体是 wiki 资产);命中包装伪文档(Source=wiki/Type=entity,有 ArticleID 直达文章页、无则 card-only 不带 URL,别名进 summary,证据链 entity_sources 带 locator——W10 证据链块级化的第一步);`entity_weight` 参数与 hybrid_rrf 默认权重接线;检索实验室自动多出 entity 路面板(routes 数组泛化的红利)+studio 参数 EntityWeight。与 graph 路分工:graph=关系遍历(种子→邻居),entity=实体本身词面/别名直接命中("查支付网关"不依赖关系存在)。单测:包装(带/不带文章页/证据链)、融合数学(entity 权重 2 的 2/(k+1) 贡献)、静音路零贡献仍出文档。**教训**:core.Config.ServerInfo 是指针,零值 Config 直接赋 Endpoint 必 panic(测试 swap 照 graph 测试的指针构造)。**W10 余项**:/entity/card 契约的 Go 侧实现(形状待与 @infinilabs/entity-ui 组件对齐)、编辑传播提议(article_refresh)、实体证据链块级化(依赖 W3)
> **批 1/W2-L0.5 落地(2026-10-08,格式补全,未提交待评审)**:①**共享 htmlToText**(html.go,goquery 既有依赖):h1-h6→markdown 标题级(W3 面包屑的最便宜信号)、表格一行 `单元格 | 单元格`、script/style 丢弃、无结构标记回落全文。②**EPUB 原生**(epub.go):container.xml→OPF→spine 阅读序,一章节一页,DC 标题作首页首行;css/图片不入页。③**XMind 原生**(xmind.go,content.json 直接递归类型):**Tika 都不认的真空白**;每画布一页缩进大纲,notes 引用块,无名画布回落 Sheet N;legacy content.xml 明确报错。④**MHTML 原生**(mhtml.go,mime/multipart stdlib):最大非广告 text/html part 胜出(googleads/doubleclick/adnxs 过滤),base64/quoted-printable 按 part 头解码。⑤file_type_detection 补 epub/xmind/mhtml/mht/html/htm/xhtml 映射+归 doc 类(此前这些扩展 content_category 为空,类目筛选不可见)。降级纪律:epub/mhtml 失败回落 Tika,xmind 无回落(原生即后端)。测试:epub(spine 序/标题/拒绝非 zip)、xmind(大纲/notes/无名画布/legacy 拒绝)、mhtml(广告 part 跳过/QP 解码/非 multipart 拒绝)、htmlToText(标题级/表格/inline 专名不注空格——期望写错修的是测试不是函数)、类型映射 12 扩展。**教训**:本工具链 quotedprintable 在 mime/ 下(encoding/ 下无);参数名 path 会遮蔽 path 包
> **批 1/W2 落地(2026-10-08,L0 前四件,未提交待评审)**:①魔数纠偏落在 text_attachment_extraction(file_type_detection 只有文件名没有字节)——effectiveExt 路由前嗅探头 8 字节:OLE 魔数的 .pptx/.pptm→".ppt-ole" 走 Tika(**顺手修了 OLE .ppt 直送原生 zip 解析器的既有缺陷**,原 switch 把 .ppt 也送 processPptx)、ZIP 魔数的 .xls/.ppt→.xlsx/.pptx 享原生路;mismatch 记 metadata.magic_ext_corrected(时间线可见);Tika 路 content-sniff 自纠无需纠偏。②**Excel 原生结构化路**(xlsx.go,zip/XML 全 stdlib,pptx.go 先例):workbook+rels 取表序表名(r:id 命名空间双字段兜底)、sharedStrings、每表一页 `# 表名`+每行 `列名: 值`、合并单元格左上主值填充、首行表头开关(默认 true,表头行不再重复出为数据行)、=DISPIMG/_xlfn 图片公式剔除、行上限 1 万;失败降级 Tika。③保守页眉页脚剔除(pages.go):仅每页首/末非空行、≤80 rune、数字归一为 # 后 ≥60% 页命中才删;<5 页不删(证据不足);页码页脚("Page 3"≡"Page 4")可删。④乱码页检测:U+FFFD 占比 >0.5% 标记 metadata.garbled_pages+warn(只标记不重路由,强制 OCR 属 L1)。测试:魔数 7 情形、xlsx fixture 测试内 zip 自建(表头/合并填充/无表头字母列名/非 zip 拒绝)、页眉页脚 3 组(可删/证据不足不删/内容行不误删)、乱码 3 组。**W2 余项**:Tika XHTML 结构路(h1-h6/表格进 IR)、PDF outline
> **批 1/W1 落地(2026-10-08,未提交待评审)**:生命周期三字段(status: indexing|completed|failed / error_message 截 500 / embedding_model 模型戳);process_documents 终态盖章+运行记录(metadata.pipeline_runs 上限 5 新者在前,passthrough 保留 reason 且**不洗白 failed 态**)+入库去重闸门(同 content_hash+source.id 异 id → 丢弃+刷幸存者 updated,INGEST_DEDUP_GATE 可关,默认开);端点:POST /document/:id/_reprocess(幂等闸门:completed+指纹未变跳过,force 豁免;清块/摘要/insights→标 indexing→重入队)、POST /document/datasource/:id/_reprocess(批量,同闸门,上限 2000,truncated 如实上报)、GET /document/:id/_timeline(status/error/模型/运行史);创建与编辑入队路径标 indexing(在途可见);embedding 处理器成功即盖 embedding_model=provider/model;engine-capability 增 by_embedding_model terms 聚合(orm.Aggregate 正道,别用 QueryBuilder 造聚合)。单测:处理器 5 组(成败/错误截断/passthrough 不洗白/运行史上限最新在前/闸门开关)+handler 6 组(跳过/force 清旧产物入队/failed 免 force/404/timeline/创建标态)。教训:提升字段(ID)不能进结构体字面量;Go 函数参数不自动取址
> **批 0/S1 落地(2026-10-08,未提交待评审)**:secretbox 包(AES-256-GCM,enc:v1: 前缀,明文兼容读,幂等写,错钥→空串绝不把密文当凭据发出;敏感键匹配连字符归一化)+ orm 写钩子(priority 200,ModelProvider.APIKey/MCPServer.Config/DataSource.Connector.Config 三模型,含 StdioConfig.Env 结构体形态)+ 四读咽喉解密(GetModelProvider 覆盖 langchain/embedding/rerank/engine-AI、GetMCPServersByID、GetDatasourceConfig、dispatcher.syncDatasource);主钥环境变量 COCO_SECRET_KEY(hex64 或口令 SHA-256 派生),未设=明文兼容模式+一次性警告。**v1 后置项**:存量明文回填任务(读侧兼容不急)、provider 编辑 UI 对密文字段的展示优化(GET /model_provider/:id 从回显明文变为回显 enc:v1: 密文——严格更安全,UX 待优化)。教训两枚:sync.Once 会吞掉 ResetMasterKey(改互斥锁+双检);interface{} 装结构体值时类型开关进不了 map 分支(StdioConfig 显式分支,map 引用语义就地下章)
> 更新时间:2026-10-07
> 状态基线(2026-09-29):P0 引擎探针+语义自愈/P1 embedding 统一(8e2a1045)、P1.5 管道策展(63f08caa)、D1 查重引擎、D2 多路 RRF 含 wiki 路(cad6447a)、D3 图谱路(ed3b3912)均已落地,CI 8/8;R1 与三大营销能力、本体四部曲(W1-W4)此前已合入
> 红线不变:PR 内不引入 license 相关文件(go.mod/go.sum、config/generated.go、.public/static.go、widget dist、二进制、.claw/);AI 只产草稿/提议,发布必须人工(D1)

## 一、现状速览

### 已完成

| 模块 | 内容 | 提交 |
|---|---|---|
| 本体 O1-O4 | 词汇层 schema / 实体抽取接 schema / 图查询+逆关系+实体卡 / wikilink 胶囊+反链+实体页+实体管理页 | 6100925b / 15e761f9 / 24386444 / 70c88e4e+e0cdc76c+192c3e55 |
| 实体 schema 归属 | 实体写入按 KB 级 schema 校验(kb_id 透传) | a982487a |
| 双引擎检索 | BM25+kNN 双路召回、RRF 融合参数可调、检索实验室现场试跑 | f38e9d0a |
| 知识加工 | 8+2 节管道模板(解析/切分/富化/实体抽取/向量化/双写入库) | 9cbbfcd9 |
| 权限与脱敏 | 索引/文档/字段三级收敛 + 动态脱敏(召回后交模型前) | 5c757cd7 |
| 代码整理 R1 | hybrid_rrf 拆分、字段收敛收口 field_access.go、settings 表驱动+首存兜底 | 87f191d4 |
| i18n | search-studio 菜单+页面全量中英 | 1b828733 / c74f22c0 |

### Wiki 模型已有资产(本次调研核对的代码事实)

`core/wiki.go` 的 WikiArticle 已具备 LLM Wiki 骨架:page_type(entity/concept/source 三分类)、Sources 溯源(doc_id/excerpt/locator 到 page|section|clause)、Confidence(high/medium/low)、Status 人工状态机(draft→reviewed→published→archived)、LinkedPages 交叉引用、WikiVersion 不可变快照。治理扫描(governance.go)已产出 stale/duplicate/conflict/low-quality/orphan 五类提议,全部走人工处理。

## 二、行业调研:LLM Wiki 正反合(2026-09-29)

来源:JitKnow《万字长文!LLM Wiki 技术深度拆解与落地实践》+ Knowly《别再迷信 LLM WIKI:大模型根本撑不起知识库的持续进化》。

### 正方(JitKnow):从"每次重查"到"一次编译"
- 三层架构:原始资料层(不可变只追加)/ 知识层(人读的 Markdown wiki 页,YAML frontmatter 带 sources/entities/concepts/pending)/ Schema 规则层(AGENTS.md 定义怎么编译、怎么链接、怎么判冲突)
- 六阶段闭环:摄入→编译→链接→冲突→检索→迭代;冲突**只标记不覆盖**(双方都留+时间戳+存疑页+人工工单)
- 三路混合检索:BM25 / 图谱关系遍历(沿预建交叉引用)/ 向量语义 → 加权合并 → 专用 reranker(Qwen3-Reranker-4B);宣称召回 +15-20%、语义命中率 +30pp、时延持平
- 与 RAG 分层协同:核心(Wiki 编译层直接读)+ 边缘(RAG 兜底)+ 融合生成带引用;宣称准确率 +20%、运营成本 -40%
- 成本逻辑:算力从查询时挪到入库时(编译一次),token 省 84.6-95%;适用前提是查询/摄入比高
- 产品面:知识图谱一键生成、RAG 分析看板(检索策略分布/命中块/低召回告警/响应时长)、多租户 2.0(组织/部门+审批流)、助手 API 化、MCP 服务

### 反方(Knowly):LLM 无状态,撑不起持续进化
- 核心难题是**知识查重**:同内容改文件名、复制到不同目录、旧/新/final 版混放、Word→PDF 同内容——LLM 判不了,靠 title/path/mtime 启发式"半吊子"(用户投诉不看内容)
- 解法是纯传统算法四层:①精确哈希(完全相同)②内容特征指纹(simhash 类,同内容不同名/格式)③文本指纹相似度(小改动)④文件名特征(疑似新旧版本);全部本地跑,1454 文件 10 分钟出 81 组,结果持久化复用
- 产品行为:分组带四档标签+相似度%+证据(路径/大小/时间);**系统只推荐、绝不擅自删**;动作:移回收站 / 保留但排除出知识库 / 标记非重复(不再问) / 暂缓
- 哲学:"先让知识库里的内容可信,再谈让它持续进化"——工程(查重/版本/冲突治理)先于 LLM 策展

### 合题(Coco 的位置)
两篇一正一反,Coco 的既有架构恰是合题:**确定性底座**(Easysearch 存状态:版本/提议/指纹;管道阶段确定性执行)+ **LLM 策展**(编译/链接/冲突标记,governance 的 LLM judge)+ **人工裁决**(D1 红线,与 Knowly"系统推荐不擅自删"、JitKnow"机器发现人拍板"完全同构)。JitKnow 的"知识编译"主张我们有引擎侧天然优势(Easysearch 即底层);Knowly 的"先可信"恰好指出我们当前最薄的一环(见 D1)。

### 第三篇(J.B.《如何打造一个每周都在变聪明的公司大脑》):学习回路与导航(2026-09-29)

- 问题:公司里最聪明的 AI 属于个人(聊透的对话/调教好的配置),每人一个"公司版本"——公司处处在学习,无处记忆
- 公司大脑四层:记忆(事实/决策/历史)、判断(规则/标准)、能力(可复用技术与 worker)、学习(**经审查的纠正**)
- 公司地图:每个 agent 先读一张小地图——做什么/当前优先级/**来源优先级**(冲突时按优先级裁决)/导航/规则(用最新批准信息、说缺什么、**绝不把猜测变成公司知识**);"更多上下文=更多困惑",解法是更好的导航而非更多记忆
- 纠正回路(全篇核心):每次纠正问"为什么旧信息还在?",修的是层不是点;纠正路由表——缺失/过期事实→公司知识、新战略选择→决策记录、反复偏好→政策、验证过的技术→技能、可重复流程→worker、危险动作→机械护栏;循环:工作→纠正→路由→**审查**→共享→未来工作更好
- HQ(总部)防碎片化:统一根据地,变更经审查进 HQ 即时全公司共享——否则大脑碎回 N 个个人大脑(Claude Code/Cursor/Codex 各一套)

对照:Coco 的定位就是文章的"HQ"——中心化知识中枢+审查发布+版本快照,红线"AI 只产草稿、发布必须人工"与文章"绝不把猜测变成公司知识/审查后才共享"完全同构;from_chat 已是"个人对话→公司资产"的通道;治理提议队列天然就是"经审查的纠正"的审查机构。**真缺口有二**(见三-4/5):纠正信号只有后台体检没有工作侧入口;没有地图页与来源优先级。落为 D7 新工作项 + D6 扩展。

### 第四篇(一安《混合检索+Rerank实战:把RAG答对率从78%干到94%》):检索工程三件套(2026-10-04)

- 战役:公司规则问答 bot,600 篇文档,答对率 78%→94%(+16);错误归因先行——12 错中 7 个词面不匹配(文档说"年终双薪"用户问"十三薪")、3 个排序下沉、2 个文档根本没有:**先归因再动手,不是哪热修哪**
- 三件套:①混合检索(BM25+向量,RRF k=60 融合取 20)②rerank(交叉编码器精排 20→4,~200ms,失败降级原序)③查询改写(便宜模型把人话→文档话,原 query+改写 query 各查一遍一起进 RRF——"稳妥起见",改写只增路不替路)
- 工程纪律:每一环独立超时+降级,任何单路挂了主链路照常;改写结果缓存(高频口语不重复打模型);参数起点 topK=20 / threshold 0.3 / rerank 留 4
- **评估账(全篇最重的一课)**:50 道真题评估集,每次调参/换模型/改分块回归跑一遍——"评估集比任何参数都值钱";改写+评估集两步再 +6 分;没有评估集,调参是盲调
- 文档补录清单:最后 3 错中 2 个是"文档缺失",答案是不硬编、给管理员列补录清单——知识库没有的别让 AI 编
- 遗留难题:大表格被切块切碎,按章节重切才救回——切块策略是深水区(对应我们 W7 的处理侧后置项)

对照:三件套 coco 已有其二且更宽——多路 RRF 是**四路**(keyword/semantic/wiki/graph,文章只有两路;k=60 同款约定),rerank 已落地(D4:融合后 top-50 重打分,超时降级,纪律同构);文章"文档补录清单"与 D5 缺口闭环同构,且我们更进一步是**自动的**(≥3 零命中自动生成提议,不止人工清单)。**真缺口有二**:①**查询改写**——检索入口只有原 query,词面不匹配全靠语义路兜底(正是文章 78% 时代的老问题);实体别名只在图谱路 substring 生效,救不了"十三薪→年终双薪"这种文档侧词汇;②**评估集**——检索实验室是单发手试,没有金标题集回归打分,RRF 权重/rerank 窗口调整目前无客观回归手段。落为候选 D8/D9,等 review 排期(R1 整理已完成,87f191d4)。

### 第五篇(WeKnora 开源对照):机制级拆解与可借清单(2026-10-07)

来源:Tencent/WeKnora(MIT,~32k star,v0.8.2)。本节不是 README 泛读——官方文档站(weknora.weixin.qq.com)机制级拆解,全部落到数据模型/管道阶段/API 形态。定位与 coco 高度同构:Go 后端、单二进制(Lite 模式 SQLite+内存队列)、知识库/RAG/Agent/Wiki 一体;但产品哲学相反——WeKnora 功能最大化(27 家模型商、8 种向量库、10 个 IM 渠道、Docker/E2B 沙箱、真浏览器控制),coco 引擎原生(Easysearch 既是状态库又是检索/向量化引擎)+人工闸门。结论:**抄机制、不抄架构**。

**值得借的十个机制(按建议优先序;S1 工作量最小、风险最实,可与 D10 并行先行)**:

1. **分块三级自适应+父子块 small-to-big**——W7 后置"切块策略深水区"的直接答案(第四篇遗留难题:大表格切碎)。coco 现状 SplitPagesToChunks 定长切块(fileproc/chunk.go)。WeKnora 方案:①文档画像(标题总数/密度/主导层级)→ heading→启发式→递归 逐级试,每级过验证器(空产出/单块超 2×chunk_size/非末块 <50 字/最大块 <1/4 或 >2× 均拒绝降级);②**ContextHeader 标题面包屑**持久化到 chunk,embedding 输入=面包屑+正文——改一块只重嵌一块;③**父子块**:子块 ~384 字带重叠(固定 1/5)入索引,命中**返回父块 ~4096 内容**——嵌入粒度与上下文粒度解耦;④预览端点与生产管道共享同一配置归一化函数(coco Pipeline Studio dry-run 现成骨架);⑤配置逐级覆盖(单次上传>KB>默认)。**与 D2 溯源联动:面包屑天然就是 section 级 locator**
2. **凭据加密 enc:v1:**——安全缺口实锤:coco 的 ModelProvider.APIKey 明文 keyword 落 ES(core/llm_provider.go:31),仅 API 出参脱敏;连接器凭据同理。WeKnora 机制:环境变量主钥 AES-256-GCM,密文带 enc:v1: 前缀,**读侧兼容历史明文**(渐进迁移零停机),凭据走只写端点、永不回显
3. **MCP 端点令牌模型**——coco /mcp 单端点走会话鉴权,"把某个 KB 发布给 Cursor/Claude"做不到也不敢开。WeKnora:每空间多命名端点,Bearer token 哈希落库+创建/轮换时明文仅示一次,KB 作用域白名单,工具组四类(检索读/QA/wiki/写默认关),每分钟限速,last_used_at 审计。coco 的 api.MCPTool 权限过滤已是同构底座,缺的只是端点实体与 Bearer 鉴权
4. **FAQ 知识库类型**——coco 只有文档库,QA 对(制度问答/客服话术)无处安放。WeKnora:FAQ 一条一 chunk(标准问+相似问拼入索引,负例问题绝不入索引),查询期混合召回→**负例问题精确命中整条出局**→唯一条目不足且向量饱和则迭代倍增 TopK(≤5 轮);chat 侧 top 分≥直答阈值标记 exact、模型直接采用标准答案;内容归一化链(去 URL/繁转简/全半角)做 ContentHash 幂等——与 D1 指纹哲学同源
5. **长期记忆五类+确认制**——J.B. 篇"记忆层"的工程化,D7 的读侧兄弟(D7=用户告诉你错在哪,memory=系统记住你是谁)。profile/preference/fact/task/interest;auto 模式后台蒸馏会话(延迟 90s、每人限频、重启不丢消息的游标机制),产出一律 **pending、不确认不注入提示词**;替换型推断确认前旧条目保持生效;fact/task 查询期向量召回注入,profile/preference 常驻限长;search_memory 工具给 agent。与 D1 红线完全同构——落地形态即治理提议队列新增 memory 类提议,确认=人工放行
6. **rerank 组合分与邻块扩展**(D4 增强)——WeKnora 精排=0.6×模型分+0.3×基础分+0.1×来源权重;MMR λ=0.7 多样性去冗(0.7×相关−0.3×最大 Jaccard);阈值回退梯(全灭但 top1≥0.15 保 top1;空结果阈值 ×0.7 重试至 0.3);**邻块扩展**:命中块 <350 字沿前后块链扩到 ~850 字;**上轮引用注入**(与上轮引文 Jaccard≥0.15 才注入、×0.6、≤3 条)——多轮对话不丢上下文。D4 现为纯模型分排序,组合分与 MMR 零模型依赖
7. **引用协议 fail-closed**(D2 溯源链的生成侧消费)——回答前注册 cN(chunk)/wN(web) 别名,系统提示词要求自闭合 `<ref id="cN"/>` 且禁止造新别名,**流式解码器把未知别名直接删除**——结构性防幻觉引用。coco 引用元数据(wiki_sources/locator)已有,缺生成侧协议
8. **本地多查询扩展(零 LLM)**(D8 伴侣)——停用词/语序重排/关键短语/检索式分词造变体,**仅当首轮召回 <TopK 才触发**,变体并发查(信号量 16)。D8 改写救"词面鸿沟"(要模型),本地扩展救"召回不足"(零成本)——触发条件设计值得照抄,不打无谓之仗
9. **连接器纪律**——①增量游标按资源粒度(obj_edit_time/last_edited_time/页面 version/commit-diff);②**流式断点续传**:每 50 节点/30s 存 checkpoint,429 退避 2/4/8s×3;③**删除护栏**:本轮将丢失 ≥20 页且 ≥80% 存量即中止——防凭证过期把全库"同步"没了。coco dispatcher 轮询已就绪,这三件是护栏级加固
10. **SSRF 白名单**——出站 URL 白名单模式(启用时 DNS 解析前拒绝),连接器 HTTP 客户端**每次重定向/拨号重验**。coco scraper/连接器/MCP fetch 目前全裸奔,对外开放前应收口

另录:WeKnora 自动打标**只从既有标签池选、永不新建、永不删手工标签**(skip_if_tagged 默认真)——有界生成,extract_tags 可借鉴;其任务死信表(task_dead_letters,可重试/取消/清空+审计)+启动重入队(task_pending_ops)的**模式**,记作 Mock backlog 死信队列 UI 的参考实现。

**coco 已领先、明确不抄的面**(防 cargo-cult):评估集(持久化+人工标注+CI 金样例+缺口回流 vs WeKnora 固定样例 parquet、结果内存态重启即失、无逐题报告);查重引擎(WeKnora 无对应物,D1/D1.5 独有);检索融合(五路 RRF+权重可调+引擎原生 processor vs 两路 RRF+外置 pgvector/Milvus——Easysearch 原生是护城河不是债);wiki 信任模型(人工闸门+治理提议 vs agent 自动生成改写);连接器广度(25 vs 9)。另:Neo4j 不抄(图谱在 ES,重依赖违背单二进制);多向量库抽象不抄;Python docreader 不抄(Tika+原生 pptx 够用);Redis/Asynq 不抄(进程内队列配单二进制);IM 渠道/沙箱/真浏览器控制/fork-rewind/Langfuse OTLP/桌面 Lite——产品级决策,先记不排。

**落为候选(等 review 排期)**:D10 分块升级(最高)、S1 凭据加密、P5 MCP 端点令牌、D11 FAQ 库、D12 长期记忆(详见第四章新条目);并入既有项:rerank 组合分+MMR+邻块扩展(→D4.5)、引用协议(→D2 消费侧)、本地多查询扩展(→D8 触发式)、连接器断点+删除护栏(→W6)。

### 第五篇补遗:知识库域深对照(2026-10-07,medcl 指定细抠)

细抠范围:WeKnora v0.8.2 源码级(internal/types/knowledgebase.go/chunk.go、knowledge_process*.go、块编辑路由、docparser 引擎注册表、kb_activity 审计)对 coco 代码级(core/document.go、fileproc/chunk.go、config/setup/pipeline.tpl、modules/document/service.go)。切块算法侧第六篇(RAGFlow)施工图已覆盖,本补遗不重复;此处补**检索侧、生命周期侧、组织侧**。

**coco 现状代码事实(本轮新发现,1-3 是缺陷级)**:

1. **检索粒度=整文档**:语义路只查 `ai_insights.embedding.embedding1024`(每文档一个向量,modules/document/service.go:27-29);而 document_embedding 给每块写的 embedding1024 **从未被任何查询消费——块向量是死重**(plugins/processors/embedding/processor.go:198-235 批量写并校验 1024 维);BM25 默认字段仅 title.keyword^100/title^10/title.pinyin^4/combined_fulltext(service.go:46),**块文本映射了 text 却不在任何检索字段里**,响应侧还被 field_access 整体排除(field_access.go:18)
2. **PUT /document 编辑不触发重切块/重嵌入**(document.go:323-355 仅 orm.Save+指纹重盖)——内容改了,块与向量静默过期;**POST /document 与 /datasource/:id/_doc 直建文档完全绕过富化管道**(仅 orm.Create;引擎默认管道只嵌 ai_insights.text,缺字段 ignore_missing 直过)——无切块/摘要/标签/实体/向量
3. **无重处理端点与模型版本戳**:改切块参数/换嵌入模型后存量文档无 _reprocess,新旧向量静默混查(engine-capability 已报 with_vectors,可扩混模检测);processed 仅布尔,无在途富化状态;Pipeline Studio dry-run 是配置侧,无每文档处理时间线
4. 切块定长 7000 字符(pipeline.tpl:53)、**零重叠**、ChunkRange{起止页}无消费方——wiki locator 的 page|section|clause 在文档侧实际无着落点(D2 溯源的文档侧断点)
5. 无文档目录树;extract_tags 自由生成(仅从 ai_insights 出发,3-8 自由词),无受控词表;tags 只做 UI 聚合面,检索参数无 tag 作用域
6. 原文件不落库(连接器来源 URL 指回源)+附件/OCR 进 KV bucket——与 WeKnora 对象存储+预览防穿越各有取舍,**不抄**

**WeKnora 知识库域可借机制(与第六篇互补,逐条对缺口)**:

1. **块模型 12 型**(text/parent_text/image_ocr/image_caption/summary/entity/relationship/faq/web_search/table_summary/table_column/wiki_page):SourceContent 不可变+ContentRevision;**SourceLocators 按文件形态**(pdf=页+bbox/docx=块/slide=页/sheet=行区间/section/ASR=时间轴)每个定位附 ≤300 字 quote 供 UI 高亮;**块编辑带修订**:乐观锁 expected_revision(冲突 409)/回滚/图片子块禁用不删(回滚即复活)/父块按偏移倒序覆盖重建/单块重嵌(含生成问题向量行)
2. **文档状态机** pending→processing→**finalizing**→completed:finalizing=富化子任务在途(pending_subtasks_count 原子减,归零转 completed;富化失败不挂文档;取消即止 LLM 花费);stall 区分 queued(积压)/stalled(死);每文档 GET stages/spans 逐阶段时间线带尝试号;**上传去重**(FileHash+文件名 scoped 租户+KB,重复刷 created_at 返回既有实体);reparse=清理旧向量/块/图/在途→重入队→**重戳嵌入模型**;删除按序传播(向量→wiki 先于块→图命名空间→文件→配额)
3. **富化编排**:单一 post_process 处理器,原子翻 finalizing 后再发子任务(摘要×1/生成问题按 20 块批/图谱逐块/wiki 去抖);auto-tag best-effort 不计数(永不阻塞完成);**生成式问题双用**——每块生成的问题既作独立向量行(SourceID=块ID-问题ID)又拼进 rerank passage;rerank passage 组装=title+ContextHeader+正文+问题/图注/OCR
4. **图像多模态门控**:attr 观察(一次合并 observe+describe VLM 调用)→纯代码 DecideOCR 决定是否二轮 OCR——装饰图省一次调用;图注/OCR 作子块挂 ParentChunkID 即刻入索引
5. **KB 实体=配置单元**:IndexingStrategy 四开关(vector/keyword/wiki/graph,至少开一);每 KB 绑 embedding/summary/VLM/ASR 模型;配置三级覆盖(上传>KB>默认)且**覆盖持久化进文档 metadata,reparse 可复现**;capabilities 计算字段(前端/agent 编辑器按能力过滤 KB);多 KB 联查要求同嵌入模型身份(Name+BaseURL)
6. **组织与权限**:文档虚拟目录 folder_path(重命名/移动改路径**不重解析**);标签实体+m2m,auto-tag 只从 top-500 池选、永不新建、永不删手工标签,标签删除异步清向量;检索参数 tag_ids/scope_tag_ids 作用域;下载=Contributor+,预览防穿越(危险类型强制附件+nosniff)
7. **检索配置三级**:租户 RetrievalConfig(权重/k/阈值/topk)→agent 覆盖→调用级 SearchParams(capped 200);跨库近重去重硬编码 token 重叠 ≥0.85+尺寸比≤3+包含关系——coco hash 折叠管精确重复,此件管块级近似重复,D14 后可产品化为旋钮

**不抄(补遗级)**:对象存储多后端+配额(附件 KV+源 URL 够用)、向量库绑定不可变(引擎原生)、docreader Python 边车+多引擎注册表(D13 的 L1"可插拔后端"已覆盖)、文档级版本快照(块级修订已够)。

**落为候选(补遗新增,详第四章)**:D14 块级检索与溯源位(与 D10 写入侧配对的检索侧)、D15 文档生命周期与重处理闭环(修缺陷 1-3)、D16 知识组织与受控标签。

### 第五篇补遗二:文档解析专页细抠(03-document-parsing,2026-10-07,medcl 指定页)

官方解析文档全量抽取,与源码级调研互证。总分工:docreader(Python gRPC 侧车)只管"文件→Markdown+图片引用",分块/图片存储/OCR/VLM/向量化全在 Go 侧——与 coco"解析后置处理在管道"同构;**抄纪律与启发式,不抄 Python 边车**。

**逐项可借点(按对 coco 的价值排序)**:

1. **PDF 逐页分诊路由**:每页独立判 text/scanned——图片面积覆盖率 ≥0.5 或文本层字符 <10 且有图 → 扫描页(渲染 JPEG 交 OCR),文本页走文本层。**零模型纯启发式**;任何失败回退"全页渲染"兜底,pdf_force_scanned 可 per-KB/per-upload 强制——降级纪律与检索路同构。渲染纪律:DPI 200/JPEG 质量 85/长边钳 2000px
2. **文本页几何重建(全启发式)**:XY-cut 递归切列(多栏按列线性化)、边栏/竖排水印列剔除(arXiv 侧栏)、按字间距推断空格、**按行高相对中位数把大字号行升级 Markdown 标题**(直接喂 D10 heading 切分)、页码行/占位符/矢量图坐标轴碎屑清理、**保守页眉页脚剔除**(仅每页首尾行+须短+出现在 ≥60% 文本页才删);**过滤隐藏文本(render-mode 3)与页外字形——防提示注入**(Tika 路径同样要防,安全细节);Figure N caption 上方矢量图区域渲染成图(矢量图表变可 OCR 图)
3. **Excel 结构化路**:"大表格切碎"的结构性答案(与第六篇表格单元 IR 同向):逐 sheet **每行转 `列名: 值` 键值块**(sheet+行区间 locator);首行表头开关(xlsx_first_row_as_header);**合并单元格先填充**(左上主值复制到覆盖区——openpyxl 只存左上);魔数检测真实格式(xlsx/xlsb/ods/WPS .et/改名 csv→LibreOffice 归一化)+sharedStrings 修复;剔除 =DISPIMG/_xlfn.IMAGE 图片函数串
4. **格式补全(Go 原生零依赖,coco 全缺)**:EPUB(zip+XHTML→markdown,TOC 序+DC 元数据+图抽)、XMind(content.json→缩进大纲)、MHTML(MIME 解析,最大非广告 part+Content-Location 图别名回写)、本地 HTML 文件;音频 ingestion 可走既有 enterprise ASR 插件(时间轴 locator)
5. **引擎选择=配置而非代码**:per-KB parser_engine_rules(文件类型→引擎+类型化覆盖项);**FirstParser 首胜链**(按序试,第一个产出非空即胜)作为多后端选择原语——D13 L1 直接可用(Tika 保底→可选远程视觉后端);**魔数纠偏**(OLE 魔数的".docx"路由去 DOC——WPS/改名文件容忍,file_type_detection 处理器可加)
6. **网页/URL 摄取质量**:Playwright+trafilatura 正文抽取;**SSRF 双重防护**(导航前校验+每子请求/重定向 route 拦截,内网/环回/云 metadata/直连 IP/危险端口全拦)——S2 的实现参考;SPA 等待(networkidle 10s+可见文本 ≥80 字符 15s)、微信公众号适配(无扩展名图 mmbiz/#js_content)——中文场景实用
7. **多模态路由细节**:图片按 image_source_type 分流(扫描页→专用 OCR prompt;插图→VLM 描述);自定义 VLM 指令只拼进描述 prompt、**绝不拼进 OCR prompt**(防指令污染);表格 HTML→Markdown 转换(合并单元格转不了的保 HTML 按行切块)——与第六篇"表头随片"一致
8. **并发与超时纪律**:命名信号量按解析器限流(重引擎各 1)、pdfium 全局锁(C 库非线程安全→进程内串行)、扫描页进程池渲染(并行度 min(4,cpu),失败透明回退串行)、LibreOffice 独立 profile+3 次退避重试;**嵌套超时预算**(引擎 90m<单次调用 100m<文档 2h)——coco 现在只有 Tika 360s 单层,多后端后需同构预算
9. **排查指南产品化**:该文档自带"解析质量排查表"(版面差→换引擎/扫描缺字→查视觉模型或强制扫描/表头被吞→开表头开关)——Pipeline Studio 可长出同款"解析质量提示"面板
10. 补 D14 细节:locator 对齐**只比字母数字**(忽略 markdown 语法/空白/标点),quote ≤300 字,老格式(doc/ppt/xls/web)回退文本搜索定位

**与既有项合流**:1/2/5(魔数)→D13 L0/L1 施工图大充实;3→D10 表格路(与第六篇 #3 合流);4→D13 新增"格式补全";6→S2 实现参考;7→D16(图像门控已记,补"指令不进 OCR prompt");8→D13 L1 随多后端;9→Pipeline Studio 候选;10→D14。

### 第六篇(RAGFlow 开源对照):文档预处理全栈拆解与可借清单(2026-10-07)

来源:infiniflow/ragflow(Apache-2.0,~60k star,master cc72ecb0,2026-10-06;浅克隆 /tmp/ragflow 本地逐文件研读)。本篇聚焦**文档预处理**(解析→切分→入库前),与第五篇 WeKnora 的检索/记忆面互补。**关键事实:RAGFlow 主干已 Go 化**——go.mod/cmd/internal 全套 Go 实现,Python 只剩 rag/prompts;这对 coco 意味着可以直接代码级对照借鉴,不再是"Python 机制翻译成 Go"。其预处理架构:coco 式管道 DSL(File→Parser→Chunker→Extractor→Tokenizer→Index),但**切分方法选择=管道模板选择**——12 个内置模板(general/qa/manual/table/paper/book/laws/presentation/picture/one/audio/email)是 embed.FS 内嵌 JSON(internal/ingestion/pipeline/template/),注册表+别名(naive→general)保持存量数据可跑。结论同第五篇:**抄机制、不抄架构**。

**值得借的机制(按建议优先序;1-5 是 D10 的直接施工图)**:

1. **结构化单元 IR 贯穿全管道**——RAGFlow 解析器输出的不是纯文本而是带类型与坐标的单元流(`schema.ChunkDoc`:text/table/image/heading + ck_type + page_number + positions bbox 五元组 + context_above/below),切分器消费 IR 而非字符串。coco 现状:Tika text/plain 页文本 → `SplitPagesToChunks` 定长 rune 窗(fileproc/chunk.go),`DocumentChunk` 只有 Range/Text/Embedding(core/document.go:134)——**表格/图片/正文在 coco 是同一种东西**,这正是 W7"大表格切碎"的病根。D10 的第一性改动不是换切分算法,是**换中间表示**
2. **Token 预算+硬上限 merge 契约**(token.go mergeUnits,Python naive_merge 的收敛移植)——①预算单位是 cl100k_base BPE token(内嵌 BPE 表,internal/tokenizer),不是字符——中文 rune≈token、英文 4 字符≈token,定长 rune 窗对双语语料实际块尺寸差 2-3 倍;②**硬上限契约**:任何产出的文本块 ≤ chunk_token_size——超预算单元先按句子边界展开(分隔符保留、无损),仍超则按 token 硬切(hardSplitPiece O(L) 单次编码,切点永不落在坐标 tag 内);③**UNDER_CAP 贪心合并**:threshold = target×(100-overlap)/100,预留 overlap 空间;joinSep 会加 token,故合并前对实际拼接文本再复核一次;④overlap 前缀无条件前置且**修边到不超上限**(overlapFitPrefix);⑤**无损性纪律**:裸分隔符保留在段尾,所有切片拼回=原文逐字节——coco 现状是硬切 rune,句中腰斩。coco 的 chunk_size 是 rune 数、无 overlap、无硬上限契约,三件都可借
3. **表格/图片不进文本合并流 + 上下文注入**——table/image 单元永不并入文本 run(遇到即 flush);超预算 HTML 大表切分时 **caption+表头每片复制**(splitLargeHTMLTable),行对齐 positions 矩阵按片切,对不齐宁可丢坐标不指错行;媒体块带 ContextAbove/Below(token 预算、句子感知截取、表格边界不跨界)。这是"大表格切碎救不回"(第四篇遗留)的完整答案:**表格是结构化单元不是文本**,切碎时表头随行,检索命中表格块时上下文自带
4. **父子块的索引落地形态**(indexdoc/parent_child.go)——子块切分时把切前单元存 `mom`,入库时 MaterializeParentChunks:xxhash(dataset\0doc\0mom) 确定性 ID 生成**隐藏父行**(available_int:0 不参与检索),子行带 mom_id。与 WeKnora 固定尺寸父子(子384/父4096)不同:RAGFlow 的父=切分前单元,**天然段落/小节尺寸**,不需要额外一道父切分。对 coco:ES nested chunks(D10 若做)可用同构形态——子块检索命中、返回 mom 文本,mom_id 锚定 D2 locator 的 section 级
5. **分块变体家族=模板族**——GeneralChunker(按格式分策略:PDF 阅读序/DOCX 文本跨媒体续接/Markdown 语义/表格/通用)、TokenChunker(纯文本 token 合并)、TitleChunker 族(heading 层级→栈建树→DFS 路径,**祖先标题串进每个叶子块**——面包屑的另一种实现;hierarchy 深度参数;chunk_token_cap 校验 128-8000 默认 512)、QAChunker(Excel 双列/doc Q-A 前缀/嵌套 question stack,一对一块 "Q::…A::…"+行坐标)。**与 WeKnora 三级试错不同,RAGFlow 是显式模板选择**——用户选"论文/法律/手册"即选模板,无逐级验证器。对 coco(D10):两者可合体——coco 管道模板已有(9cbbfcd9 的 8+2 节),把"分块方法"做成管道模板参数即可复用 Pipeline Studio dry-run(预览=生产同一 DSL,第五篇机制④同构)
6. **Markdown 语义切分细节**(general.go mergeMarkdownUnits)——**<50 token 的短标题强制与后续单元合并**(防"孤儿标题块");heading 遇非 heading 前任则起新块;图片作为块附件并入文本单元(画布拼接有像素上限 8192/32M);表格吸收前置短标题。全是 coco Tika 路径(层级信息全丢)拿不到的,但 wiki/Markdown 文档路(coco 自有 Markdown 渲染管线)可直接借
7. **PDF 深度解析栈(DeepDoc,全部原生 Go)**——这是 RAGFlow 的立身之本,机制拆解:①**per-page worker pool**(进程级有界并发,页结果按页号收集,完成序无关);②**字符质量分诊**:ExtractChars→IsGarbledPage/IsScanNoise/英文检测——干净字符走"检测合并成行框"(不重 OCR),乱码/无字符才全页 OCR detect+recognize,OCR 失败回退字符路径,三级降级链;③216 DPI 渲染+**per-page zoom 重试**(默认 zoom 无框则 ×3 重渲至多 9×,只有需要的页付内存代价);④**DLA 布局分析**:YOLOv10 ONNX 进程内推理(cgo),10 类版面(title/text/reference/figure/figure caption/table/table caption/equation),1024 输入 ≤300 框;⑤**TSR 表结构识别**:6 类(row/column/column header/projected row header/spanning cell)→HTML 表格重建;⑥PDF outline(书签)提取挂首块;⑦阅读序排列(连续 positioned run 按行排序,无坐标单元是排序屏障不沉底);⑧TOC/页眉页脚移除+dehyphen 连字符修复;⑨**8 个可插拔远程视觉后端**(mineru/paddleocr/docling/opendataloader/somark/tcadp/plain_text+monkeyocr vlm)按 setup 选择,响应上限纪律(512MB/万 zip 成员/2GB 解压)。coco 现状:Tika text/plain+X-Tika-PDFextractInlineImages——无版面、无表结构、无阅读序、乱码检测靠 Tika 内部。**务实路线见 D13:不抄模型先抄纪律**
8. **bbox 级位置溯源**——每块带页号+bbox;merge 时坐标 extend;split 时**按可见文本比例垂直切**(slicePositionsByTextRatio,坐标 tag 不计高);overlap 只携带尾段坐标(overlapTailItems 防链式超携带);chunk **截图预览**(cropImageChunks 按需裁页区域→img_id,文本块也可裁)。对 coco D2:locator 从 page|section 可精到 bbox;截图预览是检索体验的差异化项(命中即见原文截图)
9. **QA/FAQ 形态**(D11 施工参考)——QAChunker 三源(Excel 双列/doc 前缀/markdown),一对一块+行号坐标;chunk 侧 questions/keywords 在入库边界转为 question_kwd/important_kwd 数组列(cleanupConsumedChunkFields)——FAQ"相似问入索引"的字段落位可照此
10. **大文档工程纪律**——JSON 坐标数组 O(n) 拼接 fast path(内联去括号直拼,10k 单元 PDF 实测省 ~11GB 分配);tokenize 记账用 running-sum 不重编码;4 goroutine 分块 fan-out+按索引合流(完成序无关);远程后端响应上限+zip 成员上限(zip 炸弹防御);cgo/nocgo 构建标签(无 cgo 时 OCR/TSR 优雅降级而非编译失败)——coco 单二进制红线下的可移植模式
11. **知识编译查重的批审判**(可选小件)——knowledge_compile 的 Deduper:KNN 找候选组+**单轮 LLM 批量判重合并**(DecideBatch,一组一次调用而非逐对)。coco D1 是确定性指纹(路线正确,RAGFlow 无对应物、coco 已领先),但治理队列未来若加"LLM 确认"步骤,批审判比逐对省 token 一个量级

**coco 已领先、明确不抄的面**(防 cargo-cult):**查重**(RAGFlow 是 KNN+LLM 语义判重,零确定性指纹——coco D1 四层确定性算法更可信更便宜);**治理/人工闸门**(RAGFlow knowledge_compiler 是 LLM 自动编译自动合并(RAPTOR/wiki/mindmap/structure graph),无"AI 只产草稿、发布必须人工"概念——与 D1 红线相反;其 wiki 编译输出可作 coco AI 草稿源,合并决策不抄);**引擎原生检索**(RAGFlow 七引擎抽象 ES/Infinity/OceanBase/ClickHouse/kvrocks/serenedb/NATS 是其历史包袱,Easysearch 原生是 coco 护城河);**权限模型**(tenant 级,无 coco 索引/文档/字段三级收敛);**C++ RAGAnalyzer 绑定**(coco 引擎侧分词+pinyin 已够,单二进制不引 C++ 依赖);Python 残留/多租户 SaaS 化/EE 插件机制——产品级决策,先记不排。另注意:RAGFlow 的 BPE 表(cl100k_base)内嵌二进制约 ~1.7MB,若 coco 借 token 预算需接受该体积或用引擎侧 _analyze 计数(需实测,ES _analyze 每块一往返可能太贵——倾向内嵌表)。

**落为候选(等 review 排期)**:D10 分块升级施工图充实(IR/token 预算/硬上限/表格保护/父子块/模板=管道参数,本篇 1-5+6);**D13 PDF 解析分级强化(新增,见第四章)**;D11 QA 落位参考(本篇 9);D2 locator bbox 增强+chunk 截图预览(本篇 8,截图可作为检索体验候选);治理 LLM 确认批审判(本篇 11,并入 D1 可选增强)。

### 第七篇(知识编译器专题):把数据源编译成 wiki 知识——RAGFlow knowledge_compiler 全拆解+外部格局(2026-10-07)

来源:ragflow 本地代码(internal/ingestion/component/knowledge_compiler/{wiki,structure,tree}、internal/ingestion/knowledge_compile/{scheduler,wiki_dirty,wiki_contribution,dedup});外部:Karpathy《LLM Wiki》gist(2026-04)、STORM/Co-STORM、GraphRAG/LightRAG、Zep/Graphiti。本篇回答一个问题:**D2/D6 说了"编译",编译流水线本体长什么样**——coco 的 wiki 检索路(D2)与编译规则(D6)都有了,缺的是把数据源变成 wiki 页的机器。

**问题框架(Karpathy《LLM Wiki》gist,2026-04)**:RAG 是每次查询从零重新发现知识——无积累,跨文档综合题每问一次重拼一次;解法是**像编译源代码**:原始资料一次性编译成互链 markdown wiki,之后**查 wiki 不查原文**。wiki 是人类可读可改的(人工维护是特性不是负担)——与 coco wiki-first 路线同向,而"人工维护"正好落在 coco 治理闸门上,这是 coco 相对一切自动编译系统的结构优势。

**RAGFlow wiki 变体机制拆解(MAP→REDUCE→PLAN→REFINE,wiki.go ~3300 行)**:

1. **MAP 抽取**:块按 ~2048 token 预算成批(刻意远低于上下文窗口,给 JSON 输出留余量防截断)→ LLM 抽 wikiExtract{entities(name/type/aliases)/concepts(term/definition)/claims(statement/subject/confidence)/relations(from/to/type)/topics(path)},**五类全部带 SourceChunkIDs**——证据回溯到块粒度是后面一切(合并/撤回/引用)的地基
2. **版本化 MAP 缓存**(wiki_map_cache.go):per-chunk 结果不可变,key=tenant+dataset+doc+**模板指纹+LLM 指纹+schema 版本(现值 "topic-path-v2")+块内容 hash**——换 prompt/换模型/升 schema 自动整体失效,未变块零 LLM 成本;结果按块拆回存,支持块粒度增删证据
3. **REDUCE 纯确定性零 LLM**:normKey 归一(大小写/空白)合并多批 extracts——LLM 只在两头(抽取+写作),中间全是代码
4. **PLAN 双模式**:实体模式一实体一页(确定性零 LLM);主题模式 LLM 规划器,批配额+**全局页数目标与硬上限**,溢出计 capacity_excluded(可观测不静默丢);与既有页 reconcile=embedding 候选+邻居扩展+rerank→CREATE/UPDATE
5. **REFINE 每页一次写作**:输入=证据块+来源上下文+**全部 plan slug 列表([[wikilink]] 一次埋好)**+UPDATE 时带既有页内容;See-also 确定性追加不进 LLM
6. **增量编译机制群**(这是 wiki 变体最值钱的部分):activeState 快照(chunks+plan)diff→只动 AffectedPageSlugs/RemovedPageSlugs;文档级 trailing-edge debounce 20s(重复事件合并 chunk-ID、10min lease 续期);数据集级 durable backlog 有序日志(**同 doc 末事件赢**)+claim token(token 不匹配必须弃写防双主)+消息队列仅尽力唤醒(DB 是调度真相);**wiki_contribution 每 doc→pages 贡献记录**——删 doc 时精确知道哪些页失了证据

**外部格局四条**(各取一个教训):
- **STORM/Co-STORM**:写前先问——perspective 人设引导提问→多视角收集→outline→成文。借**编译前 gap 提问**为可选预研步(对数据源问"还缺什么",输出喂 D5 缺口榜);不借多 agent 辩论主流程(成本/延迟不成比例)
- **GraphRAG**:Leiden 层次社区+社区报告,local/global/DRIFT 检索强,但**增量更新是公认痛点**(issue #511,社区重算代价高)——教训:编译器必须块级缓存+页级 diff,永不做全局重编译
- **LightRAG**:双层(实体低层/主题高层)增量实体关系图,效果接近 GraphRAG 成本是其零头——教训:**增量友好>表达力**;coco D3 轻量图路线正确,不为编译器另起重型全局结构
- **Zep/Graphiti 双时间线**(valid time+transaction time):边失效不删除、重叠检测——对齐 coco conflict 提议语义:裁决=旧主张失效(留历史)而非覆盖;D3 图路径远期对齐

**明确不抄(防 cargo-cult)**:①**LLM 自动判重自动合并**(CosineDecider 相似即 merge、编译结果直接生效无人工)——与 D1 红线相反,coco 一切编译产物=治理提议/草稿,发布必人工;②**多租户调度面**(MySQL 真相+NATS 唤醒双组件)——coco 单二进制,Easysearch 是唯一状态真相,debounce/backlog/租约引擎内实现即可;③RAPTOR 树/structure 图全量变体——v1 只做 wiki 页形态(图=D3 既有路径,树不排);④STORM 全场辩论。**coco 已领先面**:D1 确定性指纹(输入侧去重比其 KNN+LLM 便宜可信);治理队列即人工闸门(RAGFlow 无对应物);四路 RRF 已含 wiki 消费侧;wiki_article 证据模型(Sources 带 locator)比其 slug 引用更完整。

**落为候选(等 review 排期)**:**D17 知识编译器(新增,见第四章)**;D6 编译规则与 D17 对齐(规则字段=编译器参数);D5 缺口闭环可接 STORM 式 gap 提问(可选);D3 图路径远期对齐双时间线失效语义。

## 三、对照差距:五个真缺口

1. **查重候选靠标题重叠**(governance.go:162 "candidate pairs come from title overlap")——正是 Knowly 批评的失败模式:改名的副本、Word→PDF 同内容全部漏检。全仓库无 content_hash/simhash 字段。→ **D1 已落地**
2. **检索层完全不覆盖 wiki 文章**(internal_search.go / search.go 零 wiki 引用)——JitKnow"核心层直接读、边缘层兜底"的分层协同在我们这里断开:策展好的知识检索不到,双路 RRF 只打 coco_document。→ **D2 已落地**
3. **无图路径**——JitKnow 三路检索里有图谱遍历一路;我们有实体图(正逆关系/反链)但没接进检索。→ D3
4. **纠正回路只有扫描侧、没有工作侧**(J.B. 篇核心机制)——治理提议全部来自后台体检(governance 扫描),而聊天侧零反馈入口(已核实:assistant API 与 chat 页面无任何 feedback/dislike/rating)。"每一次纠正回流大脑"在我们这里断开:用户在聊天中纠正了 AI,知识库毫不知情,同样的错误下周还会再犯。→ **D7**
5. **无公司地图与来源优先级**——wiki 页型仅 entity/concept/source,无导航入口页;治理有 conflict 提议但无"冲突按来源优先级裁决"的规则;能力层(技能/MCP server)游离在大脑导航之外。→ **D6 扩展 + P3**

次要差距:融合后无 rerank(Mock backlog 已列,应提前);RAG 可观测性(低召回查询榜等,归入 P2 运营概览);编译规则(粒度/引用/冲突策略)未产品化为配置。

## 四、产品设计完善:新增工作项 D1-D7(候选 D8/D9 已落地;D10-D17/P5/S1/S2 待排期)

### D1. 查重引擎(确定性四层,无 LLM 依赖)— 新增最高优先

**原则**:Knowly 路线,纯本地确定性算法,LLM 只做可选确认。
- **管道指纹阶段**:8+2 管道新增 fingerprint 富集节——归一化文本(去空白/统一全半角)sha256 为 `content_hash`,全文 simhash64 为 `content_simhash`(Document 新增两个 keyword 字段,入库时算好,增量不重算)
- **查重组件**:①hash 精确分组 ②simhash 海明距离 ≤3 为候选 ③段落 minhash 相似度(小改动) ④文件名/时间启发(疑似新旧版本);governance 的查重候选从"标题重叠"换成指纹候选(标题重叠仅兜底),LLM judge 降为可选确认
- **查重评审 UI**:四档标签(文件完全相同/内容完全相同/内容高度相似/疑似新旧版本)+ 相似度% + 证据(路径/大小/时间);动作四选:移回收站 / 保留但排除检索 / 标记非重复(持久化,不再问) / 暂缓——系统只推荐,绝不自动删(D1 红线同构)
- **检索感知**:检索结果按 content_hash 折叠,组内取代表位,附"另有 N 个副本"提示
- 参考 Knowly 实测量级:1454 文件 10 分钟 81 组——单机全量可接受,增量随管道走

### D2. 检索接入 Wiki 层(wiki-first 分层协同)

- **多路 RRF 泛化**:rrf.go 从双路参数化改为 `[]route`(数学不变:每路 weight/(k+rank)),hybrid_rrf 加第三路 **wiki**:对 wiki_article 的 combined_fulltext(title/summary/aliases/tags 已 copy_to)跑 BM25,带 KB 权限过滤
- **隔离规则**:仅 status=published 进检索;confidence=low 或单源概念页(sources<2)隔离在治理队列不进检索——噪声闸门,对齐 JitKnow pending 页语义
- **溯源链**:命中 wiki 页返回 Sources(含 locator),chat 引用链打通到 page/section/clause 级
- **检索实验室**:加 wiki 路面板,三路分数+贡献并列可试跑

### D3. 图路径(graph route)

- 查询 → 实体识别(ontology schema 词汇表+别名匹配,零 LLM 可先做)→ 关系遍历(graph.go 已有正逆关系端点)→ 相关实体/文章排序 → 作为 RRF 一路参与融合
- 典型链路:故障现象→可能原因→诊断步骤;产品→部件→工单;概念→关联概念→来源文献

### D4. rerank(从 Mock backlog 提前)

- 融合后接 reranker(model provider 配置,Qwen3-Reranker 类);无 reranker 时降级纯 RRF(与语义路降级同理)
- 检索实验室可对比融合前/后 top-K,给出 rerank 增益证据

### D5. 知识缺口闭环(迭代,归入 P2 运营概览)

- /ops/overview 增加:检索策略分布、低召回查询榜(零命中/低分)、平均响应时长
- **信号源双通道**:低召回 query 自动生成治理提议("缺 X 的概念页",JitKnow"持续集成"式迭代)+ D7 的 correction 提议——J.B. 篇"每一次犹豫、每一次纠正都在告诉你大脑缺什么",两路信号汇入同一治理队列,知识找人
- 离线全检已有(orphan/断链/冲突),补查重组复扫周期

### D6. 编译规则产品化

- ontology schema 扩展为"知识编译规则":页面粒度 / 引用必须性 / 冲突策略(标记 vs 隔离) / 概念页最少源数——schema 编辑器已有,加规则字段
- **来源优先级**:规则里加来源排序(datasource/文档类别级),治理 conflict 提议裁决时按优先级给推荐方向——J.B. 篇"冲突按来源优先级裁决"的落地
- **公司地图页(page_type=map)**:每 KB(或全局)一张人工维护的入口页——我们做什么/当前优先级/来源优先级表/导航(到决策、政策、技能、学习);P3 MCP 提供 `get_company_map`,外部 agent 先读地图再干活。对应 J.B. 篇"更多上下文=更多困惑,解法是导航而非更多记忆"——地图页是给 agent 的入口,不替代检索
- 增量全局更新语义明确为"编译一次":新文档入库 → 受影响实体/概念页刷新提议 + 交叉引用重算(governance/freshness 已有钩子)——流水线本体见 D17(第七篇),本条的规则字段即编译器参数面

### D7. 纠正回路(工作侧学习信号,J.B. 篇核心机制)

**原则**:每一次纠正回流大脑,但**审查后才共享**(与 D1 红线同构,绝不把用户一次纠正直接写成公司知识)。
- **correction 提议类型**:治理提议新增 type=correction,负载带上下文(query/被纠正的答案/当时引用的文档)+ 路由提示(事实缺失/事实过期/偏好/技术/来源冲突)——复用既有治理队列与裁决 UI,零新队列
- **聊天反馈入口**:助手答案加轻量纠错动作(提交即产 correction 提议;不做评分体系,避免噪声)
- **裁决路由表**(人工拍板后落位,对齐文章路由表):事实缺失/过期→对应 wiki 文章新版本草稿(仍走人工发布);反复出现的偏好→D6 编译规则/政策;验证过的技术→技能库;无法归类→知识缺口(并入 D5 榜)
- v1 范围:提议类型+API+治理队列展示先行;聊天 UI 反馈按钮与 MCP 反馈工具(report_correction)随后
- 与 D5 的关系:D5 是"系统自己发现缺什么"(低召回),D7 是"用户告诉你错在哪"——一读一写,汇入同一队列

### D8. 查询改写(rewrite 路)— 候选,第四篇产出,待 review 排期

- **检索入口加改写步**:便宜 chat 模型(默认模型回落即可)把口语 query 改写为文档正式表述,prompt 只输出改写结果不带解释;超时 2-3s,失败/为空返回原 query——零风险降级,与语义路降级纪律同构
- **稳妥路(不替路)**:原 query 与改写 query 各跑一路 keyword(语义路可选),结果一起进既有 RRF——纯加路,融合数学零改动;改写命中+原 query 未命中时 RRF 自然抬权
- **改写缓存**:query 归一化哈希(normalizeSearchQuery 已有)→ kv 缓存 TTL 24h 级,高频口语不重复打模型(文章:改写每次几百 ms,缓存是必须不是优化)
- 可观测:检索实验室加改写面板(显示改写结果+两路贡献分解);SearchLog 加 rewritten 标记,ops 概览可量化改写增益(改写路的命中占比)
- 与图谱别名路互补:别名救**实体名**(身份证号→统一社会信用代码),改写救**表述**(十三薪→年终双薪是文档词汇不是实体);两路都零 LLM 依赖时走别名,改写是补最后的词面鸿沟

### D9. 评估集(golden queries 回归)— 候选,第四篇产出,待 review 排期

- **金标题集**:每 KB 一组 Q→期望命中文档(人工标注,50 条量级起步,正好用演示栈走查用例当种子);检索实验室加"评估集"页,一键跑全量,出 top-4 命中率/MRR
- **调参前必跑**:RRF 权重/rerank 窗口/graph_weight/D8 改写开关——任何检索侧改动前后各跑一遍对比;文章最重的一课:"评估集比任何参数都值钱",没有它调参是盲调
- 结果入 SearchLog/ops 概览:答对率趋势线可见(文章 78%→94% 那条线,我们的版本是评估分走势)
- 可选:MCP 工具 run_eval(外部 agent/CI 也能跑);零命中但评估集里有期望答案的条目自动回流 D5 缺口榜——评估集与缺口闭环互喂

### D10. 分块升级(三级自适应+面包屑+父子块)— 候选,第五篇产出,兼修 W7 后置项;第六篇补施工图

- heading→启发式→递归逐级试+劣质切分验证器(空产出/超长单块/碎块/比例失衡均拒绝降级);表格/代码/LaTeX 保护 span 不切断;ContextHeader 面包屑落 chunk 并拼入 embedding 输入(单块编辑单块重嵌);父子块 small-to-big(子 ~384 入索引、父 ~4096 返回,重叠 1/5);Pipeline Studio 预览与生产共用配置归一化;配置逐级覆盖(上传>KB>数据源默认)
- **第六篇施工图(RAGFlow,Go 同源可直接对照)**:先换中间表示(类型化单元 IR:text/table/image+坐标,fileproc 出 Tika HTML 结构而非 text/plain)再换算法;预算改 cl100k token(内嵌 BPE 表 ~1.7MB)并守硬上限契约(超预算单元句子边界展开→token 硬切兜底,裸分隔符保留、切片拼回=原文);overlap 前缀修边不超上限;表格/图片不进文本合并流、大表切分表头随片、媒体块带上下文窗口;父子块落地形态=mom 隐藏行(available_int:0 不检索)+子行 mom_id(父=切前单元,天然段尺寸,不必另设父切分);分块方法=管道模板参数(复用既有 8+2 管道与 Pipeline Studio dry-run)
- 存量文档不强制重切:新管道生效、重建索引才迁移——与 D1.5 指纹同理(写入路径盖戳,历史留白)
- 联动:面包屑=section 级 locator(D2 溯源精度跟涨);父子块=实体抽取/图谱可吃父块上下文

### D11. FAQ 知识库类型 — 候选,第五篇产出

- KB(或 doc 子类型)新增 faq:标准问+相似问入索引、负例问题绝不入;查询期混合召回→负例精确过滤→唯一条目不足且饱和则迭代倍增 TopK;直答阈值标记 exact(模型直接采用标准答案);归一化 ContentHash 幂等(复用 D1 指纹归一化链);FAQ 命中作 RRF 独立一路或 rerank 加权(与 wiki 路同构:加一路=一个名字+一个权重)

### D12. 长期记忆(确认制) — 候选,第五篇产出

- 五类(profile/preference/fact/task/interest);会话蒸馏异步产出=pending 治理提议(memory 类,带上下文与路由提示),**确认后生效**——人工闸门同 D1/D7,绝不静默注入;fact/task 查询期向量召回注入,profile/preference 常驻限长;search_memory 作 MCP 工具(第 46 个);按用户隔离;蒸馏限频+游标防重启丢消息

### D13. 解析分级强化(PDF 深水+格式补全)— 候选,第六篇产出,第五篇补遗二充实,兼修 W7"OCR 后置"

RAGFlow DeepDoc 全栈(布局模型/TSR)与 coco 单二进制红线冲突,按"先纪律后模型"分级:
- **L0 零模型可先做**(RAGFlow+WeKnora 补遗二双来源施工图):PDF outline(书签)提取入 metadata(标题层级→D10 heading 切分的输入);**Tika 切 HTML 路径拿结构**(现状只用 text/plain,TikaGetTextHtml 已有);**逐页 text/scanned 分诊**(图片覆盖率 ≥0.5 或字符 <10 且有图→渲染 OCR;渲染纪律 DPI 200/JPEG 85/长边 2000px;全页渲染兜底+pdf_force_scanned 覆盖);几何重建启发式(XY-cut 切列/边栏剔除/字距空格/**字号→标题升级**/页码与碎屑清理/**保守页眉页脚**——首尾行+短+≥60% 文本页才删);**隐藏文本(render-mode 3)与页外字形过滤防提示注入**;乱码页检测(char 分布异常→强制 OCR 标记);**魔数纠偏**(OLE 魔数的".docx"→DOC,file_type_detection 加);Excel 结构化路(合并单元格填充→每行 `列名: 值` 块+sheet/行区间 locator+首行表头开关→并 D10 表格路)
- **L0.5 格式补全(Go 原生零依赖,coco 全缺)**:EPUB/XMind/MHTML/本地 HTML;音频 ingestion 走 enterprise ASR(时间轴 locator)
- **L1 外部后端可插拔**:解析后端抽象(现状隐式 Tika 单后端)——Tika 保底+可选 mineru/docling 类远程视觉服务(PDF→Markdown+表格+阅读序),**FirstParser 首胜链**为多后端选择原语,per-KB/数据源 parser_engine_rules(文件类型→引擎+类型化覆盖项);SSRF 白名单/响应上限/**嵌套超时预算**(引擎<单次调用<文档)随 S2 一并收口
- **L2 模型级(暂不排期)**:进程内 ONNX 布局/表结构(对标 RAGFlow DLA/TSR)——模型体积与 cgo 依赖需单独立项评估,先看 L0/L1 的实际收益

### D14. 块级检索与溯源位 — 候选,第五篇补遗产出(与 D10 配对:D10 写入侧,本项检索侧)

- 前提事实(补遗缺陷 1):块向量在写从未被查、块文本不在 BM25 字段——检索粒度是整文档单向量。升级三选(按侵入度):
  - **a 客户端腿扩展(最小)**:clientSemanticRecall 从"拉 ai_insights 单向量"扩为"拉候选文档块向量取最优块",命中记块号——不动索引
  - **b 块向量平铺顶层字段**:document_chunk 向量提升为文档级多值字段(引擎 nested knn 本版不可用,P0 探针),text 腿字段补块文本
  - **c 块独立索引(推荐,与第六篇 mom 形态同构)**:coco_chunk 索引——子块可检索+mom 隐藏行(不进检索)带 mom_id;text/semantic 腿改打块索引,块命中映射回文档参与 RRF(数学不变,加路不加复杂度)
- **溯源位**:块携带 SourceLocators(页/slide/sheet 行区间/section;D13 L0 的 outline/结构先行)+quote≤300 字;引用链从文档级升页/节级,**D2 locator 的文档侧断点接通**;顺手落 Mock backlog 高亮(命中块 quote 高亮);对齐纪律(补遗二):**只比字母数字**(忽略 markdown 语法/空白/标点),老格式(doc/ppt/xls/web)回退文本搜索定位
- 依赖:D10(先有像样的块)/D13 L0;评估集护航(D9,块级改动前后跑分)
- 配套:跨文档块级近重去重(token 重叠 ≥0.85)产品化为检索实验室旋钮——hash 折叠管精确重复,此件管近似重复

### D15. 文档生命周期与重处理闭环 — 候选,第五篇补遗产出(修补遗缺陷 1-3)

- **状态机**:processed 布尔 → status(processing/finalizing/completed/failed/cancelled)+在途富化计数——富化在途可查可取消、富化失败不挂文档(摘要/标签/图谱全 best-effort);stall 区分积压/卡死;每文档处理时间线(逐步耗时+尝试号),运营页出失败文档榜
- **_reprocess 端点**:清理旧块/向量/在途→重跑管道→**重戳 embedding_model 版本**;数据源级批量。嵌入模型治理并入:per-doc 模型戳+混模检测(engine-capability 扩)+模型切换提示批量重处理——现在换模型是新旧向量静默混查
- **编辑/入口闭环(缺陷修复)**:PUT /document 内容变更触发重切块+重嵌入(现在静默过期);POST /document 与 /datasource/:id/_doc 补跑默认富化管道(现在完全绕过)
- **上传去重**:指纹前移到入库闸门(scoped 数据源,重复刷 created_at 返回既有)——D1.5 从事后扫描前移
- **富化面**:生成式问题(每块 N 问题,独立向量行+rerank passage 拼接双用,喂 D14/D4.5);rerank passage 组装=title+面包屑+正文+问题/OCR(→D4.5)

### D16. 知识组织与受控标签 — 候选,第五篇补遗产出

- 文档虚拟目录 folder_path(重命名/移动改路径不重解析,聚合出树);标签受控词表(extract_tags 只从池中选、永不新建、永不删手工标签,池=数据源 top-N);检索参数加 tags 作用域(text 腿 filter,与 disabled 过滤同构)
- 数据源/KB 能力位(vector/keyword/wiki/graph)暴露给助手编辑器与 MCP search_documents schema——WeKnora capabilities 字段同构,agent 侧按能力过滤
- 细节借鉴:KB GeneratedProfile(聚合 hash 不变跳模型、永不覆盖人工描述——AI 摘要的幂等省钱版);图像 OCR 门控(一次 VLM 观察调用决定是否二轮 OCR,装饰图省成本)

### D17. 知识编译器(数据源→wiki 知识编译)— 候选,第七篇产出

Karpathy"compile, don't interpret"框架+RAGFlow wiki 变体机制落到 coco 既有资产;**红线不变:编译器一切产物=治理提议/AI 草稿(draft 状态),发布必人工;"系统只推荐、绝不擅自删"同样适用于撤证据(只标 stale 提议)**。RAGFlow 是 LLM 自动合并直接生效——这条明确不抄,其余机制全可借。
- **四段流水线**(与 RAGFlow wiki.go 同构,输出改道治理队列):MAP(块按 ~2048 token 成批→抽实体/概念/主张/关系/主题,五类全带 SourceChunkIDs)→REDUCE(normKey 确定性合并,零 LLM——LLM 只在抽取与写作两头)→PLAN(实体页一实体一页确定性+主题页配额与硬上限(capacity_excluded 可观测);对既有页 reconcile:embedding 候选+rerank→CREATE/UPDATE)→REFINE(每页一次写作:证据块+**全 slug 列表一次埋好 [[wikilink]]**+UPDATE 带既有页内容)——REFINE 产物是**提议负载**(新页草稿/页更新 diff/冲突标记),入治理队列人工裁决
- **增量与缓存(编译成本三道闸,全部可借)**:①MAP 结果按块内容指纹+**模板/模型/schema 指纹**缓存——未变块零 LLM、换 prompt 自动失效(与 D1 指纹哲学同源);②activeState 快照 diff 只产受影响页提议;③触发=文档入库/重处理完成(D15 状态机钩子)+trailing-edge debounce;**贡献记录(doc→pages)**支持撤源时定位失证据页→按 D6 引用必须性检查→stale 提议(不自动删)
- **复用清单(近零新基建)**:wiki_article 页模型(page_type entity/concept/topic、status、Sources)零改动;AnchoredProposalID/AnchoredEntityID 防裁决 TOCTOU;Sources{doc_id,excerpt,locator} 直接承接 SourceChunkIDs(D14 溯源位接通 locator 断点→证据可点回原文);四路 RRF wiki 路即消费侧(D2);ontology schema=D6 编译规则(页面粒度/引用必须性/来源优先级/概念页最少源数=编译器参数);D9 评估集护航(编译前后 wiki 路命中回归对比)
- **冲突语义**:编译期不做 LLM 裁决——矛盾主张只**标记**产出 conflict 提议(带双方证据块),裁决在治理队列人工+D6 来源优先级;远期对齐 Zep 双时间线(旧主张失效留历史,不覆盖)
- **范围裁剪与成本护栏**:v1 只 wiki 页变体(structure 图/RAPTOR 树不排);STORM 式 gap 提问=可选预研步(输出喂 D5);调度不引外部组件——debounce/backlog/租约全部 Easysearch 内实现(索引当 durable log,同 doc 末事件赢)
- **验收口径**:同批文档二次编译**零 LLM 调用**(全缓存命中);增删单文档只产受影响页提议(不全局重编译);每条提议证据链可点(SourceChunkIDs→excerpt+locator);未裁决提议不进检索(wiki 路只打 published/reviewed)

### P5. MCP 端点令牌与 KB 作用域 — 候选,第五篇产出

- 端点实体:命名+Bearer token(哈希落库/明文仅示一次/可轮换)+KB 白名单(空=全部)+工具组(检索读/QA/wiki/写默认关)+每分钟限速+last_used_at;外部 agent(Cursor/Claude)按端点接入;api.MCPTool 权限过滤复用,端点作用域再收窄

### S1. 凭据加密(enc:v1:) — 候选,第五篇产出,安全

- 主钥环境变量,AES-256-GCM,密文 enc:v1: 前缀;**读侧兼容明文**(渐进迁移零停机);凭据字段只写端点、出参既脱敏也永不回显;覆盖模型商/连接器/MCP server 三处凭据

### S2. SSRF 收口 — 候选,第五篇产出,安全

- 出站(连接器/scraper/MCP client fetch)统一 URL 校验器;重定向逐跳重验;白名单模式可配(默认关,开放部署/托管开)

## D8/D9 落地记录(2026-10-04)

第四篇调研产出当天实现,检索侧第 5 路 + 评估回归面:

- **D8 查询改写**:`modules/document/query_rewrite.go`——默认语言模型改写口语 query(系统提示词:同义术语/缩写展开/口语转书面,只输出改写文本),2.5s 超时、归一化哈希缓存(TTL 24h,上限 4096 条,溢出整体清空)、三连失败断路器(冷却 60s,下线模型不给每次检索加超时)、`SEARCH_QUERY_REWRITE=off` 全局开关、`rewrite=0`/`rewrite_weight=0` 按请求静音;清洗剥 <think>/围栏/"改写:"前缀/包裹引号,超长视为解说词拒收。融合为**加路不替路**:原 query 四路照跑,改写文本多跑一路 keyword(`rrfRouteRewrite`,RRF 数学零改动,坏改写只能加候选不能删)。SearchLog 加 `rewritten` 字段,ops 概览出"改写 N 次/其中零命中 M 次"(对照总零命中率即增益观测面);Warning 头带 `query rewritten: ...`。与图谱别名路互补:别名救实体名,改写救表述
- **D9 评估集**:`core/search_eval.go`(SearchEvalCase/SearchEvalRun/SearchEvalCaseResult)+ `modules/document/eval_set.go`——CRUD 按 query 归一化幂等(重标注即覆盖);`POST /search/studio/eval/_run` 逐条跑**生产融合管线**(克隆请求保留权限头,pin from=0/size=top-N,rerank 含在内),评分 top-4 命中率 + MRR + 平均耗时,run 落库出趋势;零命中用例(有期望文档却召回为空)直接回流治理队列 knowledge_gap 提议(evidence 带 source=eval_set 与期望清单,复用 query 哈希幂等锚,仅推荐)。MCP 工具 `run_search_eval`(第 45 个);索引体检 6→8(eval_cases/eval_runs)。前端:检索实验室底部评估集卡(用例表 + 添加表单——期望文档直接用线上搜索挑选 + 跑分按钮 + 每用例命中位次/Top-4 返回 + 最近跑分趋势),改写卡显示 原→改写/缓存命中/耗时/未启用原因
- 测试:改写(应用+缓存命中、同文不改写也缓存、失败三连断路器后不再打模型、长度/无模型/总开关守卫、清洗各形态)、评估(空集零除、top-4 边界(rank4 算/5 不算)、MRR 数学、ID 去重/标题对齐);tsc 棘轮维持 598

**Live 走查记录(2026-10-04,f13fe921)**:演示栈全链验证通过——索引体检 8/8(新增 eval_cases/eval_runs);MCP 44→45(run_search_eval);评估集 CRUD/跑分/趋势落库(top-4 命中率+MRR+每用例命中位次,mysql 用例 rank1、词面错配用例诚实记 miss);零召回用例自动回流 knowledge_gap 提议(evidence.source=eval_set,重跑幂等仍 1 条);studio 改写区块在无语言模型配置时正确降级并说明原因;ops 概览 rewritten 字段就位;SPA entry 含 i18n 键、懒加载分片含 top4_rate/rewrite_weight 代码。**抓出并修复两个真 bug**:①D1.5 的 content_simhash 用 uint64 写 ES long(有符号)——simhash 高位为 1 时(约一半文档)写入直接 document_parsing_exception,改 int64 位模式不变(853b9004);②评估 run 的 Top4Rate 忘记赋值(响应与落库恒 0),补一行计算。教训:无符号整型落有符号列是必炸组合,写库前对齐类型;另注意演示栈 /query 用户面有数据源权限过滤,无数据源的内部文档只在 /document/_search 可见——评估用例的期望文档必须选用户面可达的文档。

**D8 生效路径验证(2026-10-04,f465b081 收尾)**:本机无 ollama、演示栈供应商无 key——起本地 mock OpenAI 兼容端点(/tmp/mock-llm.py,:9199,"十三薪"→"年终双薪 发放 政策"),注册供应商 mock_local 并设为默认语言模型,**不改写任何真实凭据**。全链实测:hybrid 检索 Warning 头出 `query rewritten: 年终双薪 发放 政策`;词面错配查询召回"年终双薪发放政策"(原 text 路召不回);改写路独立命中 top-1;**缓存命中**(mock 调用数不增长);SearchLog rewritten=1 进 ops 概览;评估趋势 0.25→0.5("十三薪"用例 miss→rank1)——文章"改写+评估集再 +6 分"叙事的微缩复刻。过程中又抓一真 bug:**评估用例写入不等刷新**,操作员"加用例→点跑分"读到旧快照(实测两次)——用例 CRUD 补 WaitForRefresh(f465b081),新建用例当轮可见验证通过。演示栈现状:默认语言模型指 mock_local(mock 进程存活期间改写路可演示;换真实模型:设置→默认模型)。

## 五、遗留工作项(沿承)

### W5. 本地环境(优先级:高)

| # | 事项 | 状态 |
|---|---|---|
| 5.1 | 配置默认模型(解锁语义路/实体抽取/AI 摘要/治理 LLM 确认) | 待 medcl 提供 LLM 密钥 |
| 5.2 | 磁盘大扫除(97%,数据卷 32Gi 余量;2026-09-30 已完成只读分析供 medcl 决策:JetBrains 缓存 60G 为最大单块,go src 30G + mod 9.4G + go-build 4.2G,Yarn/Google/claude-cli/LarkShell/pip 各 1.4-3.9G,~/go/infini.sh.bak.zip 1.6G、go1.21.5 旧工具链 1.4G;**未动任何文件**,清单与建议命令见会话报告) | 分析完成,待 medcl 清理 |
| 5.3 | REPO_PAT 换新 | 已恢复,本会话 integration_test 多次通过 |
| 5.4 | 本地测试 ES(2026-09-29 就绪):medcl 提供的 easysearch-2.4.0-2963-mac-arm64-bundle 起在 **http://127.0.0.1:9200**(security 关,数据/日志独立在 ~/es-d3,bundle 目录保持原样);coco 测试实例配置 ~/coco-it-9003 已对上 | ✅ ES 侧就绪 |
| 5.5 | Cisco AnyConnect 过滤器仍拦 coco 进程(本会话新证据:**Developer ID 真证书签名也无效**,2913 listen 调用本身超时;桌面版 app pid 存活说明过滤器按进程放行已签名 app);live HTTP 走查(引擎 AI/查重/实验室四路)继续挂账,过滤器放行或断开 AnyConnect 后几分钟可补完 | 阻塞 |
| 5.6 | release.infinilabs.com 大文件传输不稳(90.9%/49.6% 断流+挂死,疑似也导致 medcl 手动下载了本地 bundle);CI 已三层加固:下载重试×3(4b7b1547)+ 单次 timeout 600(c4d37269)+ **~/.curlrc 全局断点续传 --continue-at -/--retry 8/失速 60s 断**(79df5a15,curl 每次调用都读,安装器内部的下载也继承)——加固后当轮即绿;服务器侧健康仍建议引擎团队排查 | 已缓解,观察 |

### W6. 工程质量(持续)

- tsc 棘轮门禁已建(598 ≤ 基线 601),保持
- widget 独立构建同步(ui-search 深色变体源码已修,dist 不入库)
- ui-common#104 审发

### W7. 明确后置(不排期)

- 权限组织树(部门/角色维度分享)、OCR、图谱规模化(>500 实体分页/聚焦子图)
- Mock backlog 其余:多跳检索、查询路由、高亮、目录同步/密级、审计哈希链、死信队列 UI、联邦 RSA

## 六、建议执行顺序

```
第 0 步  R1(87f191d4)review                                    ← 已过,开整
第 1 步  P0 引擎探针+语义路自愈 → P1 embedding 统一              ✅ 8e2a1045(2026-09-29)
第 1.5 步 P1.5 管道策展(引擎 AI 设置/下发/回显)                 ✅ 63f08caa(2026-09-29,CI 8/8)
第 2 步  D1 查重引擎(确定性、无 LLM 依赖、可独立演示)            ✅ 本批(2026-09-29)
第 3 步  D2 wiki 检索接入(多路 RRF)→ D3 图路径                   ✅ 均落地(2026-09-29)
第 4 步  D4 rerank + P2 运营概览(含 D5 缺口闭环、D7 纠正回路)     ✅ 5ad99954(2026-09-29,CI 8/8)
第 5 步  P3 MCP 工具族 → D6 编译规则(含地图页+来源优先级)→ P4 索引体检   ✅ 3aa6acc5(2026-09-29,CI 8/8)
收尾批   聊天纠正按钮 + index-health UI + 地图模板 + D1.5 指纹落库/折叠     ✅ d4e2146c+1545f7ae(2026-09-30,CI 8/8)
live 走查 全端到端实测于本机(ES 9200 + coco 9003),抓出并修复 4 个真缺陷  ✅ b28ec482(2026-09-30,CI 8/8);tsc 棘轮 601→598 ✅ 14c99500
W6 补账  SearchLog 每日保留清理(默认 30 天,SEARCH_LOG_RETENTION_DAYS/INTERVAL 可调可关) ✅ af5d79e3
```

### Live 走查记录(2026-09-30,b28ec482)——挂账清零

本机 Cisco 过滤器当日未拦(此前两天连 Developer ID 签名都拦),coco+ES 全链起后完成全部新面实测:

- **全绿项**:引擎能力三层规划判决/引擎 AI 期望-实际回显/查重报告/实验室四路(text/semantic skipped 带原因/wiki/graph 带 note)/rerank 降级说明/运营概览/索引体检 6/6/编译规则 PUT-GET 往返/公司地图(模板自动播种)/纠正提议(幂等计数)/指纹落库+两路检索折叠/**图谱路全链**(查询"payment service"→识别种子→depends_on 遍历→邻居实体页 rank1→graph+wiki 双路融合第一)/6 个新 MCP 工具全部列出(共 44)/零命中≥3 次→knowledge_gap 提议自动生成+低召回榜标记
- **抓出并修复 4 缺陷**(全部 live 验证后收绿):
  1. wiki/图谱路 total 用 NewGeneralTotal(util.MapStr 命名类型),GetTotal() 类型开关只认裸 map → 路计数对融合不可见(total=0 但命中在场)——D2/D3 潜伏缺陷
  2. queryWithRRF 融合 total 同病 → 零命中遥测全记 -1
  3. Warning 头只在 else 分支设,hybrid_rrf 主路径丢 rerank/语义说明
  4. 遥测 goroutine 用 req.Context()(响应即取消)→ 日志写入竞态、缺口检查必败、提议从未生成;改 context.WithoutCancel
  5. (另)折叠漏挂 /document/_search(API 检索路)
- **教训**:CI 全绿 ≠ 能跑。util.MapStr 与裸 map 在 JSON 序列化上不可区分、在类型开关上是两个世界——凡被 GetTotal() 消费的 total 一律裸 map;异步 goroutine 一律 WithoutCancel。已 live 验证的环境:ES=~/es-d3(bundle 2963,http 9200),coco=~/coco-it-9003(admin@coco.test/CocoDev_2026!),均已留驻可复用
- **自查**:本会话提交零触碰红线;新端点权限全部落位(ops 权限/登录+文章检索权/纠错仅登录即可报);纠正并发首报存在双提议窗口(评审合并即可,良性)

### 收尾批落地记录:聊天纠正 + 索引体检 UI + 地图模板 + D1.5 指纹落库(2026-09-30,d4e2146c)

- **聊天纠正按钮(D7 UI 闭环)**:ui-search 新增 onCorrectAnswer 宿主钩子(与 onSaveToWiki 同构,Fullscreen→Chat→ChatContent→MessageActions 全链 + 双语标签);助手消息旗标按钮 → 宿主纠正弹窗(问题类型五选 + 纠正文本)→ POST /wiki/governance/_correction,人工闸门不变
- **索引体检 UI(P4 补面)**:运营页新增体检卡(存储/状态/文档数,unavailable 悬浮显示原因)
- **地图页模板(D6 补面)**:crud PrepareCreate 对 page_type=map 空正文播种导航骨架(做什么/优先级/来源优先级/导航/规则)——API 与 MCP 建的地图页同样得到;作者自带内容绝不覆盖
- **D1.5 指纹落库**:Document 增 content_hash/content_simhash;orm 全局 pre-hook(create+update)在**所有写入路径**(document API/datasource API/管道/MCP)统一盖指纹——每次写入重算(同内容幂等,内容变了绝不留旧指纹;空/过短跳过);检索结果页内同 hash 折叠到最高位命中,代表位携带 fingerprint_duplicates{count,ids},遗留文档与伪命中直通——深清理仍在查重报告人工裁决
- 单测:指纹幂等/归一化稳定/短文本拒绝、钩子对非 Document 直通、折叠(计数/IDs/无指纹直通/空入参)、地图模板三情形
- 事故一枚:core/document.go 字段标签没 gofmt,CI format_check 拦下,1545f7ae 修正——提交前 `gofmt -l` 应进个人肌肉记忆
- WORKPLAN 计划内工作项至此全部落地;余项:live 走查(等 Cisco 放行)、索引体检卡接入 alerting、检索折叠的全局化(现页内折叠,ES collapse 可为后续优化)

### 第 5 步落地记录:P3 工具族 + D6 编译规则 + P4 索引体检(2026-09-29,3aa6acc5)

- **P3 MCP 工具族**(框架把任意路由暴露成工具,api.MCPTool 选项,权限照走):get_document(引用链终点:命中与 wiki 页携带 doc_id,agent 可读全文)/search_documents(BM25)/list_article_versions(知识演化史)/report_correction(D7 回路开放给外部 agent,人工闸门不变)/get_company_map(agent 入口仪式)
- **D6 编译规则**:WikiCompileRules 进本体 schema(core 类型,wiki 编辑器与 document 闸门读同一份):concept_min_sources(检索闸门从硬编码 2 改为读规则,tenant 级 60s 缓存,绝不逐命中查库)/conflict_strategy(mark|isolate)/source_priority;本体页新增"知识编译规则"卡片(源数下限/冲突策略/来源优先级 tags 输入,中英);仅规则的 schema 文档不再被丢弃(此前 entity_types 为空即判无 schema)
- **来源优先级裁决**:duplicate 提议落卷时解析双方引用文档→datasource,按优先级给推荐方向(evidence.priority_hint,"prefer X — cites higher-priority source Y");同源/未配置不给建议——推荐,不替人决定
- **公司地图页**:page_type=map(建文表单已加选项);GET /wiki/company-map 返回最新已发布地图(kb 可选过滤,权限复用 wiki 文章检索);地图页为人工维护,不受概念页源数下限约束
- **P4 索引体检**:GET /search/ops/index-health——六个中枢库(documents/wiki/entities/versions/governance/search_logs)各一次 size-0 计数,缺失报 unavailable 不失败;端点先行,UI 下批挂运营页
- 单测:规则归一化/优先级排序/闸门参数化、schema 规则往返与钳制、优先级提示三情形、公司地图四情形(登录/发布胜草稿/无权 403/按 kb 隔离);测试基建:新测试文件自备独立 sqlite 库(共享库按最后注册绑路由,文件排序会互相污染——踩坑两次,已隔离)
- 未做:聊天 UI 纠正按钮(端点已就绪)、index-health 的 UI 卡片、地图页模板预填

### 第 4 步落地记录:D4 rerank + P2 运营概览 + D5 缺口闭环 + D7 纠正回路(2026-09-29)

- **D4 rerank**:新模型类型 rerank(默认模型设置第四个选择器,ModelSelect 按 model.type=rerank 过滤)+ Jina/Cohere/vLLM 通用的 `/rerank` 客户端(仅 openai 兼容 API 类型,其余类型带原因关闭);生产挂在 RRF 融合之后(候选窗 50,标题+摘要为文档文本,10s 超时),未配置/调用失败降级纯 RRF,状态走 Warning 头(rerank applied/degraded);实验室新增 rerank 对照卡(RRF 位次 vs 重排名次+Δ+相关度,纯 RRF 卡保持基线不动);单测抓出一个真 bug:窗口内未打分命中会丢失(现按 RRF 序垫后,零丢失)
- **P2 检索留痕+运营概览**:SearchLog(query 归一化小写折叠/策略/总数/耗时/零命中,异步写,失败只记日志不打挂检索);GET /search/ops/overview(新权限 search/ops):检索次数/零命中率/平均与最大耗时/策略分布/低召回榜(按未命中次数排序,top 20,已提交缺口的查询带标记);新页面 /search-ops(菜单"检索运营")
- **D5 缺口闭环**:同一查询零命中 ≥3 次(单次是笔误,三次是缺口)自动提交 knowledge_gap 治理提议(ArticleID=归一化查询哈希,复用扫描器的 ArticleID|Type 幂等键;开放中提议只刷新计数)——低召回榜(系统发现缺什么)与纠正提议(用户告诉你错在哪)汇入同一治理队列,J.B. 篇"每一次犹豫和纠正都在告诉你大脑缺什么"的双通道
- **D7 纠正回路(v1)**:POST /wiki/governance/_correction(登录即可报,带 query/被纠正答案摘录/引用文档/路由提示 fact_missing|fact_outdated|preference|technique|source_conflict/评论);幂等锚=消息 ID(或查询)+评论哈希——同纠正重复提交只加计数,同答案不同纠正各自成提议(第一份证据不被覆盖);治理队列新增 knowledge_gap/correction 两类标签与跳转(无文章锚,落 KB 列表);聊天 UI 反馈按钮与 MCP report_correction 留给 P3 批次
- 单测:rerank 六情形(重排/平局保序/未打分垫后/非法索引忽略/降级/窗口上限)、聚合四情形(空/全量/板排序与截断/归一化)、纠正三组(校验/创建/幂等)
- 未做:SearchLog 保留周期(暂无清理,W6 观察);rerank 相关度阈值(全部按序返回);纠正的裁决路由执行(裁决后落位动作仍人工在文章/规则面操作)

### D3 落地记录(2026-09-29)

- **图路径(graph_route.go,零 LLM)**:查询文本按"子串包含"识别实体(名字+别名,lowercase 匹配;单字符词不参与防误触发;一词多匹配取最长——更具体的实体锚定遍历;上限 5 种子);仅 reviewed/published 实体参与(与 wiki 噪声闸门同哲学:proposed=草稿层不进检索)
- **双向一跳遍历**:正关系(种子 Relations 展开)+反关系(同一次有界扫描内过滤指向种子的边,零额外查询——关系内联未索引,反关系本来就要扫);邻居按"路径数"排序(互指边=2 路径);上限 50 邻居;扫描池外邻居按 ID 分块补拉(200/批 terms 安全批)
- **命中落位**:种子实体页优先(最具体优先),邻居页按路径数次之;复用 wikiArticleToHit 包装(伪文档+溯源+直达 URL)+图路径元数据(graph_route/graph_seeds/graph_via 含关系名与引出实体);文章侧仍过 passesWikiNoiseGate(发布/置信度/概念页源数);权限复用 coco#wiki/article/search,无权跳路不影响其余路
- **有界性**:词表扫描上限 1000 实体(Include 限定字段:仅 id/name/aliases/type/status/article_id/relations),与 wiki 图谱画布同一权衡;文章按 window 截断
- **生产/实验室接线**:hybrid_rrf 第四路 text/semantic/wiki/graph;新参数 graph_weight(权重语义修正:显式传 0=静音该路,此前 URL 参数传 0 无法与缺省区分);实验室 routes 数组自动多出 graph 路,note 显示识别到的种子(或"no entity recognized")
- 单测:识别(包含/别名/状态闸门/单字符拒绝/最长优先/上限)、遍历(正/反/互指路径计数/自环排除/种子互不为邻)、包装(graph_route 标记/via 边/缺邻居名不 panic)、四路融合数学(graph 权重 2 的贡献 2/(k+rank))、静音路零贡献仍出文档
- 提交 ed3b3912,CI 8/8(integration_test 因下载服务器不稳折腾三轮后以 curlrc 断点续传加固收绿,见 W5.6)
- 未做:两跳遍历、拼音/形态变体识别(词表包含是精确匹配)、概念页按 linked_pages 反查(该字段 enabled:false 未索引)、实体卡直出(检索空间是文档,实体以页面呈现)——待图谱规模化(W7)一并考虑

### D2 落地记录(2026-09-29)

- **多路 RRF 泛化(rrf.go 重写)**:RRFConfig{K, Weights map[route]weight},`rrfFuseMulti(routes)` 统一数学(Σ weight/(k+rank));breakdown 泛化为 ranks/route_scores/contributions 三张 map;平局规则泛化:分数→召回路数→最佳排名→ID。加一路 = 一个名字+一个权重,零数学改动
- **wiki 路(wiki_route.go)**:BM25 打 wiki_article(title^20/pinyin/summary/tags/combined_fulltext),仅 status=published 且 confidence≠low;**噪声闸门**:概念页需 ≥2 独立来源才进检索(治理队列同规则的读侧);命中包装为伪文档(Source=wiki)并携带溯源(wiki_sources 含 doc_id/title/locator/excerpt 进 Metadata)+ 文章直达 URL;权限复用 coco#wiki/article/search(无权则该路跳过,不影响其余路);search 主链路对 wiki 命中跳过文档级 Refine,防 URL 覆盖
- **生产 hybrid_rrf 三路**:text/semantic(能力规划)/wiki 并行,单路失败降级融合;新查询参数 wiki_weight(与 text_weight/semantic_weight 同型,0=静音该路)
- **实验室泛化**:响应改为 routes 数组(name/took/total/hits/route/note/error)+ 通用 fused breakdown;前端动态渲染任意路(每路一张表,融合表按路由出排名列与贡献堆叠条),wiki 权重滑杆与 engine/client/skipped 状态徽标
- 单测:三路融合数学(含三路全平局)、权重翻转、静音路、空路、噪声闸门六情形、wiki 命中转换(溯源/URL/标志)
- 未做:chat 引用链消费 wiki_sources(等 P3 MCP 工具族一起);per-KB 精细 ACL(v1 语义:published=组织内可搜)

### D1 落地记录(2026-09-29)

- **指纹(fingerprint.go,纯函数)**:NFKC 归一化(全半角折叠、大小写、空白)→ sha256(精确)+ simhash64(unicode 双字,中英通吃);海明 ≤3 近同、4-8 高相似;**两段式验证**:simhash 只做候选提名,bigram Jaccard ≥0.5 才认边(实测短文本无关对海明可低至 7,无验证必误报)
- **扫描(dedup.go)**:分页拉取(默认上限 5000,可到 5 万)→ 内存指纹(存量文档零重索引即刻可查)→ 四档边(精确 hash/近同/高相似/同题异文疑似版本)→ union-find 成簇(同文变体自动合并)→ 组内推荐保留人(最新更新优先);候选生成用**直接两两海明**而非块桶(鸽巢只保证半径 ≤3,块桶会漏 4-8 档;5 千文档 1250 万次海明约秒级)
- **评审闭环**:分组带四档标签+相似度+证据(标题/来源/大小/时间);动作三选——保留但排除检索(disabled,可逆)/删除(二次确认+独立删除权限校验)/标记非重复(DocumentDedupDismissal 持久化,永不再报);**系统只推荐,绝不自动删**;报告 60 秒缓存,动作后失效重扫
- 端点:GET /document/dedup/report(可 refresh)、POST dismiss、POST action;设置页新增"内容查重" tab(统计卡+分组表+动作,中英)
- **未做(留 D1.5)**:指纹落库(Document 加 content_hash/content_simhash 字段+管道 fingerprint 节)、检索结果按 hash 折叠("另有 N 个副本")——需要写入路径配合,待查重评审链验证后再上

### P0/P1 落地记录(2026-09-29,8e2a1045)

对本机 Easysearch 2.4.0 的穷举探测(探针索引+报错预言机)结论:
- `semantic` 查询存在,参数仅 query_text/image_text/candidates/query_strategy/boost,**无 query_vector、无逐查询服务选择**;未配引擎侧 embedding 服务时 500(embeddingRequest null)
- `knn` 查询对所有字段/所有形态(嵌套对象/裸数组/顶层节)一律 "does not support"——**本版本不可用**
- script_score 的 cosineSimilarity 函数缺失、knn_dense_float_vector 的 doc values 对 painless 不可访问
- `/_inference` 无服务管理路由;引擎配置文件无 embedding 段

因此 P1 的"Coco 端向量化"落地为**客户端语义重排**:BM25 召回(限已向量化文档)→ 按取回向量与查询向量(Coco 默认 embedding 模型)余弦重排。三层规划:engine semantic(引擎配好服务时自动启用,60s 探针缓存)→ client rerank → keyword 降级(Warning 头注明)。新端点 GET /search/engine-capability;检索实验室每路报 engine/client/skipped。单测覆盖分类/规划/余弦/重排/分页。**CI 8/8 全绿(含 integration_test)。**

### 重大修正:Easysearch 的 processor 体系(2026-09-29,medcl 指出后实测定案)

上面"引擎侧只剩一条路"的结论**不完整**——引擎的 embedding 与 RRF 以 **processor 体系**交付(ai 插件,plugins/ai/ai-2.4.0.jar),已在 9200 全链路实测打通:

| 层 | processor | 关键配置 |
|---|---|---|
| ingest | `text_embedding` | text_field/vector_field/vendor/api_key(支持掩码)/url/model_id/dims/batch_size/ignore_missing/ignore_failure;引擎写入时调 OpenAI 兼容服务向量化(实测向量已入库) |
| search request | `semantic_query_enricher` | default_model_id/vendor/api_key/url/vector_field_model_id(按字段选模型);把 query_text 嵌入成向量注入 semantic 查询——**之前缺的"embedding 服务"就是这个** |
| search results | `hybrid_ranker_processor` | combination.technique=rrf + parameters.rank_constant(归一化硬编码 rrf);对 hybrid 查询的各子路结果做 RRF 融合 |

search pipeline 顶层键为 Easysearch 自有 schema:`description/version/rewrite_processors/enrich_processors/rerank_processors`(非 ES 的 request/response_processors)。用法:`?search_pipeline=<name>`。裸 hybrid 查询不配 ranker 会返回未融合的原始堆积分数——**hybrid 必须与 ranker 成对出现**。RRF 数学实测吻合(1/(60+1)+1/(60+2)=0.0325≈实测 0.0328)。

**Coco 侧后续(P1.5)——✅ 已落地(2026-09-29,63f08caa,CI 8/8)**:
1. **管道策展**:设置新增"引擎 AI"段(enabled/embedding 模型/批量/RRF 常数/字段覆盖),保存即后台下发;`GET /search/engine-ai` 回显期望 vs 已部署(api_key 掩码)+逐管道漂移标记,`POST /search/engine-ai/sync` 手动同步;管道固定名 `coco-embedding`/`coco-semantic-rrf`,文档索引自动设 `index.default_pipeline`,模型回落默认 embedding 模型——一处配置,写入与查询两侧生效
2. **探针修正**:金丝雀带 `?search_pipeline=` 重试,"pipeline is not defined" 归类为确定性不可用(提示同步引擎 AI 设置)——启用未同步的配置自动降级客户端重排/keyword,不会打挂检索
3. 客户端重排保留为兜底层;后续(D4 时)检索实验室可并排展示引擎 RRF 与客户端 RRF 的分数分解
- 设置页"引擎 AI" tab:模型选择/高级参数/同步按钮/期望-已部署对照面板(中英)
- 待环境恢复(Cisco AnyConnect 过滤器仍在拦本机 coco 实例):live 走查设置页 + 端到端语义/混合检索

每步完成后:web 构建 → vfs → go build → 重启 → curl/浏览器实测 → 提交推送 → 盯 CI。

**演示数据重播种(2026-10-04)**:`~/coco-it-9003/seed-demo.sh`(实例工具,不进 PR)——幂等重建走查数据:演示 KB、四个 product 实体(支付服务/订单服务/收银台页面/支付网关,reviewed 态)+ depends_on 关系链、三篇 published 页(实体页/概念页/公司地图 map)、一条 D7 纠正提议。终态验证:图谱 9 节点/10 边;**图谱路复活**("支付服务挂了影响什么"→ seeds: 支付服务,hits: 1);company-map found:true(模板自动播种);纠正提议幂等锚生效。写脚本踩的坑:crud create 路径带尾斜杠(/wiki/kb/、/wiki/entity/、/wiki/article/),_search 的 filter 是查询参数非 body;实体类型过本体 schema 校验(演示 schema 已声明 product.depends_on→product,service/page/vendor 会被拒);macOS bash 3.2 无关联数组;发布后即查有 ES 刷新延迟(验证前 sleep)。文中 [[service:xx]] 前缀与 product 实体类型不匹配的 wikilink 会留 unresolved 节点——正好演示 O2 断链修复入口,不修。

## 七、风险与依赖

| 风险 | 影响 | 对策 |
|---|---|---|
| REPO_PAT 过期 | CI 停 7/8 | W5.3 换新后重跑 |
| LLM 密钥未配 | 语义路 500(已降级)、实体抽取/AI 摘要/治理确认不可用 | D1/D3(实体识别)设计为零 LLM 依赖先行 |
| simhash 对短文本不稳 | 查重误报/漏报 | 阈值分档(短文本只用 hash+文件名启发);标记非重复持久化纠错 |
| 查重误删 | 用户信任受损 | 绝不自动删,只推荐;回收站可恢复;动作全部留审计 |
| LLM 编译 token 成本 | 入库变贵 | 成本前移的前提是查询/摄入比高;管道级开关+按 KB 配置编译深度 |
| O4 后文章渲染管线再动 | 回归风险 | D2 检索侧改动不碰渲染;wiki 路走独立查询构造 |

**走查固化为集成测试(2026-10-04,f082d4a8+94559f6c)**:三轮 live 走查抓的 4 个 bug 全在 CI 盲区(全栈接线,单测覆盖不到),把走查路径写成 `tests/knowledge-hub/scenario1.dsl`——CI 的 integration_test 自动发现并执行,每文件独立干净栈。场景锁:simhash 回归(年假/年终双薪原文档 200)、本体 schema(product.depends_on)、KB+实体+依赖链、公司地图、三篇 published 页(内容页延后发布)、评估集 4 用例(精确锁 total_cases=4/top4_hits=2/top4_rate=0.5/mrr=0.5)、缺口回流 2 条+重跑幂等、纠正幂等(report_count 1→2、总量 1)、改写降级 applied:false、索引体检 healthy:8。**写场景又抓一真 bug(#4):评估缺口回流与纠正提议写库不等刷新**——提案落库了但操作员下一跳(打开治理队列)读到空,与 #3 同类;两处补 WaitForRefresh(f082d4a8)。
- **确定性设计的关键认知**(试错一下午换来,前端调参/排障同样适用):①text 路对文档按 `source.id` 数据源硬过滤,无数据源文档进不了 eval 的 text 路;②wiki 路只要查询非空就匹配(should-only),**任何查询都锁不死 wiki 路零召回**——零召回只能靠"当时没有过噪音闸门的已发布页"达成(map 页不过闸);③新建数据源必须 `enabled:true`,否则被禁用过滤器排除;④crud 数据源只收 connector 型——借内置 connector(boot 自播,milvus 等)建新的空数据源,admin 所有,范围内只有场景自己的期望文档,排名恒 1-2;⑤~~eval/studio 的合成请求与生产 /document/_search 行为有分歧~~ **已排查定案,见下条**
- loadgen 新 DSL 解析器(CI 装最新)的坑:断言空元组 `(200, )`/`(200, {})` 都不行(生成非法 condition,报"missing or invalid condition"且容易误读成下一个请求的错);注释必须 `#//` 裸 `# 文字` 会被当指令;request 块后要空 `#` 行;`hits.hits.0._id` 数组下标 **register 可用**(assert 不行)
- 本地复刻 CI 栈:同 bundle 再起一个 Easysearch(ES_PATH_CONF 指独立 config,security off,9201)+ coco 9004(注意演示实例的 config/generated.go 是烘焙配置会覆盖 -config,删掉;badger 数据目录被删时 SIGTERM 会挂死,kill -9 后清数据重启)。本地两轮干净栈全绿(35 请求 0 断言失败)后提交

**"eval/studio 与生产检索分歧"排查定案(2026-10-04,0f8f8946)**:上一条⑤标记的"分歧"追到底后**不是管线缺口,是对照基准选错了**——`/document/_search`(document.go searchDocs)是文档管理页的 BM25 列表检索(NewQueryBuilderFromRequest 三默认字段 + created desc,不看 search_type、不记 search log),而 `/query/_search`(search.go search)才是 app/widget/MCP 的用户检索端点;eval 的 executeEvalCase 直调 queryWithRRF,与 `/query/_search?search_type=hybrid_rrf` 的 text 腿**逐子节同构**(慢日志实证:12 子句 BuildFuzzinessQueryClauses 形状一致;此前看到的 3 子句 should 形状就是 searchDocs 的)。排查路上顺手修了三个真缺陷(同笔提交):①scenario1.dsl 的"hybrid 播种"一直打错端点(/document/_search),从未真正跑过融合管线、也从未播过 search log——改到 /query/_search 后实测 coco_searchlog 真的有播种日志了;②search.go fuzziness 解析条件写反(err != nil,4732b292d 引入):合法 ?fuzziness=N 被静默忽略、非法值反而置 0——实测修后 fuzziness=1 出 4 子句形状、fuzziness=abc 回落默认 3(与无参逐字节一致);③/document/_search 与 /query/_search 两个 MCP 工具都叫 search_documents,mcp-go 注册表按名覆盖(s.tools[name]=entry),后者静默遮蔽前者且权限元数据错位——CRUD 端改名 search_documents_bm25。**遗留产品决策(已按推荐方案落地,见下条"默认搜索模式接线")**:`/query/_search` 的 search_type 默认 keyword,而 web 搜索页只透传 URL 里的 search_type、widget 完全不发、SearchSettings 也只有 {enabled,integration}——**生产默认流量全走 keyword 腿,D4-D8 建的 wiki/graph/rewrite 融合管线目前只有 eval/studio 和手工 URL 触达**。评估数字测的是"目标管线"而非"当前默认流量";若要让用户实际吃到,需决策:SearchSettings 加默认 search_type(推荐,可灰度)/前端显式发 hybrid_rrf/暂缓保持 eval-only。本地验证配方:9201 开 `index.search.slowlog.threshold.query.info:0s` 抓全量查询 source,直查比对形状。

**默认搜索模式接线(2026-10-04,5fba0886)**:上述产品决策点落地为**可灰度开关**——search_settings 增 `search_type`(keyword/semantic/hybrid/hybrid_rrf,保存时校验非法 400),`/query/_search` 解析顺序:显式 ?search_type= > 配置默认 > keyword;**缺省/旧存档一律 keyword,零行为变化**,想全量切 hybrid_rrf 时改一处设置即可。suggest(下拉联想)刻意留 keyword(逐键融合太重,已注释)。设置页加选择器(中英)、MCP search_documents schema 补 search_type 参数说明、单测锁 DefaultType 回落矩阵(nil/空/陈旧值→keyword)。scenario1.dsl 尾部锁全链:香蕉 400 → 设置 PUT/GET 回显 → 无 search_type 参数的 /query/_search 打 no-such-datasource(空域 text 腿 0、wiki 腿不受限出全部已发布页)断言 total:3——**keyword 默认下同请求 total:0,构成融合生效证明**(本地两模式都实测)。**过程中的确定性认知**:①eval 零召回用例的"text 腿 0"靠的是数据源限定,不是查询串本身——**pinyin 分析器按字母粒度切分 + operator OR,任何非空查询都能在 text 腿模糊命中中文语料**("xqzwjvqz" 全语料命中 24 篇、2 篇小语料全中),这也是当初"宽松匹配 47-57"之谜的机制;②wiki 腿逐命中打分有时序:concept 页的闸门输入发布后需数秒落定,断言前加 4s 等待(与 company-map 断言同法);③loadgen assert 仍不支持数组下标(只有 register 支持),hits.0._index 断言不可用;④本地 it-reset 期间不要用带 -m 超时的 POST 探测 setup 完成度——中途超时会让 replay 半途而废,索引半建,下一次 setup 必撞 already-exists(等脚本的 RESET_DONE 标志即可)。

**pinyin 宽松匹配调查与评估网扩展(2026-10-04,72fca84c)**:session 内发现的"任何非空查询都能命中"之谜拆到分析器级——`pinyin_analyzer` 对拉丁输入产出**单字母 token**("xqzwjvqz"→x/xqzwjvqz/q/z/w/j/v/q/z),对 CJK 产出 字母+缩写+音节("支付服务"→z/zhi/zffw/f/fu/w/wu),match 子句 operator OR → 垃圾串靠字母撞库("xqzwjvqz" 全语料 24/44);而 "zf"→z/zf/f、"zhifu"→音节 是**刻意设计的缩写/全拼检索**。评估过 obvious fix:text 腿 should 加 minimum_should_match=2——ES 实测杀掉垃圾(xqzwjvqz 2→0)、中文探针不受影响,但 **"zf"/"zhifu" 也归零**(它们只靠 title.pinyin 单子句命中)——否决;只留 prefix 会杀 "fuwu" 类非前缀音节输入,同样否决。**结论:宽松度收紧是分析器/索引模板级调优,需先扩评估集护航,单独立项**。本提交把拼音能力锁进黄金集:zhifu/zf 两用例(限定场景数据源)实测均 rank 1,总量锁 6 cases/top4 4/rate=MRR=2/3,零召回用例照旧回流 2 gap;未来任何破坏拼音输入的"收紧"会在 CI 当场爆,且 run_search_eval 一键出前后对比。

**拼音黄金用例的平分坑(CI 抓到、本地抓不到,72fca84c 后续修复)**:"zf" 对两篇支付标题近平分,谁第一是**环境级平分决断**——本地 manual 第一(mrr 2/3),CI trouble 第一(mrr 7/12 断言炸)。修法=拼音用例与 CJK 探针同构挂**双期望**(first-expected 恒 rank 1,平分决断无关;真回归=两篇都不进窗,断言照样炸)。教训入册:**期望集必须覆盖平分集合,单文档期望只在分数悬殊时可用**。另:CI 日志见引擎侧 `SemanticQueryBuilder.doRewrite` NPE(EmbeddingRequest.dimensions null,path /coco_document-v2/_search,semantic 腿降级时被抑制的警告,coco 降级路兜住、测试未受影响)——**引擎侧健壮性问题,待转告引擎同学**(无模型时该报清晰错误而非 NPE)。

**D5 缺口闭环被 pinyin 宽松性废掉(量化实证,2026-10-04,待分析器决策一并处理)**:缺口回流/低召回榜/零命中率三个运营信号全部锚在 `ZeroHit: total==0`(search_log.go,3 次阈值),而 pinyin 字母效应让 text 腿对任何非空查询必有命中——本地实测:"nonexistentwordxyz" **70/70 全语料命中**,"vvvv" 7、"xyzzy" 15、"12345" 3,唯一归零的是纯 emoji"🤖🚀"(分析器丢弃)。**推论:有机流量下缺口提案永不触发、低召回榜恒空、zero_hit_rate≈0% 失真**——D5 的"系统发现缺什么"半环在生产等于死环(场景/演示里能触发全靠空数据源域或 emoji 类查询)。分数阈值不可行(BM25 在该语料上真实命中也只有 ~0.016,与字母巧合无法区分)。**根治=分析器级收紧(见"pinyin 宽松匹配调查"条)**;在那之前 D5 的三个信号只能当"演示性指标"理解,不建议在 UI 上强调。转告 medcl:分析器决策现在连带三个运营特性。

**三路代码审查与修复批次(2026-10-04,46cb21e9+179d39d5+5422a448)**:检修指令下三路并行审查(检索路径/数据存储层/wiki 治理)共 **30 项发现**,逐项亲自验证后分三级处理。

**本批已修(16 项,三笔提交,本地干净栈 scenario 0 断言失败+全模块单测绿)**:
- *安全与正确性(46cb21e9)*:①wiki create 接受客户端 status(D1 违规实锤:PrepareCreate 校验但保留传入值,一个 POST/MCP create 绕过评审闸门)→ 强制 draft;②指纹 orm 钩子漏 OpSave(框架 orm.go:697 Save→OpSave,PUT /document/:id 全部漏盖章)→ 三操作齐挂 + 短内容清旧指纹(原来 !ok 早退保留上次写的 hash,缩写后无关文档永远折叠);③dedup action ctx 无 direct 读写(非管理员审查者对跨数据源文档必失败);④dedup report 挂 document:read(人人皆有→跨数据源枚举泄露,DirectReadAccess 扫描无视共享边界)→ 与 dismiss/action 同挂 document:update(独立 document_dedup 权限需 UI/角色接线,留产品决策);⑤batchDeleteDoc 后显式 refresh(框架 DeleteByQuery 不转发 refresh,实测 UI 立即重查仍见已删行)。
- *检索路径健壮性(179d39d5)*:⑥search 处理器 6 处 panic(err==nil 检查写反那次留下的地雷,hybrid_rrf 成为可配置默认后引擎抖动=裸 500)→ builder 错 400/引擎错 500;⑦检索页 size 无上界 → maxSearchPageSize=100(keyword builder/semantic 分支/RRF 窗口/assistant 填充四处);⑧hybrid_rrf 融合 total=各路 max 但 hits 是并集 → total 以 len(fused) 为下限;⑨改写熔断把父 ctx 取消(客户端挂断)计为失败 → 三次弃搜开闸静音全员,豁免 errors.Is(Canceled)&&父 ctx 活着;⑩graph 腿词表扫描打满 1000 上限时静默漏识 → note 明说;⑪isDefinitiveCapability 漏 "pipeline is not defined" 串(每搜重探)→ 三个 definitive 串收敛为共享常量(漂移正是这次 bug 的病根)。
- *一致性与诚实性(5422a448)*:⑫search log 写入与 gap 提案写库等 refresh(gap 检查读自己刚写的日志,不刷新必少数漏阈值——与 f082d4a8 修的 #4 同类);⑬wiki 五处 WaitForRefresh(toc 保存/governance 状态迁移/文章发布/版本快照/AI 编辑——发布可见性、连续 toc 变更、快速双存(版本号重复)都依赖读刚落地的写);⑭generate worker pool:results 无缓冲,SSE emit 失败早退 → 全池 worker 永久阻塞泄漏 → 改排干后退出;⑮eval 列表 500 vs run 只评估最老 200 → 列表钳到 200,run 多取 1 条探测溢出并在响应报 evaluated/truncated;⑯duplicatePriorityHint 只读 tenant schema → loadOntologySchemaForKB(kb.ID)(KB 自有 source_priority 被 tenant 规则否决),测试签名同步。

**留档 medcl 决策(11 项,不盲动)**:⑴TOCTOU 幂等——纠正/实体提案"查重再创建"两步间无锁,并发双写可重复。**已修(2026-10-05,锚定 ID)**:ID 即锁——core/anchor.go 纯函数 `AnchoredProposalID(anchor,type,generation)`=md5("proposal|type|anchor[|#N]")、`AnchoredEntityID(name)`=md5("entity|name 归一化(trim+lower)");并发双写者落在同一 ID,输家覆盖字节相同内容而非产重复行(orm.Create 尊重预设 ID,空才生成 uuid)。**代数后缀**是关键决策:同锚点重开(纠正 resolve 后再报/扫描器重探)吃 #2/#3 新行,resolved 行保留为审计历史——单纯锚定会覆盖掉历史。覆盖四条写路径:createCorrection、fileProposal(治理扫描器)、maybeFileKnowledgeGap(缺口回流)、FileEntityGovernanceProposal(实体冲突);实体抽取 resolveOrCreateEntity 创建前 SetID(AnchoredEntityID)——**只锚管线创建,手动建实体不锚**(保持生成 ID 与自己的评审流,防同名异义手动实体被管线覆盖,输家覆盖只损 provenance 且下轮抽取会补回 sources)。proposalAnchorState 助手按扫描而非排序位置找 open 行(同 tick 创建的行排序并列,hits[0] 不稳,fold-or-refile 判断不能赌店端返回顺序);knowledge-gap 的语义刻意保守:已有任何行(含 dismissed)即不重报,防止被驳回的缺口每次搜索重新刷屏。测试:core/anchor_test.go 纯函数(确定性/代数区分/类型区分/实体名归一化)+ TestCorrectionReopenAfterResolve handler 级(gen1 锚定 ID→resolve→重报产 gen2 新行且 resolved 行保留→第三次并入 open 行 report_count 递增仍 2 行);⑵**connector 摄取绕过指纹钩子(已实证,2026-10-04 深挖)**:链路=connector 插件 BatchCollect→queue.Push(indexing_documents)→pipeline process_documents→queue(documents_processed)→pipeline merge_documents 的框架处理器 indexing_merge(直连 elasticsearch "prod" 写 coco_document-v2,key_field id)→queue(merged_documents)→pipeline ingest_documents 的 bulk_indexing——**全程不过 orm,指纹钩子永远不触发**;datasource API 三处 orm.Create/Update 则正常盖章。后果:connector 来源文档无 content_hash/simhash→检索折叠不生效、查重报告扫不到。**修复已落地(2026-10-04,源侧盖章)**:比新处理器插件更优的落点——所有 connector 都经 plugins/connectors/common 的 BatchCollect 入队,webhook 亦直推同一队列,故在**入队前就地盖章**(纯代码修复,零配置变更,覆盖全部 connector+webhook);指纹数学原样迁到 modules/common/fingerprint 共享包(纯函数,无依赖),modules/document 的钩子与 document 侧函数全部委托同一实现——两前端永远产出相同值。**本地 E2E 三重证明**:①重置栈 Hugo 摄取后 70/70 文档 content_hash 全有(修复前 0/70);②API 路径(orm 钩子)与 connector 路径(源侧盖章)对同一内容产出逐字节相同的 hash(622f8cde...);③检索折叠生效:同内容 API 副本被折进原件,metadata.fingerprint_duplicates 带 count+ids。**遗留:存量回填**(2026-10-04 前摄取的文档仍无指纹,需一次性 scroll+计算+bulk 脚本,待 medcl 定时机;只读统计口径已交:对生产 ES `GET coco_document-v2/_count {"query":{"bool":{"must_not":{"exists":{"field":"content_hash"}}}}}` 先看规模再定跑法与时机);另注意 indexing_merge 按字段合并,短内容时显式写空串/0 以冲掉旧值(已按此实现)。**活数据复核**:本地隔离栈 Hugo 连接器摄取后,coco_document-v2 里 70/70 文档(两个 connector 数据源:54+16)content_hash 全空——绕过端到端坐实,非理论推演。fingerprint_persist.go 头注释已同步改口;⑶引擎探测在全局锁下串行化(singleflight 可改观,非正确性);⑷loadDedupDismissals Size(10000) 无排序无溢出信号(超过即静默失效)——**已做最小诚实修(2026-10-05)**:按 created DESC 排序(咬到上限时丢最旧)+ 命中 dedupDismissalCap 打 warn 日志(被驳回对重现会被误读成 dedup bug,实为加载上限);规模化正解=查询期过滤(不进内存 map),与 ⑸ 合并成"dedup 扫描规模化"一个事项,等真上大库再排;⑸dedup 扫描 from/max ≤50000 逼近 index.max_result_window(大库需 search_after);⑹⑦**已修(2026-10-05,28c43a4a,含同族通知)**:共同根因=DirectWrite 写入无主 + orm 搜索钩子对非管理员强制 owner_id should 子句 → 恰好对该看见的人不可见。修法三分:①治理队列读挂 crud CtxDecorate 的 DirectRead(队列是全租户审查面,可见性由路由 wiki_governance 权限决定——扫描器/缺口回流提案天然无属主);②AI 草稿(aiGenerate 流与 _from_chat)盖请求者为 owner;③wiki 通知盖收件人为 owner(通知不能 DirectRead——那会让持通知读权限者看到所有人的通知,泄露)。**E2E 实证**(本地栈建 wiki_reviewer 角色+非管理员用户):持权限非管理员投稿纠正后队列 total:1(修复前恒 0);无权限用户 403;管理员视角不变;评审员 _from_chat 存的草稿带 owner 落位、自己列表可见;ai-draft 通知 _system.owner_id=收件人。设计注记:knowledge-gap/纠正提案保持无属主(系统件),队列按权限全租户可见是刻意语义;**顺带发现本机磁盘 99%(剩 17Gi)触发 ES flood-stage 只读锁**——测试集群已用绝对空闲量水位(8gb/4gb/2gb)解除,真实清理留 medcl 决策;⑻WikiEntity 数据模型无 kb_id → entityNeighbors 只能解析 tenant schema(per-KB schema 对实体邻域失效,数据模型决策);⑨D5 缺口闭环失活(前条,系于分析器决策)——**演示级标注已落地(2026-10-05,方案 B)**:/search/ops/overview 响应加 signals_caveat 常量字段(handler 设置,aggregateSearchLogs 保持纯函数),web 检索运营页按其存在渲染中英警示横幅(零命中率/低召回榜/缺口回流三信号勿作运营依据);根治(分析器收紧)仍独立立项,评估集护栏已就位(zhifu/zf 黄金用例锁 CI);⑩引擎 SemanticQueryBuilder.doRewrite NPE 转告引擎同学;⑪eval run 取最老 200(ASC)而列表最新优先(DESC)——窗口与方向双不一致,超 200 时两界面看的不是同一批(本批已把不一致显式化,方向统一留产品定)——**已修(2026-10-05,medcl 拍板同向)**:run 改 DESC 取最新 200,与列表同向;溢出探测(evalCaseCap+1 多取 1 条报 truncated)语义不变,只是截的是最旧一批;现存场景断言 4 用例不超窗不受影响。

**误报撤销(1 项,教训入册)**:审查代理称"DBQ body 里 size:2 会让每天静默删零"——Easysearch 实测**忽略体里的 size**(3 条匹配全删),不报错不欠删。验证代理结论必须实证,尤其"静默失败"类断言。
