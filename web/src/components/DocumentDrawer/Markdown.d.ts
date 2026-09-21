declare const Markdown: (props: {
  content?: string;
  fontSize?: number;
  fontFamily?: string;
  onContextMenu?: (e: React.MouseEvent) => void;
  onDoubleClickCapture?: (e: React.MouseEvent) => void;
}) => React.ReactElement;

export default Markdown;
