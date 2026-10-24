import { RotateCcw } from 'lucide-react';
import { SearchOutlined } from '@ant-design/icons';
import { Button, Card, Empty, Form, Input, Modal, Popconfirm, Space, Table, Tag, Typography, message } from 'antd';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';

import { request } from '@/service/request';

/**
 * FAQ entry management (W5): the compiled question documents of this
 * datasource — create (standard/similar/negative/answer), delete, and a
 * live test box against /query/_faq that shows the negative gate and the
 * exact direct answer. Negatives never index; the answer is returned,
 * not searched.
 */

interface FaqEntry {
  id: string;
  title: string;
  answer?: string;
  similar?: string[];
  negative?: string[];
}

interface FaqTestResult {
  exact: boolean;
  entry?: { id: string; title: string; answer?: string };
  hits?: { id: string; title: string; answer?: string }[];
}

export default function FaqManagement({ id: datasourceID }: { id: string }) {
  const { t } = useTranslation();
  const { hasAuth } = useAuth();
  const [items, setItems] = useState<FaqEntry[]>([]);
  const [loading, setLoading] = useState(false);
  const [createOpen, setCreateOpen] = useState(false);
  const [form] = Form.useForm();
  const [testQuery, setTestQuery] = useState('');
  const [testResult, setTestResult] = useState<FaqTestResult | null>(null);
  const [testing, setTesting] = useState(false);

  const canEdit = hasAuth('coco#document/create') && hasAuth('coco#document/update');

  const load = useCallback(async () => {
    if (!datasourceID) return;
    setLoading(true);
    try {
      const res: any = await request({
        method: 'post',
        url: '/document/_search',
        data: {
          query: { bool: { filter: [{ term: { 'source.id': datasourceID } }, { term: { type: 'faq' } }] } },
          size: 100,
          sort: [{ created: 'desc' }]
        }
      });
      const hits = res?.data?.hits?.hits ?? [];
      setItems(
        hits.map((h: any) => {
          const s = h._source;
          const faq = s?.metadata?.faq ?? {};
          return {
            id: h._id,
            title: s?.title ?? '',
            answer: faq.answer,
            similar: faq.similar ?? [],
            negative: faq.negative ?? []
          };
        })
      );
    } catch (e: any) {
      message.error(e?.response?.data?.message ?? e?.message ?? t('page.faq.loadFailed'));
    } finally {
      setLoading(false);
    }
  }, [datasourceID, t]);

  useEffect(() => {
    load();
  }, [load]);

  const createEntry = useCallback(async () => {
    const values = await form.validateFields();
    try {
      const res: any = await request({
        method: 'post',
        url: '/document/faq',
        data: {
          datasource_id: datasourceID,
          entries: [
            {
              standard: values.standard,
              similar: splitLines(values.similar),
              negative: splitLines(values.negative),
              answer: values.answer ?? ''
            }
          ]
        }
      });
      const first = res?.data?.created?.[0];
      if (first?.result === 'duplicate') {
        message.info(t('page.faq.duplicate'));
      } else {
        message.success(t('page.faq.created'));
      }
      setCreateOpen(false);
      form.resetFields();
      load();
    } catch (e: any) {
      message.error(e?.response?.data?.message ?? e?.message ?? t('page.faq.loadFailed'));
    }
  }, [datasourceID, form, load, t]);

  const removeEntry = useCallback(
    async (entryId: string) => {
      try {
        await request({ method: 'delete', url: `/document/${entryId}` });
        message.success(t('page.faq.deleted'));
        load();
      } catch (e: any) {
        message.error(e?.response?.data?.message ?? e?.message ?? t('page.faq.loadFailed'));
      }
    },
    [load, t]
  );

  const runTest = useCallback(async () => {
    if (!testQuery.trim()) return;
    setTesting(true);
    setTestResult(null);
    try {
      const res: any = await request({
        method: 'get',
        url: '/query/_faq',
        params: { q: testQuery.trim(), datasource: datasourceID }
      });
      setTestResult(res?.data ?? null);
    } catch (e: any) {
      message.error(e?.response?.data?.message ?? e?.message ?? t('page.faq.loadFailed'));
    } finally {
      setTesting(false);
    }
  }, [testQuery, datasourceID, t]);

  const columns = [
    { title: t('page.faq.standard'), dataIndex: 'title', ellipsis: true },
    {
      title: t('page.faq.answerCol'),
      dataIndex: 'answer',
      ellipsis: true,
      render: (v: string) => v || '-'
    },
    {
      title: t('page.faq.similarCol'),
      dataIndex: 'similar',
      width: 220,
      render: (v: string[]) =>
        v?.length ? (
          <Space size={4} wrap>
            {v.slice(0, 3).map((s, i) => (
              <Tag key={i}>{s}</Tag>
            ))}
            {v.length > 3 ? <Tag>+{v.length - 3}</Tag> : null}
          </Space>
        ) : (
          '-'
        )
    },
    {
      title: t('page.faq.negativeCol'),
      dataIndex: 'negative',
      width: 180,
      render: (v: string[]) =>
        v?.length ? (
          <Space size={4} wrap>
            {v.slice(0, 2).map((s, i) => (
              <Tag key={i} color="red">
                {s}
              </Tag>
            ))}
            {v.length > 2 ? <Tag color="red">+{v.length - 2}</Tag> : null}
          </Space>
        ) : (
          '-'
        )
    },
    {
      title: t('page.faq.actions'),
      key: 'actions',
      width: 90,
      render: (_: unknown, r: FaqEntry) =>
        canEdit ? (
          <Popconfirm title={t('page.faq.deleteConfirm')} onConfirm={() => removeEntry(r.id)}>
            <Button size="small" danger>
              {t('page.faq.delete')}
            </Button>
          </Popconfirm>
        ) : null
    }
  ];

  return (
    <div className="p-16px flex flex-col gap-16px">
      <Card
        size="small"
        title={t('page.faq.title')}
        extra={
          <Space>
            <Button size="small" icon={<RotateCcw className="w-14px h-14px" />} onClick={load} loading={loading}>
              {t('page.faq.refresh')}
            </Button>
            {canEdit ? (
              <Button size="small" type="primary" onClick={() => setCreateOpen(true)}>
                {t('page.faq.newEntry')}
              </Button>
            ) : null}
          </Space>
        }
      >
        <Typography.Text type="secondary" className="text-12px">
          {t('page.faq.hint')}
        </Typography.Text>
        <Table<FaqEntry>
          className="mt-8px"
          rowKey="id"
          size="small"
          loading={loading}
          columns={columns}
          dataSource={items}
          pagination={{ pageSize: 10, hideOnSinglePage: true }}
          locale={{ emptyText: <Empty description={t('page.faq.empty')} /> }}
        />
      </Card>

      <Card size="small" title={t('page.faq.testTitle')}>
        <Space.Compact style={{ width: '100%' }}>
          <Input
            placeholder={t('page.faq.testPlaceholder')}
            value={testQuery}
            onChange={(e) => setTestQuery(e.target.value)}
            onPressEnter={runTest}
          />
          <Button type="primary" icon={<SearchOutlined />} loading={testing} onClick={runTest}>
            {t('page.faq.testRun')}
          </Button>
        </Space.Compact>
        {testResult ? (
          <div className="mt-12px">
            {testResult.exact ? (
              <div className="flex flex-col gap-4px">
                <Tag color="green">{t('page.faq.exactHit')}</Tag>
                <Typography.Text strong>{testResult.entry?.title}</Typography.Text>
                <Typography.Text>{testResult.entry?.answer ?? '-'}</Typography.Text>
              </div>
            ) : (testResult.hits?.length ?? 0) > 0 ? (
              <div className="flex flex-col gap-4px">
                <Tag>{t('page.faq.rankedHits', { n: String(testResult.hits!.length) })}</Tag>
                {testResult.hits!.slice(0, 5).map((h) => (
                  <Typography.Text key={h.id}>· {h.title}</Typography.Text>
                ))}
              </div>
            ) : (
              <Tag>{t('page.faq.noHit')}</Tag>
            )}
          </div>
        ) : null}
      </Card>

      <Modal
        title={t('page.faq.newEntry')}
        open={createOpen}
        onOk={createEntry}
        onCancel={() => setCreateOpen(false)}
        okText={t('page.faq.createOk')}
      >
        <Form form={form} layout="vertical">
          <Form.Item name="standard" label={t('page.faq.standard')} rules={[{ required: true }]}>
            <Input placeholder={t('page.faq.standardPlaceholder')} />
          </Form.Item>
          <Form.Item name="similar" label={t('page.faq.similarCol')} tooltip={t('page.faq.linesHint')}>
            <Input.TextArea rows={2} placeholder={t('page.faq.linesPlaceholder')} />
          </Form.Item>
          <Form.Item name="negative" label={t('page.faq.negativeCol')} tooltip={t('page.faq.negativeHint')}>
            <Input.TextArea rows={2} placeholder={t('page.faq.linesPlaceholder')} />
          </Form.Item>
          <Form.Item name="answer" label={t('page.faq.answerCol')}>
            <Input.TextArea rows={3} />
          </Form.Item>
        </Form>
      </Modal>
    </div>
  );
}

function splitLines(v?: string): string[] {
  if (!v) return [];
  return v
    .split('\n')
    .map((s) => s.trim())
    .filter(Boolean);
}
