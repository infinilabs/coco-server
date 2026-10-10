import { BellOutlined, CheckOutlined } from '@ant-design/icons';
import { Badge, Button, Drawer, Empty, List, Segmented, Tag, Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { markWikiNotificationRead, searchWikiNotifications } from '@/service/api';

const ACTION_COLOR: Record<string, string> = {
  ai_generated: 'purple',
  auto_updated: 'orange',
  published: 'green',
  reviewed: 'blue'
};

/**
 * Message center for the wiki app: freshness-loop and review notifications, filtered by
 * read state, one-click through to the touched article.
 */
export function NotificationDrawer({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useTranslation();
  const nav = useNavigate();
  const [items, setItems] = useState<Api.Wiki.Notification[]>([]);
  const [loading, setLoading] = useState(false);
  const [filter, setFilter] = useState<'all' | 'unread'>('all');

  const fetchList = () => {
    setLoading(true);
    searchWikiNotifications().then(res => {
      setItems((res as any).data || []);
      setLoading(false);
    });
  };

  useEffect(() => {
    if (open) fetchList();
  }, [open]);

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
      onClose();
    }
  };

  return (
    <Drawer
      title={
        <div className="flex items-center justify-between" style={{ paddingRight: 24 }}>
          <span>{t('page.wiki.notification.title')}</span>
          <Tooltip title={t('page.wiki.notification.markAll')}>
            <Button
              disabled={unread === 0}
              icon={<CheckOutlined />}
              onClick={markAll}
              size="small"
              type="text"
            >
              {t('page.wiki.notification.markAll')}
            </Button>
          </Tooltip>
        </div>
      }
      open={open}
      width={380}
      onClose={onClose}
    >
      <Segmented
        block
        className="mb-12px"
        value={filter}
        onChange={v => setFilter(v as 'all' | 'unread')}
        options={[
          { value: 'all', label: t('page.wiki.notification.all') },
          { value: 'unread', label: `${t('page.wiki.notification.unread')} (${unread})` }
        ]}
      />
      {shown.length === 0 && !loading ? (
        <Empty description={t('page.wiki.notification.empty')} image={Empty.PRESENTED_IMAGE_SIMPLE} />
      ) : (
        <List
          dataSource={shown}
          loading={loading}
          renderItem={n => (
            <List.Item
              className="cursor-pointer"
              onClick={() => openTarget(n)}
              extra={
                !n.read ? (
                  <Badge color="#1677ff" status="processing" />
                ) : (
                  <span className="text-xs text-gray-400">{n.created_at}</span>
                )
              }
            >
              <List.Item.Meta
                description={n.message}
                title={
                  <span className={!n.read ? 'font-500' : 'text-gray-500'}>
                    {n.action && (
                      <Tag bordered={false} color={ACTION_COLOR[n.action] || 'default'}>
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
    </Drawer>
  );
}
