import BasicLayout from './Layout';
import SearchBox from './SearchBox';
import Logo from './Logo';
import Aggregations from './Aggregations';
import ResultHeader from './ResultHeader';
import { LIST_TYPES } from './ResultList';
import { useEffect, useMemo, useRef, useState } from 'react';
import { formatESResult } from './utils/es';
import Welcome from './Welcome';
import AIOverviewWrapper from './AIOverview/AIOverviewWrapper';

function generateRandomString(size) {
  const characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';
  let result = '';
  for (let i = 0; i < size; i++) {
    const randomIndex = Math.floor(Math.random() * characters.length);
    result += characters.charAt(randomIndex);
  }
  return result;
}

const rootID = generateRandomString(16)

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
  } = props;

  const [currentQuery, setCurrentQuery] = useState({
    from: 0,
    size: 10,
    filters: {}
  })
  const [result, setResult] = useState(formatESResult())
  const [askBody, setAskBody] = useState()
  const [loading, setLoading] = useState(false)

  const [isMobile, setIsMobile] = useState(false);


  const handleSearch = (query, shouldAsk) => {
    setCurrentQuery({
      ...query,
      _t: shouldAsk ? new Date().valueOf() : query._t
    })
    const shouldAgg = query.filters && Object.keys(query.filters).length === 0
    onSearch(query, (res) => {
      let rs
      if (res && !res.error) {
        rs = formatESResult(res);
        setResult((os) => ({
          ...rs,
          aggregations: res?.aggregations ? rs.aggregations : os.aggregations
        }))
      } else {
        setResult()
      }
      if (shouldAsk) {
        setAskBody({
          message: JSON.stringify({
            query: query.keyword,
            result: rs.hits
          }),
          _t: new Date().valueOf()
        })
      }
    }, setLoading, shouldAgg)
  }

  useEffect(() => {
    const checkScreenSize = () => {
      setIsMobile(window.innerWidth < 768);
    };
    
    checkScreenSize();
    window.addEventListener('resize', checkScreenSize);
    
    return () => window.removeEventListener('resize', checkScreenSize);
  }, []);

  useEffect(() => {
    window.onsearch = (keyword) => handleSearch({...currentQuery, from: 0, keyword}, true)
    return () => {
      window.onsearch = undefined
    }
  }, [currentQuery])

  const listType = useMemo(() => {
    if (!LIST_TYPES || LIST_TYPES.length === 0) return undefined
    return LIST_TYPES.find((item) => item.type === type) || LIST_TYPES[0]
  }, [type])

  const commonProps = { isMobile }

  const { keyword, from, size, filters } = currentQuery;

  const { hits, aggregations } = result

  return (
    <BasicLayout
      rootID={rootID}
      isFirst={isFirst}
      loading={loading}
      logo={<Logo isFirst={isFirst} {...commonProps} {...logo}/>}
      welcome={welcome ? <Welcome {...commonProps} text={welcome} /> : null}
      searchbox={<SearchBox {...commonProps} placeholder={placeholder} keyword={keyword} onSearch={(keyword) => handleSearch({...currentQuery, from: 0, keyword}, true)}/>}
      rightMenuWidth={rightMenuWidth}
      aggregations={<Aggregations {...commonProps} config={config.aggregations} aggregations={aggregations} filters={filters} onSearch={(filters) => handleSearch({...currentQuery, filters})}/>}
      resultHeader={<ResultHeader hits={hits} {...commonProps}/>}
      aiOverview={aiOverview?.enabled ? <AIOverviewWrapper askBody={askBody} config={aiOverview} onAsk={onAsk}/> : null}
      resultList={listType ? <listType.component {...commonProps} getDetailContainer={() => root.getElementById(rootID)} from={from} size={size} hits={hits} onSearch={(from, size) => handleSearch({...currentQuery, from, size })}/> : null}
      widgets={(
        <>
          {
            widgets.map((item, index) => (
              <AIOverviewWrapper key={index} askBody={askBody} config={item} onAsk={onAsk}/>
            ))
          }
        </>
      )}
      {...commonProps}
    >
    </BasicLayout>
  );
};

export default Fullscreen;