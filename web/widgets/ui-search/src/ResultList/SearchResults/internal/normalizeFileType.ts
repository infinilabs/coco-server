import type { SearchResultFileType } from "../types";

export function normalizeFileType(input?: string): SearchResultFileType | undefined {
  const value = input?.trim().toLowerCase();
  if (!value) return undefined;
  if (value === "pdf") return "pdf";
  if (value === "doc" || value === "docx" || value === "word") return "doc";
  if (value === "ppt" || value === "pptx") return "ppt";
  if (value === "xls" || value === "xlsx" || value === "excel") return "xls";
  if (value === "link" || value === "url" || value === "html") return "link";
  if (value === "txt" || value === "text") return "text";
  return "unknown";
}

