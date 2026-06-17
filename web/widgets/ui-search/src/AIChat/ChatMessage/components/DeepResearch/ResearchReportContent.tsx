import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";
import Markdown from "@infinilabs/markdown";
import Pdf from "../../../../ResultDetail/DocDetail/DocDetail/components/Preview/components/Pdf";
import HtmlDoc from "../../../../ResultDetail/DocDetail/DocDetail/components/Preview/components/HtmlDoc";
import loadingSvg from "../../../../icons/file-loading.svg";
import loadingFailedSvg from "../../../../icons/file-loading-failed.svg";

const StatusPlaceholder = ({ icon, message, className }: { icon: string; message: string; className?: string }) => (
  <div className={`flex flex-col items-center justify-center text-center ${className || "px-6 h-full"}`}>
    <img className="mb-16px w-80px h-80px" src={icon} />
    <div className="text-sm text-[#999] dark:text-[#666] max-w-[520px] leading-relaxed">
      {message}
    </div>
  </div>
);

export interface ResearchReportData {
  title?: string;
  url?: string;
  created?: string;
  attachment?: string;
  format?: string;
}

export interface ResearchReportContentProps {
  data?: ResearchReportData;
  formatUrl?: (data: any) => string;
  requestHeaders?: Record<string, string>;
  t?: TFunction;
  theme?: string;
}

export const ResearchReportContent = ({
  data,
  formatUrl,
  requestHeaders,
  t: tProp,
  theme,
}: ResearchReportContentProps) => {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | undefined>();

  useEffect(() => {
    setError(undefined);
  }, [data])

  const onLoadingChange = (loading: boolean) => {
    if (loading) {
      setLoading(true);
    } else {
      setTimeout(() => {
        setLoading(false);
      }, 500);
    }
  }

  const formatContent = useMemo(() => {
    if (!data?.url) return null;
    const finalUrl = formatUrl ? formatUrl({ url: data.url }) : data.url;

    if (data?.format === "html") {
      return (
        <HtmlDoc
          data={{}}
          url={finalUrl}
          requestHeaders={requestHeaders}
          onLoadingChange={onLoadingChange}
          onLoadError={(error) => {
            setError(error);
          }}
          theme={(theme as any)}
        />
      )
    }
    if (data?.format === "markdown") {
      return (
        <Markdown url={finalUrl} requestHeaders={requestHeaders} onLoadingChange={onLoadingChange} dark={theme === "dark"} />
      );
    }
    if (data?.format === "pdf") {
      return (
        <Pdf
          data={{}}
          url={finalUrl}
          requestHeaders={requestHeaders}
          onLoadingChange={onLoadingChange}
          onLoadError={(error) => {
            setError(error);
          }}
        />
      )
    }
    return null;
  }, [data?.url, data?.format, requestHeaders, theme]);


  if (!data?.url) {
    return <StatusPlaceholder icon={loadingSvg} message={t("deepResearch.report.generatingTitle")} />;
  }

  if (error && !loading) {
    return <StatusPlaceholder icon={loadingFailedSvg} message={t("deepResearch.report.loadFailed")} />;
  }

  let classes = "px-24px pb-24px";

  if (data?.format === "html") {
    classes = "h-full";
  } else if (data?.format === "pdf") {
    classes = "h-full overflow-hidden p-0";
  }

  return (
    <div className={`w-full relative ${classes} ${loading ? "h-full" : ""}`}>
      {loading && (
        <StatusPlaceholder icon={loadingSvg} message={t("deepResearch.report.loadingTitle")} className="absolute inset-0 z-50" />
      )}

      <div className={loading ? "invisible h-0 overflow-hidden" : "w-full h-full"}>
        {formatContent}
      </div>
    </div>
  );
};