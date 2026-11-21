import { RobotOutlined } from '@ant-design/icons';
import { Alert, Button, Input, Modal, Space, Spin } from 'antd';
import { useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { editWikiArticleAI } from '@/service/api';

/** instruction-based AI edit (design doc C4/D1c): streams the rewrite of
 * the whole article or a selected fragment; the applied result lands as a
 * new ai-generated version while the status machine stays untouched. */
export function AIEditModal({
  articleId,
  open,
  onClose,
  onApplied
}: {
  articleId: string;
  open: boolean;
  onClose: () => void;
  onApplied: () => void;
}) {
  const { t } = useTranslation();
  const [instruction, setInstruction] = useState('');
  const [selection, setSelection] = useState('');
  const [running, setRunning] = useState(false);
  const [streamText, setStreamText] = useState('');
  const [error, setError] = useState('');
  const [applied, setApplied] = useState<{ version?: number } | null>(null);
  const abortRef = useRef<AbortController | null>(null);

  const reset = () => {
    setStreamText('');
    setError('');
    setApplied(null);
  };

  const start = () => {
    if (!instruction.trim()) return;
    reset();
    setRunning(true);
    abortRef.current = new AbortController();
    editWikiArticleAI(
      articleId,
      { instruction: instruction.trim(), selection: selection.trim() || undefined },
      ev => {
        if (ev.chunk?.text) setStreamText(prev => prev + ev.chunk!.text);
        if (ev.done) {
          setRunning(false);
          if (ev.done.reason) {
            setError(ev.done.reason);
          } else {
            setApplied({ version: ev.done.version });
            onApplied();
          }
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
              <Button disabled={!instruction.trim()} icon={<RobotOutlined />} type="primary" onClick={start}>
                {t('page.wiki.aiEdit.start')}
              </Button>
            </>
          )}
        </Space>
      }
      open={open}
      title={
        <Space>
          <RobotOutlined />
          {t('page.wiki.aiEdit.title')}
        </Space>
      }
      width={720}
      onCancel={() => {
        stop();
        onClose();
      }}
    >
      <div className="flex flex-col gap-4">
        <div>
          <div className="mb-1 font-medium">{t('page.wiki.aiEdit.instruction')}</div>
          <Input.TextArea
            disabled={running}
            placeholder={t('page.wiki.aiEdit.instructionPlaceholder')}
            rows={2}
            value={instruction}
            onChange={e => setInstruction(e.target.value)}
          />
        </div>
        <div>
          <div className="mb-1 font-medium">{t('page.wiki.aiEdit.selection')}</div>
          <Input.TextArea
            disabled={running}
            placeholder={t('page.wiki.aiEdit.selectionPlaceholder')}
            rows={2}
            value={selection}
            onChange={e => setSelection(e.target.value)}
          />
        </div>

        {error && <Alert showIcon type="error" message={error} />}
        {applied && (
          <Alert showIcon type="success" message={t('page.wiki.aiEdit.done', { version: applied.version ?? '' })} />
        )}

        {(running || streamText) && (
          <div>
            <div className="mb-1 font-medium">{t('page.wiki.aiEdit.stream')}</div>
            <Spin spinning={running}>
              <pre className="max-h-360px overflow-auto rounded border border-solid p-3 font-mono text-xs whitespace-pre-wrap" style={{ borderColor: 'var(--ant-color-border)' }}>
                {streamText}
              </pre>
            </Spin>
          </div>
        )}
      </div>
    </Modal>
  );
}
