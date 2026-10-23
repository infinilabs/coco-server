import { CompassOutlined, FileAddOutlined, RobotOutlined, ThunderboltOutlined } from '@ant-design/icons';
import { Button, Descriptions, Drawer, Empty, List, Modal, Space, Tag, message } from 'antd';
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

import Markdown from '@/components/DocumentDrawer/Markdown';
import { createWikiArticle, getWikiCompanyMap, runWikiCompileAgent } from '@/service/api';
import { useAuth } from '@/hooks/business/auth';

const STATUS_COLOR: Record<string, string> = {
  draft: 'default',
  reviewed: 'processing',
  published: 'success',
  archived: 'warning'
};

type AgentReport = { iterations: number; patches: number; no_action: boolean; notes?: string[] };

type CompanyMapInfo = {
  found: boolean;
  id?: string;
  map?: { kb_id: string; title: string; summary?: string; content?: string; updated?: string };
};

/**
 * KB landing tab: headline numbers, recently touched articles, quick actions — mrdoc-style divided stat band over a
 * dotted-leader recent list.
 */
export function KbOverview({
  kb,
  articles,
  onNewArticle,
  onAiGenerate
}: {
  readonly kb: Api.Wiki.Kb;
  readonly articles: Api.Wiki.Article[];
  readonly onNewArticle: () => void;
  readonly onAiGenerate: () => void;
}) {
  const { t } = useTranslation();
  const nav = useNavigate();
  const { hasAuth } = useAuth();
  const canUpdateKb = hasAuth('coco#wiki_kb/update');
  // the agent run is synchronous and LLM-slow — the report lands in a modal
  const [agentLoading, setAgentLoading] = useState(false);
  const [agentReport, setAgentReport] = useState<AgentReport | null>(null);
  // the company map card: the KB's navigation entry page, rendered inline
  const [companyMap, setCompanyMap] = useState<CompanyMapInfo | null>(null);
  const [mapOpen, setMapOpen] = useState(false);
  const [creatingMap, setCreatingMap] = useState(false);

  useEffect(() => {
    getWikiCompanyMap(kb.id)
      .then(setCompanyMap)
      .catch(() => setCompanyMap({ found: false }));
  }, [kb.id]);

  // one-click create: the backend seeds the map skeleton on empty content —
  // the author lands in the editor with the template instead of a blank page
  const createMap = () => {
    setCreatingMap(true);
    createWikiArticle({ kb_id: kb.id, title: t('page.wiki.companyMap.defaultTitle'), page_type: 'map' })
      .then(res => {
        const newId = ((res as any)?._id as string) || ((res as any)?.id as string) || '';
        if (newId) nav(`/wiki/article/${newId}?kb=${kb.id}`);
      })
      .finally(() => setCreatingMap(false));
  };

  const runAgent = () => {
    setAgentLoading(true);
    runWikiCompileAgent(kb.id)
      .then(report => {
        setAgentReport(report ?? null);
      })
      .finally(() => setAgentLoading(false));
  };

  const published = articles.filter(a => a.status === 'published').length;
  const aiGenerated = articles.filter(a => a.ai_generated).length;
  const recent = [...articles].sort((a, b) => (b.updated_at > a.updated_at ? 1 : -1)).slice(0, 6);

  return (
    // full container width: the header/tabs above span the whole column, so a
    // capped overview column read as "not adaptive" on wide screens
    <div className='w-full flex flex-col gap-14px'>
      {companyMap?.found && companyMap.id ? (
        <section className='wiki-panel'>
          <div className='wiki-panel-head'>
            <span className='wiki-panel-title'>
              <CompassOutlined className='color-[var(--wiki-accent)]' />
              {t('page.wiki.companyMap.title')}
            </span>
            <Space>
              <span className='text-12px color-[var(--wiki-text-3)]'>
                {companyMap.map?.updated
                  ? new Date(companyMap.map.updated).toISOString().slice(0, 16).replace('T', ' ')
                  : ''}
              </span>
              <Button
                size='small'
                onClick={() => setMapOpen(true)}
              >
                {t('page.wiki.companyMap.view')}
              </Button>
              <Button
                size='small'
                type='text'
                onClick={() => nav(`/wiki/article/${companyMap.id}?kb=${kb.id}`)}
              >
                {t('page.wiki.companyMap.open')}
              </Button>
            </Space>
          </div>
          <div className='px-16px py-10px text-13px color-[var(--wiki-text-2)]'>
            {companyMap.map?.summary || companyMap.map?.title}
          </div>
        </section>
      ) : (
        companyMap &&
        !companyMap.found &&
        canUpdateKb && (
          <section className='wiki-panel border-dashed'>
            <div className='flex items-center justify-between gap-12px px-16px py-10px'>
              <span className='text-13px color-[var(--wiki-text-3)]'>{t('page.wiki.companyMap.createHint')}</span>
              <Button
                icon={<CompassOutlined />}
                loading={creatingMap}
                size='small'
                onClick={createMap}
              >
                {t('page.wiki.companyMap.create')}
              </Button>
            </div>
          </section>
        )
      )}
      <div className='wiki-statband'>
        <div className='wiki-stat'>
          <div className='wiki-stat-value'>{articles.length}</div>
          <div className='wiki-stat-label'>{t('page.wiki.overview.articles')}</div>
        </div>
        <div className='wiki-stat'>
          <div className='wiki-stat-value'>{published}</div>
          <div className='wiki-stat-label'>{t('page.wiki.status.published')}</div>
        </div>
        <div className='wiki-stat'>
          <div className='wiki-stat-value'>{aiGenerated}</div>
          <div className='wiki-stat-label'>{t('page.wiki.overview.aiGenerated')}</div>
        </div>
        <div className='wiki-stat'>
          <div className='wiki-stat-value'>{kb.datasource_ids?.length ?? 0}</div>
          <div className='wiki-stat-label'>{t('page.wiki.overview.datasources')}</div>
        </div>
      </div>

      <section className='wiki-panel'>
        <div className='wiki-panel-head'>
          <span className='wiki-panel-title'>{t('page.wiki.overview.recent')}</span>
          <Space>
            {canUpdateKb && (
              <Button
                icon={<ThunderboltOutlined />}
                loading={agentLoading}
                onClick={runAgent}
              >
                {t('page.wiki.kb.agentCompile')}
              </Button>
            )}
            <Button
              icon={<RobotOutlined />}
              onClick={onAiGenerate}
            >
              {t('page.wiki.kb.aiGenerate')}
            </Button>
            <Button
              icon={<FileAddOutlined />}
              type='primary'
              onClick={onNewArticle}
            >
              {t('page.wiki.kb.newArticle')}
            </Button>
          </Space>
        </div>
        <div className='p-16px'>
          {recent.length === 0 ? (
            <Empty
              description={t('page.wiki.hub.empty')}
              image={Empty.PRESENTED_IMAGE_SIMPLE}
            />
          ) : (
            <div className='flex flex-col'>
              {recent.map(a => (
                <div
                  className='flex cursor-pointer items-center gap-12px rounded-8px px-8px py-7px transition-colors -mx-8px hover:bg-[var(--wiki-panel-2)]'
                  key={a.id}
                  onClick={() => nav(`/wiki/article/${a.id}?kb=${kb.id}`)}
                >
                  <span className='min-w-0 truncate text-14px'>{a.title}</span>
                  {/* dotted leader line filling the gap between title and meta —
                      the classic catalog row */}
                  <span
                    className='h-1px flex-1 border-t border-dotted'
                    style={{ borderColor: 'var(--wiki-border)' }}
                  />
                  <Tag
                    className='shrink-0'
                    color={STATUS_COLOR[a.status]}
                  >
                    {t(`page.wiki.status.${a.status}`)}
                  </Tag>
                  <span className='shrink-0 text-xs color-[var(--wiki-text-3)]'>{a.updated_at}</span>
                </div>
              ))}
            </div>
          )}
        </div>
      </section>

      <Drawer
        title={companyMap?.map?.title || t('page.wiki.companyMap.title')}
        width={680}
        onClose={() => setMapOpen(false)}
        open={mapOpen}
      >
        <Markdown content={companyMap?.map?.content || ''} />
      </Drawer>

      <Modal
        footer={null}
        onCancel={() => setAgentReport(null)}
        open={!!agentReport}
        title={t('page.wiki.kb.agentReportTitle')}
      >
        {agentReport && (
          <>
            <Descriptions column={1} size='small'>
              <Descriptions.Item label={t('page.wiki.kb.agentIterations')}>{agentReport.iterations}</Descriptions.Item>
              <Descriptions.Item label={t('page.wiki.kb.agentPatches')}>{agentReport.patches}</Descriptions.Item>
              <Descriptions.Item label={t('page.wiki.kb.agentVerdict')}>
                {agentReport.no_action ? t('page.wiki.kb.agentNoAction') : t('page.wiki.kb.agentFiled')}
              </Descriptions.Item>
              {agentReport.notes && agentReport.notes.length > 0 && (
                <Descriptions.Item label={t('page.wiki.kb.agentNotes')}>{agentReport.notes.join('；')}</Descriptions.Item>
              )}
            </Descriptions>
            {agentReport.patches > 0 && (
              <div className='text-13px color-[var(--ant-color-text-tertiary)]'>{t('page.wiki.kb.agentHint')}</div>
            )}
          </>
        )}
      </Modal>
    </div>
  );
}
