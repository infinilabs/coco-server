import Logo from "../Logo";
import Recommends from "../Recommends";
import SearchBox from "../SearchBox";
import Welcome from "../Welcome";
import HomeLayout from "../Layout/HomeLayout";

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
}) {
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
      recommends={<Recommends onRecommend={(callback) => onRecommend("hot_topics_for_homepage", callback)}/>}
    />
  );
}