import { ActionButton, Preview } from 'ui-search/source';
import { Download } from 'lucide-react';
import { type FC } from 'react';

interface PreviewContentProps {
  data: any;
  redirectUrl?: string;
  contentBlobUrl?: string;
  downloadFilename?: string;
  requestHeaders?: Record<string, string>;
  theme?: 'auto' | 'dark' | 'light';
}

export const PreviewContent: FC<PreviewContentProps> = (props) => {
  const { data, redirectUrl, contentBlobUrl, downloadFilename, requestHeaders, theme } = props;
  const { t } = useTranslation();

  const handleDownload = () => {
    if (!contentBlobUrl) return;

    const a = document.createElement('a');
    a.href = contentBlobUrl;
    a.download = downloadFilename || 'download';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  };

  if (redirectUrl) {
    return (
      <div className='h-full flex flex-col items-center justify-center px-4'>
        <div className='w-[640px] rounded-lg border border-[#EDEDED] bg-[#FAFAFA] p-6 dark:border-[#303030] dark:bg-[#1C1C1C] flex flex-col gap-4'>
          <div className='text-sm font-bold leading-relaxed text-[#333] dark:text-[#CCC]'>
            {t('page.preview.hints.leave')}
          </div>

          <div className='text-sm font-normal leading-relaxed text-[#333] dark:text-[#CCC]'>
            {t('page.preview.hints.externalLinkWarning')}
          </div>

          <div className='break-all text-sm text-[#999] dark:text-[#666]'>
            {redirectUrl}
          </div>

          <div className='flex justify-start mt-6'>
            <ActionButton
              alwaysExpanded
              className='!px-4'
              size='large'
              onClick={() => {
                window.open(redirectUrl);
              }}
            >
              {t('page.preview.buttons.continueVisiting')}
            </ActionButton>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className='h-full flex flex-col'>
      <div className='mb-4 shrink-0 flex items-start justify-between gap-4'>
        <div>
          <div className='text-xl font-bold'>{data?.title}</div>
          <div className='text-sm text-[#999] dark:text-[#666]'>
            {data?.source?.name}
            {data?.category ? ` / ${data.category}` : ''}
          </div>
        </div>

        {contentBlobUrl && (
          <ActionButton
            icon={<Download />}
            onClick={handleDownload}
          >
            {t('page.preview.buttons.download')}
          </ActionButton>
        )}
      </div>

      <div className='flex-1 min-h-0 overflow-hidden'>
        <Preview
          data={data}
          requestHeaders={requestHeaders}
          theme={theme}
        />
      </div>
    </div>
  );
};

export default PreviewContent;
