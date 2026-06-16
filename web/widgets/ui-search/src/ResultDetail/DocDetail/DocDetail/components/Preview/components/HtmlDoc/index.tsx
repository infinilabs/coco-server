import { useState, useEffect, type FC } from "react";
import { DocDetailProps } from "@/ResultDetail/DocDetail/DocDetail";

interface HtmlIframeProps extends DocDetailProps {
  url: string;
  onLoadingChange?: (loading: boolean) => void;
  onLoadError?: (error: any) => void;
}

const HtmlDoc: FC<HtmlIframeProps> = (props) => {
  const { url, requestHeaders, onLoadingChange, onLoadError, theme } = props;
  const [renderUrl, setRenderUrl] = useState<string>("");

  useEffect(() => {
    let isCurrent = true;
    let generatedBlobUrl = "";

    if (url) {
      onLoadingChange?.(true);

      fetch(url, { headers: requestHeaders })
        .then((res) => {
          if (!res.ok) throw new Error(`HTTP error! status: ${res.status}`);
          return res.text();
        })
        .then((htmlText) => {
          if (!isCurrent) return;

          const injectedStyles = `
            <style>
              * {
                scrollbar-width: thin;
                scrollbar-color: rgba(144, 147, 153, 0.3) transparent !important;
              }
              *::-webkit-scrollbar-thumb {
                background-color: rgba(144, 147, 153, 0.3) !important;
                border-radius: 7px !important;
              }
              *::-webkit-scrollbar-thumb:hover {
                background-color: rgba(144, 147, 153, 0.3) !important;
                border-radius: 7px !important;
              }
              *::-webkit-scrollbar {
                width: 7px !important;
                height: 7px !important;
              }
              *::-webkit-scrollbar-track-piece {
                background-color: rgba(0, 0, 0, 0) !important;
                border-radius: 10px !important;
              }

              body {
                background: #fff !important;
                margin: 0 !important;
                padding-left: 24px !important;
                padding-right: 24px !important;
                padding-bottom: 24px !important;
              }
            </style>
          `;

          const finalHtmlContent = injectedStyles + htmlText;

          const htmlBlob = new Blob([finalHtmlContent], { type: "text/html;charset=utf-8" });
          generatedBlobUrl = URL.createObjectURL(htmlBlob);
          setRenderUrl(generatedBlobUrl);
        })
        .catch((err) => {
          setRenderUrl("");
          onLoadingChange?.(false);
          onLoadError?.(err);
        })
    }

    return () => {
      isCurrent = false;
      if (generatedBlobUrl) {
        URL.revokeObjectURL(generatedBlobUrl);
      }
    };
  }, [url, JSON.stringify(requestHeaders)]);

  return (
    renderUrl ? (
      <iframe
        src={renderUrl}
        className="w-full border-0 h-full"
        style={{ minHeight: 600 }}
        sandbox="allow-same-origin allow-scripts"
        title="research-report"
        onLoad={() => {
          onLoadingChange?.(false);
        }}
        onError={(error) => {
          setRenderUrl("");
          onLoadingChange?.(false);
          onLoadError?.(error);
        }}
      />
    ) : null
  );
};

export default HtmlDoc;
