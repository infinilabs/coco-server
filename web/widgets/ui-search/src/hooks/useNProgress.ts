import { useEffect } from "react";
import scopedNProgress from "../utils/nprogress";

scopedNProgress.configure({ easing: 'ease', speed: 500 });

/**
 * Toggle the widget's own scoped progress bar based on a `loading` flag.
 *
 * This intentionally does NOT use the `nprogress` package. See
 * `src/utils/nprogress.ts` for why (the host app also uses `nprogress`, and the
 * shared module/CSS would otherwise break the host's top-of-page bar).
 */
export default function useNProgress(loading?: boolean) {
  useEffect(() => {
    if (loading) {
      scopedNProgress.start();
    } else {
      scopedNProgress.done();
    }
    return () => {
      scopedNProgress.done();
    };
  }, [loading]);
}