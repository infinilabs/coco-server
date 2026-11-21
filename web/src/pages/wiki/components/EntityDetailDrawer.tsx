import { Button, Drawer, Space, Spin } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import { getWikiEntity } from '@/service/api';

import { EntityArticleSections } from './EntityArticleSections';

/**
 * Shared entity detail drawer — the in-place face for entities that have no
 * curated article page yet. The article page's wikilink capsules open this
 * instead of deep-linking the graph tab (a full-page jump lost the reading
 * context); the graph stays reachable as a secondary header action, and
 * clicking a related entity inside swaps the drawer to that entity.
 */
export function EntityDetailDrawer({
  entityId,
  kbId,
  open,
  onClose,
  onOpenEntity
}: {
  /** entity to show; null closes */
  entityId: string | null;
  kbId?: string;
  open: boolean;
  onClose: () => void;
  /** related-entity chips swap the drawer in place instead of jumping */
  onOpenEntity?: (entityId: string) => void;
}) {
  const { t } = useTranslation();
  const nav = useNavigate();
  const [entity, setEntity] = useState<Api.Wiki.EntityInfo | null>(null);
  const [loading, setLoading] = useState(false);

  useEffect(() => {
    if (!open || !entityId) {
      setEntity(null);
      return;
    }
    let cancelled = false;
    setLoading(true);
    getWikiEntity(entityId)
      .then(e => {
        if (!cancelled) setEntity(e);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [open, entityId]);

  return (
    <Drawer
      title={entity ? t('page.wiki.entityManager.detailTitle', { name: entity.name }) : ''}
      width={560}
      open={open && !!entityId}
      onClose={onClose}
      extra={
        entity && (
          <Space>
            {entity.article_id && (
              <Button size="small" onClick={() => nav(`/wiki/article/${entity.article_id}?kb=${kbId}`)}>
                {t('page.wiki.entityManager.viewArticle')}
              </Button>
            )}
            <Button size="small" onClick={() => nav(`/wiki/kb/${kbId || ''}?tab=graph&entity=${entityId}`)}>
              {t('page.wiki.entityManager.viewInGraph')}
            </Button>
          </Space>
        )
      }
    >
      {loading ? (
        <div className="flex justify-center py-40px">
          <Spin />
        </div>
      ) : (
        entityId && (
          <EntityArticleSections
            key={entityId}
            entityId={entityId}
            kbId={kbId}
            onOpenEntity={onOpenEntity}
          />
        )
      )}
    </Drawer>
  );
}

export default EntityDetailDrawer;
