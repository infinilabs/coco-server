import Aggregations, { FIELD_LABEL_KEYS, OWNER_FILTER_FIELD } from '../Aggregations';
import LoadingIcon from '../components/LoadingIcon';
import AIOverviewWrapper from '../AIOverview/AIOverviewWrapper';
import Categories from '../Categories';
import BasicLayout from '../Layout/BasicLayout';
import Logo from '../Logo';
import ResultDetail from '../ResultDetail';
import ResultHeader from '../ResultHeader';
import SearchBox from '../SearchBox';
import Recommends from '../Recommends';
import { LIST_TYPES } from '../ResultList';
import { EmptyList } from '../ResultList/EmptyList';
import MediaLayout from '../Layout/MediaLayout';
import PreviewConnector from '../components/PreviewConnector';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { X } from 'lucide-react';
import FilterIcon from '../icons/FilterIcon';
import HistogramIcon from '../icons/HistogramIcon';
import { Button } from 'antd';
import Toolbar from '../Toolbar';
import Histogram from '../Histogram';
import {
  ACTION_TYPE_SEARCH_KEYWORD,
  DEFAULT_SEARCH_SORT,
  normalizeSearchFuzziness,
  normalizeSearchSort
} from '../SearchBox/ActionBar/SearchActions';

interface SearchProps {
  readonly aggregations?: any;
  readonly aiOverview?: Record<string, any>;
  readonly askBody?: any;
  /** per-category availability from the aggregation query; drives which result
   * tabs are worth showing (null = no data, show all tabs) */
  readonly categoryCounts?: { image: number; nonImage: number } | null;
  readonly commonProps?: Record<string, any>;
  readonly config?: Record<string, any>;
  readonly data?: any[];
  readonly getContainer?: () => HTMLElement | null;
  readonly handleLogoClick?: () => void;
  readonly hits?: any;
  readonly hasMore?: boolean;
  readonly initContainer?: (ref: HTMLDivElement | null) => void;
  readonly loading?: boolean;
  /** pass null to hide the widget's own logo (host app already shows its brand) */
  readonly logo?: Record<string, any> | null;
  readonly onAsk?: (...args: any[]) => void;
  readonly onSearchFilter?: (aggfilter: Record<string, any>) => void;
  readonly onSearch?: (...args: any[]) => void;
  readonly placeholder?: string;
  readonly queryParams?: Record<string, any>;
  readonly rightMenuWidth?: number;
  readonly showFullScreenSpin?: boolean;
  readonly setQueryParams?: (params: any) => void;
  readonly theme?: string;
  readonly onSuggestion?: (...args: any[]) => void;
  readonly onRecommend?: (...args: any[]) => void;
  readonly onChatContinue?: (session_id: string) => void;
  readonly getFieldsMeta?: (...args: any[]) => any;
  readonly onUpload?: (...args: any[]) => void;
  readonly attachments?: any[];
  readonly setAttachments?: (attachments: any[]) => void;
  readonly settings?: Record<string, any>;
  readonly onLoadMore?: () => void;
  readonly histogramData?: { date: string; count: number }[];
  [key: string]: any;
}

export default function Search({
  aggregations,
  aiOverview,
  askBody,
  categoryCounts,
  commonProps,
  config,
  data,
  getContainer,
  handleLogoClick,
  hits,
  hasMore,
  initContainer,
  loading,
  logo,
  onAsk,
  onSearchFilter,
  onSearch,
  placeholder,
  queryParams,
  rightMenuWidth,
  showFullScreenSpin,
  setQueryParams,
  theme,
  onSuggestion,
  onRecommend,
  onChatContinue,
  getFieldsMeta,
  onUpload,
  attachments,
  setAttachments,
  settings,
  onLoadMore,
  onCategoryChange,
  histogramData
}: SearchProps) {
  const { query, filter, aggfilter = {}, search_type = ACTION_TYPE_SEARCH_KEYWORD } = queryParams || {};
  const { t } = useTranslation();
  const fuzziness = normalizeSearchFuzziness(queryParams?.fuzziness);
  const sort = normalizeSearchSort(queryParams?.sort || DEFAULT_SEARCH_SORT);
  const dateRange = typeof queryParams?.date_range === 'string' ? queryParams.date_range : 'all-time';
  const start =
    typeof queryParams?.start === 'string' || typeof queryParams?.start === 'number'
      ? String(queryParams.start)
      : undefined;
  const end =
    typeof queryParams?.end === 'string' || typeof queryParams?.end === 'number' ? String(queryParams.end) : undefined;
  const content_category = queryParams?.['metadata.content_category'];
  // when the facet skeleton shows from mount (first search in flight) the
  // sider starts expanded, so the first paint already sits at the final
  // column geometry instead of rendering one sider-less frame first
  const [siderCollapse, setSiderCollapse] = useState(
    () => Boolean(commonProps?.isMobile) || !(loading && !data?.length && !aggregations?.length)
  );
  const [detailCollapse, setDetailCollapse] = useState(true);
  const [recommendsCollapse, setRecommendsCollapse] = useState(true);
  // the record previewed in the layout's right pane; owned here so the pane can
  // replace the recommends column while staying click-through operable
  const [detailRecord, setDetailRecord] = useState<Record<string, any> | undefined>();
  const openDetail = useCallback((record: Record<string, any>) => setDetailRecord(record), []);
  const closeDetail = useCallback(() => setDetailRecord(undefined), []);
  const [toolbarVisible, setToolbarVisible] = useState(false);
  const [histogramVisible, setHistogramVisible] = useState(false);
  const [filterFieldsMeta, setFilterFieldsMeta] = useState({});
  const [hasRecommendsData, setHasRecommendsData] = useState(false);
  const ownerCacheRef = useRef<Record<string, any>>({});
  const pendingOwnerIdsRef = useRef<Set<string>>(new Set());
  const [ownerVersion, setOwnerVersion] = useState(0);
  const getUserEntities = commonProps?.getUserEntities as
    | ((ids: string[], callback?: (data: any) => void) => any)
    | undefined;
  const handleRecommendsDataLoaded = useCallback((hasData: boolean) => {
    setHasRecommendsData(hasData);
  }, []);

  const handleSearch = useCallback(
    (params: Record<string, any>, ...args: any[]) => {
      const nextParams = { ...params };
      delete nextParams.dateRange;
      const hasDateRangeParam = Object.hasOwn(nextParams, 'date_range');
      const hasStartParam = Object.hasOwn(nextParams, 'start');
      const hasEndParam = Object.hasOwn(nextParams, 'end');

      if (!hasDateRangeParam && !hasStartParam && !hasEndParam) {
        nextParams.date_range = queryParams?.date_range;
        nextParams.start = queryParams?.start;
        nextParams.end = queryParams?.end;
      }

      if (nextParams.start && nextParams.end) {
        nextParams.date_range = undefined;
      } else {
        nextParams.start = undefined;
        nextParams.end = undefined;
        if (nextParams.date_range === 'all-time') {
          nextParams.date_range = undefined;
        }
      }

      onSearch?.(nextParams, ...args);
    },
    [onSearch, queryParams?.date_range, queryParams?.end, queryParams?.start]
  );

  const handleSearchTypeChange = useCallback(
    (type: string) => {
      handleSearch({ ...(queryParams || {}), search_type: type, from: 0 }, true, true);
    },
    [handleSearch, queryParams]
  );

  const handleFuzzinessChange = useCallback(
    (value: number) => {
      const nextFuzziness = normalizeSearchFuzziness(value);
      handleSearch({ ...(queryParams || {}), fuzziness: nextFuzziness, from: 0 }, true, true);
    },
    [handleSearch, queryParams]
  );

  const handleSortChange = useCallback(
    (value: string) => {
      const nextSort = normalizeSearchSort(value);
      handleSearch({ ...(queryParams || {}), sort: nextSort, from: 0 }, true, true);
    },
    [handleSearch, queryParams]
  );

  const handleDateRangeChange = useCallback(
    (value: string) => {
      handleSearch(
        { ...(queryParams || {}), date_range: value, start: undefined, end: undefined, from: 0 },
        true,
        true
      );
    },
    [handleSearch, queryParams]
  );

  const handleCustomDateRangeChange = useCallback(
    (range: { start?: string; end?: string }) => {
      handleSearch(
        { ...(queryParams || {}), start: range.start, end: range.end, date_range: undefined, from: 0 },
        true,
        true
      );
    },
    [handleSearch, queryParams]
  );

  const listType = useMemo(() => {
    if (!LIST_TYPES || LIST_TYPES.length === 0) return undefined;
    return LIST_TYPES.find(item => item.type === content_category) || LIST_TYPES[0];
  }, [content_category]);

  // tabs only appear for categories the current result set actually contains —
  // switching to an empty view is worse than not offering it
  const availableCategories = useMemo(() => {
    if (!categoryCounts) return undefined;
    const list = ['all'];
    if (categoryCounts.nonImage > 0) list.push('doc');
    if (categoryCounts.image > 0) list.push('image');
    return list;
  }, [categoryCounts]);

  // the active tab can become empty as the query/filters change (e.g. still on
  // the image tab after refining to a text-only query) — fall back to 全部
  const categoryResetRef = useRef(false);
  useEffect(() => {
    if (!availableCategories || categoryResetRef.current) return;
    if (content_category && content_category !== 'all' && !availableCategories.includes(content_category)) {
      categoryResetRef.current = true;
      handleSearch({ ...(queryParams || {}), 'metadata.content_category': '', from: 0 }, true, false);
    }
  }, [availableCategories, content_category, handleSearch, queryParams]);
  useEffect(() => {
    categoryResetRef.current = false;
  }, [content_category]);

  const hasSearchParams = useMemo(() => {
    return Boolean(query) || Object.keys(filter || {}).length > 0 || Object.keys(aggfilter || {}).length > 0;
  }, [query, JSON.stringify(filter), JSON.stringify(aggfilter)]);

  const hasAggFilter = useMemo(() => {
    return Object.keys(aggfilter || {}).length > 0;
  }, [JSON.stringify(aggfilter)]);

  const isEmptyResult = hasSearchParams && !loading && (hits?.total || 0) === 0 && (data?.length || 0) === 0;

  useEffect(() => {
    if (!Array.isArray(data) || data.length === 0) return;

    data.forEach(item => {
      const ownerId = item?._system?.owner_id;
      if (ownerId && item?.owner && ownerCacheRef.current[ownerId] === undefined) {
        ownerCacheRef.current[ownerId] = item.owner;
      }
    });

    if (typeof getUserEntities !== 'function') return;

    const ownerIds = Array.from(new Set(data.map(item => item?._system?.owner_id).filter(Boolean)));
    const missingOwnerIds = ownerIds.filter(
      id => ownerCacheRef.current[id] === undefined && !pendingOwnerIdsRef.current.has(id)
    );

    if (missingOwnerIds.length === 0) return;

    missingOwnerIds.forEach(id => pendingOwnerIdsRef.current.add(id));

    const markMissingOwnersAsLoaded = () => {
      missingOwnerIds.forEach(id => {
        ownerCacheRef.current[id] = null;
        pendingOwnerIdsRef.current.delete(id);
      });
      setOwnerVersion(prev => prev + 1);
    };

    try {
      const request = getUserEntities(missingOwnerIds, (res: any) => {
        const entities = Array.isArray(res) ? res : Array.isArray(res?.data) ? res.data : [];
        const entityMap = new Map<string, any>();
        entities.forEach((entity: any) => {
          if (entity?.id) {
            entityMap.set(entity.id, entity);
          }
        });

        missingOwnerIds.forEach(id => {
          ownerCacheRef.current[id] = entityMap.get(id) ?? null;
          pendingOwnerIdsRef.current.delete(id);
        });

        setOwnerVersion(prev => prev + 1);
      });

      Promise.resolve(request).catch(markMissingOwnersAsLoaded);
    } catch {
      markMissingOwnersAsLoaded();
    }
  }, [data, getUserEntities]);

  const dataWithOwners = useMemo(() => {
    if (!Array.isArray(data) || data.length === 0) return data;

    return data.map(item => {
      const ownerId = item?._system?.owner_id;
      const owner = ownerId ? ownerCacheRef.current[ownerId] : undefined;

      if (!owner || item?.owner === owner) return item;

      return {
        ...item,
        owner
      };
    });
  }, [data, ownerVersion]);

  const handleGenerateAnswer = useCallback(() => {
    onSearch?.({
      query,
      attachments,
      mode: 'chat',
      action: 'deepthink',
      assistant_id: settings?.deep_think_assistant_entity?.id
    });
  }, [onSearch, query, attachments, settings]);

  const handleSearchFilter = useCallback(
    (nextAggFilter: Record<string, any>) => {
      if (onSearch) {
        handleSearch(
          {
            ...(queryParams || {}),
            aggfilter: nextAggFilter,
            fuzziness,
            sort
          },
          false,
          false
        );
        return;
      }
      onSearchFilter?.(nextAggFilter);
    },
    [fuzziness, handleSearch, onSearch, onSearchFilter, queryParams, sort]
  );

  // clicking a category/tag in the preview panel narrows the current search by
  // that facet value (toggles off when already active); the overlay variants
  // close first so the refreshed result list is visible
  const applyMetaFilter = useCallback(
    (field: string, value: string) => {
      const current = aggfilter[field];
      const currentValues = (Array.isArray(current) ? current : current ? [current] : []).map(String);
      const active = currentValues.includes(value);
      const nextValues = active ? currentValues.filter(v => v !== value) : [...currentValues, value];
      const next = { ...aggfilter };
      if (nextValues.length > 0) next[field] = nextValues;
      else delete next[field];
      handleSearchFilter(next);
    },
    [aggfilter, handleSearchFilter]
  );

  // selected facet values, surfaced in the center column as removable chips so
  // active filters stay visible even with the facet rail collapsed (mobile)
  const removeFilterValue = useCallback(
    (field: string, value: string) => {
      const values = aggfilter[field];
      const rest = (Array.isArray(values) ? values : [values]).filter((v: any) => String(v) !== String(value));
      const next = { ...aggfilter };
      if (rest.length > 0) next[field] = rest;
      else delete next[field];
      handleSearchFilter(next);
    },
    [aggfilter, handleSearchFilter]
  );

  // Backspace on an empty query clears every active filter condition: the
  // aggregation chips above the results and the search box field filters
  const handleEmptyBackspace = useCallback(() => {
    const hasAgg = Object.values(aggfilter || {}).some((v: any) => (Array.isArray(v) ? v.length > 0 : !!v));
    const hasBoxFilters = Object.keys(filter || {}).length > 0;
    if (!hasAgg && !hasBoxFilters) return false;
    handleSearch(
      {
        ...(queryParams || {}),
        query: '',
        aggfilter: {},
        filter: {},
        fuzziness,
        sort
      },
      false,
      false
    );
    return true;
  }, [aggfilter, filter, queryParams, handleSearch, fuzziness, sort]);

  const filterChips = useMemo(() => {
    const entries = Object.entries(aggfilter || {}).flatMap(([field, values]) =>
      (Array.isArray(values) ? values : [values]).filter(Boolean).map(value => ({ field, value: String(value) }))
    );
    if (entries.length === 0) return null;

    const valueName = (field: string, value: string) => {
      // the owner facet carries user ids — resolve the display name from the
      // owner cache (no aggregation behind this field)
      if (field === OWNER_FILTER_FIELD) {
        const owner = ownerCacheRef.current[value];
        return owner?.title ?? owner?.username ?? owner?.name ?? value;
      }
      const agg = (aggregations || []).find((a: any) => a?.key === field);
      const item = agg?.list?.find((i: any) => String(i?.key) === value);
      return item?.name || item?.key || value;
    };
    const fieldLabel = (field: string) =>
      FIELD_LABEL_KEYS[field] ? t(FIELD_LABEL_KEYS[field], { defaultValue: field }) : field;

    return (
      <div className='flex flex-wrap items-center gap-8px'>
        {entries.map(({ field, value }) => (
          <span
            className='group max-w-full inline-flex items-center gap-4px rounded-14px bg-[#F1F3F4] py-3px pl-10px pr-4px text-12px text-[#3C4043] dark:bg-white/10 dark:text-white/80'
            key={`${field}:${value}`}
            title={`${fieldLabel(field)}: ${valueName(field, value)}`}
          >
            <span className='truncate'>
              {fieldLabel(field)}: {valueName(field, value)}
            </span>
            <span
              className='flex-none cursor-pointer rounded-50% p-2px text-[#5F6368] hover:bg-black/10 dark:text-white/60 hover:text-black dark:hover:bg-white/15 dark:hover:text-white'
              onClick={() => removeFilterValue(field, value)}
            >
              <X size={11} />
            </span>
          </span>
        ))}
        <span
          className='cursor-pointer px-4px text-12px text-[var(--ant-color-primary)] hover:underline'
          onClick={() => handleSearchFilter({})}
        >
          {t('labels.clearFilters')}
        </span>
      </div>
    );
  }, [JSON.stringify(aggfilter), aggregations, ownerVersion, removeFilterValue, handleSearchFilter, t]);

  const resultList = isEmptyResult ? (
    <EmptyList
      query={query}
      settings={settings}
      variant={hasAggFilter ? 'filtered' : 'search'}
      onClearFilters={() => handleSearchFilter({})}
      onGenerateAnswer={handleGenerateAnswer}
    />
  ) : listType ? (
    <listType.component
      {...commonProps}
      data={dataWithOwners}
      detailRecord={detailRecord}
      getDetailContainer={getContainer as (() => HTMLElement) | undefined}
      hasMore={hasMore}
      loading={loading}
      query={query}
      setDetailCollapse={setDetailCollapse}
      settings={settings}
      total={hits?.total || 0}
      onCloseDetail={closeDetail}
      onGenerateAnswer={handleGenerateAnswer}
      onLoadMore={onLoadMore}
      onMetaFilter={applyMetaFilter}
      onOpenDetail={openDetail}
    />
  ) : null;

  // while the aggregation query is in flight (fresh search, nothing loaded
  // yet) a skeleton facet rail keeps the sider column reserved from the first
  // paint, so the center column starts at its final width and real facets
  // swap in without squeezing the results. Keyed to "no result has arrived"
  // rather than "no facets" — the agg response lands before the search
  // response, and facets arriving early must not end the skeleton phase or
  // the summary row flashes "0 results" until the real count comes in.
  const isInitialLoading = loading && !data?.length && !(hits?.total > 0);
  const aggregationsNode =
    aggregations?.length > 0 ? (
      <Aggregations
        {...commonProps}
        aggregations={aggregations}
        config={config?.aggregations}
        filter={aggfilter}
        dateRangeValue={dateRange}
        onDateRangeChange={handleDateRangeChange}
        onSearch={handleSearchFilter}
      />
    ) : isInitialLoading ? (
      <div className='flex justify-center pt-32px'>
        <LoadingIcon size={40} />
      </div>
    ) : null;

  useEffect(() => {
    const keys = Object.keys(filter);
    if (keys.length === 0) return;
    const rawKeys = keys.map(k => (k.startsWith('!') ? k.slice(1) : k));
    getFieldsMeta?.(rawKeys, (res: any) => {
      setFilterFieldsMeta(res);
    });
  }, [JSON.stringify(filter)]);

  const toolbar = toolbarVisible ? (
    <Toolbar
      dateRange={dateRange}
      end={end}
      fuzziness={fuzziness}
      searchType={search_type}
      sort={sort}
      start={start}
      onCustomDateRangeChange={handleCustomDateRangeChange}
      onDateRangeChange={handleDateRangeChange}
      onFuzzinessChange={handleFuzzinessChange}
      onSearchTypeChange={handleSearchTypeChange}
      onSortChange={handleSortChange}
    />
  ) : null;

  const histogram =
    histogramData && histogramVisible ? (
      <Histogram
        data={histogramData}
        theme={theme}
        onCustomDateRangeChange={handleCustomDateRangeChange}
      />
    ) : null;

  const layoutCommonProps = {
    ...commonProps,
    getContainer,
    initContainer,
    loading: showFullScreenSpin,
    rightMenuWidth,
    siderCollapse,
    setSiderCollapse,
    // logo === null hides the banner entirely (host app already shows its brand)
    logo:
      logo === null ? null : (
        <Logo
          onLogoClick={handleLogoClick}
          {...commonProps}
          {...logo}
        />
      ),
    resultHeader: (
      <ResultHeader
        {...commonProps}
        hasAggregations={aggregations?.length > 0 || isInitialLoading}
        hits={hits}
        initialLoading={isInitialLoading}
        recommendsCollapse={recommendsCollapse}
        setRecommendsCollapse={setRecommendsCollapse}
        setSiderCollapse={setSiderCollapse}
        siderCollapse={siderCollapse}
        toolbar={toolbar}
      />
    ),
    resultList,
    searchbox: (
      <SearchBox
        {...commonProps}
        attachments={attachments}
        filterFieldsMeta={filterFieldsMeta}
        fuzziness={fuzziness}
        minimize={true}
        placeholder={placeholder}
        queryParams={queryParams}
        searchType={search_type}
        setAttachments={setAttachments}
        setQueryParams={setQueryParams}
        settings={settings}
        sort={sort}
        onSearch={handleSearch}
        onEmptyBackspace={handleEmptyBackspace}
        onSearchTypeChange={handleSearchTypeChange}
        onSuggestion={onSuggestion}
        onUpload={onUpload}
      />
    ),
    tabs: (
      <Categories
        category={content_category}
        categories={availableCategories}
        onChange={category => {
          onCategoryChange?.();
          let shouldAgg = false;
          const shouldAsk = category !== 'image';
          if (category !== content_category) {
            shouldAgg = true;
          }
          handleSearch(
            {
              ...queryParams,
              fuzziness,
              sort,
              'metadata.content_category': category !== 'all' ? category : ''
            },
            shouldAsk,
            shouldAgg
          );
        }}
      />
    ),
    tools: (
      <div className='flex items-center gap-8px'>
        <Button
          className={`px-0 ${toolbarVisible ? 'text-[var(--ant-color-primary)]' : 'text-[#333] dark:text-[#E5E7EB]'}`}
          color='default'
          variant='link'
          onClick={() => setToolbarVisible(visible => !visible)}
        >
          <FilterIcon size={16} />
        </Button>
        {histogramData ? (
          <Button
            className={`px-0 ${histogramVisible ? 'text-[var(--ant-color-primary)]' : 'text-[#333] dark:text-[#E5E7EB]'}`}
            color='default'
            variant='link'
            onClick={() => setHistogramVisible(visible => !visible)}
          >
            <HistogramIcon size={16} />
          </Button>
        ) : null}
      </div>
    ),
    histogram
  };

  if (listType?.type === 'image') {
    return (
      <MediaLayout
        {...layoutCommonProps}
        detailCollapse={detailCollapse}
        aggregations={aggregationsNode}
      />
    );
  }

  const isMobile = Boolean(commonProps?.isMobile);
  // the record preview renders as the layout's inline right pane on wide
  // screens and takes over the full page below the header on narrow ones and
  // mobile — there is no floating-card variant anymore
  const detailPreview = detailRecord ? (
    <ResultDetail
      inline
      apiConfig={commonProps?.apiConfig}
      data={detailRecord}
      theme={theme as 'auto' | 'dark' | 'light'}
      onMetaFilter={applyMetaFilter}
      onClose={closeDetail}
    />
  ) : null;

  return (
    <>
      <BasicLayout
        {...layoutCommonProps}
        filterChips={filterChips}
        hasRecommendsData={hasRecommendsData}
        preview={detailPreview}
        recommendsCollapse={recommendsCollapse}
        setRecommendsCollapse={setRecommendsCollapse}
        aggregations={aggregationsNode}
        aiOverview={
          listType?.showAIOverview && aiOverview?.enabled ? (
            <AIOverviewWrapper
              askBody={askBody}
              config={aiOverview}
              requestHeaders={commonProps?.apiConfig?.headers}
              theme={theme as 'light' | 'dark' | 'auto' | undefined}
              onAsk={onAsk!}
              onChatContinue={onChatContinue}
            />
          ) : null
        }
        recommends={
          <Recommends
            showTitle={true}
            onDataLoaded={handleRecommendsDataLoaded}
            onRecommend={callback => onRecommend?.('hot_topics_for_search_result', callback)}
          />
        }
      />
      {/* leader line from the selected row to the inline preview pane; hidden
          automatically when the pane folds to the full-page variant */}
      {!isMobile && detailRecord?.id != null ? (
        <PreviewConnector activeId={detailRecord.id} getContainer={getContainer} />
      ) : null}
    </>
  );
}
