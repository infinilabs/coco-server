import { Tooltip } from 'antd';
import { SquareArrowOutUpRight, X } from 'lucide-react';
import { ActionButton, DocDetail } from './DocDetail';
import { type FC, useMemo } from 'react';
import CommonDrawer from '../Layout/CommonDrawer';
import { useTranslation } from 'react-i18next';
import { filesize } from 'filesize';
import dayjs from 'dayjs';
import 'dayjs/locale/zh-cn';
import { isString } from 'lodash';
import { formatDate } from '../utils/date';

interface ResultDetailProps {
  readonly getContainer?: () => HTMLElement;
  readonly data?: Record<string, any>;
  readonly isMobile?: boolean;
  readonly open?: boolean;
  readonly onClose?: () => void;
  readonly apiConfig?: Record<string, any>;
  readonly theme?: 'auto' | 'dark' | 'light';
  /** render as an in-flow panel that fills its parent instead of a floating drawer */
  readonly inline?: boolean;
  /**
   * forwarded to the preview: clicking a filterable meta entry (category, tag, type, owner) re-runs the current search
   * narrowed by that value
   */
  readonly onMetaFilter?: (field: string, value: string) => void;
}

export function DateTime(props: {
  readonly value: string | number;
  readonly showTooltip?: boolean;
  /** display a short relative time (e.g. "10 天前") instead of the full timestamp; hover reveals the exact time */
  readonly relative?: boolean;
}) {
  const { value, showTooltip = true, relative = false } = props;
  const { i18n } = useTranslation();
  if (!value || !dayjs(value).isValid()) return '-';

  const full = formatDate(value);
  // fromNow must follow the UI language, not dayjs's global default locale
  const display = relative
    ? dayjs(value)
        .locale(i18n.language?.toLowerCase().startsWith('zh') ? 'zh-cn' : 'en')
        .fromNow()
    : full;

  if (showTooltip) {
    return (
      <Tooltip title={relative ? full : isString(value) ? value : full}>
        <span className='whitespace-nowrap'>{display}</span>
      </Tooltip>
    );
  }

  return display;
}

export const ResultDetail: FC<ResultDetailProps> = props => {
  const { getContainer, data = {}, isMobile, open, onClose, apiConfig, theme, inline, onMetaFilter } = props;
  const { t } = useTranslation();
  const detailData = useMemo<Record<string, any>>(
    () => ({
      ...(data || {}),
      // localize known content types (labels.value_image etc.); unknown ones pass through
      type: data?.type ? t(`labels.value_${data.type}`, { defaultValue: data.type }) : undefined,
      // DocDetail filters by the raw bucket value while showing the localized
      // label above
      typeFilterValue: data?.type,
      // a missing size should hide the row, not claim "0 B"
      size: Number(data?.size) > 0 ? filesize(data.size) : undefined,
      created: data?.created ? (
        <DateTime
          showTooltip={false}
          value={data?.created}
        />
      ) : null,
      updated: data?.updated ? (
        <DateTime
          showTooltip={false}
          value={data?.updated}
        />
      ) : null,
      // short relative variant for the one-line meta header (full timestamp stays in the expanded info grid)
      updatedCompact: data?.updated ? (
        <DateTime
          relative
          value={data?.updated}
        />
      ) : null
    }),
    [data, t]
  );

  const content = (
    <>
      <X
        className='absolute right-24px top-24px z-1 cursor-pointer color-[#bbb]'
        onClick={onClose}
      />
      <DocDetail
        data={detailData}
        mode='embedded'
        requestHeaders={apiConfig?.headers}
        theme={theme}
        actionButtons={[
          <ActionButton
            icon={<SquareArrowOutUpRight />}
            key='open'
            onClick={() => {
              const url = detailData?.url;
              // http(s) links and app-relative URLs (e.g. "/#/preview/…",
              // which the server emits when the endpoint is relative) all
              // open in a new tab; custom schemes like coco:// stay inert —
              // the browser has nothing to hand them to
              if (url && (url.startsWith('http') || url.startsWith('/') || url.startsWith('#'))) {
                window.open(url, '_blank');
              }
            }}
          >
            {t('labels.openSource')}
          </ActionButton>
        ]}
        i18n={{
          labels: {
            type: t('labels.type'),
            size: t('labels.size'),
            format: t('labels.format'),
            source: t('labels.source'),
            connector: t('labels.connector'),
            createdBy: t('labels.createdBy'),
            createdAt: t('labels.createdAt'),
            updatedAt: t('labels.updatedAt'),
            location: t('labels.location'),
            link: t('labels.link'),
            tag: t('labels.tag'),
            noMoreInfo: t('labels.noMoreInfo'),
            preview: t('labels.preview'),
            previewUnavailableTitle: t('labels.previewUnavailableTitle'),
            previewUnavailableDescription: t('labels.previewUnavailableDescription'),
            openSource: t('labels.openSourceFromPreview'),
            aiInterpretation: t('labels.aiInterpretation')
          }
        }}
        onMetaFilter={onMetaFilter}
      />
    </>
  );

  if (inline) {
    return (
      // mirrors the drawer variant's body (p-24px, hidden overflow); no card
      // rounding/shadow — the layout pins this pane full-height below the
      // fixed header, so it reads as a panel, not a floating card
      <div className='relative h-full w-full overflow-hidden bg-[var(--ant-color-bg-container)] p-24px'>{content}</div>
    );
  }

  return (
    <CommonDrawer
      destroyOnHidden
      clickOutsideToClose={Boolean(isMobile)}
      getContainer={getContainer}
      open={open}
      placement='right'
      size={isMobile ? undefined : 800}
      classNames={{
        wrapper: `${isMobile ? '!left-0px !right-0px !w-full !top-122px !bottom-0px' : '!right-24px !top-88px !bottom-24px'}`,
        body: '!p-24px !overflow-hidden !h-full'
      }}
      onClose={onClose}
    >
      {content}
    </CommonDrawer>
  );
};

export default ResultDetail;
