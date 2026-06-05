import { useEffect } from "react";
import NProgress from "nprogress";
import "nprogress/nprogress.css";

NProgress.configure({ easing: 'ease', speed: 500 });

export default function useNProgress(loading) {
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
