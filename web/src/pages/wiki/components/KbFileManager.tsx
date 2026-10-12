/* Copyright © INFINI LTD. All rights reserved.
 * Web: https://infini.ltd */

import { InboxOutlined, PaperClipOutlined } from '@ant-design/icons';
import { Button, Card, Table, Tag, Upload, message } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { useNavigate } from 'react-router-dom';

import { getWikiKbFiles, uploadWikiKbFiles } from '@/service/api';
import { useAuth } from '@/hooks/business/auth';

interface KbFileAttachment {
  id: string;
  name: string;
  size?: number;
  created?: string;
}

interface KbFileRow {
  id: string;
  title: string;
  source_name?: string;
  uploaded?: boolean;
  size?: number;
  status?: string;
  processed?: boolean;
  updated?: string;
  attachments?: KbFileAttachment[];
  articles?: { id: string; title: string }[];
}

const STATUS_COLOR: Record<string, string> = {
  completed: 'green',
  indexing: 'blue',
  failed: 'red'
};

/** KB-level raw material: every document behind the knowledge (bound
 * datasources + manual uploads), its attachments, and the articles that
 * cite it — plus the upload entry that feeds the pipeline by hand. */
export function KbFileManager({ kbId }: { kbId: string }) {
  const { t } = useTranslation();
  const nav = useNavigate();
  const canUpdate = useAuth().hasAuth('coco#wiki_kb/update');
  const [files, setFiles] = useState<KbFileRow[]>([]);
  const [loading, setLoading] = useState(true);
  const [uploading, setUploading] = useState(false);

  const fetchData = useCallback(() => {
    setLoading(true);
    getWikiKbFiles(kbId)
      .then(res => setFiles(((res as any)?.files || []) as KbFileRow[]))
      .catch(() => setFiles([]))
      .finally(() => setLoading(false));
  }, [kbId]);

  useEffect(() => {
    fetchData();
  }, [fetchData]);

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
      dataIndex: 'title',
      render: (_: any, row: KbFileRow) => (
        <a
          className="text-13px"
          onClick={() => nav(`#/preview/document/${row.id}`)}
        >
          {row.title}
        </a>
      )
    },
    {
      title: t('page.wiki.fileManager.columns.source'),
      dataIndex: 'source_name',
      width: 150,
      render: (v: string, row: KbFileRow) =>
        row.uploaded ? <Tag color="purple">{t('page.wiki.fileManager.manualUpload')}</Tag> : v || '—'
    },
    {
      title: t('page.wiki.fileManager.columns.status'),
      dataIndex: 'status',
      width: 100,
      render: (v: string, row: KbFileRow) =>
        v ? <Tag color={STATUS_COLOR[v] || 'default'}>{v}</Tag> : row.processed ? <Tag color="green">✓</Tag> : '—'
    },
    {
      title: t('page.wiki.fileManager.columns.attachments'),
      key: 'attachments',
      width: 90,
      render: (_: any, row: KbFileRow) => (row.attachments || []).length
    },
    {
      title: t('page.wiki.fileManager.columns.updated'),
      dataIndex: 'updated',
      width: 130
    },
    {
      title: t('page.wiki.fileManager.columns.articles'),
      key: 'articles',
      render: (_: any, row: KbFileRow) =>
        (row.articles || []).length === 0 ? (
          <span className="text-12px text-[var(--ant-color-text-tertiary)]">—</span>
        ) : (
          <div className="flex flex-col gap-2px">
            {row.articles!.map(a => (
              <a
                className="text-12px"
                key={a.id}
                onClick={() => nav(`/wiki/article/${a.id}?kb=${kbId}`)}
              >
                {a.title}
              </a>
            ))}
          </div>
        )
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
        pagination={{ pageSize: 20, hideOnSinglePage: true }}
        rowKey="id"
        size="small"
        expandable={{
          rowExpandable: row => (row.attachments || []).length > 0,
          expandedRowRender: row => (
            <div className="flex flex-col gap-4px pl-24px">
              {(row.attachments || []).map(att => (
                <a
                  className="flex items-center gap-6px text-12px text-[var(--ant-color-link)]"
                  href={`/attachment/${att.id}`}
                  key={att.id}
                  rel="noreferrer"
                  target="_blank"
                >
                  <PaperClipOutlined />
                  {att.name}
                  {att.size ? ` (${att.size < 1024 ? '<1' : (att.size / 1024).toFixed(0)} KB)` : ''}
                  {att.created ? ` · ${att.created}` : ''}
                </a>
              ))}
            </div>
          )
        }}
      />
    </Card>
  );
}
