import { type FC, useState } from "react";
import Markdown from "@infinilabs/markdown";
import { Spin } from "antd";

import Pdf from "./components/Pdf";
import Docx from "./components/Docx";
import Pptx from "./components/Pptx";
import Image from "./components/Image";
import Video from "./components/Video";
import { DocDetailProps, MetadataContentType } from "../..";

const Preview: FC<DocDetailProps> = (props) => {
  const { data } = props;
  const [loading, setLoading] = useState(false);

  const renderFile = (type: MetadataContentType, url: string) => {
    if (type === "markdown") {
      return (
        <Markdown url={url} requestHeaders={props.requestHeaders} onLoadingChange={setLoading} />
      );
    }

    if (type === "pdf") {
      return <Pdf url={url} {...props} onLoadingChange={setLoading} />;
    }

    if (type === "docx") {
      return <Docx url={url} {...props} onLoadingChange={setLoading} />;
    }

    if (type === "pptx") {
      return <Pptx url={url} {...props} onLoadingChange={setLoading} />;
    }

    return null;
  };

  const { url } = data;
  const type = data?.metadata?.content_type;

  if (!type || !url) return null;

  if (type === "image") {
    return <Image {...props} />;
  }

  if (type === "video") {
    return (
      <Spin spinning={loading}>
        <Video url={url} requestHeaders={props.requestHeaders} onLoadingChange={setLoading} />
      </Spin>
    );
  }

  return (
    <Spin spinning={loading}>
      {renderFile(type, url)}
    </Spin>
  );
};

export default Preview;
