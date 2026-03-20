import { useRef, useState } from "react";

import Fullscreen from "./Fullscreen";
import { isEmpty } from "lodash";

const FullscreenPage = (props) => {
  const { onSearch, queryParams, setQueryParams, onLogoClick } = props;

  const [isHome, setIsHome] = useState(true);
  const isHomeRef = useRef(true);

  return (
    <Fullscreen
      {...props}
      isHome={queryParams?.query || !isEmpty(queryParams?.filter) ? false : isHome}
      onSearch={(query, callback, setLoading, shouldAgg) => {
        if (isHomeRef.current) {
          setIsHome(false);
          isHomeRef.current = false;
        }
        onSearch(query, callback, setLoading, shouldAgg);
      }}
      onLogoClick={() => {
        setIsHome(true)
        onLogoClick && onLogoClick()
      }}
    />
  );
};

export default FullscreenPage;
