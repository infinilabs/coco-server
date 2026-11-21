import Logo from "../Logo";
import Recommends from "../Recommends";
import SearchBox from "../SearchBox";
import Welcome from "../Welcome";
import HotSearches from "../HotSearches";
import HomeLayout from "../Layout/HomeLayout";

interface HomeProps {
  commonProps?: Record<string, any>;
  loading?: boolean;
  /** pass null to hide the banner logo (host app already shows its brand) */
  logo?: Record<string, any> | null;
  /** page background image per theme: { light, dark } */
  background?: Record<string, any>;
  /** banner slot max height in px (width stays adaptive) */
  logoMaxHeight?: number;
  /** welcome text font size in px (default 30) */
  welcomeFontSize?: number;
  /** brand-gradient welcome text (default on) */
  welcomeGradient?: boolean;
  onSearch?: (...args: any[]) => void;
  placeholder?: string;
  welcome?: string;
  queryParams?: Record<string, any>;
  setQueryParams?: (params: any) => void;
  onSuggestion?: (...args: any[]) => void;
  onRecommend?: (...args: any[]) => void;
  onHotSearches?: (callback: (res: { items?: { query: string; count: number }[] }) => void) => void;
  onUpload?: (...args: any[]) => void;
  attachments?: any[];
  setAttachments?: (attachments: any[]) => void;
  settings?: Record<string, any>;
  [key: string]: any;
}

export default function Home({ 
    commonProps, 
    loading, 
    logo, 
    background,
    logoMaxHeight,
    welcomeFontSize,
    welcomeGradient,
    onSearch, 
    placeholder, 
    welcome, 
    queryParams,
    setQueryParams, 
    onSuggestion,
    onRecommend,
    onHotSearches,
    onUpload,
    attachments,
    setAttachments,
    settings,
    defaultSearchType
}: HomeProps) {
  return (
    <HomeLayout
      {...commonProps}
      loading={loading}
      background={background}
      logoMaxHeight={logoMaxHeight}
      logo={logo === null ? null : (
        <Logo
          isHome={true}
          {...commonProps}
          {...logo}
        />
      )}
      searchbox={
        <SearchBox
          {...commonProps}
          placeholder={placeholder}
          queryParams={queryParams}
          setQueryParams={setQueryParams}
          onSearch={onSearch}
          onSuggestion={onSuggestion}
          onUpload={onUpload}
          attachments={attachments}
          setAttachments={setAttachments}
          settings={settings}
          defaultSearchType={defaultSearchType}
        />
      }
          welcome={
            welcome ? (
              <Welcome
                {...commonProps}
                fontSize={welcomeFontSize}
                gradient={welcomeGradient}
                text={welcome}
              />
            ) : null
          }
      recommends={
        <>
          <Recommends
            visibleCount={3}
            onRecommend={(callback: any) => onRecommend?.("hot_topics_for_homepage", callback)}
            onPick={query => setQueryParams?.({ ...(queryParams || {}), query, from: 0 })}
          />
          {/* hot searches (search-log aggregation) sit under the recommend
              topics — picking one runs it as a search via the URL params,
              same flow as typing it into the box */}
          <HotSearches
            onHotSearches={callback => onHotSearches?.(callback)}
            onPick={query => setQueryParams?.({ ...(queryParams || {}), query, from: 0 })}
          />
        </>
      }
    />
  );
}