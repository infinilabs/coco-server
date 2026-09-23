import { FileAddOutlined, RobotOutlined, SearchOutlined } from '@ant-design/icons';
import { Button, Card, Empty, List, Space, Statistic, Tag } from 'antd';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

const STATUS_COLOR: Record<string, string> = {
  draft: 'default',
  reviewed: 'processing',
  published: 'success',
  archived: 'warning'
};

/** KB landing tab: headline numbers, recently touched articles, quick actions. */
export function KbOverview({
  kb,
  articles,
  onNewArticle,
  onAiGenerate
}: {
  kb: Api.Wiki.Kb;
  articles: Api.Wiki.Article[];
  onNewArticle: () => void;
  onAiGenerate: () => void;
}) {
  const { t } = useTranslation();
  const nav = useNavigate();

  const published = articles.filter(a => a.status === 'published').length;
  const aiGenerated = articles.filter(a => a.ai_generated).length;
  const recent = [...articles]
    .sort((a, b) => (b.updated_at > a.updated_at ? 1 : -1))
    .slice(0, 6);

  return (
    <div className="flex flex-col gap-12px">
      <div className="grid grid-cols-2 gap-12px sm:grid-cols-4">
        <Card bordered={false} className="card-wrapper">
          <Statistic title={t('page.wiki.overview.articles')} value={articles.length} />
        </Card>
        <Card bordered={false} className="card-wrapper">
          <Statistic title={t('page.wiki.status.published')} value={published} />
        </Card>
        <Card bordered={false} className="card-wrapper">
          <Statistic title={t('page.wiki.overview.aiGenerated')} value={aiGenerated} />
        </Card>
        <Card bordered={false} className="card-wrapper">
          <Statistic title={t('page.wiki.overview.datasources')} value={kb.datasource_ids?.length ?? 0} />
        </Card>
      </div>

      <Card bordered={false} className="card-wrapper">
        <div className="mb-8px flex items-center justify-between">
          <span className="font-500">{t('page.wiki.overview.recent')}</span>
          <Space>
            <Button icon={<SearchOutlined />} onClick={() => nav('/wiki/list')}>
              {t('page.wiki.overview.search')}
            </Button>
            <Button icon={<RobotOutlined />} onClick={onAiGenerate}>
              {t('page.wiki.kb.aiGenerate')}
            </Button>
            <Button icon={<FileAddOutlined />} type="primary" onClick={onNewArticle}>
              {t('page.wiki.kb.newArticle')}
            </Button>
          </Space>
        </div>
        {recent.length === 0 ? (
          <Empty description={t('page.wiki.hub.empty')} image={Empty.PRESENTED_IMAGE_SIMPLE} />
        ) : (
          <List
            dataSource={recent}
            renderItem={a => (
              <List.Item
                className="cursor-pointer"
                extra={
                  <Space>
                    <Tag color={STATUS_COLOR[a.status]}>{t(`page.wiki.status.${a.status}`)}</Tag>
                    <span className="text-xs text-gray-400">{a.updated_at}</span>
                  </Space>
                }
                onClick={() => nav(`/wiki/article/${a.id}?kb=${kb.id}`)}
              >
                <List.Item.Meta description={a.summary?.slice(0, 90)} title={a.title} />
              </List.Item>
            )}
          />
        )}
      </Card>
    </div>
  );
}
