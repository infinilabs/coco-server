import { Button, Typography } from 'antd';
import { ExternalLink, FileWarning } from 'lucide-react';
import { type FC } from 'react';
import { useTranslation } from 'react-i18next';

import Markdown from '@/components/DocumentDrawer/Markdown';

interface CapturedContentProps {
  readonly data: any;
  /** the document's original external URL, when the raw_content probe returned one */
  readonly sourceUrl?: string;
  /** non-fatal raw_content probe error — the captured fields below still render */
  readonly fetchWarning?: string;
}

// External-link documents (RSS / Hugo / web_page) never stream a file: the
// raw_content endpoint answers with a redirect. When the connector captured a
// body (content) or at least a summary, show it here instead of a bare
// "leave" card — content wins, summary is the fallback — with the original
// link riding along so the user can still jump out.
export const CapturedContent: FC<CapturedContentProps> = ({ data, sourceUrl, fetchWarning }) => {
  const { t } = useTranslation();

  const content: string = data?.content || '';
  const summary: string = content ? '' : data?.summary || '';
  const sourceName: string = data?.source?.name || '';

  return (
    <div className='h-full overflow-auto'>
      <div className='mx-auto w-full max-w-860 px-24px py-24px'>
        {fetchWarning && (
          <div className='mb-16px flex items-start gap-8px rounded-lg border border-[#F0D48A] bg-[#FDF6E3] p-12px text-12px leading-relaxed text-[#8C6D1F] dark:border-[#5C4A1E] dark:bg-[#3A3018] dark:text-[#E3C77A]'>
            <FileWarning className='mt-2px h-14px w-14px shrink-0' />
            <div>
              <div>{t('page.preview.hints.originalUnavailable')}</div>
              <div className='opacity-70'>{fetchWarning}</div>
            </div>
          </div>
        )}

        <Typography.Title className='!mb-4px' level={4}>
          {data?.title}
        </Typography.Title>

        {sourceUrl && (
          <div className='mb-8px flex flex-wrap items-center gap-x-12px gap-y-4px text-13px'>
            {sourceName && <span className='color-[var(--ant-color-text-secondary)]'>{sourceName}</span>}
            <Typography.Text className='!text-12px break-all' copyable ellipsis>
              {sourceUrl}
            </Typography.Text>
          </div>
        )}

        <div className='my-12px h-1px bg-[#EDEDED] dark:bg-[#303030]' />

        {content ? (
          <>
            {sourceUrl && (
              <div className='mb-8px'>
                <Button
                  icon={<ExternalLink size={14} />}
                  onClick={() => {
                    window.open(sourceUrl);
                  }}
                >
                  {t('page.preview.buttons.viewOriginal')}
                </Button>
              </div>
            )}
            <div className='text-14px leading-relaxed'>
              <Markdown content={content} />
            </div>
          </>
        ) : (
          <div className='text-14px leading-relaxed'>
            <div className='mb-8px color-[var(--ant-color-text-secondary)]'>
              {t('page.preview.hints.capturedSummary')}
            </div>
            <Markdown content={summary} />
            {sourceUrl && (
              <div className='mt-16px'>
                <Button
                  icon={<ExternalLink size={14} />}
                  type='primary'
                  onClick={() => {
                    window.open(sourceUrl);
                  }}
                >
                  {t('page.preview.buttons.viewOriginal')}
                </Button>
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
};

export default CapturedContent;
