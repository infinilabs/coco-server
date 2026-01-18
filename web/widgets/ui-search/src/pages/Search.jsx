import { ChartColumn, ListFilter } from "lucide-react";
import Aggregations from "../Aggregations";
import AIOverviewWrapper from "../AIOverview/AIOverviewWrapper";
import Categories from "../Categories";
import BasicLayout from "../Layout/BasicLayout";
import Logo from "../Logo";
import ResultHeader from "../ResultHeader";
import SearchBox from "../SearchBox";
import Welcome from "../Welcome";
import Recommends from "../Recommends";
import { LIST_TYPES } from "../ResultList";
import MediaLayout from "../Layout/MediaLayout";

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
  welcome,
  onSuggestion,
  onRecommend,
}) {

  const { query, filter } = queryParams;
  const content_category = queryParams?.['metadata.content_category']

  const listType = useMemo(() => {
    if (!LIST_TYPES || LIST_TYPES.length === 0) return undefined;
    return LIST_TYPES.find(item => item.type === content_category) || LIST_TYPES[0];
  }, [content_category]);

  if (listType.type === 'image') {
    return (
      <MediaLayout
        {...commonProps}
        getContainer={getContainer}
        initContainer={initContainer}
        loading={showFullScreenSpin}
        rightMenuWidth={rightMenuWidth}
        aggregations={
          aggregations?.length > 0 ? (
            <Aggregations
              {...commonProps}
              aggregations={aggregations}
              config={config.aggregations}
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
            hits={hits}
            {...commonProps}
          />
        }
        resultList={
          listType ? (
            <listType.component
              {...commonProps}
              data={data}
              getDetailContainer={getContainer}
              hasMore={hasMore}
              loading={loading}
              query={query}
              total={hits?.total || 0}
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
          />
        }
        tabs={
          <Categories
            category={content_category}
            onChange={category => {
              setQueryParams({
                ...queryParams,
                'metadata.content_category': category !== 'all' ? category : '',
                t: new Date().valueOf()
              });
            }}
          />
        }
        tools={
          <div className='h-46px flex items-center gap-8px'>
            <ListFilter className='h-16px w-16px' />
            <ChartColumn className='h-16px w-16px' />
          </div>
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
      aggregations={
        aggregations?.length > 0 ? (
          <Aggregations
            {...commonProps}
            aggregations={aggregations}
            config={config.aggregations}
            filter={filter}
            onSearch={onSearchFilter}
          />
        ) : null
      }
      aiOverview={
        listType?.showAIOverview && aiOverview?.enabled ? (
          <AIOverviewWrapper
            askBody={askBody}
            config={aiOverview}
            theme={theme}
            onAsk={onAsk}
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
          hits={hits}
          {...commonProps}
        />
      }
      resultList={
        listType ? (
          <listType.component
            {...commonProps}
            data={data}
            getDetailContainer={getContainer}
            hasMore={hasMore}
            loading={loading}
            query={query}
            total={hits?.total || 0}
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
        />
      }
      tabs={
        <Categories
          category={content_category}
          onChange={category => {
            setQueryParams({
              ...queryParams,
              'metadata.content_category': category !== 'all' ? category : '',
              t: new Date().valueOf()
            });
          }}
        />
      }
      tools={
        <div className='h-46px flex items-center gap-8px'>
          <ListFilter className='h-16px w-16px' />
          <ChartColumn className='h-16px w-16px' />
        </div>
      }
      recommends={<Recommends showTitle={true} onRecommend={(callback) => onRecommend("hot_topics_for_search_result", callback)} />}
    />
  );
}