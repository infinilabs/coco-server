import Aggregations from "../Aggregations";
import AIOverviewWrapper from "../AIOverview/AIOverviewWrapper";
import Categories from "../Categories";
import BasicLayout from "../Layout/BasicLayout";
import Logo from "../Logo";
import ResultHeader from "../ResultHeader";
import SearchBox from "../SearchBox";
import Recommends from "../Recommends";
import { LIST_TYPES } from "../ResultList";
import { EmptyList } from "../ResultList/EmptyList";
import MediaLayout from "../Layout/MediaLayout";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import FilterIcon from "../icons/FilterIcon";
import HistogramIcon from "../icons/HistogramIcon";
import { Button } from "antd";
import Toolbar from "../Toolbar";
import Histogram from "../Histogram";
import { ACTION_TYPE_SEARCH_KEYWORD, DEFAULT_SEARCH_SORT, normalizeSearchFuzziness, normalizeSearchSort } from "../SearchBox/ActionBar/SearchActions";

interface SearchProps {
  aggregations?: any;
  aiOverview?: Record<string, any>;
  askBody?: any;
  commonProps?: Record<string, any>;
  config?: Record<string, any>;
  data?: any[];
  getContainer?: () => HTMLElement | null;
  handleLogoClick?: () => void;
  hits?: any;
  hasMore?: boolean;
  initContainer?: (ref: HTMLDivElement | null) => void;
  loading?: boolean;
  logo?: Record<string, any>;
  onAsk?: (...args: any[]) => void;
  onSearchFilter?: (aggfilter: Record<string, any>) => void;
  onSearch?: (...args: any[]) => void;
  placeholder?: string;
  queryParams?: Record<string, any>;
  rightMenuWidth?: number;
  showFullScreenSpin?: boolean;
  setQueryParams?: (params: any) => void;
  theme?: string;
  onSuggestion?: (...args: any[]) => void;
  onRecommend?: (...args: any[]) => void;
  onChatContinue?: (session_id: string) => void;
  getFieldsMeta?: (...args: any[]) => any;
  onUpload?: (...args: any[]) => void;
  attachments?: any[];
  setAttachments?: (attachments: any[]) => void;
  settings?: Record<string, any>;
  onLoadMore?: () => void;
  histogramData?: { date: string; count: number }[];
  [key: string]: any;
}

export default function Search({
  aggregations,
  aiOverview,
  askBody,
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
  const fuzziness = normalizeSearchFuzziness(queryParams?.fuzziness);
  const sort = normalizeSearchSort(queryParams?.sort || DEFAULT_SEARCH_SORT);
  const dateRange = typeof queryParams?.date_range === 'string' ? queryParams.date_range : 'all-time';
  const start = typeof queryParams?.start === 'string' || typeof queryParams?.start === 'number' ? String(queryParams.start) : undefined;
  const end = typeof queryParams?.end === 'string' || typeof queryParams?.end === 'number' ? String(queryParams.end) : undefined;
  const content_category = queryParams?.['metadata.content_category']
  const [siderCollapse, setSiderCollapse] = useState(true)
  const [detailCollapse, setDetailCollapse] = useState(true)
  const [recommendsCollapse, setRecommendsCollapse] = useState(true)
  const [toolbarVisible, setToolbarVisible] = useState(false)
  const [histogramVisible, setHistogramVisible] = useState(false)
  const [filterFieldsMeta, setFilterFieldsMeta] = useState({})
  const [hasRecommendsData, setHasRecommendsData] = useState(false)
  const ownerCacheRef = useRef<Record<string, any>>({});
  const pendingOwnerIdsRef = useRef<Set<string>>(new Set());
  const [ownerVersion, setOwnerVersion] = useState(0);
  const getUserEntities = commonProps?.getUserEntities as ((ids: string[], callback?: (data: any) => void) => any) | undefined;
  const handleRecommendsDataLoaded = useCallback((hasData: boolean) => {
    setHasRecommendsData(hasData);
  }, []);

  const handleSearch = useCallback((params: Record<string, any>, ...args: any[]) => {
    const nextParams = { ...params };
    delete nextParams.dateRange;
    const hasDateRangeParam = Object.prototype.hasOwnProperty.call(nextParams, 'date_range');
    const hasStartParam = Object.prototype.hasOwnProperty.call(nextParams, 'start');
    const hasEndParam = Object.prototype.hasOwnProperty.call(nextParams, 'end');

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
  }, [onSearch, queryParams?.date_range, queryParams?.end, queryParams?.start]);

  const handleSearchTypeChange = useCallback((type: string) => {
    handleSearch({ ...(queryParams || {}), search_type: type, from: 0 }, true, true);
  }, [handleSearch, queryParams]);

  const handleFuzzinessChange = useCallback((value: number) => {
    const nextFuzziness = normalizeSearchFuzziness(value);
    handleSearch({ ...(queryParams || {}), fuzziness: nextFuzziness, from: 0 }, true, true);
  }, [handleSearch, queryParams]);

  const handleSortChange = useCallback((value: string) => {
    const nextSort = normalizeSearchSort(value);
    handleSearch({ ...(queryParams || {}), sort: nextSort, from: 0 }, true, true);
  }, [handleSearch, queryParams]);

  const handleDateRangeChange = useCallback((value: string) => {
    handleSearch({ ...(queryParams || {}), date_range: value, start: undefined, end: undefined, from: 0 }, true, true);
  }, [handleSearch, queryParams]);

  const handleCustomDateRangeChange = useCallback((range: { start?: string; end?: string }) => {
    handleSearch({ ...(queryParams || {}), start: range.start, end: range.end, date_range: undefined, from: 0 }, true, true);
  }, [handleSearch, queryParams]);

  const listType = useMemo(() => {
    if (!LIST_TYPES || LIST_TYPES.length === 0) return undefined;
    return LIST_TYPES.find(item => item.type === content_category) || LIST_TYPES[0];
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

    data.forEach((item) => {
      const ownerId = item?._system?.owner_id;
      if (ownerId && item?.owner && ownerCacheRef.current[ownerId] === undefined) {
        ownerCacheRef.current[ownerId] = item.owner;
      }
    });

    if (typeof getUserEntities !== 'function') return;

    const ownerIds = Array.from(new Set(data.map((item) => item?._system?.owner_id).filter(Boolean)));
    const missingOwnerIds = ownerIds.filter((id) => ownerCacheRef.current[id] === undefined && !pendingOwnerIdsRef.current.has(id));

    if (missingOwnerIds.length === 0) return;

    missingOwnerIds.forEach((id) => pendingOwnerIdsRef.current.add(id));

    const markMissingOwnersAsLoaded = () => {
      missingOwnerIds.forEach((id) => {
        ownerCacheRef.current[id] = null;
        pendingOwnerIdsRef.current.delete(id);
      });
      setOwnerVersion((prev) => prev + 1);
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

        missingOwnerIds.forEach((id) => {
          ownerCacheRef.current[id] = entityMap.get(id) ?? null;
          pendingOwnerIdsRef.current.delete(id);
        });

        setOwnerVersion((prev) => prev + 1);
      });

      Promise.resolve(request).catch(markMissingOwnersAsLoaded);
    } catch {
      markMissingOwnersAsLoaded();
    }
  }, [data, getUserEntities]);

  const dataWithOwners = useMemo(() => {
    if (!Array.isArray(data) || data.length === 0) return data;

    return data.map((item) => {
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
      assistant_id: settings?.deep_think_assistant_entity?.id,
    });
  }, [onSearch, query, attachments, settings]);

  const handleSearchFilter = useCallback((nextAggFilter: Record<string, any>) => {
    if (onSearch) {
      handleSearch({
        ...(queryParams || {}),
        aggfilter: nextAggFilter,
        fuzziness,
        sort,
      }, false, false);
      return;
    }
    onSearchFilter?.(nextAggFilter);
  }, [fuzziness, handleSearch, onSearch, onSearchFilter, queryParams, sort]);

  const resultList = isEmptyResult ? (
    <EmptyList
      query={query}
      settings={settings}
      variant={hasAggFilter ? "filtered" : "search"}
      onClearFilters={() => handleSearchFilter({})}
      onGenerateAnswer={handleGenerateAnswer}
    />
  ) : listType ? (
    <listType.component
      {...commonProps}
      data={dataWithOwners}
      getDetailContainer={getContainer as (() => HTMLElement) | undefined}
      hasMore={hasMore}
      loading={loading}
      query={query}
      total={hits?.total || 0}
      settings={settings}
      onGenerateAnswer={handleGenerateAnswer}
      setDetailCollapse={setDetailCollapse}
      onLoadMore={onLoadMore}
    />
  ) : null;

  useEffect(() => {
    const keys = Object.keys(filter)
    if (keys.length === 0) return;
    const rawKeys = keys.map(k => k.startsWith('!') ? k.slice(1) : k);
    getFieldsMeta?.(rawKeys, (res: any) => {
      setFilterFieldsMeta(res)
    })
  }, [JSON.stringify(filter)])

  const toolbar = toolbarVisible ? (
    <Toolbar
      searchType={search_type}
      onSearchTypeChange={handleSearchTypeChange}
      fuzziness={fuzziness}
      onFuzzinessChange={handleFuzzinessChange}
      sort={sort}
      onSortChange={handleSortChange}
      dateRange={dateRange}
      onDateRangeChange={handleDateRangeChange}
      start={start}
      end={end}
      onCustomDateRangeChange={handleCustomDateRangeChange}
    />
  ) : null;

  const histogram = histogramData && histogramVisible ? (
    <Histogram data={histogramData} theme={theme} onCustomDateRangeChange={handleCustomDateRangeChange}/>
  ) : null;

  const layoutCommonProps = {
    ...commonProps,
    getContainer,
    initContainer,
    loading: showFullScreenSpin,
    rightMenuWidth,
    siderCollapse,
    setSiderCollapse,
    logo: (
      <Logo
        onLogoClick={handleLogoClick}
        {...commonProps}
        {...logo}
      />
    ),
    resultHeader: (
      <ResultHeader
        {...commonProps}
        hits={hits}
        hasAggregations={aggregations?.length > 0}
        siderCollapse={siderCollapse}
        setSiderCollapse={setSiderCollapse}
        recommendsCollapse={recommendsCollapse}
        setRecommendsCollapse={setRecommendsCollapse}
        toolbar={toolbar}
      />
    ),
    resultList,
    searchbox: (
      <SearchBox
        {...commonProps}
        minimize={true}
        placeholder={placeholder}
        queryParams={queryParams}
        setQueryParams={setQueryParams}
        searchType={search_type}
        onSearchTypeChange={handleSearchTypeChange}
        fuzziness={fuzziness}
        sort={sort}
        onSearch={handleSearch}
        onSuggestion={onSuggestion}
        onUpload={onUpload}
        filterFieldsMeta={filterFieldsMeta}
        attachments={attachments}
        setAttachments={setAttachments}
        settings={settings}
      />
    ),
    tabs: (
      <Categories
        category={content_category}
        onChange={category => {
          onCategoryChange?.();
          let shouldAgg = false;
          let shouldAsk = category !== 'image';
          if (category !== content_category) {
            shouldAgg = true;
          }
          handleSearch({
            ...queryParams,
            fuzziness,
            sort,
            'metadata.content_category': category !== 'all' ? category : '',
          }, shouldAsk, shouldAgg);
        }}
      />
    ),
    tools: (
      <div className="flex items-center gap-8px">
        <Button
          className={`px-0 ${toolbarVisible ? 'text-[var(--ant-color-primary)]' : 'text-[#333] dark:text-[#E5E7EB]'}`}
          color="default"
          variant="link"
          onClick={() => setToolbarVisible((visible) => !visible)}
        >
          <FilterIcon size={16} />
        </Button>
        {histogramData ? (
          <Button
            className={`px-0 ${histogramVisible ? 'text-[var(--ant-color-primary)]' : 'text-[#333] dark:text-[#E5E7EB]'}`}
            color="default"
            variant="link"
            onClick={() => setHistogramVisible((visible) => !visible)}
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
        aggregations={
          aggregations?.length > 0 ? (
            <Aggregations
              {...commonProps}
              aggregations={aggregations}
              config={config?.aggregations}
              filter={filter}
              onSearch={handleSearchFilter}
            />
          ) : null
        }
      />
    );
  }

  return (
    <BasicLayout
      {...layoutCommonProps}
      recommendsCollapse={recommendsCollapse}
      setRecommendsCollapse={setRecommendsCollapse}
      aggregations={
        aggregations?.length > 0 ? (
          <Aggregations
            {...commonProps}
            aggregations={aggregations}
            config={config?.aggregations}
            filter={aggfilter}
            onSearch={handleSearchFilter}
          />
        ) : null
      }
      aiOverview={
        listType?.showAIOverview && aiOverview?.enabled ? (
          <AIOverviewWrapper
            askBody={askBody}
            config={aiOverview}
            theme={theme as "light" | "dark" | "auto" | undefined}
            onAsk={onAsk!}
            onChatContinue={onChatContinue}
            requestHeaders={commonProps?.apiConfig?.headers}
          />
        ) : null
      }
      recommends={<Recommends showTitle={true} onRecommend={(callback) => onRecommend?.("hot_topics_for_search_result", callback)} onDataLoaded={handleRecommendsDataLoaded} />}
      hasRecommendsData={hasRecommendsData}
    />
  );
}