import { useEffect } from 'react';
import './script.css';

const useScript = (src: string) => {
  useEffect(() => {
    if (document.querySelector(`script[src="${src}"]`)) {
      return; // Prevent duplicate script loading
    }

    const script = document.createElement('script');
    script.src = src;
    script.async = true;
    document.body.appendChild(script);

    return () => {
      document.body.removeChild(script);
    };
  }, [src]);
};

export const useIconfontScript = () => {
  useScript('/assets/fonts/icons/iconfont.js');
};

export default useScript;
