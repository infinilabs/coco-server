import { CheckCircleOutlined, CloseCircleOutlined, LoadingOutlined, RobotOutlined } from '@ant-design/icons';
import { Alert, Button, Input, List, Modal, Progress, Space, Tag, Typography } from 'antd';
import { useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { generateWikiKb } from '@/service/api';

const PHASES = ['scope', 'cluster', 'outline', 'draft', 'deliver'] as const;

const PHASE_ICON: Record<string, React.ReactNode> = {
  done: <CheckCircleOutlined className="text-[var(--ant-color-success)]" />,
  active: <LoadingOutlined />,
  pending: <span className="inline-block h-14px w-14px rounded-full border border-solid" />
};

/** KM agent generation panel (design doc D1c/C3): drives the SSE contract
 * of POST /wiki/kb/:id/ai/generate — phases, per-article events and the
 * done summary. Generated pages land as drafts; review stays manual. */
export function GenerateModal({
  kbId,
  open,
  onClose,
  onGenerated,
  onOpenArticle
}: {
  kbId: string;
  open: boolean;
  onClose: () => void;
  onGenerated: () => void;
  onOpenArticle: (id: string) => void;
}) {
  const { t } = useTranslation();
  const [hint, setHint] = useState('');
  const [running, setRunning] = useState(false);
  const [activePhase, setActivePhase] = useState<string>('');
  const [percent, setPercent] = useState(0);
  const [articles, setArticles] = useState<Api.Wiki.Article[]>([]);
  const [done, setDone] = useState<{ generated?: number; failed?: string[]; reason?: string } | null>(null);
  const [error, setError] = useState('');
  const abortRef = useRef<AbortController | null>(null);

  const phaseState = (phase: string): 'pending' | 'active' | 'done' => {
    const idx = PHASES.indexOf(phase as (typeof PHASES)[number]);
    const activeIdx = PHASES.indexOf(activePhase as (typeof PHASES)[number]);
    if (activeIdx < 0 || idx < activeIdx) return 'done';
    if (idx === activeIdx) return 'active';
    return 'pending';
  };

  const reset = () => {
    setActivePhase('');
    setPercent(0);
    setArticles([]);
    setDone(null);
    setError('');
  };

  const start = () => {
    reset();
    setRunning(true);
    abortRef.current = new AbortController();
    generateWikiKb(
      kbId,
      { hint: hint.trim() || undefined },
      ev => {
        if (ev.progress) {
          setActivePhase(ev.progress.phase);
          setPercent(Math.round((ev.progress.progress || 0) * 100));
        }
        if (ev.article) setArticles(prev => [...prev, ev.article!]);
        if (ev.done) {
          setDone(ev.done);
          setRunning(false);
          setPercent(100);
          if ((ev.done.generated || 0) > 0) onGenerated();
        }
      },
      abortRef.current.signal
    ).catch(err => {
      if (err?.name !== 'AbortError') setError(err?.message || String(err));
      setRunning(false);
    });
  };

  const stop = () => {
    abortRef.current?.abort();
    setRunning(false);
  };

  return (
    <Modal
      footer={
        <Space>
          {running ? (
            <Button danger onClick={stop}>
              {t('page.wiki.generate.cancel')}
            </Button>
          ) : (
            <>
              <Button onClick={onClose}>{t('common.close')}</Button>
              <Button icon={<RobotOutlined />} type="primary" onClick={start}>
                {t('page.wiki.generate.start')}
              </Button>
            </>
          )}
        </Space>
      }
      okText={t('common.confirm')}
      open={open}
      title={
        <Space>
          <RobotOutlined />
          {t('page.wiki.generate.title')}
        </Space>
      }
      onCancel={() => {
        stop();
        onClose();
      }}
    >
      <div className="flex flex-col gap-4">
        <Input.TextArea
          disabled={running}
          placeholder={t('page.wiki.generate.hintPlaceholder')}
          rows={2}
          value={hint}
          onChange={e => setHint(e.target.value)}
        />

        {(running || done || activePhase) && (
          <>
            <div className="flex flex-wrap gap-x-4 gap-y-1">
              {PHASES.map(phase => (
                <Space key={phase} size={4}>
                  {PHASE_ICON[phaseState(phase)]}
                  <span
                    className={
                      phaseState(phase) === 'pending'
                        ? 'text-xs text-gray-400'
                        : phaseState(phase) === 'active'
                          ? 'text-xs font-medium'
                          : 'text-xs'
                    }
                  >
                    {t(`page.wiki.generate.phase.${phase}`)}
                  </span>
                </Space>
              ))}
            </div>
            <Progress percent={percent} size="small" status={error ? 'exception' : undefined} />
          </>
        )}

        {error && <Alert showIcon type="error" message={error} />}

        {done?.reason && <Alert showIcon type="warning" message={done.reason} />}
        {done && !done.reason && (
          <Alert
            showIcon
            type={(done.generated || 0) > 0 ? 'success' : 'warning'}
            message={t('page.wiki.generate.done', { count: done.generated || 0 })}
          />
        )}
        {done?.failed && done.failed.length > 0 && (
          <div>
            <div className="mb-1 text-xs text-gray-400">{t('page.wiki.generate.failed')}</div>
            <Space size={4} wrap>
              {done.failed.map(title => (
                <Tag color="error" key={title}>
                  <CloseCircleOutlined className="mr-1" />
                  {title}
                </Tag>
              ))}
            </Space>
          </div>
        )}

        {articles.length > 0 && (
          <div>
            <div className="mb-1 font-medium">{t('page.wiki.generate.generatedArticles')}</div>
            <List
              dataSource={articles}
              renderItem={article => (
                <List.Item
                  className="cursor-pointer"
                  onClick={() => {
                    onOpenArticle(article.id);
                  }}
                >
                  <List.Item.Meta
                    description={
                      <Space size={4} wrap>
                        <Tag color="purple">{t('page.wiki.status.draft')}</Tag>
                        {article.confidence && (
                          <Typography.Text className="text-xs" type="secondary">
                            {t(`page.wiki.confidence.${article.confidence}`)}
                          </Typography.Text>
                        )}
                      </Space>
                    }
                    title={article.title}
                  />
                </List.Item>
              )}
            />
          </div>
        )}
      </div>
    </Modal>
  );
}
