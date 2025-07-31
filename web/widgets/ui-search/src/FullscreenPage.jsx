import { useRef, useState } from 'react';
import Fullscreen from './Fullscreen';

const FullscreenPage = (props) => {

  const { 
    onSearch,
  } = props;

  const [isFirst, setIsFirst] = useState(true);
  const isFirstRef = useRef(true);

  return (
    <Fullscreen
      {...props}
      isFirst={isFirst}
      onSearch={(query, callback, setLoading, shouldAgg) => {
        if (isFirstRef.current) {
            setIsFirst(false)
            isFirstRef.current = false
        }
        onSearch(query, callback, setLoading, shouldAgg)
      }}
    />
  );
};

export default FullscreenPage;