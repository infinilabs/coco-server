import type { FC } from "react";
import Markdown from "@infinilabs/markdown";
import { DocDetailProps } from "../..";

const AIInterpretation: FC<DocDetailProps> = (props) => {
  const { data } = props;

  return <Markdown content={data?.ai_insights?.text} />;
};

export default AIInterpretation;
