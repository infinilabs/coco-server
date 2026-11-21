/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

import { DatabaseOutlined, InboxOutlined, PaperClipOutlined } from '@ant-design/icons';
import { Button, Card, List, Progress, Table, Tag, Upload, message } from 'antd';
import { useCallback, useEffect, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import {
  cancelWikiCompile,
  getWikiCompileJob,
  getWikiKbFiles,
  startWikiCompile,
  uploadWikiKbFiles,
  type WikiCompileJob
} from '@/service/api';
import { useAuth } from '@/hooks/business/auth';

interface KbFileAttachment {
  id: string;
  name: string;
  size?: number;
  created?: string;
}

interface KbFileOwner {
  id: string;
  title: string;
  owner_type: 'article' | 'document';
  source_name?: string;
  uploaded?: boolean;
  status?: string;
  articles?: { id: string; title: string }[];
}

interface KbFileRow {
  id: string;
  name: string;
  size?: number;
  mime_type?: string;
  created?: string;
  owners: KbFileOwner[];
}

const STATUS_COLOR: Record<string, string> = {
  completed: 'green',
  indexing: 'blue',
  failed: 'red'
};

/** KB-level raw material: every document behind the knowledge (bound
 * datasources + manual uploads), its attachments, and the articles that
 * cite it — plus the upload entry that feeds the pipeline by hand. */
export function KbFileManager({
  kbId,
  datasources
}: {
  kbId: string;
  datasources: Api.Wiki.DatasourceInfo[];
}) {
  const { t } = useTranslation();
  const nav = useNavigate();
  const canUpdate = useAuth().hasAuth('coco#wiki_kb/update');
  const [files, setFiles] = useState<KbFileRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [uploading, setUploading] = useState(false);

  const fetchData = useCallback(() => {
    setLoading(true);
    getWikiKbFiles(kbId)
      .then(res => setFiles(((res as any)?.attachments || []) as KbFileRow[]))
      .catch(() => setFiles([]))
      .finally(() => setLoading(false));
  }, [kbId]);

  // compile console: one job per KB, polled while running
  const [job, setJob] = useState<WikiCompileJob | null>(null);
  const [compileLoading, setCompileLoading] = useState(false);
  const pollRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const stopPoll = () => {
    if (pollRef.current) {
      clearInterval(pollRef.current);
      pollRef.current = null;
    }
  };
  const poll = useCallback(() => {
    getWikiCompileJob(kbId).then((p: any) => {
      if (!p || p.status === 'idle') return;
      setJob(p);
      if (p.status === 'running') return;
      stopPoll();
      fetchData();
    }).catch(() => {});
  }, [kbId, fetchData]);
  useEffect(() => {
    // re-attach to a running compile after navigation
    getWikiCompileJob(kbId).then((p: any) => {
      if (p?.status === 'running') {
        setJob(p);
        pollRef.current = setInterval(poll, 2000);
      }
    }).catch(() => {});
    return stopPoll;
  }, [kbId]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

  const startCompile = () => {
    setCompileLoading(true);
    startWikiCompile(kbId)
      .then(() => {
        setCompileLoading(false);
        pollRef.current = setInterval(poll, 2000);
        poll();
      })
      .catch((err: any) => {
        setCompileLoading(false);
        message.error(String(err)?.includes('already running') ? t('page.wiki.fileManager.compileAlready') : t('page.wiki.fileManager.compileStartFailed'));
      });
  };

  const cancelCompileRun = () => {
    cancelWikiCompile(kbId).then(() => poll()).catch(() => {});
  };

  const doUpload = async (fileList: any[]) => {
    if (!fileList.length) return;
    setUploading(true);
    try {
      const res = await uploadWikiKbFiles(kbId, fileList);
      const created = (res as any)?.created ?? 0;
      const failed = ((res as any)?.failed || []) as { file: string; error: string }[];
      if (failed.length) {
        message.warning(t('page.wiki.fileManager.uploadPartial', { created, failed: failed.length }));
      } else {
        message.success(t('page.wiki.fileManager.uploadDone', { count: created }));
      }
      fetchData();
    } catch {
      message.error(t('page.wiki.fileManager.uploadFailed'));
    } finally {
      setUploading(false);
    }
  };

  const columns: any[] = [
    {
      title: t('page.wiki.fileManager.columns.title'),
      dataIndex: 'name',
      render: (_: any, row: KbFileRow) => (
        <a
          className="flex items-center gap-6px text-13px"
          href={`/attachment/${row.id}`}
          rel="noreferrer"
          target="_blank"
        >
          <PaperClipOutlined />
          {row.name}
        </a>
      )
    },
    {
      title: t('page.wiki.fileManager.columns.size'),
      dataIndex: 'size',
      width: 90,
      render: (v: number) => (v ? (v < 1024 ? '<1 KB' : `${(v / 1024).toFixed(0)} KB`) : '—')
    },
    {
      title: t('page.wiki.fileManager.columns.created'),
      dataIndex: 'created',
      width: 110
    },
    {
      title: t('page.wiki.fileManager.columns.hangFrom'),
      key: 'owners',
      render: (_: any, row: KbFileRow) => (
        <div className="flex flex-col gap-2px">
          {(row.owners || []).map((o, i) => (
            <div className="flex flex-wrap items-center gap-4px text-12px" key={`${o.id}-${i}`}>
              {o.owner_type === 'article' ? (
                <a className="text-[var(--ant-color-link)]" onClick={() => nav(`/wiki/article/${o.id}?kb=${kbId}`)}>
                  {o.title || o.id}
                </a>
              ) : (
                <a className="text-[var(--ant-color-link)]" onClick={() => nav(`#/preview/document/${o.id}`)}>
                  {o.title || o.id}
                </a>
              )}
              {o.source_name && <Tag className="m-0">{o.source_name}</Tag>}
              {o.uploaded && <Tag className="m-0" color="purple">{t('page.wiki.fileManager.manualUpload')}</Tag>}
            </div>
          ))}
        </div>
      )
    },
    {
      title: t('page.wiki.fileManager.columns.articles'),
      key: 'articles',
      width: 200,
      render: (_: any, row: KbFileRow) => {
        const arts: { id: string; title: string }[] = [];
        (row.owners || []).forEach(o => (o.articles || []).forEach(a => arts.push(a)));
        if (!arts.length) return <span className="text-12px text-[var(--ant-color-text-tertiary)]">—</span>;
        const uniq = arts.filter((a, i) => arts.findIndex(x => x.id === a.id) === i);
        return (
          <div className="flex flex-col gap-2px">
            {uniq.map(a => (
              <a className="text-12px text-[var(--ant-color-link)]" key={a.id} onClick={() => nav(`/wiki/article/${a.id}?kb=${kbId}`)}>
                {a.title}
              </a>
            ))}
          </div>
        );
      }
    }
  ];

  return (
    <Card
      className="card-wrapper"
      title={
        <span className="flex items-center gap-8px">
          {t('page.wiki.fileManager.title')}
          <span className="text-13px font-normal text-[var(--ant-color-text-tertiary)]">
            {t('common.totalItems', { total: files.length })}
          </span>
        </span>
      }
    >
      {/* compile console: trigger, progress, logs, cancel */}
      <Card
        className="card-wrapper mb-12px"
        size="small"
        title={t('page.wiki.fileManager.compileTitle')}
        extra={
          job?.status === 'running' ? (
            <Button danger size="small" onClick={cancelCompileRun}>
              {t('page.wiki.fileManager.compileCancel')}
            </Button>
          ) : (
            <Button loading={compileLoading} size="small" type="primary" onClick={startCompile}>
              {t('page.wiki.fileManager.compileStart')}
            </Button>
          )
        }
      >
        {job && job.status !== 'idle' ? (
          <div className="flex flex-col gap-6px">
            <div className="flex flex-wrap items-center gap-10px text-12.5px">
              <Tag color={job.status === 'running' ? 'processing' : job.status === 'done' ? 'green' : job.status === 'failed' ? 'red' : 'default'}>
                {t(`page.wiki.fileManager.compileStatus.${job.status}`)}
              </Tag>
              <span>{t('page.wiki.fileManager.compileStage', { stage: t(`page.wiki.fileManager.compileStages.${job.stage}`, { defaultValue: job.stage || '-' }) })}</span>
              {!!job.incremental && (
                <Tag color='cyan'>
                  {t('page.wiki.fileManager.compileIncremental', { ops: job.ops || 0, pages: job.affected_pages || 0 })}
                </Tag>
              )}
              <span>
                {t('page.wiki.fileManager.compileBatches', {
                  done: job.batches_done || 0,
                  total: job.batches || 0,
                  cached: job.batches_cached || 0,
                  failed: job.batches_failed || 0
                })}
              </span>
              {!!job.topics && (
                <span>{t('page.wiki.fileManager.compileTopics', { done: job.topics_done || 0, total: job.topics || 0 })}</span>
              )}
              <span>{t('page.wiki.fileManager.compileCounts', { docs: job.docs || 0, extracts: job.extracts || 0, reduced: job.reduced || 0, proposals: job.proposals || 0 })}</span>
            </div>
            {job.status === 'running' && (
              <Progress
                percent={
                  job.stage === 'compose' && job.topics
                    ? Math.round(((job.topics_done || 0) / job.topics) * 100)
                    : job.batches
                      ? Math.round(((job.batches_done || 0) / job.batches) * 100)
                      : 0
                }
                size="small"
                status="active"
              />
            )}
            {job.current && job.status === 'running' && (
              <div className="text-12px text-[var(--ant-color-text-tertiary)]">{t('page.wiki.fileManager.compileCurrent', { current: job.current })}</div>
            )}
            {(job.logs || []).length > 0 && (
              <div
                className="max-h-160px overflow-auto rounded-4px border border-solid px-8px py-4px font-mono text-11px leading-16px"
                style={{ borderColor: 'var(--ant-color-border)', background: 'var(--ant-color-bg-layout)' }}
              >
                {(job.logs || []).map((l, i) => (
                  <div key={i}>{l}</div>
                ))}
              </div>
            )}
          </div>
        ) : (
          <div className="text-12.5px text-[var(--ant-color-text-tertiary)]">{t('page.wiki.fileManager.compileIdle')}</div>
        )}
      </Card>
      {/* bound datasources: the synced half of the raw material */}
      <Card
        className="card-wrapper mb-12px"
        size="small"
        title={
          <span className="flex items-center gap-8px">
            <DatabaseOutlined />
            {t('page.wiki.fileManager.datasources')}
          </span>
        }
      >
        <List
          className="wiki-row-list"
          dataSource={datasources}
          locale={{ emptyText: t('page.wiki.fileManager.noDatasources') }}
          renderItem={ds => (
            <List.Item
              actions={[
                <Tag
                  color={ds.status === 'connected' ? 'green' : ds.status === 'syncing' ? 'blue' : ds.status === 'error' ? 'red' : 'default'}
                  key="status"
                >
                  {ds.status}
                </Tag>
              ]}
            >
              <List.Item.Meta
                title={ds.name}
                description={`${ds.document_count} ${t('page.wiki.kb.docsUnit')} · ${t('page.wiki.kb.lastSynced')} ${ds.last_synced}`}
              />
            </List.Item>
          )}
        />
      </Card>
      {canUpdate && (
        <div
          className="mb-12px flex items-center justify-between gap-10px rounded-6px border border-dashed border-solid px-12px py-8px"
          style={{ borderColor: 'var(--ant-color-border)' }}
        >
          <span className="text-12.5px text-[var(--ant-color-text-tertiary)]">
            {t('page.wiki.fileManager.uploadHint')}
          </span>
          <Upload
            beforeUpload={() => false}
            multiple
            showUploadList={false}
            onChange={({ fileList }) => {
              const pending = fileList.map(f => f.originFileObj).filter(Boolean) as any[];
              if (pending.length && !uploading) {
                doUpload(pending);
                fileList.length = 0;
              }
            }}
          >
            <Button icon={<InboxOutlined />} loading={uploading} size="small" type="primary">
              {t('page.wiki.fileManager.upload')}
            </Button>
          </Upload>
        </div>
      )}
      <Table
        columns={columns}
        dataSource={files}
        loading={loading}
        pagination={{ pageSize: 20, hideOnSinglePage: true, showSizeChanger: true }}
        rowKey="id"
        size="small"
      />
    </Card>
  );
}
