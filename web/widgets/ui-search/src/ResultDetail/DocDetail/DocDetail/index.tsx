import { Button, Collapse, Typography } from 'antd';
import { ChevronDown, ChevronRight, Dot, Minus, SquareArrowOutUpRight } from 'lucide-react';
import { motion } from 'motion/react';

import { type FC, type HTMLAttributes, type ReactNode, useMemo, useState } from 'react';
import Preview from './components/Preview';
import AIInterpretation from './components/AIInterpretation';
import { clsx } from 'clsx';
import PreviewIcon from '../../../icons/PreviewIcon';
import AIInsightIcon from '../../../icons/AIInsightIcon';
import { AuthImage } from '../../../ResultList/AuthImage';
import loadingFailedSvg from '../../../icons/file-loading-failed.svg';

const { Text } = Typography;

export type MetadataContentType = 'docx' | 'image' | 'markdown' | 'pdf' | 'pptx' | 'video' | 'xlsx';

export interface DocDetailProps extends HTMLAttributes<HTMLDivElement> {
  readonly data: {
    id?: string;
    created?: ReactNode;
    updated?: ReactNode;
    _system?: {
      owner_id?: string;
      parent_path?: string;
      tenant_id?: string;
    };
    metadata?: {
      colors?: string[];
      content_type?: MetadataContentType;
      height?: number;
      mime_type?: string;
      raw_content_returns_file?: boolean;
      users?: null | unknown;
      width?: number;
      raw_content?: string;
    };
    source?: {
      type?: string;
      name?: string;
      id?: string;
    };
    type?: string;
    category?: string;
    title?: string;
    summary?: string;
    icon?: string;
    thumbnail?: string;
    cover?: string;
    tags?: string[];
    url?: string;
    size?: ReactNode;
    owner?: {
      type?: string;
      id?: string;
      icon?: string;
      title?: string;
      subtitle?: string;
      cover?: string;
      username?: string;
      name?: string;
    };
    ai_insights?: {
      text?: string;
    };
  };
  readonly i18n?: {
    labels?: {
      type?: string;
      size?: string;
      createdBy?: string;
      createdAt?: string;
      updatedAt?: string;
      preview?: string;
      previewUnavailableTitle?: string;
      previewUnavailableDescription?: string;
      openSource?: string;
      aiInterpretation?: string;
    };
  };
  readonly requestHeaders?: Record<string, string>;
  readonly actionButtons?: ReactNode[];
  readonly mode?: 'embedded' | 'standalone';
  readonly theme?: 'auto' | 'dark' | 'light';
}

const DocDetail: FC<DocDetailProps> = props => {
  const { data, i18n, actionButtons, requestHeaders, className, mode, theme, ...rest } = props;

  const [expandMore, setExpandMore] = useState(false);

  const ownerName = data?.owner?.title ?? data?.owner?.username ?? data?.owner?.name;

  const moreInfo = [
    {
      label: i18n?.labels?.type ?? 'Type',
      value: data?.type
    },
    {
      label: i18n?.labels?.size ?? 'Size',
      value: data?.size
    },
    {
      label: i18n?.labels?.createdBy ?? 'Created By',
      value: ownerName
    },
    {
      label: i18n?.labels?.createdAt ?? 'Created At',
      value: data?.created
    },
    {
      label: i18n?.labels?.updatedAt ?? 'Updated At',
      value: data?.updated
    }
  ];

  const contentType = data?.metadata?.content_type;
  const hasPreviewSource = Boolean(contentType) && Boolean(data?.metadata?.raw_content);
  const isInlinePreview = hasPreviewSource && (contentType === 'image' || contentType === 'video');
  const hasCollapsiblePreview =
    hasPreviewSource &&
    (contentType === 'markdown' || contentType === 'pdf' || contentType === 'docx' || contentType === 'pptx');
  // For file types that we don't have a dedicated renderer for, we can still embed the raw content
  // in a generic iframe/object preview box. This is useful when the raw_content endpoint returns a
  // file stream that the browser can render (e.g. text, html, plain documents).
  // We only show this for real raw content (raw_content_returns_file === true); for external links we
  // do not display the content because the raw_content endpoint just returns a JSON wrapper.
  const hasGenericPreview =
    data?.metadata?.raw_content_returns_file === true &&
    Boolean(data?.metadata?.raw_content) &&
    !isInlinePreview &&
    !hasCollapsiblePreview;

  const collapseItems = useMemo(() => {
    const items: { key: string; label: ReactNode | string; children: ReactNode }[] = [];

    if (hasCollapsiblePreview) {
      items.push({
        key: 'preview',
        label: (
          <div className='inline-flex items-center gap-8px'>
            <PreviewIcon
              className='shrink-0 text-[--ant-color-primary]'
              size={16}
            />
            <div className='text-16px text-#333 leading-22px dark:text-#666'>{i18n?.labels?.preview ?? 'Preview'}</div>
          </div>
        ),
        children: (
          <Preview
            {...props}
            loadingHeight='h-[calc(100cqh-394px)]'
          />
        )
      });
    }

    if (data?.ai_insights?.text) {
      items.push({
        key: 'ai-interpretation',
        label: (
          <div className='inline-flex items-center gap-8px'>
            <AIInsightIcon
              className='shrink-0 text-[--ant-color-primary]'
              size={16}
            />
            <div className='text-16px text-#333 leading-22px dark:text-#666'>
              {i18n?.labels?.aiInterpretation ?? 'AI Interpretation'}
            </div>
          </div>
        ),
        children: <AIInterpretation {...props} />
      });
    }

    return items;
  }, [hasCollapsiblePreview, data?.ai_insights?.text, i18n, props]);

  const isContentEmpty = !isInlinePreview && collapseItems.length === 0 && !hasGenericPreview;
  const canOpenSource = data?.url?.startsWith('http');

  return (
    <div
      className={clsx('flex flex-col h-full overflow-hidden @container/detail', className)}
      {...rest}
    >
      <div
        className={clsx(
          'text-20px text-[#1A0CAB] dark:text-[#8AB4F8] break-words',
          mode === 'embedded' ? 'pr-24px' : ''
        )}
      >
        <AuthImage
          className='mr-8px inline-block h-20px w-20px align-middle'
          requestHeaders={requestHeaders}
          src={data?.icon}
        />
        <span className='align-middle'>{data?.title}</span>
      </div>

      <div className='my-2 flex items-center justify-between'>
        <Text className='inline-flex items-center gap-0.5 text-3 text-[#666] dark:text-white/80'>
          <div>{data?.source?.name ?? '-'}</div>
          {data?.category && (
            <>
              <ChevronRight className='size-3' />
              <div>{data?.category}</div>
            </>
          )}
          {ownerName && (
            <>
              <Minus className='size-3 rotate-90' />
              <div>{ownerName}</div>
            </>
          )}
          {data?.updated && (
            <>
              <Dot className='size-3' />
              <div>{data?.updated}</div>
            </>
          )}

          <ChevronDown
            className={clsx('ml-2 size-3 hover:text-[--ant-color-primary] transition cursor-pointer', {
              '-scale-y-100': expandMore
            })}
            onClick={() => {
              setExpandMore(prev => !prev);
            }}
          />
        </Text>

        <div className='inline-flex gap-2'>{actionButtons}</div>
      </div>

      <motion.div
        className='overflow-hidden rounded-lg bg-black/3 dark:bg-white/4'
        initial={false}
        animate={{
          height: expandMore ? 'auto' : 0,
          opacity: expandMore ? 1 : 0,
          marginBottom: expandMore ? '1rem' : 0
        }}
      >
        <div className='flex flex-wrap gap-row-2 p-4'>
          {moreInfo.map(item => {
            const { label, value } = item;

            return (
              <div
                className='w-1/2 inline-flex items-center <sm:w-full'
                key={label}
              >
                <Text
                  className='w-24'
                  type='secondary'
                >
                  {label}
                </Text>

                <span className='text-3.5'>{value ?? '-'}</span>
              </div>
            );
          })}
        </div>
      </motion.div>

      <div className='flex flex-col flex-1 gap-4 overflow-auto'>
        {isInlinePreview && (
          <Preview
            {...props}
            loadingHeight='aspect-4/3'
          />
        )}
        {collapseItems.length > 0 && (
          <Collapse
            defaultActiveKey={[collapseItems[0]?.key]}
            expandIconPlacement='end'
            items={collapseItems}
            size='small'
            classNames={{
              root: 'bg-transparent border-[#F0F0F0] dark:border-[#303030] [&_.ant-collapse-panel]:border-[#F0F0F0]! dark:[&_.ant-collapse-panel]:border-[#303030]!',
              body: 'p-24px!',
              header: 'px-16px! py-9px!',
              title: 'flex items-center',
              icon: 'text-[#999]! dark:text-[#666]! [&_.ant-collapse-arrow]:text-16px!'
            }}
          />
        )}
        {hasGenericPreview && (
          <div className='min-h-360px flex-1 overflow-hidden border border-[#F0F0F0] rounded-lg dark:border-[#303030]'>
            <iframe
              className='h-full w-full'
              sandbox='allow-same-origin'
              src={data?.metadata?.raw_content}
              title={data?.title || 'File preview'}
            />
          </div>
        )}
        {isContentEmpty && (
          <div className='min-h-360px flex flex-1 items-center justify-center px-24px text-center'>
            <div className='flex flex-col items-center'>
              <img
                className='mb-16px h-80px w-80px'
                src={loadingFailedSvg}
              />
              <div className='text-14px text-[#999] leading-22px dark:text-[#666]'>
                <div>{i18n?.labels?.previewUnavailableTitle ?? "This file can't be previewed"}</div>
                <div>
                  {i18n?.labels?.previewUnavailableDescription ??
                    'The file format may be unsupported or the content is temporarily unavailable'}
                </div>
              </div>
              {canOpenSource && (
                <Button
                  className='mt-24px min-w-120px rounded-20px'
                  icon={<SquareArrowOutUpRight className='size-14px' />}
                  type='primary'
                  onClick={() => window.open(data.url)}
                >
                  {i18n?.labels?.openSource ?? 'Open Source'}
                </Button>
              )}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default DocDetail;
