import Aggregations from "../Aggregations";
import AIOverviewWrapper from "../AIOverview/AIOverviewWrapper";
import Categories from "../Categories";
import BasicLayout from "../Layout/BasicLayout";
import Logo from "../Logo";
import ResultHeader from "../ResultHeader";
import SearchBox from "../SearchBox";
import Recommends from "../Recommends";
import { LIST_TYPES } from "../ResultList";
import MediaLayout from "../Layout/MediaLayout";
import { useCallback, useEffect, useMemo, useState } from "react";

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
  onCategoryChange
}: SearchProps) {

  const { query, filter, aggfilter = {} } = queryParams || {};
  const content_category = queryParams?.['metadata.content_category']
  const [siderCollapse, setSiderCollapse] = useState(true)
  const [detailCollapse, setDetailCollapse] = useState(true)
  const [recommendsCollapse, setRecommendsCollapse] = useState(true)
  const [filterFieldsMeta, setFilterFieldsMeta] = useState({})
  const [hasRecommendsData, setHasRecommendsData] = useState(false)
  const handleRecommendsDataLoaded = useCallback((hasData: boolean) => {
    setHasRecommendsData(hasData);
  }, []);

  const listType = useMemo(() => {
    if (!LIST_TYPES || LIST_TYPES.length === 0) return undefined;
    return LIST_TYPES.find(item => item.type === content_category) || LIST_TYPES[0];
  }, [content_category]);

  useEffect(() => {
    const keys = Object.keys(filter)
    if (keys.length === 0) return;
    const rawKeys = keys.map(k => k.startsWith('!') ? k.slice(1) : k);
    getFieldsMeta?.(rawKeys, (res: any) => {
      setFilterFieldsMeta(res)
    })
  }, [JSON.stringify(filter)])

  if (listType?.type === 'image') {
    return (
      <MediaLayout
        {...commonProps}
        getContainer={getContainer}
        initContainer={initContainer}
        loading={showFullScreenSpin}
        rightMenuWidth={rightMenuWidth}
        siderCollapse={siderCollapse}
        setSiderCollapse={setSiderCollapse}
        detailCollapse={detailCollapse}
        aggregations={
          aggregations?.length > 0 ? (
            <Aggregations
              {...commonProps}
              aggregations={aggregations}
              config={config?.aggregations}
              filter={filter}
              onSearch={onSearchFilter}
            />
          ) : null
        }
        logo={
          <Logo
            onLogoClick={handleLogoClick}
            {...commonProps}
            {...logo}
          />
        }
        resultHeader={
          <ResultHeader
            {...commonProps}
            hits={hits}
            hasAggregations={aggregations?.length > 0}
            siderCollapse={siderCollapse}
            setSiderCollapse={setSiderCollapse}
            recommendsCollapse={recommendsCollapse}
            setRecommendsCollapse={setRecommendsCollapse}
          />
        }
        resultList={
          listType ? (
            <listType.component
              {...commonProps}
              data={data}
              getDetailContainer={getContainer as (() => HTMLElement) | undefined}
              hasMore={hasMore}
              loading={loading}
              total={hits?.total || 0}
              setDetailCollapse={setDetailCollapse}
              onLoadMore={onLoadMore}
            />
          ) : null
        }
        searchbox={
          <SearchBox
            {...commonProps}
            minimize={true}
            placeholder={placeholder}
            queryParams={queryParams}
            setQueryParams={setQueryParams}
            onSearch={onSearch}
            onSuggestion={onSuggestion}
            filterFieldsMeta={filterFieldsMeta}
            onUpload={onUpload}
            attachments={attachments}
            setAttachments={setAttachments}
            settings={settings}
          />
        }
        tabs={
          <Categories
            category={content_category}
            onChange={category => {
              onCategoryChange?.();
              setQueryParams?.({
                ...queryParams,
                'metadata.content_category': category !== 'all' ? category : '',
                t: new Date().valueOf()
              });
            }}
          />
        }
      />
    );
  }

  return (
    <BasicLayout
      {...commonProps}
      getContainer={getContainer}
      initContainer={initContainer}
      loading={showFullScreenSpin}
      rightMenuWidth={rightMenuWidth}
      siderCollapse={siderCollapse}
      setSiderCollapse={setSiderCollapse}
      recommendsCollapse={recommendsCollapse}
      setRecommendsCollapse={setRecommendsCollapse}
      aggregations={
        aggregations?.length > 0 ? (
          <Aggregations
            {...commonProps}
            aggregations={aggregations}
            config={config?.aggregations}
            filter={aggfilter}
            onSearch={onSearchFilter}
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
          />
        ) : null
      }
      logo={
        <Logo
          onLogoClick={handleLogoClick}
          {...commonProps}
          {...logo}
        />
      }
      resultHeader={
        <ResultHeader
          {...commonProps}
          hits={hits}
          siderCollapse={siderCollapse}
          hasAggregations={aggregations?.length > 0}
          setSiderCollapse={setSiderCollapse}
          recommendsCollapse={recommendsCollapse}
          setRecommendsCollapse={setRecommendsCollapse}
        />
      }
      resultList={
        listType ? (
          <listType.component
            {...commonProps}
            data={data}
            getDetailContainer={getContainer as (() => HTMLElement) | undefined}
            hasMore={hasMore}
            loading={loading}
            query={query}
            total={hits?.total || 0}
            setDetailCollapse={setDetailCollapse}
            onLoadMore={onLoadMore}
          />
        ) : null
      }
      searchbox={
        <SearchBox
          {...commonProps}
          minimize={true}
          placeholder={placeholder}
          queryParams={queryParams}
          setQueryParams={setQueryParams}
          onSearch={onSearch}
          onSuggestion={onSuggestion}
          onUpload={onUpload}
          filterFieldsMeta={filterFieldsMeta}
          attachments={attachments}
          setAttachments={setAttachments}
          settings={settings}
        />
      }
      tabs={
        <Categories
          category={content_category}
          onChange={category => {
            onCategoryChange?.();
            let shouldAgg = false
            if (category !== content_category) {
              shouldAgg = true
            }
            onSearch?.({
              ...queryParams,
              'metadata.content_category': category !== 'all' ? category : '',
            }, false, shouldAgg);
          }}
        />
      }
      recommends={<Recommends showTitle={true} onRecommend={(callback) => onRecommend?.("hot_topics_for_search_result", callback)} onDataLoaded={handleRecommendsDataLoaded} />}
      hasRecommendsData={hasRecommendsData}
    />
  );
}