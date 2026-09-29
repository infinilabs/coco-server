import { SearchOutlined } from '@ant-design/icons';
import { Empty, Input, List, Modal, Tag } from 'antd';
import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { searchWikiArticles, searchWikiKbs } from '@/service/api';

interface Props {
  open: boolean;
  onClose: () => void;
}

export function SearchModal({ open, onClose }: Props) {
  const { t } = useTranslation();
  const nav = useNavigate();
  const [query, setQuery] = useState('');
  const [articles, setArticles] = useState<Api.Wiki.Article[]>([]);
  const [kbs, setKbs] = useState<Api.Wiki.Kb[]>([]);
  const [loading, setLoading] = useState(false);

  const runSearch = (value: string) => {
    const q = value.trim();
    setQuery(q);
    if (!q) {
      setArticles([]);
      setKbs([]);
      return;
    }
    setLoading(true);
    Promise.all([searchWikiArticles({ query: q }), searchWikiKbs({ query: q })]).then(([a, k]) => {
      setArticles(((a as any).data || []) as Api.Wiki.Article[]);
      setKbs(((k as any).data || []) as Api.Wiki.Kb[]);
      setLoading(false);
    });
  };

  const goArticle = (article: Api.Wiki.Article) => {
    onClose();
    nav(`/wiki/article/${article.id}?kb=${article.kb_id}`);
  };

  const goKb = (kb: Api.Wiki.Kb) => {
    onClose();
    nav(`/wiki/kb/${kb.id}`);
  };

  const empty = !loading && query && articles.length === 0 && kbs.length === 0;

  return (
    <Modal footer={null} open={open} title={t('page.wiki.search.title')} width={640} onCancel={onClose}>
      <Input.Search
        allowClear
        autoFocus
        prefix={<SearchOutlined className='text-gray-400' />}
        size='large'
        onChange={e => runSearch(e.target.value)}
        onSearch={runSearch}
        placeholder={t('page.wiki.search.placeholder')}
      />
      {empty && <Empty className='mt-6' description={t('page.wiki.search.empty')} image={Empty.PRESENTED_IMAGE_SIMPLE} />}
      {kbs.length > 0 && (
        <div className='mt-4'>
          <div className='mb-2 font-medium text-gray-500'>{t('page.wiki.search.kbs')}</div>
          <List
            dataSource={kbs}
            loading={loading}
            renderItem={kb => (
              <List.Item
                className='cursor-pointer'
                onClick={() => goKb(kb)}
                extra={<span className='text-3#999'>{`${kb.article_count} ${t('page.wiki.hub.articlesUnit')}`}</span>}
              >
                <List.Item.Meta
                  avatar={<span className='text-2em'>{kb.icon}</span>}
                  description={kb.description}
                  title={kb.name}
                />
              </List.Item>
            )}
          />
        </div>
      )}
      {articles.length > 0 && (
        <div className='mt-4'>
          <div className='mb-2 font-medium text-gray-500'>{t('page.wiki.search.articles')}</div>
          <List
            dataSource={articles}
            loading={loading}
            renderItem={article => (
              <List.Item
                className='cursor-pointer'
                extra={article.page_type ? <Tag>{t(`page.wiki.pageType.${article.page_type}`)}</Tag> : null}
                onClick={() => goArticle(article)}
              >
                <List.Item.Meta description={article.summary} title={article.title} />
              </List.Item>
            )}
          />
        </div>
      )}
    </Modal>
  );
}
