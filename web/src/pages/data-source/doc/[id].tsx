/*
 * 文档详情整页（P0 产物可见性第一批）：
 * - 抽屉空间不足，详情改为 URL 可寻址的完整页面 /data-source/doc/:id
 * - 新增「AI 加工结果」页签：summary / ai_insights 直接读文档字段，无需新端点
 * - 分块页签补向量化元信息（embedding_model）
 * - 解析记录步骤可点击，跳转到对应产物页签（处理器 → 产物映射）
 * - 布局对齐参考原型：左侧页签内容 + 右侧「基本信息」侧栏卡片
 */
import { ArrowLeftOutlined, DownOutlined, ExportOutlined } from '@ant-design/icons';
import { Download } from 'lucide-react';
import { Avatar, Button, Dropdown, Empty, Image, Space, Spin, Tabs, Tag, Typography } from 'antd';

import ChunkTable from '@/components/DocumentDrawer/ChunkTable';
import Markdown from '@/components/DocumentDrawer/Markdown';
import ProcessingTimeline, { STEP_TAB_MAP } from '@/components/DocumentDrawer/ProcessingTimeline';
import { getDocument } from '@/service/api';
import { useRoute } from '@sa/simple-router';
import { useNavigate } from 'react-router-dom';

// bytes → 人类可读（与列表展示口径一致，原始值不可读）
const formatSize = (size?: number) => {
  if (size == null) return '-';
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  if (size < 1024 * 1024 * 1024) return `${(size / 1024 / 1024).toFixed(1)} MB`;
  return `${(size / 1024 / 1024 / 1024).toFixed(1)} GB`;
};

// 「基本信息」侧栏行：label 灰色左对齐定宽，value 右侧自适应（对齐参考原型）
function InfoRow({ label, children }: { label: string; children?: React.ReactNode }) {
  return (
    <div className='flex items-start gap-12px py-5px'>
      <span className='w-80px shrink-0 text-12px color-[var(--ant-color-text-tertiary)] leading-22px'>
        {label}
      </span>
      <div className='flex flex-1 flex-wrap items-center gap-4px min-w-0 text-13px leading-22px'>
        {children ?? <span>-</span>}
      </div>
    </div>
  );
}

function UserCell({ user }: { user?: { username?: string } }) {
  if (!user?.username) return <span>-</span>;
  return (
    <>
      <Avatar size={20} className='bg-[var(--ant-color-primary)] text-12px!'>
        {user.username.slice(0, 1).toUpperCase()}
      </Avatar>
      <span className='truncate'>{user.username}</span>
    </>
  );
}

export function Component() {
  const route = useRoute();
  const navigate = useNavigate();
  // hash 路由跳转时 route.params 会先于本组件卸载更新为下个路由的参数，
  // 导致 getDocument 用错误 id 重发请求（404）——只在自家路由激活时更新 docId
  const [docId, setDocId] = useState(route.params.id as string);
  useEffect(() => {
    if (route.name === 'data-source_doc' && route.params.id) {
      setDocId(route.params.id);
    }
  }, [route.name, route.params.id]);
  const [searchParams, setSearchParams] = useSearchParams();
  const { t } = useTranslation();

  const [doc, setDoc] = useState<any>(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    if (!docId) return;
    setLoading(true);
    getDocument(docId)
      .then(res => {
        if (res.data?.found === true) {
          setDoc(res.data._source || {});
        }
      })
      .finally(() => setLoading(false));
  }, [docId]);

  const activeKey = searchParams.get('tab') || 'ai';
  const setTab = (key: string) => {
    try {
      setSearchParams({ tab: key });
    } catch {
      // URL 同步尽力而为，失败时仅内存态切换
    }
  };

  const onStepClick = (name?: string) => {
    const key = name ? STEP_TAB_MAP[name] : undefined;
    if (key) setTab(key);
  };

  const rawContent = doc?.metadata?.raw_content;
  // raw_content URL 是后端无条件合成的（RefineRawContentURL），
  // 只有 raw_content_returns_file=true（local_fs/s3 连接器采集时写入）才真有文件可下载；
  // 头部按钮组（下载原文件）与附件 tab 空态判断共用
  const hasRawFile = rawContent && doc?.metadata?.raw_content_returns_file === true;

  // ai_insights 由 document_summarization 生成，格式为 Markdown+Mermaid；
  // 复用 wiki 详情页同一 Markdown 渲染器（含代码高亮/mermaid 样式）
  const aiTab = (
    <div className='flex flex-col gap-16px'>
      {doc?.summary ? (
        <section className='flex flex-col gap-8px'>
          <Typography.Text strong className='text-13px'>
            {t('page.datasource.doc.aiSummary')}
          </Typography.Text>
          <Markdown content={doc.summary} />
        </section>
      ) : null}
      {doc?.ai_insights?.text ? (
        <section className='flex flex-col gap-8px'>
          <Typography.Text strong className='text-13px'>
            {t('page.datasource.doc.aiInsights')}
          </Typography.Text>
          <Markdown content={doc.ai_insights.text} />
        </section>
      ) : null}
      {!doc?.summary && !doc?.ai_insights?.text ? (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description={t('page.datasource.doc.noAiResult')}
        />
      ) : null}
    </div>
  );

  const tabItems = [
    {
      key: 'ai',
      label: t('page.datasource.doc.tabAi'),
      children: aiTab
    },
    {
      key: 'text',
      label: t('page.datasource.doc.tabText'),
      children: doc?.content ? (
        <div className='leading-[24px] text-12px'>
          <Markdown content={doc.content} />
        </div>
      ) : (
        <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('page.datasource.doc.noText')} />
      )
    },
    {
      key: 'chunks',
      label: t('page.datasource.doc.tabChunks'),
      children: <ChunkTable docId={docId} embeddingModel={doc?.embedding_model} />
    },
    {
      key: 'attachments',
      label: t('page.datasource.doc.tabAttachments'),
      children: (() => {
        // 生成图像：cover/thumbnail 由 generate_cover 处理器产出，
        // getDocument 响应里已是转好的完整 /attachment/:id URL（RefineCoverThumbnail），
        // 前端零后端改动即可直接展示；内嵌附件列表等后端补 mapping 后再接入
        const images = [
          { key: 'cover', url: doc?.cover as string | undefined, label: t('page.datasource.doc.cover') },
          { key: 'thumbnail', url: doc?.thumbnail as string | undefined, label: t('page.datasource.doc.thumbnail') }
        ].filter(img => !!img.url);

        return (
          <div className='flex flex-col gap-12px'>
            {images.length > 0 && (
              <div className='flex flex-wrap gap-16px'>
                {images.map(img => (
                  <div key={img.key} className='flex flex-col gap-6px'>
                    <Typography.Text type='secondary' className='text-12px'>
                      {img.label}
                    </Typography.Text>
                    <Image
                      src={img.url}
                      width={220}
                      height={160}
                      className='rounded-lg bg-#F6F8FA object-contain dark:bg-[rgb(var(--layout-bg-color))]'
                      style={{ objectFit: 'contain' }}
                    />
                    <Button
                      size='small'
                      className='self-start'
                      icon={<Download className='h-14px w-14px' />}
                      onClick={() => img.url && window.open(img.url, '_blank')}
                    >
                      {t('page.datasource.doc.download')}
                    </Button>
                  </div>
                ))}
              </div>
            )}
            {/* 附件 tab 只放衍生品；原文件下载已上移到头部「原文预览」按钮组
                （raw_content URL 是后端无条件合成的，raw_content_returns_file 才真有文件） */}
            {images.length === 0 ? (
              <Empty image={Empty.PRESENTED_IMAGE_SIMPLE} description={t('page.datasource.doc.noAttachment')} />
            ) : null}
          </div>
        );
      })()
    },
    {
      key: 'runs',
      label: t('page.datasource.doc.tabRuns'),
      children: (
        <div className='flex flex-col gap-8px'>
          <Typography.Text type='secondary' className='text-12px'>
            {t('page.datasource.doc.stepsJumpHint')}
          </Typography.Text>
          <ProcessingTimeline docId={docId} onStepClick={onStepClick} />
        </div>
      )
    }
  ];

  return (
    <Spin spinning={loading}>
      {/* pb：布局层 h-full+p-16px 的底边距在内容溢出滚动时会被跳过，页面级补回同样的 16px；
          w-full 防御性保证根容器吃满内容区；默认 stretch 使窄屏列布局下两张卡片都占满宽度
          （items-start 会收缩到内容宽），lg 行布局下再用 items-start 保持顶对齐 + 自然高度 */}
      <div className='flex w-full flex-col gap-16px pb-16px lg:flex-row lg:items-start'>
        {/* 左卡片：标题头 + 页签内容 */}
        <ACard className='min-w-0 flex-1' styles={{ body: { padding: '16px 24px' } }}>
          <div className='flex flex-col gap-12px'>
            {/* 文件路径式面包屑：数据源名（可点，回列表）/ 文档标题；doc 未加载时退回返回按钮 */}
            <div className='flex w-full items-center gap-8px'>
              {!loading && doc?.source?.id ? (
                <>
                  <Typography.Link
                    className='shrink-0 text-14px'
                    onClick={() => navigate(`/data-source/detail/${doc.source.id}`)}
                  >
                    {doc.source.name || doc.source.id}
                  </Typography.Link>
                  <span className='color-[var(--ant-color-text-tertiary)]'>/</span>
                </>
              ) : (
                <Button
                  type='text'
                  size='small'
                  className='shrink-0'
                  icon={<ArrowLeftOutlined />}
                  onClick={() => navigate(-1)}
                >
                  {t('page.datasource.doc.backToList')}
                </Button>
              )}
              <Typography.Title className='mb-0! min-w-0 flex-1 truncate! text-16px!'>
                {doc?.title || docId}
              </Typography.Title>
              {docId &&
                (hasRawFile ? (
                  // 原文预览 + 下载原文件是同一源文件的两种消费方式，收进一个按钮组；
                  // 「下载原件」是文档级动作，放头部而非附件 tab（附件 tab 只放衍生品）。
                  // 预览与操作菜单拆成两个独立点击目标 + click 触发，避免 hover 菜单导致误触预览；
                  // 右侧为分裂式下拉箭头，后续文档级操作（重试加工、导出等）可直接加进菜单
                  <Space.Compact size='small'>
                    <Button size='small' onClick={() => window.open(`/preview/document/${docId}`, '_blank')}>
                      {t('page.datasource.doc.openPreview')}
                    </Button>
                    <Dropdown
                      placement='bottomRight'
                      trigger={['click']}
                      menu={{
                        items: [
                          {
                            key: 'download',
                            icon: <Download className='h-14px w-14px' />,
                            label: t('page.datasource.doc.downloadOriginal')
                          }
                        ],
                        onClick: ({ key }) => {
                          if (key === 'download') window.open(rawContent, '_blank');
                        }
                      }}
                    >
                      <Button size='small' icon={<DownOutlined className='text-10px' />} />
                    </Dropdown>
                  </Space.Compact>
                ) : (
                  <Button size='small' onClick={() => window.open(`/preview/document/${docId}`, '_blank')}>
                    {t('page.datasource.doc.openPreview')}
                  </Button>
                ))}
            </div>

            <Tabs className='flex-1' activeKey={activeKey} items={tabItems} onChange={setTab} destroyOnHidden={false} />
          </div>
        </ACard>

        {/* 右卡片：基本信息侧栏（对齐参考原型布局） */}
        <ACard className='w-full shrink-0 lg:w-336px' styles={{ body: { padding: '12px 16px' } }}>
          <Typography.Text strong className='text-14px'>
            {t('page.datasource.doc.basicInfo')}
          </Typography.Text>
          <div className='mt-8px flex flex-col'>
            <InfoRow label={t('page.datasource.labels.createdBy')}>
              <UserCell user={doc?.owner} />
            </InfoRow>
            <InfoRow label={t('page.datasource.labels.type')}>{doc?.type || '-'}</InfoRow>
            <InfoRow label={t('page.datasource.labels.size')}>{formatSize(doc?.size)}</InfoRow>
            <InfoRow label={t('page.datasource.labels.created')}>
              <DateTime value={doc?.created} />
            </InfoRow>
            <InfoRow label={t('page.datasource.labels.updatedBy')}>
              <UserCell user={doc?.last_updated_by?.user} />
            </InfoRow>
            <InfoRow label={t('page.datasource.labels.updated')}>
              <DateTime value={doc?.last_updated_by?.timestamp} />
            </InfoRow>
            <InfoRow label={t('page.datasource.doc.lang')}>{doc?.lang || '-'}</InfoRow>
            <InfoRow label={t('page.datasource.doc.docLink')}>
              {doc?.url ? (
                <Typography.Link
                  className='truncate'
                  copyable
                  onClick={() => doc.url && window.open(doc.url, '_blank')}
                >
                  {doc.url}
                  <ExportOutlined className='ml-2px text-12px' />
                </Typography.Link>
              ) : (
                <span>-</span>
              )}
            </InfoRow>
            <InfoRow label='Tags'>
              {doc?.tags?.length > 0 ? (
                doc.tags.map((tag: string, i: number) => (
                  <Tag
                    className='mr-0 bg-#E8E8E8 color-#101010 border-0 dark:bg-[var(--ant-color-fill-secondary)] dark:color-[var(--ant-color-text)]'
                    key={i}
                  >
                    {tag}
                  </Tag>
                ))
              ) : (
                <span>-</span>
              )}
            </InfoRow>
          </div>
        </ACard>
      </div>
    </Spin>
  );
}
