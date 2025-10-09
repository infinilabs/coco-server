import { useEffect, useMemo, useRef, useState } from "react";

import BasicLayout from "./Layout";
import SearchBox from "./SearchBox";
import Logo from "./Logo";
import Aggregations from "./Aggregations";
import ResultHeader from "./ResultHeader";
import { LIST_TYPES } from "./ResultList";
import { formatESResult } from "./utils/es";
import Welcome from "./Welcome";
import AIOverviewWrapper from "./AIOverview/AIOverviewWrapper";

function generateRandomString(size) {
  const characters =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  let result = "";
  for (let i = 0; i < size; i++) {
    const randomIndex = Math.floor(Math.random() * characters.length);
    result += characters.charAt(randomIndex);
  }
  return result;
}

const rootID = generateRandomString(16);

const Fullscreen = (props) => {
  const {
    logo = {},
    placeholder,
    welcome,
    type,
    aiOverview,
    widgets = [],
    onSearch,
    onAsk,
    config = {},
    root,
    isFirst = false,
    rightMenuWidth,
    queryParams = {},
    setQueryParams,
  } = props;

  const [result, setResult] = useState(formatESResult());
  const [askBody, setAskBody] = useState();
  const [loading, setLoading] = useState(false);

  const [isMobile, setIsMobile] = useState(false);
  const shouldAskRef = useRef(true);

  const handleSearch = (query, shouldAsk) => {
    shouldAskRef.current = shouldAsk;
    setQueryParams({
      ...query,
      t: new Date().valueOf(),
    });
  };

  useEffect(() => {
    const checkScreenSize = () => {
      setIsMobile(window.innerWidth < 768);
    };

    checkScreenSize();
    window.addEventListener("resize", checkScreenSize);

    return () => window.removeEventListener("resize", checkScreenSize);
  }, []);

  useEffect(() => {
    if (queryParams.query) {
      const shouldAgg =
        queryParams.filter && Object.keys(queryParams.filter).length === 0;
      onSearch(
        queryParams,
        (res) => {
          let rs;
          if (res && !res.error) {
            rs = formatESResult(res);
            setResult((os) => ({
              ...rs,
              aggregations: res?.aggregations
                ? rs.aggregations
                : os.aggregations,
            }));
          } else {
            setResult(formatESResult());
          }
          if (shouldAskRef.current) {
            shouldAskRef.current = false;
            setAskBody({
              message: JSON.stringify({
                query: queryParams.query,
                result: rs.hits,
              }),
              t: new Date().valueOf(),
            });
          }
        },
        setLoading,
        shouldAgg,
      );
    }
  }, [queryParams]);

  useEffect(() => {
    window.onsearch = (query) =>
      handleSearch({ ...queryParams, from: 0, query }, true);
    return () => {
      window.onsearch = undefined;
    };
  }, [queryParams]);

  const listType = useMemo(() => {
    if (!LIST_TYPES || LIST_TYPES.length === 0) return undefined;
    return LIST_TYPES.find((item) => item.type === type) || LIST_TYPES[0];
  }, [type]);

  const commonProps = { isMobile };

  const { query, from, size, filter } = queryParams;

  const { hits, aggregations } = result;

  const handleLogoClick = () => {
    // Return to start by clearing search query and resetting to first page
    setQueryParams({
      from: 0,
      size: 10,
      query: '',
      filter: {},
      sort: ''
    })
  }

  return (
    <BasicLayout
      rootID={rootID}
      isFirst={isFirst}
      loading={loading}
      logo={<Logo isFirst={isFirst} onLogoClick={handleLogoClick} {...commonProps} {...logo}/>}
      welcome={welcome ? <Welcome {...commonProps} text={welcome} /> : null}
      searchbox={
        <SearchBox
          {...commonProps}
          placeholder={placeholder}
          query={query}
          onSearch={(query) =>
            handleSearch({ ...queryParams, from: 0, query }, true)
          }
        />
      }
      rightMenuWidth={rightMenuWidth}
      aggregations={
        <Aggregations
          {...commonProps}
          config={config.aggregations}
          aggregations={aggregations}
          filter={filter}
          onSearch={(filter) => handleSearch({ ...queryParams, filter })}
        />
      }
      resultHeader={<ResultHeader hits={hits} {...commonProps} />}
      aiOverview={
        !aiOverview?.enabled ? (
          <AIOverviewWrapper
            askBody={askBody}
            config={aiOverview}
            onAsk={onAsk}
          />
        ) : null
      }
      resultList={
        listType ? (
          <listType.component
            {...commonProps}
            getDetailContainer={() => root.getElementById(rootID)}
            from={from}
            size={size}
            hits={hits}
            query={query}
            onSearch={(from, size) =>
              handleSearch({ ...queryParams, from, size })
            }
          />
        ) : null
      }
      widgets={
        <>
          {widgets.map((item, index) => (
            <AIOverviewWrapper
              key={index}
              askBody={askBody}
              config={item}
              onAsk={onAsk}
            />
          ))}
        </>
      }
      {...commonProps}
    ></BasicLayout>
  );
};

export default Fullscreen;
