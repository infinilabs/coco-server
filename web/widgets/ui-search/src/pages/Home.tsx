import Logo from "../Logo";
import Recommends from "../Recommends";
import SearchBox from "../SearchBox";
import Welcome from "../Welcome";
import HomeLayout from "../Layout/HomeLayout";

interface HomeProps {
  commonProps?: Record<string, any>;
  loading?: boolean;
  logo?: Record<string, any>;
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
      logo={
        <Logo
          isHome={true}
          {...commonProps}
          {...logo}
        />
      }
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
            text={welcome}
          />
        ) : null
      }
      recommends={<Recommends onRecommend={(callback: any) => onRecommend?.("hot_topics_for_homepage", callback)}/>}
    />
  );
}