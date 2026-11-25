import { CommentOutlined, DeleteOutlined, UserOutlined } from '@ant-design/icons';
import { Avatar, Button, Empty, Input, List, Popconfirm, Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { createWikiComment, deleteWikiComment, searchWikiComments } from '@/service/api';

/**
 * Discussion under an article. Comments are lightweight user content (no
 * review machine); identity is stamped on create and immutable after.
 */
export function ArticleComments({ articleId, currentUserId, currentUserName }: { articleId: string; currentUserId: string; currentUserName: string }) {
  const { t } = useTranslation();
  const [items, setItems] = useState<Api.Wiki.Comment[]>([]);
  const [loading, setLoading] = useState(false);
  const [draft, setDraft] = useState('');
  const [sending, setSending] = useState(false);

  const fetchList = () => {
    setLoading(true);
    searchWikiComments(articleId).then(res => {
      setItems((res as any) || []);
      setLoading(false);
    });
  };

  useEffect(() => {
    if (articleId) fetchList();
  }, [articleId]);

  const onSubmit = () => {
    const content = draft.trim();
    if (!content) return;
    setSending(true);
    createWikiComment({ article_id: articleId, user_id: currentUserId, user_name: currentUserName, content }).then(res => {
      setSending(false);
      if ((res as any)?._id) {
        setDraft('');
        fetchList();
      }
    });
  };

  const onDelete = (id: string) => {
    deleteWikiComment(id).then(() => {
      setItems(prev => prev.filter(c => c.id !== id));
    });
  };

  return (
    <div>
      <div className="mb-8px flex items-center gap-6px font-500">
        <CommentOutlined />
        {t('page.wiki.comments.title')}
        <span className="font-400 color-[var(--ant-color-text-tertiary)]">({items.length})</span>
      </div>
      <Input.TextArea
        autoSize={{ minRows: 2, maxRows: 6 }}
        maxLength={4000}
        placeholder={t('page.wiki.comments.placeholder')}
        value={draft}
        onChange={e => setDraft(e.target.value)}
      />
      <div className="mt-8px mb-16px text-right">
        <Button disabled={!draft.trim()} loading={sending} type="primary" onClick={onSubmit}>
          {t('page.wiki.comments.submit')}
        </Button>
      </div>
      {items.length === 0 && !loading ? (
        <Empty description={t('page.wiki.comments.empty')} image={Empty.PRESENTED_IMAGE_SIMPLE} />
      ) : (
        <List
          dataSource={items}
          loading={loading}
          renderItem={c => (
            <List.Item
              actions={
                c.user_id === currentUserId
                  ? [
                      <Popconfirm key="del" title={t('page.wiki.comments.deleteConfirm')} onConfirm={() => onDelete(c.id)}>
                        <Tooltip title={t('common.delete')}>
                          <Button danger icon={<DeleteOutlined />} size="small" type="text" />
                        </Tooltip>
                      </Popconfirm>
                    ]
                  : undefined
              }
            >
              <List.Item.Meta
                avatar={<Avatar icon={<UserOutlined />}>{c.user_name?.[0]?.toUpperCase()}</Avatar>}
                description={
                  <div className="whitespace-pre-wrap break-words">{c.content}</div>
                }
                title={
                  <span>
                    {c.user_name || t('page.wiki.comments.anonymous')}
                    <span className="ml-8px text-xs font-400 color-[var(--ant-color-text-tertiary)]">{c.created_at}</span>
                  </span>
                }
              />
            </List.Item>
          )}
        />
      )}
    </div>
  );
}
