import { type FC, useState } from "react";
import { useTranslation } from "react-i18next";
import Markdown from "@infinilabs/markdown";

import Pdf from "./components/Pdf";
import Docx from "./components/Docx";
import Pptx from "./components/Pptx";
import Image from "./components/Image";
import Video from "./components/Video";
import { DocDetailProps, MetadataContentType } from "../..";
import loadingSvg from "../../../../../icons/file-loading.svg";
import loadingFailedSvg from "../../../../../icons/file-loading-failed.svg";

const Preview: FC<{ loadingHeight?: string } & DocDetailProps> = (props) => {
  const { data, theme, loadingHeight = '' } = props;
  const { t } = useTranslation();
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<Error | undefined>();

  const onLoadingChange = (loading: boolean) => {
    if (loading) {
      setLoading(true);
    } else {
      setTimeout(() => {
        setLoading(false);
      }, 500);
    }
  }

  const renderFile = (type: MetadataContentType, url: string) => {
    if (type === "image") {
      return (
        <Image
          {...props}
          onLoadingChange={onLoadingChange}
          onLoadError={(error) => {
            setError(error);
          }}
        />
      );
    }

    if (type === "markdown") {
      return (
        <Markdown url={url} requestHeaders={props.requestHeaders} onLoadingChange={onLoadingChange} dark={theme === "dark"} />
      );
    }

    if (type === "pdf") {
      return (
        <Pdf
          url={url} {...props}
          onLoadingChange={onLoadingChange}
          className="mx-[-24px]"
          onLoadError={(error) => {
            setError(error);
          }}
        />
      );
    }

    if (type === "docx") {
      return (
        <Docx
          url={url} {...props}
          onLoadingChange={onLoadingChange}
          onLoadError={(error) => {
            setError(error);
          }}
        />
      );
    }

    if (type === "pptx") {
      return (
        <Pptx
          url={url} {...props}
          onLoadingChange={onLoadingChange}
          onLoadError={(error) => {
            setError(error);
          }}
        />
      );
    }

    if (type === "video") {
      return (
        <Video
          url={url}
          requestHeaders={props.requestHeaders}
          onLoadingChange={onLoadingChange}
          onLoadError={(error) => {
            setError(error);
          }}
        />
      );
    }

    return null;
  };

  const type = data?.metadata?.content_type;
  const url = data?.metadata?.raw_content;

  if (!type || !url) return null;

  if (error && !loading) {
    return (
      <div className={`px-6 flex flex-col items-center justify-center text-center ${ loadingHeight || 'h-full' }`}>
        <img className="mb-16px w-80px h-80px" src={loadingFailedSvg} />
        <div className="text-sm text-[#999] dark:text-[#666] max-w-[520px] leading-relaxed">
          {t("labels.fileLoadFailed")}
        </div>
      </div>
    );
  }

  return (
    <div className={`w-full relative ${loading ? loadingHeight : 'h-full'}`}>
      {loading && (
        <div className="absolute inset-0 z-50 flex flex-col items-center justify-center text-center">
          <img className="mb-16px w-80px h-80px" src={loadingSvg} />
          <div className="text-sm text-[#999] dark:text-[#666] max-w-[520px] leading-relaxed">
            {t("labels.fileLoading")}
          </div>
        </div>
      )}
      <div className={loading ? "invisible h-0 overflow-hidden" : "w-full h-full"}>
        {renderFile(type, url)}
      </div>
    </div>
  );
};

export default Preview;
