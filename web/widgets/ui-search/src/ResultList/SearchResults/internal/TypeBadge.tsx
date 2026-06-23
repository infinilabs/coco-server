import clsx from "clsx";
import {
  FileText,
  FileSpreadsheet,
  File,
  Presentation
} from "lucide-react";
import type React from "react";

import type { SearchResultFileType } from "../types";

function getFileTypeBgClassName(fileType: SearchResultFileType) {
  switch (fileType) {
    case "doc":
    case "word":
      return "bg-[#027FFE] text-white";
    case "pdf":
      return "bg-[#E02E2E] text-white";
    default:
      return "bg-slate-100 text-slate-700 dark:bg-slate-800 dark:text-slate-200";
  }
}

function getFileTypeIcon(fileType: SearchResultFileType) {
  switch (fileType) {
    case "xls":
      return <FileSpreadsheet className="h-5 w-5" />;
    case "ppt":
      return <Presentation className="h-5 w-5" />;
    case "pdf":
    case "doc":
    case "word":
    case "text":
      return <FileText className="h-5 w-5" />;
    default:
      return <File className="h-5 w-5" />;
  }
}

export function TypeBadge({
  fileType,
  typeIcon
}: {
  fileType?: SearchResultFileType;
  typeIcon?: React.ReactNode;
}) {
  if (typeIcon) {
    return (
      <span className="inline-flex h-5 w-5 items-center justify-center">{typeIcon}</span>
    );
  }

  if (!fileType) return null;

  return (
    <span
      className={clsx(
        "inline-flex h-6 w-6 items-center justify-center rounded-md",
        getFileTypeBgClassName(fileType)
      )}
    >
      {getFileTypeIcon(fileType)}
    </span>
  );
}
