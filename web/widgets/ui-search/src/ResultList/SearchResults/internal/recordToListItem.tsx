import { AuthImage } from "./AuthImage";
import { normalizeFileType } from "./normalizeFileType";

import type { SearchResultListItem, SearchResultsRecord } from "../types";
import { formatDateWithRelative } from "../../../utils/date";
import { cleanHighlightFragment, cleanSummary, decodeHtmlEntities } from "../../../utils/utils"

export function recordToListItem(
  record: SearchResultsRecord,
  index: number,
  onClick?: () => void
): SearchResultListItem {
  const cover = record.thumbnail ?? record.cover ?? record.metadata?.thumbnail_link;
  // prefer the ES highlight fragments (they carry <em> markers around the
  // matched terms); the fragment path strips heading debris in a marker-aware
  // way, the raw-field path is decoded and tidied as plain text
  const hl = record.highlight as Record<string, string[]> | undefined;
  const summary = hl?.content?.[0] ?? hl?.summary?.[0]
    ? cleanHighlightFragment(hl?.content?.[0] ?? hl?.summary?.[0])
    : cleanSummary(decodeHtmlEntities(record.summary ?? record.content));
  const fileType = normalizeFileType(record.metadata?.file_extension ?? record.type);

  const sourceName = record.source?.name;
  const categoryText = record.category ?? record.categories?.join(" / ");
  const breadcrumbs = [sourceName, categoryText].filter(Boolean) as string[];

  const author = record.owner?.title ?? record.owner?.username ?? record.owner?.name ?? record.last_updated_by?.user?.username;
  const date = formatDateWithRelative(record.last_updated_by?.timestamp ?? record.metadata?.last_reviewed ?? record.updated ?? record.created);

  const typeIconUrl = record.metadata?.icon_link ?? record.icon;
  const typeIcon = typeIconUrl ? (
    <AuthImage src={typeIconUrl} alt="" className="h-5 w-5 rounded-sm object-contain" />
  ) : undefined;

  const rawTitle = hl?.title?.[0] ?? record.title;

  return {
    type: "result",
    id: `${record.source?.id ?? record.url ?? record.title}-${index}`,
    title: rawTitle,
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
