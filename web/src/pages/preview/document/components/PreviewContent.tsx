import { Download } from 'lucide-react';
import { type FC, type ReactNode, useEffect, useMemo } from 'react';
import { I18nextProvider, useTranslation } from 'react-i18next';
import { useNavigate, useSearchParams } from 'react-router-dom';
import dayjs from 'dayjs';
import relativeTime from 'dayjs/plugin/relativeTime';
import { filesize } from 'filesize';

import { ActionButton, DocDetail, searchWidgetI18n } from 'ui-search/source';
import { getLocale } from '@/store/slice/app';
import { useAppSelector } from '@/hooks/business/useStore';

dayjs.extend(relativeTime);

interface PreviewContentProps {
  readonly data: any;
  readonly redirectUrl?: string;
  readonly contentBlobUrl?: string;
  readonly downloadFilename?: string;
  readonly requestHeaders?: Record<string, string>;
  readonly theme?: 'auto' | 'dark' | 'light';
}

// same timestamp format the result drawer's info grid shows
const fullDateTime = (value: string | number) => {
  const date = dayjs(value);
  return date.isValid() ? `${date.format('YYYY-MM-DD HH:mm:ss')} (GMT${date.format('ZZ')})` : undefined;
};

interface FilterableDetailProps {
  readonly data: any;
  readonly labels: Record<string, string>;
  readonly actionButtons?: ReactNode[];
  readonly onMetaFilter: (field: string, value: string) => void;
  readonly requestHeaders?: Record<string, string>;
  readonly theme?: 'auto' | 'dark' | 'light';
}

// rendered under the widget's i18n provider, so content-type names localize
// through the widget instance and follow its language reactively
const FilterableDetail: FC<FilterableDetailProps> = ({
  data,
  labels,
  actionButtons,
  onMetaFilter,
  requestHeaders,
  theme
}) => {
  const { t, i18n } = useTranslation();

  // mirrors the result drawer's ResultDetail enrichment: display-ready values
  // for the info grid plus the raw value a meta filter narrows by
  const detailData = useMemo(
    () => ({
      ...(data || {}),
      // localize known content types (labels.value_image etc.); unknown ones pass through
      type: data?.type ? t(`labels.value_${data.type}`, { defaultValue: data.type }) : undefined,
      typeFilterValue: data?.type,
      size: Number(data?.size) > 0 ? filesize(data.size) : undefined,
      created: data?.created ? fullDateTime(data.created) : undefined,
      updated: data?.updated ? fullDateTime(data.updated) : undefined,
      updatedCompact: data?.updated
        ? dayjs(data.updated)
            .locale(i18n.language?.toLowerCase().startsWith('zh') ? 'zh-cn' : 'en')
            .fromNow()
        : undefined
    }),
    [data, t, i18n.language]
  );

  return (
    <DocDetail
      actionButtons={actionButtons}
      data={detailData}
      i18n={{ labels }}
      mode='standalone'
      requestHeaders={requestHeaders}
      theme={theme}
      onMetaFilter={onMetaFilter}
    />
  );
};

export const PreviewContent: FC<PreviewContentProps> = props => {
  const { data, redirectUrl, contentBlobUrl, downloadFilename, requestHeaders, theme } = props;
  const { t } = useTranslation();
  const navigate = useNavigate();
  const locale = useAppSelector(getLocale);
  const [searchParams] = useSearchParams();
  const embedded = searchParams.get('mode') === 'embedded';

  // widget-internal strings (preview loading/error states, content-type names)
  // resolve through the widget's own i18n instance; this page lives outside the
  // app layout that normally provides it, so supply it here and keep the
  // language in sync with the app locale
  useEffect(() => {
    searchWidgetI18n.changeLanguage(locale);
  }, [locale]);

  const handleDownload = () => {
    if (!contentBlobUrl) return;

    const a = document.createElement('a');
    a.href = contentBlobUrl;
    a.download = downloadFilename || 'download';
    document.body.appendChild(a);
    a.click();
    document.body.removeChild(a);
  };

  // clicking a filterable meta entry (category, tag, type, owner) re-runs the
  // search narrowed by that facet — the standalone-page counterpart of the
  // result drawer's onMetaFilter; the filter travels as the search page's
  // aggfilter URL param (`field:value`)
  const handleMetaFilter = (field: string, value: string) => {
    const target = `/search?aggfilter=${encodeURIComponent(`${field}:${value}`)}`;
    // an embedded preview renders inside a host page's iframe — navigating the
    // frame would tear the embed down, so hand the filtered search to a new tab
    if (embedded) {
      window.open(`${window.location.origin}${window.location.pathname}#${target}`, '_blank');
      return;
    }
    navigate(target);
  };

  if (redirectUrl) {
    return (
      <div className='h-full flex flex-col items-center justify-center px-4'>
        <div className='w-[640px] flex flex-col gap-4 border border-[#EDEDED] rounded-lg bg-[#FAFAFA] p-6 dark:border-[#303030] dark:bg-[#1C1C1C]'>
          <div className='text-sm text-[#333] font-bold leading-relaxed dark:text-[#CCC]'>
            {t('page.preview.hints.leave')}
          </div>

          <div className='text-sm text-[#333] font-normal leading-relaxed dark:text-[#CCC]'>
            {t('page.preview.hints.externalLinkWarning')}
          </div>

          <div className='break-all text-sm text-[#999] dark:text-[#666]'>{redirectUrl}</div>

          <div className='mt-6 flex justify-start'>
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
    <I18nextProvider i18n={searchWidgetI18n}>
      <FilterableDetail
        data={data}
        requestHeaders={requestHeaders}
        theme={theme}
        actionButtons={
          contentBlobUrl
            ? [
                <ActionButton
                  icon={<Download />}
                  key='download'
                  onClick={handleDownload}
                >
                  {t('page.preview.buttons.download')}
                </ActionButton>
              ]
            : undefined
        }
        labels={{
          type: t('page.preview.labels.type'),
          size: t('page.preview.labels.size'),
          format: t('page.preview.labels.format'),
          source: t('page.preview.labels.source'),
          connector: t('page.preview.labels.connector'),
          createdBy: t('page.preview.labels.createdBy'),
          createdAt: t('page.preview.labels.createdAt'),
          updatedAt: t('page.preview.labels.updatedAt'),
          location: t('page.preview.labels.location'),
          link: t('page.preview.labels.link'),
          tag: t('page.preview.labels.tag'),
          noMoreInfo: t('page.preview.labels.noMoreInfo'),
          preview: t('page.preview.labels.preview'),
          previewUnavailableTitle: t('page.preview.labels.previewUnavailableTitle'),
          previewUnavailableDescription: t('page.preview.labels.previewUnavailableDescription'),
          openSource: t('page.preview.labels.openSourceFromPreview'),
          aiInterpretation: t('page.preview.labels.aiInterpretation')
        }}
        onMetaFilter={handleMetaFilter}
      />
    </I18nextProvider>
  );
};

export default PreviewContent;
