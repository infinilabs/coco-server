import { AuthImage } from "./AuthImage";
import { normalizeFileType } from "./normalizeFileType";

import type { SearchResultListItem, SearchResultsRecord } from "../types";
import { formatDateWithRelative } from "../../../utils/date";

export function recordToListItem(
  record: SearchResultsRecord,
  index: number,
  onClick?: () => void
): SearchResultListItem {
  const cover = record.thumbnail ?? record.cover ?? record.metadata?.thumbnail_link;
  const summary = record.summary ?? record.content;
  const fileType = normalizeFileType(record.metadata?.file_extension ?? record.type);

  const sourceName = record.source?.name;
  const categoryText = record.category ?? record.categories?.join(" / ") ?? "Categories";
  const breadcrumbs = [sourceName, categoryText].filter(Boolean) as string[];

  const author = record.owner?.title ?? record.owner?.username ?? record.owner?.name ?? record.last_updated_by?.user?.username;
  const date = formatDateWithRelative(record.last_updated_by?.timestamp ?? record.metadata?.last_reviewed ?? record.updated ?? record.created);

  const typeIconUrl = record.metadata?.icon_link ?? record.icon;
  const typeIcon = typeIconUrl ? (
    <AuthImage src={typeIconUrl} alt="" className="h-5 w-5 rounded-sm object-contain" />
  ) : undefined;

  return {
    type: "result",
    id: `${record.source?.id ?? record.url ?? record.title}-${index}`,
    title: record.title,
    href: record.url,
    summary,
    cover,
    fileType,
    typeIcon,
    breadcrumbs: breadcrumbs.length ? breadcrumbs : undefined,
    author,
    date,
    onClick,
    isActive: !!record.isActive
  };
}
