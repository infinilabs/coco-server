import { CheckOutlined, CloseOutlined, SafetyCertificateOutlined } from '@ant-design/icons';
import { Badge, Button, Card, Empty, List, Popconfirm, Segmented, Space, Tag, Tooltip } from 'antd';
import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { useTranslation } from 'react-i18next';
import { searchWikiGovernance, updateWikiGovernanceStatus, type GovernanceProposal } from '@/service/api';
import { WikiShell } from '../components/WikiShell';

const TYPE_COLOR: Record<string, string> = {
  stale: 'warning',
  duplicate: 'purple',
  conflict: 'error',
  low_quality: 'default',
  orphan: 'cyan'
};

/**
 * Knowledge-governance queue: the background scanner files proposals (stale
 * pages, duplicates, conflicts, low quality, orphans) and this page is the
 * human gate — every fix happens on the article face, marking resolved or
 * dismissing only records the decision (D1).
 */
export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  const [status, setStatus] = useState<string>('open');
  const [proposals, setProposals] = useState<GovernanceProposal[]>([]);
  const [loading, setLoading] = useState(true);

  const fetchData = (st: string = status) => {
    setLoading(true);
    searchWikiGovernance({ status: st === 'all' ? undefined : st }).then(res => {
      setProposals(((res as any)?.data || []) as GovernanceProposal[]);
      setLoading(false);
    });
  };

  useEffect(() => {
    fetchData(status);
  }, [status]);

  const act = (proposal: GovernanceProposal, to: string) => {
    updateWikiGovernanceStatus(proposal.id, to).then(() => {
      window.$message?.success(t('common.updateSuccess'));
      fetchData();
    });
  };

  const goArticle = (proposal: GovernanceProposal) =>
    nav(`/wiki/article/${proposal.article_id}?kb=${proposal.kb_id}`);

  const openCount = proposals.length;

  return (
    <WikiShell>
      <Card
        bordered={false}
        className="card-wrapper"
        title={
          <Space>
            <SafetyCertificateOutlined />
            <span>{t('page.wiki.governance.title')}</span>
          </Space>
        }
        extra={
          <Segmented
            value={status}
            onChange={v => setStatus(v as string)}
            options={[
              { value: 'open', label: t('page.wiki.governance.statusOpen') },
              { value: 'resolved', label: t('page.wiki.governance.statusResolved') },
              { value: 'dismissed', label: t('page.wiki.governance.statusDismissed') },
              { value: 'all', label: t('page.wiki.governance.statusAll') }
            ]}
          />
        }
      >
        {t('page.wiki.governance.subtitle')}
        <List
          className="mt-4"
          dataSource={proposals}
          loading={loading}
          locale={{ emptyText: <Empty description={t('page.wiki.governance.empty')} image={Empty.PRESENTED_IMAGE_SIMPLE} /> }}
          pagination={{ pageSize: 10, hideOnSinglePage: true }}
          renderItem={proposal => {
            const dupOf = proposal.evidence?.duplicate_of;
            return (
              <List.Item
                actions={[
                  <Popconfirm
                    key="resolve"
                    title={t('page.wiki.governance.resolveConfirm')}
                    onConfirm={() => act(proposal, 'resolved')}
                  >
                    <Button icon={<CheckOutlined />} size="small" type="primary" ghost>
                      {t('page.wiki.governance.resolve')}
                    </Button>
                  </Popconfirm>,
                  <Popconfirm
                    key="dismiss"
                    title={t('page.wiki.governance.dismissConfirm')}
                    onConfirm={() => act(proposal, 'dismissed')}
                  >
                    <Button icon={<CloseOutlined />} size="small">
                      {t('page.wiki.governance.dismiss')}
                    </Button>
                  </Popconfirm>
                ]}
                className="cursor-pointer"
                onClick={() => goArticle(proposal)}
              >
                <List.Item.Meta
                  description={
                    <Space direction="vertical" size={2}>
                      <span>{proposal.reason}</span>
                      {dupOf && (
                        <Tooltip title={t('page.wiki.governance.duplicateOf')}>
                          <Tag
                            className="cursor-pointer"
                            color="purple"
                            onClick={e => {
                              e.stopPropagation();
                              nav(`/wiki/article/${dupOf.id}?kb=${proposal.kb_id}`);
                            }}
                          >
                            {String(dupOf.title || dupOf.id)}
                          </Tag>
                        </Tooltip>
                      )}
                      {proposal.evidence?.pending_version ? (
                        <span className="text-xs text-gray-400">
                          {t('page.wiki.governance.pendingVersion', { version: String(proposal.evidence.pending_version) })}
                        </span>
                      ) : null}
                    </Space>
                  }
                  title={
                    <Space>
                      <Badge count={status === 'open' ? '!' : 0} offset={[10, 0]}>
                        <Tag color={TYPE_COLOR[proposal.type] || 'default'}>{t(`page.wiki.governance.type.${proposal.type}`)}</Tag>
                      </Badge>
                      <a>{proposal.article_title || proposal.article_id}</a>
                    </Space>
                  }
                />
              </List.Item>
            );
          }}
        />
        {status === 'open' && openCount > 0 && (
          <div className="mt-2 text-xs text-gray-400">{t('page.wiki.governance.hint', { count: String(openCount) })}</div>
        )}
      </Card>
    </WikiShell>
  );
}
