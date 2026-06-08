import { type FC } from "react";
import Markdown from "@infinilabs/markdown";

import Pdf from "./components/Pdf";
import Docx from "./components/Docx";
import Pptx from "./components/Pptx";
import Image from "./components/Image";
import Video from "./components/Video";
import { DocDetailProps, MetadataContentType } from "../..";

const Preview: FC<DocDetailProps> = (props) => {
  const { data } = props;

  const renderFile = (type: MetadataContentType, url: string) => {
    if (type === "markdown") {
      return <Markdown url={url} requestHeaders={props.requestHeaders} />;
    }

    if (type === "pdf") {
      return <Pdf url={url} {...props} />;
    }

    if (type === "docx") {
      return <Docx url={url} {...props} />;
    }

    if (type === "pptx") {
      return <Pptx url={url} {...props} />;
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
    return <Video url={url} requestHeaders={props.requestHeaders} />;
  }

  return renderFile(type, url);
};

export default Preview;
