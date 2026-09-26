declare const Markdown: (props: {
  content?: string;
  fontSize?: number;
  fontFamily?: string;
  onContextMenu?: (e: React.MouseEvent) => void;
  onDoubleClickCapture?: (e: React.MouseEvent) => void;
  /** first refusal on every rendered link; a non-undefined return replaces the anchor */
  renderLink?: (href: string, children: React.ReactNode) => React.ReactNode | undefined;
}) => React.ReactElement;

export default Markdown;
