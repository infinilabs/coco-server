/* Wiki module fixtures — ported (trimmed) from the coco-wiki prototype mock data.
 * Used only while the /wiki/* backend (WS10 knowledge hub) is not wired yet;
 * see src/service/api/wiki.ts (USE_WIKI_MOCK) and flip it off when endpoints land.
 * Api.* types are ambient (src/types/api.d.ts). */

export const wikiMembers: Api.Wiki.Member[] = [
  { id: 'u1', name: 'Medcl', avatar: 'MC', role: 'owner', email: 'medcl@infini.ltd' },
  { id: 'u2', name: '刘佳鑫', avatar: 'LJ', role: 'editor', email: 'liujiaxin@infini.ltd' },
  { id: 'u3', name: '姚季平', avatar: 'YJ', role: 'editor', email: 'yaojiping@infini.ltd' },
  { id: 'u4', name: 'KM Agent', avatar: 'AI', role: 'agent' }
];

export const wikiKbs: Api.Wiki.Kb[] = [
  {
    id: 'kb-coco',
    name: 'Coco AI 产品文档',
    description: 'Coco AI 统一搜索与 AI 助手产品知识库：架构设计、安装部署、数据源配置、LLM 集成、RAG 搜索。',
    icon: '🥥',
    visibility: 'team',
    workspace_id: 'ws-default',
    datasource_ids: ['ds-gitlab', 'ds-github', 'ds-yuque'],
    assistant_id: 'wiki-agent',
    sync_strategy: 'scheduled',
    article_count: 4,
    last_updated: '30 分钟前',
    ai_status: 'ready',
    members: wikiMembers,
    datasources: [
      { id: 'ds-gitlab', type: 'gitlab', name: 'GitLab - coco-server', status: 'connected', last_synced: '1 小时前', document_count: 95 },
      { id: 'ds-github', type: 'github', name: 'GitHub - coco-app', status: 'connected', last_synced: '2 小时前', document_count: 112 },
      { id: 'ds-yuque', type: 'yuque', name: '语雀 - Coco 产品文档', status: 'connected', last_synced: '3 小时前', document_count: 38 }
    ]
  },
  {
    id: 'kb-frontend',
    name: '前端组件库文档',
    description: 'React 组件设计规范、用法示例、API 参考和最佳实践。',
    icon: '🎨',
    visibility: 'team',
    workspace_id: 'ws-default',
    datasource_ids: ['ds-comp', 'ds-design'],
    assistant_id: 'wiki-agent',
    sync_strategy: 'manual',
    article_count: 2,
    last_updated: '2 天前',
    ai_status: 'ready',
    members: wikiMembers.slice(0, 3),
    datasources: [
      { id: 'ds-comp', type: 'github', name: 'GitHub - component-lib', status: 'connected', last_synced: '1 小时前', document_count: 156 },
      { id: 'ds-design', type: 'yuque', name: '语雀 - 前端设计文档', status: 'syncing', last_synced: '10 分钟前', document_count: 23 }
    ]
  }
];

const src = (doc_id: string, source_name: string, title: string, excerpt: string, locator: string, url?: string): Api.Wiki.SourceRef => ({
  doc_id,
  source_type: source_name.split(' ')[0].toLowerCase(),
  source_name,
  title,
  excerpt,
  locator,
  url
});

export const wikiArticles: Api.Wiki.Article[] = [
  {
    id: 'art-rag',
    kb_id: 'kb-coco',
    title: 'RAG 检索增强生成',
    summary: 'Coco AI 混合检索（关键词 + 语义）与引用生成的整体机制。',
    page_type: 'concept',
    subtype: 'method',
    aliases: ['Retrieval Augmented Generation', '检索增强'],
    tags: ['RAG', '搜索', '架构'],
    status: 'published',
    ai_generated: true,
    confidence: 'high',
    entity_id: '',
    created_by: wikiMembers[3],
    contributors: [wikiMembers[3], wikiMembers[1]],
    created_at: '2026-09-18 10:00',
    updated_at: '2026-09-20 15:30',
    sources: [
      src('doc-101', 'GitLab - coco-server', 'modules/document/service.go', 'SemanticQuery/HybridQuery 基于 ORM 查询构建器暴露语义与混合检索。', 'L42-L60', 'https://gitlab.com/infinilabs/coco-server'),
      src('doc-102', '语雀 - Coco 产品文档', '混合检索白皮书', '混合召回将 BM25 得分与 kNN 向量得分归一化后融合排序。', 'p3')
    ],
    content: `## 定义
RAG（Retrieval Augmented Generation）指在 LLM 生成回答前，先从企业知识索引中检索相关片段并注入上下文，使回答有据可依、可引用溯源。

## 关键特征
- 关键词、语义、混合三种检索模式，由数据源与助手配置决定
- 检索结果按 owner/team 权限过滤，实现 permission-aware retrieval
- 引用（citations）随回答流式返回前端，可回链源文档

## 应用场景
- Smart Store 一线问答：活动规则、SOP 检索后作答
- 合同审查：条款对齐与页级引用回链

## Related Concepts
- [[concept:Embedding]]
- [[concept:知识中心]]

## Related Entities
- [[entity:coco-server]]

## Mentions in Source
- 「混合召回将 BM25 得分与 kNN 向量得分归一化后融合排序」 — doc_id:doc-102, locator:p3
- 「SemanticQuery/HybridQuery 基于 ORM 查询构建器暴露语义与混合检索」 — doc_id:doc-101, locator:L42-L60

## 正文内容
Coco 的 RAG 管道在连接器同步阶段完成抽取、摘要、标签与向量化，检索阶段由助手工具 \`internal_search\` 执行权限过滤后的混合查询，引用结构 \`SourceRef\` 携带 \`doc_id\` 与 \`locator\` 保证逐条回链。`
  },
  {
    id: 'art-connector',
    kb_id: 'kb-coco',
    title: '连接器（Connector）体系',
    summary: '25+ 数据源连接器的插件化架构与增量同步机制。',
    page_type: 'concept',
    tags: ['连接器', '数据源', '插件'],
    status: 'reviewed',
    ai_generated: true,
    confidence: 'medium',
    created_by: wikiMembers[3],
    contributors: [wikiMembers[3]],
    created_at: '2026-09-17 09:00',
    updated_at: '2026-09-19 11:20',
    sources: [
      src('doc-201', 'GitLab - coco-server', 'plugins/connectors/', '连接器以 ProcessorPlugin 形态注册，Fetch 钩子负责遍历与入队。', 'README')
    ],
    content: `## 定义
连接器是 Coco 接入外部数据源的插件单元，负责认证、遍历、增量游标与文件下载，产出进入统一摄取管道。

## 关键特征
- 插件注册：pipeline.RegisterProcessorPlugin + ConnectorProcessorBase
- 增量同步：watermark / cursor 游标，支持删除传播
- 摄取管道：file_type_detection → file_extraction → summary → extract_tags → embedding

## Related Concepts
- [[concept:RAG 检索增强生成]]

## Mentions in Source
- 「连接器以 ProcessorPlugin 形态注册，Fetch 钩子负责遍历与入队」 — doc_id:doc-201, locator:README

## 正文内容
数据库类连接器（MySQL/PostgreSQL/Oracle/MSSQL）支持以用户提供的 SQL 视图作为基础查询，仅将审批子集接入索引。`
  },
  {
    id: 'art-store-entity',
    kb_id: 'kb-coco',
    title: '美心门店（示例实体页）',
    summary: '实体页示例：门店实体属性、关联活动与租赁合同。',
    page_type: 'entity',
    subtype: 'store',
    aliases: ['Maxim Store 001'],
    tags: ['门店', '示例'],
    status: 'draft',
    ai_generated: true,
    confidence: 'low',
    created_by: wikiMembers[3],
    contributors: [wikiMembers[3]],
    created_at: '2026-09-20 16:00',
    updated_at: '2026-09-20 16:40',
    sources: [
      src('doc-301', 'SharePoint - 门店主数据', '门店清单 2026.xlsx', '门店 001 位于香港中环，租赁合同 2027 年到期。', 'sheet1/L7')
    ],
    content: `## 定义
美心门店 001（中环店），示例实体页，展示实体-关系-来源引用结构。

## 关键特征
- 区域：香港岛
- 状态：营业中
- 合同到期：2027-06-30

## Related Entities
- [[entity:2026 中秋活动]]
- [[entity:租赁合同 L2020-001]]

## Mentions in Source
- 「门店 001 位于香港中环，租赁合同 2027 年到期」 — doc_id:doc-301, locator:sheet1/L7

## 正文内容
该页由 extract_entities 处理器生成（proposed 状态），待 KM 审核后发布进入实体索引。`
  },
  {
    id: 'art-comp-btn',
    kb_id: 'kb-frontend',
    title: 'Button 组件规范',
    summary: '按钮的类型、尺寸与状态规范。',
    page_type: 'source',
    tags: ['组件', '规范'],
    status: 'published',
    ai_generated: false,
    created_by: wikiMembers[1],
    contributors: [wikiMembers[1], wikiMembers[2]],
    created_at: '2026-09-10 14:00',
    updated_at: '2026-09-15 10:00',
    sources: [],
    content: `## 定义
按钮用于触发操作，支持 5 种类型与 4 种尺寸。

## 关键特征
- 类型：primary / default / dashed / text / link
- 尺寸：large / middle / small / 自定义 token

## 正文内容
按钮间距使用 8px 栅格；危险操作必须二次确认。`
  }
];

export const wikiVersions: Api.Wiki.Version[] = [
  {
    id: 'ver-rag-3',
    article_id: 'art-rag',
    version: 3,
    change_type: 'auto-updated',
    change_summary: '源文档 doc-102 更新：混合检索白皮书新增融合排序说明',
    content: '（快照内容略——自动更新产生的待审版本）',
    created_by: 'KM Agent',
    created_at: '2026-09-20 15:30'
  },
  {
    id: 'ver-rag-2',
    article_id: 'art-rag',
    version: 2,
    change_type: 'human-edited',
    change_summary: '人工修订定义段，补充 permission-aware 说明',
    content: '（快照内容略）',
    created_by: '刘佳鑫',
    created_at: '2026-09-19 09:12'
  },
  {
    id: 'ver-rag-1',
    article_id: 'art-rag',
    version: 1,
    change_type: 'ai-generated',
    change_summary: 'KM Agent 首次生成',
    content: '（快照内容略）',
    created_by: 'KM Agent',
    created_at: '2026-09-18 10:00'
  }
];

export const wikiTocs: Record<string, Api.Wiki.TocNode[]> = {
  'kb-coco': [
    {
      id: 'toc-1',
      title: '产品架构',
      type: 'folder',
      children: [
        { id: 'toc-1-1', title: 'RAG 检索增强生成', type: 'article', article_id: 'art-rag' },
        { id: 'toc-1-2', title: '连接器体系', type: 'article', article_id: 'art-connector' }
      ]
    },
    {
      id: 'toc-2',
      title: '实体',
      type: 'folder',
      children: [{ id: 'toc-2-1', title: '美心门店（示例实体页）', type: 'article', article_id: 'art-store-entity' }]
    }
  ],
  'kb-frontend': [{ id: 'toc-3', title: 'Button 组件规范', type: 'article', article_id: 'art-comp-btn' }]
};

export const wikiAssistants = [
  { id: 'wiki-agent', name: 'Wiki 知识管理 Agent', description: '从数据源生成与更新结构化 wiki 文章' },
  { id: 'deep-research', name: 'Deep Research', description: '多步研究型助手' }
];
