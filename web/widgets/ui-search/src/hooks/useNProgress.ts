import { useEffect } from "react";
import NProgress from "nprogress";

NProgress.configure({ easing: 'ease', speed: 500 });

export default function useNProgress(loading?: boolean) {
  useEffect(() => {
    if (loading) {
      NProgress.start();
    } else {
      NProgress.done();
    }
    return () => {
      NProgress.done();
    };
  }, [loading]);
}
