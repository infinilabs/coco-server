import { CheckOutlined } from '@ant-design/icons';
import { Badge, Button, Empty, List, Segmented, Tag } from 'antd';
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { markWikiNotificationRead, searchWikiNotifications } from '@/service/api';
import { WikiShell } from '../components/WikiShell';

const ACTION_COLOR: Record<string, string> = {
  ai_generated: 'purple',
  auto_updated: 'orange',
  published: 'green',
  reviewed: 'blue'
};

/**
 * Message center page — the wiki notification list promoted from the old
 * drawer to a full page, mounted like knowledge governance behind the
 * topbar entry switcher. Freshness-loop and review events, filterable by
 * read state, one-click through to the touched article.
 */
export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  const [items, setItems] = useState<Api.Wiki.Notification[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<'all' | 'unread'>('all');

  const fetchList = () => {
    setLoading(true);
    searchWikiNotifications().then(res => {
      setItems(((res as any).data || []) as Api.Wiki.Notification[]);
      setLoading(false);
    });
  };

  useEffect(() => {
    fetchList();
  }, []);

  const unread = items.filter(n => !n.read).length;
  const shown = filter === 'unread' ? items.filter(n => !n.read) : items;

  const markRead = (id: string) => {
    markWikiNotificationRead(id).then(() => {
      setItems(prev => prev.map(n => (n.id === id ? { ...n, read: true } : n)));
    });
  };

  const markAll = () => {
    items.filter(n => !n.read).forEach(n => markWikiNotificationRead(n.id));
    setItems(prev => prev.map(n => ({ ...n, read: true })));
  };

  const openTarget = (n: Api.Wiki.Notification) => {
    if (!n.read) markRead(n.id);
    if (n.target_type === 'article' && n.target_id) {
      nav(`/wiki/article/${n.target_id}`);
    }
  };

  return (
    <WikiShell>
      <div className='wiki-hub-page pb-24px'>
        <div className='wiki-panel'>
          <div className='wiki-panel-head'>
            <span className='wiki-panel-title'>{t('page.wiki.notification.title')}</span>
            <Button
              disabled={unread === 0}
              icon={<CheckOutlined />}
              onClick={markAll}
              size='small'
            >
              {t('page.wiki.notification.markAll')}
            </Button>
          </div>
          <div className='p-16px'>
            <Segmented
              block
              className='mb-12px'
              value={filter}
              onChange={v => setFilter(v as 'all' | 'unread')}
              options={[
                { value: 'all', label: t('page.wiki.notification.all') },
                { value: 'unread', label: `${t('page.wiki.notification.unread')} (${unread})` }
              ]}
            />
            {shown.length === 0 && !loading ? (
              <Empty
                description={t('page.wiki.notification.empty')}
                image={Empty.PRESENTED_IMAGE_SIMPLE}
              />
            ) : (
              <List
                className='wiki-row-list'
                dataSource={shown}
                loading={loading}
                renderItem={n => (
                  <List.Item
                    className='cursor-pointer'
                    extra={
                      !n.read ? (
                        <Badge
                          color='#1677ff'
                          status='processing'
                        />
                      ) : (
                        <span className='text-xs text-gray-400'>{n.created_at}</span>
                      )
                    }
                    onClick={() => openTarget(n)}
                  >
                    <List.Item.Meta
                      description={n.message}
                      title={
                        <span className={!n.read ? 'font-500' : 'text-gray-500'}>
                          {n.action && (
                            <Tag
                              bordered={false}
                              color={ACTION_COLOR[n.action] || 'default'}
                            >
                              {t(`page.wiki.notification.action.${n.action}`, n.action)}
                            </Tag>
                          )}
                          {n.created_at}
                        </span>
                      }
                    />
                  </List.Item>
                )}
              />
            )}
          </div>
        </div>
      </div>
    </WikiShell>
  );
}
