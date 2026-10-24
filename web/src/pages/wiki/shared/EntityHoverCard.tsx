/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

import { Button, Spin, Tag } from 'antd';
import { useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { getWikiEntity } from '@/service/api';

import { wikiTypeColor } from './WikiLinkTag';

// hover previews repeat across articles — one fetch per entity per session
const cardCache = new Map<string, Api.Wiki.EntityInfo>();

interface EntityHoverCardProps {
  entityId: string;
  type: string;
  statusLabel: Record<string, string>;
  detailsLabel: string;
  onOpen: () => void;
}

/** The mini card behind a resolved wikilink capsule: identity + type +
 * status at a glance, one click to the details (entity page or graph). */
export function EntityHoverCard({ entityId, type, statusLabel, detailsLabel, onOpen }: EntityHoverCardProps) {
  const { t } = useTranslation();
  const [entity, setEntity] = useState<Api.Wiki.EntityInfo | null>(() => cardCache.get(entityId) || null);

  useEffect(() => {
    if (entity) return;
    let alive = true;
    getWikiEntity(entityId)
      .then(e => {
        if (!alive || !e) return;
        cardCache.set(entityId, e as Api.Wiki.EntityInfo);
        setEntity(e as Api.Wiki.EntityInfo);
      })
      .catch(() => {});
    return () => {
      alive = false;
    };
  }, [entity, entityId]);

  if (!entity) {
    return (
      <div className="flex w-52 justify-center py-5">
        <Spin size="small" />
      </div>
    );
  }

  const color = wikiTypeColor(entity.type || type);
  const aliases = (entity.aliases || []).filter(Boolean).slice(0, 4);

  return (
    <div className="flex w-60 flex-col gap-4px">
      <div className="flex flex-wrap items-center gap-6px">
        <span className="text-13.5px font-550">{entity.name}</span>
        <Tag className="m-0" color={color}>
          {entity.type || type}
        </Tag>
        {entity.status ? (
          <Tag className="m-0" color={entity.status === 'published' ? 'green' : entity.status === 'reviewed' ? 'blue' : 'orange'}>
            {statusLabel[entity.status] || entity.status}
          </Tag>
        ) : null}
      </div>
      {aliases.length > 0 && (
        <div className="text-12px text-[var(--ant-color-text-tertiary)]">
          {t('page.wiki.entityManager.aliases')}: {aliases.join('、')}
          {(entity.aliases || []).length > aliases.length ? ' …' : ''}
        </div>
      )}
      {(entity.relations || []).length > 0 && (
        <div className="text-12px text-[var(--ant-color-text-tertiary)]">
          {t('page.wiki.entityManager.columns.relations')}: {(entity.relations || []).length}
        </div>
      )}
      <Button className="self-start" onClick={onOpen} size="small" type="link">
        {detailsLabel}
      </Button>
    </div>
  );
}
