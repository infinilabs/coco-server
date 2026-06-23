import clsx from "clsx";

import { RequestHeadersContext } from "./internal/RequestHeadersContext";
import { renderSection } from "./internal/renderSection";
import { recordToListItem } from "./internal/recordToListItem";

import type {
  SearchResultImageItem,
  SearchResultListItem,
  SearchResultMediaItem,
  SearchResultsAction,
  SearchResultsProps,
  SearchResultsImageGroupProps,
  SearchResultsTheme,
  SearchResultsItem,
  SearchResultsRecord,
  SearchResultsSection,
  SearchResultsVideoGroupProps
} from "./types";

export function SearchResults({
  section,
  className,
  theme,
  footerAction,
  onRecordClick,
  onItemClick,
  hideHeader,
  requestHeaders
}: SearchResultsProps) {
  const resolvedSection = applyFooterAction(resolveSection(section, onRecordClick), footerAction);
  const resolvedTheme = resolveTheme(theme);
  return (
    <RequestHeadersContext.Provider value={requestHeaders}>
      <div className={clsx(resolvedTheme === "dark" && "dark", className)}>
        {renderSection(resolvedSection, onItemClick, hideHeader)}
      </div>
    </RequestHeadersContext.Provider>
  );
}

function resolveTheme(theme: SearchResultsTheme | undefined): "light" | "dark" | undefined {
  if (!theme) return undefined;
  if (theme === "light") return "light";
  if (theme === "dark") return "dark";
  if (
    typeof window !== "undefined" &&
    typeof window.matchMedia === "function" &&
    window.matchMedia("(prefers-color-scheme: dark)").matches
  ) {
    return "dark";
  }
  return "light";
}

function applyFooterAction(
  section: SearchResultsSection,
  footerAction: SearchResultsAction | undefined
): SearchResultsSection {
  if (!footerAction || section.footerAction) return section;
  if (section.layout === "list") return { ...section, footerAction };
  if (section.layout === "mediaGrid") return { ...section, footerAction };
  return { ...section, footerAction };
}

export function SearchResultsImageGroup(props: SearchResultsImageGroupProps) {
  return <SearchResults {...props} />;
}

export function SearchResultsVideoGroup(props: SearchResultsVideoGroupProps) {
  return <SearchResults {...props} />;
}

function resolveSection(
  input: SearchResultsProps["section"],
  onRecordClick: SearchResultsProps["onRecordClick"] | undefined
): SearchResultsSection {
  if (Array.isArray(input)) return resolveArray(input, onRecordClick);
  if (isSection(input)) return input;

  const value = input as unknown as Record<string, unknown>;

  if (value.type === "imageGroup" && Array.isArray(value.items)) {
    const group = value as unknown as {
      type: "imageGroup";
      title?: string;
      items: unknown[];
      columns?: 2 | 3 | 4;
      footerAction?: SearchResultsSection["footerAction"];
      className?: string;
    };

    const first = group.items[0] as { type?: unknown } | undefined;
    if (first?.type === "media") {
      return {
        type: "section",
        title: group.title,
        layout: "mediaGrid",
        items: group.items.filter(
          (child): child is SearchResultMediaItem =>
            typeof child === "object" && !!child && (child as { type?: unknown }).type === "media"
        ),
        columns: group.columns,
        footerAction: group.footerAction,
        className: group.className
      };
    }

    return {
      type: "section",
      title: group.title,
      layout: "imageGrid",
      items: group.items.filter(
        (child): child is SearchResultImageItem =>
          typeof child === "object" && !!child && (child as { type?: unknown }).type === "image"
      ),
      columns: group.columns,
      footerAction: group.footerAction,
      className: group.className
    };
  }

  if (value.type === "videoGroup" && Array.isArray(value.items)) {
    const group = value as unknown as {
      type: "videoGroup";
      title?: string;
      items: SearchResultMediaItem[];
      columns?: 2 | 3 | 4;
      footerAction?: SearchResultsSection["footerAction"];
      className?: string;
    };
    return {
      type: "section",
      title: group.title,
      layout: "mediaGrid",
      items: group.items,
      columns: group.columns,
      footerAction: group.footerAction,
      className: group.className
    };
  }

  if (value.type === "result") {
    return {
      type: "section",
      layout: "list",
      items: [input as SearchResultListItem]
    };
  }

  if (value.type === "media") {
    return {
      type: "section",
      layout: "mediaGrid",
      items: [input as SearchResultMediaItem]
    };
  }

  if (value.type === "image" && typeof (value as { imageUrl?: unknown }).imageUrl === "string") {
    return {
      type: "section",
      layout: "imageGrid",
      items: [input as SearchResultImageItem]
    };
  }

  // const meta = (value as { metadata?: unknown }).metadata;
  // const metaContentType =
  //   typeof meta === "object" && meta
  //     ? (meta as { content_type?: unknown }).content_type
  //     : undefined;
  // const recordType =
  //   typeof metaContentType === "string"
  //     ? (metaContentType.trim().toLowerCase() as string)
  //     : typeof (value as { type?: unknown }).type === "string"
  //       ? ((value as { type: string }).type.trim().toLowerCase() as string)
  //       : undefined;
  // if (recordType === "image" || recordType === "video") {
  //   const recordId = typeof (value as { id?: unknown }).id === "string" ? (value as { id: string }).id : undefined;
  //   const cover =
  //     typeof (value as { thumbnail?: unknown }).thumbnail === "string"
  //       ? (value as { thumbnail: string }).thumbnail
  //       : typeof (value as { cover?: unknown }).cover === "string"
  //         ? (value as { cover: string }).cover
  //         : typeof (value as { metadata?: unknown }).metadata === "object" && (value as { metadata?: unknown }).metadata
  //           ? typeof ((value as { metadata: { thumbnail_link?: unknown } }).metadata.thumbnail_link) === "string"
  //             ? (value as { metadata: { thumbnail_link: string } }).metadata.thumbnail_link
  //             : undefined
  //           : undefined;

  //   if (cover) {
  //     const categoryLabel =
  //       typeof (value as { category?: unknown }).category === "string"
  //         ? (value as { category: string }).category
  //         : Array.isArray((value as { categories?: unknown }).categories)
  //           ? (value as { categories: unknown[] }).categories
  //               .filter((v): v is string => typeof v === "string")
  //               .join(" / ")
  //           : undefined;

  //     const title = typeof (value as { title?: unknown }).title === "string" ? (value as { title: string }).title : "Untitled";
  //     const url = typeof (value as { url?: unknown }).url === "string" ? (value as { url: string }).url : undefined;
  //     const sourceName =
  //       typeof (value as { source?: unknown }).source === "object" && (value as { source?: unknown }).source
  //         ? typeof ((value as { source: { name?: unknown } }).source.name) === "string"
  //           ? (value as { source: { name: string } }).source.name
  //           : undefined
  //         : undefined;

  //     const mediaItem: SearchResultMediaItem = {
  //       type: "media",
  //       id: recordId ?? `${url ?? title}-0`,
  //       mediaType: recordType === "video" ? "video" : "image",
  //       title,
  //       href: onRecordClick ? undefined : url,
  //       cover,
  //       sourceLabel: sourceName,
  //       categoryLabel,
  //       breadcrumbs: [sourceName, categoryLabel].filter(Boolean) as string[],
  //       ...(onRecordClick ? { onClick: () => onRecordClick(input as SearchResultsRecord, 0) } : {})
  //     };

  //     return {
  //       type: "section",
  //       layout: "mediaGrid",
  //       items: [mediaItem]
  //     };
  //   }
  // }

  return {
    type: "section",
    layout: "list",
    items: [
      recordToListItem(
        input as SearchResultsRecord,
        0,
        onRecordClick ? () => onRecordClick(input as SearchResultsRecord, 0) : undefined
      )
    ]
  };
}

function resolveArray(
  input: Array<SearchResultsItem | SearchResultsRecord>,
  onRecordClick: SearchResultsProps["onRecordClick"] | undefined
): SearchResultsSection {
  if (!input.length) {
    return { type: "section", layout: "list", items: [] };
  }

  if (input.every(isNormalizedItem)) {
    if (input.every((item) => item.type === "result")) {
      return { type: "section", layout: "list", items: input as SearchResultListItem[] };
    }

    if (input.every((item) => item.type === "media")) {
      return { type: "section", layout: "mediaGrid", items: input as SearchResultMediaItem[] };
    }

    if (input.every((item) => item.type === "image" && "imageUrl" in item)) {
      return { type: "section", layout: "imageGrid", items: input as SearchResultImageItem[] };
    }

    if (input.length === 1) return resolveSection(input[0], onRecordClick);
  }

  const mediaItems: SearchResultMediaItem[] = [];
  const listItems: SearchResultListItem[] = [];

  for (const [index, record] of input.entries()) {
    const media = recordToMediaItem(record, index, onRecordClick);
    if (media) {
      mediaItems.push(media);
    } else {
      listItems.push(
        recordToListItem(
          record as SearchResultsRecord,
          index,
          onRecordClick ? () => onRecordClick(record as SearchResultsRecord, index) : undefined
        )
      );
    }
  }

  if (mediaItems.length && !listItems.length) {
    const title = getCommonString(input, (record) => record.category ?? record.source?.name);
    return { type: "section", title, layout: "mediaGrid", items: mediaItems };
  }

  const title = getCommonString(input, (record) => record.category ?? record.source?.name);
  return { type: "section", title, layout: "list", items: listItems };
}

function isNormalizedItem(value: unknown): value is SearchResultsItem {
  if (!value || typeof value !== "object") return false;
  const candidate = value as { type?: unknown };
  if (candidate.type === "result") return true;
  if (candidate.type === "media") return true;
  if (candidate.type === "imageGroup") return true;
  if (candidate.type === "videoGroup") return true;
  if (candidate.type === "image" && typeof (value as { imageUrl?: unknown }).imageUrl === "string") return true;
  return false;
}

function recordToMediaItem(
  record: SearchResultsItem | SearchResultsRecord,
  index: number,
  onRecordClick: SearchResultsProps["onRecordClick"] | undefined
): SearchResultMediaItem | undefined {
  const value = record as unknown as Record<string, unknown>;

  const meta2 = (value as { metadata?: unknown }).metadata;
  const metaContentType2 =
    typeof meta2 === "object" && meta2
      ? (meta2 as { content_type?: unknown }).content_type
      : undefined;
  const recordType =
    typeof metaContentType2 === "string"
      ? (metaContentType2.trim().toLowerCase() as string)
      : typeof (value as { type?: unknown }).type === "string"
        ? ((value as { type: string }).type.trim().toLowerCase() as string)
        : undefined;
  if (recordType !== "image" && recordType !== "video") return undefined;

  const recordId = typeof (value as { id?: unknown }).id === "string" ? (value as { id: string }).id : undefined;
  const cover =
    typeof (value as { thumbnail?: unknown }).thumbnail === "string"
      ? (value as { thumbnail: string }).thumbnail
      : typeof (value as { cover?: unknown }).cover === "string"
        ? (value as { cover: string }).cover
        : typeof (value as { metadata?: unknown }).metadata === "object" && (value as { metadata?: unknown }).metadata
          ? typeof ((value as { metadata: { thumbnail_link?: unknown } }).metadata.thumbnail_link) === "string"
            ? (value as { metadata: { thumbnail_link: string } }).metadata.thumbnail_link
            : undefined
          : undefined;

  if (!cover) return undefined;

  const categoryLabel =
    typeof (value as { category?: unknown }).category === "string"
      ? (value as { category: string }).category
      : Array.isArray((value as { categories?: unknown }).categories)
        ? (value as { categories: unknown[] }).categories
            .filter((v): v is string => typeof v === "string")
            .join(" / ")
        : undefined;

  const title = typeof (value as { title?: unknown }).title === "string" ? (value as { title: string }).title : "Untitled";
  const url = typeof (value as { url?: unknown }).url === "string" ? (value as { url: string }).url : undefined;
  const sourceName =
    typeof (value as { source?: unknown }).source === "object" && (value as { source?: unknown }).source
      ? typeof ((value as { source: { name?: unknown } }).source.name) === "string"
        ? (value as { source: { name: string } }).source.name
        : undefined
      : undefined;

  const shouldUseRecordClick = !!onRecordClick && !isNormalizedItem(record);

  return {
    type: "media",
    id: recordId ?? `${url ?? title}-0`,
    mediaType: recordType === "video" ? "video" : "image",
    title,
    href: shouldUseRecordClick ? undefined : url,
    cover,
    sourceLabel: sourceName,
    categoryLabel,
    breadcrumbs: [sourceName, categoryLabel].filter(Boolean) as string[],
    ...(shouldUseRecordClick ? { onClick: () => onRecordClick(record as SearchResultsRecord, index) } : {})
  };
}

function getCommonString(
  records: Array<SearchResultsItem | SearchResultsRecord>,
  getter: (record: SearchResultsRecord) => string | undefined
): string | undefined {
  const first = records[0] as SearchResultsRecord;
  const firstValue = getter(first);
  if (!firstValue) return undefined;

  for (const record of records) {
    const value = getter(record as SearchResultsRecord);
    if (value !== firstValue) return undefined;
  }

  return firstValue;
}

function isSection(value: unknown): value is SearchResultsSection {
  if (!value || typeof value !== "object") return false;
  const candidate = value as { type?: unknown; layout?: unknown; items?: unknown };
  return (
    candidate.type === "section" &&
    (candidate.layout === "list" || candidate.layout === "imageGrid" || candidate.layout === "mediaGrid") &&
    Array.isArray(candidate.items)
  );
}
