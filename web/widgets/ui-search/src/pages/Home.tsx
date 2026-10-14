import Logo from "../Logo";
import Recommends from "../Recommends";
import SearchBox from "../SearchBox";
import Welcome from "../Welcome";
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
    onUpload,
    attachments,
    setAttachments,
    settings
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
      recommends={<Recommends onRecommend={(callback: any) => onRecommend?.("hot_topics_for_homepage", callback)}/>}
    />
  );
}