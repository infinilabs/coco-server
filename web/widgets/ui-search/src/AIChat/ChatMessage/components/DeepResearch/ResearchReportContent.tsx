import { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { type TFunction } from "i18next";
import Markdown from "@infinilabs/markdown";
import Pdf from "../../../../ResultDetail/DocDetail/DocDetail/components/Preview/components/Pdf";
import HtmlDoc from "../../../../ResultDetail/DocDetail/DocDetail/components/Preview/components/HtmlDoc";
import loadingSvg from "./loding.svg";

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
      }, 1000);
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
    return (
      <div className="px-6 h-full flex flex-col items-center justify-center text-center">
        <div className="mb-2 text-base font-medium text-[#333333] dark:text-[#E5E7EB]">
          {t("deepResearch.report.generatingTitle")}
        </div>
        <div className="text-sm text-[#999] dark:text-[#666] max-w-[520px] leading-relaxed">
          {t("deepResearch.report.generatingDescription")}
        </div>
      </div>
    );
  }

  if (error && !loading) {
    return (
      <div className="px-6 h-full flex flex-col items-center justify-center text-center">
        <div className="mb-2 text-base font-medium text-[#333333] dark:text-[#E5E7EB]">
          {t("deepResearch.report.loadFailed")}
        </div>
        <div className="text-sm text-[#999] dark:text-[#666] max-w-[520px] leading-relaxed">
          {error.toString()}
        </div>
      </div>
    );
  }

  let classes = "px-24px pb-24px";

  if (data?.format === "html") {
    classes = "h-full";
  } else if (data?.format === "pdf") {
    classes = "overflow-hidden p-0";
  }

  return (
    <div className={`w-full relative ${classes} ${loading ? "h-full" : ""}`}>
      {loading && (
        <div className="absolute inset-0 z-50 bg-white dark:bg-[#1f1f1f] flex flex-col items-center justify-center text-center">
          <img className="mb-16px w-80px h-80px" src={loadingSvg}/>
          <div className="text-sm text-[#999] dark:text-[#666] max-w-[520px] leading-relaxed">
            {t("deepResearch.report.loadingTitle")}
          </div>
        </div>
      )}
      
      <div className={loading ? "invisible h-0 overflow-hidden" : "w-full h-full"}>
        {formatContent}
      </div>
    </div>
  );
};