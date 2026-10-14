import { Button, Collapse, Tag, Tooltip, Typography } from 'antd';
import {
  CalendarPlus,
  ChevronDown,
  ChevronRight,
  Database,
  FileCode,
  FileText,
  FolderOpen,
  HardDrive,
  History,
  Link2,
  Plug,
  SquareArrowOutUpRight,
  Tags,
  UserRound
} from 'lucide-react';
import { motion } from 'motion/react';

import { type ComponentType, type FC, type HTMLAttributes, type ReactNode, useMemo, useState } from 'react';
import Preview from './components/Preview';
import AIInterpretation from './components/AIInterpretation';
import { clsx } from 'clsx';
import PreviewIcon from '../../../icons/PreviewIcon';
import AIInsightIcon from '../../../icons/AIInsightIcon';
import { OWNER_FILTER_FIELD } from '../../../Aggregations';
import { AuthImage } from '../../../ResultList/AuthImage';
import loadingFailedSvg from '../../../icons/file-loading-failed.svg';

const { Text } = Typography;

// tiny muted dot between meta entries — keeps the one-line header quiet and uniform
const MetaSeparator = () => <span className='size-3px shrink-0 rounded-full bg-current opacity-35' />;

// compact entity card for the info grid: icon + name on a bordered chip.
// With onClick it doubles as a facet filter entry (hover signals it).
const MetaEntityCard = (props: {
  readonly icon?: string;
  readonly name: string;
  readonly onClick?: () => void;
  readonly requestHeaders?: Record<string, string>;
}) => {
  const { icon, name, onClick, requestHeaders } = props;
  return (
    <span
      className={clsx(
        'inline-flex max-w-full min-w-0 items-center gap-4px rounded-6px border border-[#E8E8E8] bg-[#F7F7F7] py-2px pl-4px pr-8px dark:border-[#303030] dark:bg-white/6',
        onClick && 'cursor-pointer hover:border-[--ant-color-primary] hover:text-[--ant-color-primary]'
      )}
      onClick={onClick}
    >
      {icon ? (
        <AuthImage
          className='size-12px shrink-0 rounded-2px object-contain'
          requestHeaders={requestHeaders}
          src={icon}
        />
      ) : null}
      <span className='truncate'>{name}</span>
    </span>
  );
};

export type MetadataContentType = 'docx' | 'image' | 'markdown' | 'pdf' | 'pptx' | 'video' | 'xlsx';

export interface DocDetailProps extends HTMLAttributes<HTMLDivElement> {
  readonly data: {
    id?: string;
    created?: ReactNode;
    updated?: ReactNode;
    /** compact variant of `updated` (e.g. short relative time) for the one-line meta header; falls back to `updated` */
    updatedCompact?: ReactNode;
    _system?: {
      owner_id?: string;
      parent_path?: string;
      tenant_id?: string;
    };
    metadata?: {
      colors?: string[];
      content_type?: MetadataContentType;
      file_extension?: string;
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
      icon?: string;
      /**
       * connector behind the datasource (resolved server-side at read time, e.g. github/yuque) — powers the connector
       * entity card and its facet
       */
      connector_id?: string;
      connector_name?: string;
      connector_icon?: string;
    };
    type?: string;
    /**
     * raw (unlocalized) type value — `type` carries the localized display label, this one is what a type filter narrows
     * the search by; falls back to `type`
     */
    typeFilterValue?: string;
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
      format?: string;
      source?: string;
      connector?: string;
      createdBy?: string;
      createdAt?: string;
      updatedAt?: string;
      location?: string;
      link?: string;
      tag?: string;
      noMoreInfo?: string;
      preview?: string;
      previewUnavailableTitle?: string;
      previewUnavailableDescription?: string;
      openSource?: string;
      aiInterpretation?: string;
    };
  };
  readonly requestHeaders?: Record<string, string>;
  readonly actionButtons?: ReactNode[];
  /**
   * fired when the user clicks a filterable meta entry (category path, tag, type, owner) — the host re-runs the search
   * with this facet value; absent entries render plain
   */
  readonly onMetaFilter?: (field: string, value: string) => void;
  readonly mode?: 'embedded' | 'standalone';
  readonly theme?: 'auto' | 'dark' | 'light';
}

const DocDetail: FC<DocDetailProps> = props => {
  const { data, i18n, actionButtons, requestHeaders, onMetaFilter, className, mode, theme, ...rest } = props;

  const [expandMore, setExpandMore] = useState(false);

  const ownerName = data?.owner?.title ?? data?.owner?.username ?? data?.owner?.name;
  // the display label is localized (labels.value_image etc.) — filters need
  // the raw bucket value the aggregations rail matches on
  const typeFilterValue = data?.typeFilterValue ?? data?.type;
  const ownerFilterValue = data?._system?.owner_id;
  const metaUpdated = data?.updatedCompact ?? data?.updated;
  // http(s) links and app-relative URLs ("/#/preview/…") can be opened;
  // custom schemes like coco:// have nothing to hand off to
  const canOpenSource = Boolean(data?.url?.match(/^(https?:|\/|#)/));

  // one row per metadata field for the expandable info panel; rows with no
  // value are dropped instead of rendered as "-" noise
  const extension = data?.metadata?.file_extension?.replace(/^\./, '');
  const mimeType = data?.metadata?.mime_type;
  const formatValue = extension
    ? mimeType
      ? `${extension.toUpperCase()} · ${mimeType}`
      : extension.toUpperCase()
    : mimeType;

  const moreInfo: {
    key: string;
    label: string;
    icon: ComponentType<{ className?: string }>;
    /** span the full panel width (paths, links, tag lists) */
    full?: boolean;
    value: ReactNode;
  }[] = [
    {
      key: 'datasource',
      label: i18n?.labels?.source ?? 'Source',
      icon: Database,
      // provenance first: the datasource entity card re-filters the current
      // search by source.id when the host wires onMetaFilter
      value:
        data?.source?.name || data?.source?.id ? (
          <MetaEntityCard
            icon={data?.source?.icon}
            name={data?.source?.name || data?.source?.id}
            requestHeaders={requestHeaders}
            onClick={onMetaFilter && data?.source?.id ? () => onMetaFilter('source.id', data.source.id!) : undefined}
          />
        ) : undefined
    },
    {
      key: 'connector',
      label: i18n?.labels?.connector ?? 'Connector',
      icon: Plug,
      // the connector behind the datasource (server resolves it from the
      // datasource config); filters as source.connector_id, which the search
      // API translates into the datasources running that connector
      value: data?.source?.connector_id ? (
        <MetaEntityCard
          icon={data?.source?.connector_icon}
          name={data?.source?.connector_name || data?.source?.connector_id}
          requestHeaders={requestHeaders}
          onClick={onMetaFilter ? () => onMetaFilter('source.connector_id', data.source!.connector_id!) : undefined}
        />
      ) : undefined
    },
    {
      key: 'type',
      label: i18n?.labels?.type ?? 'Type',
      icon: FileText,
      // clicking re-filters the current search by this type when the host
      // wires onMetaFilter
      value: data?.type ? (
        <span
          className={clsx(
            onMetaFilter && typeFilterValue && 'cursor-pointer hover:text-[--ant-color-primary] hover:underline'
          )}
          onClick={onMetaFilter && typeFilterValue ? () => onMetaFilter('type', typeFilterValue) : undefined}
        >
          {data?.type}
        </span>
      ) : undefined
    },
    {
      key: 'size',
      label: i18n?.labels?.size ?? 'Size',
      icon: HardDrive,
      value: data?.size
    },
    {
      key: 'format',
      label: i18n?.labels?.format ?? 'Format',
      icon: FileCode,
      value: formatValue
    },
    {
      key: 'createdBy',
      label: i18n?.labels?.createdBy ?? 'Created By',
      icon: UserRound,
      // clickable like category/tags: narrows the search to this owner's
      // documents (_system.owner_id term filter)
      value: ownerName ? (
        <span
          className={clsx(
            onMetaFilter && ownerFilterValue && 'cursor-pointer hover:text-[--ant-color-primary] hover:underline'
          )}
          onClick={
            onMetaFilter && ownerFilterValue ? () => onMetaFilter(OWNER_FILTER_FIELD, ownerFilterValue) : undefined
          }
        >
          {ownerName}
        </span>
      ) : undefined
    },
    {
      key: 'createdAt',
      label: i18n?.labels?.createdAt ?? 'Created At',
      icon: CalendarPlus,
      value: data?.created
    },
    {
      key: 'updatedAt',
      label: i18n?.labels?.updatedAt ?? 'Updated At',
      icon: History,
      value: data?.updated
    },
    {
      key: 'location',
      label: i18n?.labels?.location ?? 'Location',
      icon: FolderOpen,
      full: true,
      // single-line with the full text on hover; clicking re-filters the
      // current search by this category when the host wires onMetaFilter
      value: data?.category ? (
        <span className='min-w-0 flex flex-1 items-center gap-6px [&_.ant-typography-copy]:flex-none'>
          <Tooltip title={data.category}>
            <span
              className={clsx(
                'min-w-0 flex-1 truncate',
                onMetaFilter && 'cursor-pointer hover:text-[--ant-color-primary] hover:underline'
              )}
              onClick={onMetaFilter ? () => onMetaFilter('category', data.category!) : undefined}
            >
              {data.category}
            </span>
          </Tooltip>
          <Text
            className='flex-none'
            copyable={{ text: data.category }}
          />
        </span>
      ) : undefined
    },
    {
      key: 'link',
      label: i18n?.labels?.link ?? 'Link',
      icon: Link2,
      full: true,
      value: canOpenSource ? (
        <span className='min-w-0 flex flex-1 items-center gap-6px [&_.ant-typography-copy]:flex-none'>
          <Tooltip title={data.url}>
            <a
              className='min-w-0 flex-1 truncate text-[#1A0CAB] dark:text-[#8AB4F8] hover:underline'
              href={data.url}
              rel='noreferrer'
              target='_blank'
            >
              {data.url}
            </a>
          </Tooltip>
          <Text
            className='flex-none'
            copyable={{ text: data.url }}
          />
        </span>
      ) : undefined
    },
    {
      key: 'tags',
      label: i18n?.labels?.tag ?? 'Tags',
      icon: Tags,
      full: true,
      value: data?.tags?.length ? (
        <div className='min-w-0 flex flex-wrap gap-4px'>
          {data.tags.map(tag => (
            <Tag
              bordered={false}
              key={tag}
              className={clsx(
                'mr-0',
                onMetaFilter && 'cursor-pointer hover:text-[--ant-color-primary] hover:opacity-80'
              )}
              onClick={onMetaFilter ? () => onMetaFilter('tags', tag) : undefined}
            >
              {tag}
            </Tag>
          ))}
        </div>
      ) : undefined
    }
  ].filter(row => {
    const { value } = row;
    return value !== undefined && value !== null && value !== '' && !(Array.isArray(value) && value.length === 0);
  });

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
            // preview holds loading/error state for one document; without the
            // key a failed load sticks when the drawer switches documents
            key={data?.id}
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
  // whether any rendered section follows the inline preview in the scroll area
  const hasContentBelow = collapseItems.length > 0 || hasGenericPreview;

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

      <div className='my-2 min-w-0 flex items-center justify-between gap-2'>
        <Text className='min-w-0 flex flex-1 items-center gap-x-6px text-3 text-[#666] dark:text-white/80'>
          <span className='shrink-0'>{data?.source?.name ?? '-'}</span>
          {data?.category && (
            <>
              <ChevronRight className='size-3 shrink-0 opacity-45' />
              <Tooltip title={data.category}>
                <span className='min-w-0 cursor-default truncate'>{data.category}</span>
              </Tooltip>
            </>
          )}
          {ownerName && (
            <>
              <MetaSeparator />
              <span className='shrink-0'>{ownerName}</span>
            </>
          )}
          {metaUpdated ? (
            <>
              <MetaSeparator />
              <span className='shrink-0 whitespace-nowrap'>{metaUpdated}</span>
            </>
          ) : null}

          <ChevronDown
            className={clsx('ml-4px size-3 shrink-0 cursor-pointer transition hover:text-[--ant-color-primary]', {
              '-scale-y-100': expandMore
            })}
            onClick={() => {
              setExpandMore(prev => !prev);
            }}
          />
        </Text>

        <div className='inline-flex shrink-0 gap-2'>{actionButtons}</div>
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
        {/* two-column info grid inside the named container; narrow panes
            (mobile drawer, split view) fall back to a single column */}
        <div className='grid grid-cols-1 gap-x-20px gap-y-6px p-12px @md/detail:grid-cols-2'>
          {moreInfo.length ? (
            moreInfo.map(row => {
              const Icon = row.icon;

              return (
                <div
                  className={clsx('flex min-w-0 items-center gap-x-6px', row.full && '@md/detail:col-span-full')}
                  key={row.key}
                >
                  <Icon className='size-3 shrink-0 text-[#999] dark:text-white/40' />
                  <span className='w-14 shrink-0 text-2.5 text-[#999] leading-16px dark:text-white/45'>
                    {row.label}
                  </span>
                  <div className='min-w-0 flex-1 truncate text-3 text-[#333] leading-16px dark:text-white/85'>
                    {row.value}
                  </div>
                </div>
              );
            })
          ) : (
            <div className='col-span-full py-4px text-center text-3 text-[#999] dark:text-white/45'>
              {i18n?.labels?.noMoreInfo ?? 'No additional information'}
            </div>
          )}
        </div>
      </motion.div>

      <div className='flex flex-col flex-1 gap-4 overflow-auto'>
        {isInlinePreview && (
          // images don't need the whole pane, but a strict quarter felt
          // cramped — cap at ~45% while sections below stay visible, or take
          // the remaining area when nothing follows; videos keep the full pane
          <div className={clsx('shrink-0', contentType === 'image' && hasContentBelow ? 'h-[45%]' : 'flex-1 min-h-0')}>
            <Preview
              {...props}
              key={data?.id}
            />
          </div>
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
