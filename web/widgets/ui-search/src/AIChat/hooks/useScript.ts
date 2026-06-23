import { useEffect } from "react";

export const useScript = (url: string) => {
  useEffect(() => {
    if (typeof document === "undefined") {
      return;
    }
    
    if (document.querySelector(`script[src="${url}"]`)) {
      return;
    }

    const script = document.createElement("script");
    script.src = url;
    script.async = true;
    document.body.appendChild(script);

    return () => {
      // We don't remove the script on unmount because other components might need it
      // and iconfonts are usually global.
    };
  }, [url]);
};

export const useIconfontScript = () => {
  // Coco Server Icons
  useScript("https://at.alicdn.com/t/c/font_4878526_cykw3et0ezd.js");
  // Coco App Icons
  useScript("https://at.alicdn.com/t/c/font_4934333_0u00aavw7iob.js");
};
