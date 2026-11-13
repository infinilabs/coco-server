import { ReloadOutlined } from '@ant-design/icons';
import { Button, Card, Col, message, Row, Statistic, Table, Tag, Tooltip } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';

import { createWikiArticle, fetchKbHealth } from '@/service/api';

const CODE_COLOR: Record<string, string> = {
  orphan: 'orange',
  thin_content: 'gold',
  no_sources: 'volcano',
  missing_provenance: 'purple'
};

const fmtTime = (iso?: string) => {
  if (!iso) return '';
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? iso : d.toISOString().slice(0, 16).replace('T', ' ');
};

/**
 * KB health report (LLM-wiki 体检, whole-KB counterpart of the per-page
 * lint): orphan pages nobody links to, red-link knowledge gaps worth
 * writing next, thin/unbacked pages and the hubs holding the graph
 * together — recomputed live on every load, advisory only.
 */
export function KbHealth({ kbId }: { readonly kbId: string }) {
  const { t } = useTranslation();
  const nav = useNavigate();

  const [report, setReport] = useState<Api.Wiki.KbHealth | null>(null);
  const [loading, setLoading] = useState(false);
  const [creatingGap, setCreatingGap] = useState<string | null>(null);

  const load = useCallback(() => {
    if (!kbId) return;
    setLoading(true);
    fetchKbHealth(kbId)
      .then(res => {
        setReport((res as any)?.data ?? res ?? null);
      })
      .finally(() => setLoading(false));
  }, [kbId]);

  useEffect(() => {
    load();
  }, [load]);

  const goArticle = (articleId: string) => nav(`/wiki/article/${articleId}?kb=${kbId}`);

  // a red link is the next page worth writing — create the stub draft
  // with real wikilinks back to its citers (born connected, never an
  // orphan) and drop the maintainer straight into it; the next scan
  // resolves the links
  const createDraftFromGap = (name: string, citers: Api.Wiki.HealthGap['cited_by']) => {
    setCreatingGap(name);
    const related = (citers || [])
      .map(ref => `- [[${ref.title}]]`)
      .join('\n');
    createWikiArticle({
      kb_id: kbId,
      title: name,
      page_type: 'concept',
      status: 'draft',
      summary: t('page.wiki.health.gapDraftSummary'),
      content:
        t('page.wiki.health.gapDraftContent', { name }) +
        (related ? `\n\n## ${t('page.wiki.health.relatedSection')}\n\n${related}\n` : '')
    })
      .then(res => {
        if ((res as any)?._id) {
          message.success(t('page.wiki.health.draftCreated'));
          nav(`/wiki/article/${(res as any)._id}?kb=${kbId}`);
        }
      })
      .finally(() => setCreatingGap(null));
  };

  const counts = report?.counts || {};

  const flaggedColumns = [
    {
      title: t('page.wiki.health.page'),
      dataIndex: 'title',
      render: (title: string, row: Api.Wiki.HealthArticle) => (
        <a onClick={() => goArticle(row.id)}>{title}</a>
      )
    },
    {
      title: t('page.wiki.health.findings'),
      dataIndex: 'codes',
      width: 260,
      render: (codes?: string[]) =>
        (codes || []).map(code => (
          <Tag key={code} color={CODE_COLOR[code]}>
            {t(`page.wiki.health.codes.${code}`)}
          </Tag>
        ))
    },
    {
      title: t('page.wiki.health.inbound'),
      dataIndex: 'inbound_links',
      width: 110,
      sorter: (a: Api.Wiki.HealthArticle, b: Api.Wiki.HealthArticle) => a.inbound_links - b.inbound_links
    },
    {
      title: t('page.wiki.health.contentLen'),
      dataIndex: 'content_len',
      width: 110,
      sorter: (a: Api.Wiki.HealthArticle, b: Api.Wiki.HealthArticle) => a.content_len - b.content_len
    }
  ];

  const gapColumns = [
    {
      title: t('page.wiki.health.gapName'),
      dataIndex: 'name',
      render: (name: string) => <span className='font-medium'>{name}</span>
    },
    {
      title: t('page.wiki.health.demand'),
      dataIndex: 'count',
      width: 110,
      sorter: (a: Api.Wiki.HealthGap, b: Api.Wiki.HealthGap) => a.count - b.count,
      render: (count: number) => <Tag color='red'>{count}</Tag>
    },
    {
      title: t('page.wiki.health.citedBy'),
      dataIndex: 'cited_by',
      render: (refs: Api.Wiki.HealthGap['cited_by']) =>
        (refs || []).map(ref => (
          <a key={ref.id} className='mr-8px' onClick={() => goArticle(ref.id)}>
            {ref.title}
          </a>
        ))
    },
    {
      title: t('page.wiki.health.actions'),
      width: 120,
      render: (_: unknown, row: Api.Wiki.HealthGap) => (
        <Button
          loading={creatingGap === row.name}
          onClick={() => createDraftFromGap(row.name, row.cited_by)}
          size='small'
          type='link'
        >
          {t('page.wiki.health.createDraft')}
        </Button>
      )
    }
  ];

  const hubColumns = [
    {
      title: t('page.wiki.health.hubPage'),
      dataIndex: 'title',
      render: (title: string, row: Api.Wiki.HealthArticle) => (
        <a onClick={() => goArticle(row.id)}>{title}</a>
      )
    },
    {
      title: t('page.wiki.health.inbound'),
      dataIndex: 'inbound_links',
      width: 140,
      sorter: (a: Api.Wiki.HealthArticle, b: Api.Wiki.HealthArticle) => a.inbound_links - b.inbound_links,
      render: (n: number) => <Tag color='geekblue'>{n}</Tag>
    }
  ];

  if (!loading && report && report.total_pages === 0) {
    return (
      <Card>
        <div className='mb-12px text-right'>
          <Button icon={<ReloadOutlined />} onClick={load} />
        </div>
        {t('page.wiki.health.empty')}
      </Card>
    );
  }

  return (
    <div className='flex flex-col gap-12px'>
      <div className='flex items-center justify-end gap-8px'>
        {report?.generated_at && (
          <span className='text-xs text-gray-400'>
            {t('page.wiki.health.generatedAt', { time: fmtTime(report.generated_at) })}
          </span>
        )}
        <Button icon={<ReloadOutlined />} loading={loading} onClick={load}>
          {t('page.wiki.health.refresh')}
        </Button>
      </div>

      <Row gutter={12}>
        <Col span={4}>
          <Card>
            <Statistic title={t('page.wiki.health.totalPages')} value={report?.total_pages ?? 0} />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title={t('page.wiki.health.orphanPages')}
              value={counts.orphan_pages ?? 0}
              valueStyle={counts.orphan_pages ? { color: '#fa8c16' } : undefined}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title={t('page.wiki.health.knowledgeGaps')}
              value={counts.knowledge_gaps ?? 0}
              valueStyle={counts.knowledge_gaps ? { color: '#cf1322' } : undefined}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title={t('page.wiki.health.thinPages')}
              value={counts.thin_pages ?? 0}
              valueStyle={counts.thin_pages ? { color: '#d4b106' } : undefined}
            />
          </Card>
        </Col>
        <Col span={5}>
          <Card>
            <Statistic
              title={t('page.wiki.health.noSourcePages')}
              value={counts.no_source_pages ?? 0}
              valueStyle={counts.no_source_pages ? { color: '#7cb305' } : undefined}
            />
          </Card>
        </Col>
      </Row>

      <Card
        title={t('page.wiki.health.gapsTitle')}
        size='small'
        extra={<Tooltip title={t('page.wiki.health.gapsHint')}>{t('page.wiki.health.hintTag')}</Tooltip>}
      >
        <Table
          columns={gapColumns}
          dataSource={report?.knowledge_gaps || []}
          loading={loading}
          pagination={{ pageSize: 5, hideOnSinglePage: true }}
          rowKey='name'
          size='small'
        />
      </Card>

      <Card title={t('page.wiki.health.flaggedTitle')} size='small'>
        <Table
          columns={flaggedColumns}
          dataSource={report?.flagged || []}
          loading={loading}
          pagination={{ pageSize: 10, hideOnSinglePage: true }}
          rowKey='id'
          size='small'
        />
      </Card>

      <Card title={t('page.wiki.health.hubsTitle')} size='small'>
        <Table
          columns={hubColumns}
          dataSource={report?.hubs || []}
          loading={loading}
          pagination={false}
          rowKey='id'
          size='small'
        />
      </Card>
    </div>
  );
}
