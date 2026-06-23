import type React from "react";

export type SearchResultFileType =
  | "pdf"
  | "doc"
  | "ppt"
  | "xls"
  | "link"
  | "word"
  | "text"
  | "unknown";

export type SearchResultsAction = {
  label: string;
  href?: string;
  target?: string;
  rel?: string;
  onClick?: () => void;
};

export type SearchResultsTheme = "light" | "dark" | "auto";

export type SearchResultCommon = {
  id: string;
  title: string;
  href?: string;
  target?: string;
  rel?: string;
};

export type SearchResultListItem = SearchResultCommon & {
  type: "result";
  summary?: string;
  meta?: string[];
  breadcrumbs?: string[];
  author?: string;
  date?: string;
  cover?: string;
  thumbnailAlt?: string;
  fileType?: SearchResultFileType;
  typeIcon?: React.ReactNode;
  source?: string;
  leading?: React.ReactNode;
  trailing?: React.ReactNode;
  onClick?: () => void;
  isActive?: boolean;
};

export type SearchResultImageItem = SearchResultCommon & {
  type: "image";
  imageUrl: string;
  imageAlt?: string;
  subtitle?: string;
  breadcrumbs?: string[];
  onClick?: () => void;
};

export type SearchResultMediaItem = SearchResultCommon & {
  type: "media";
  mediaType: "image" | "video";
  cover: string;
  thumbnailAlt?: string;
  matchCountText?: string;
  sourceLabel?: string;
  categoryLabel?: string;
  breadcrumbs?: string[];
  onClick?: () => void;
};

export type SearchResultImageGroupItem = {
  type: "imageGroup";
  id: string;
  title?: string;
  items: Array<SearchResultImageItem | SearchResultMediaItem>;
  columns?: 2 | 3 | 4;
  footerAction?: SearchResultsAction;
  className?: string;
};

export type SearchResultVideoGroupItem = {
  type: "videoGroup";
  id: string;
  title?: string;
  items: SearchResultMediaItem[];
  columns?: 2 | 3 | 4;
  footerAction?: SearchResultsAction;
  className?: string;
};

export type SearchResultsRecord = {
  source?: {
    type?: string;
    name?: string;
    id?: string;
  };
  category?: string;
  categories?: string[];
  cover?: string;
  title: string;
  summary?: string;
  type?: string;
  lang?: string;
  content?: string;
  icon?: string;
  thumbnail?: string;
  tags?: string[];
  url?: string;
  size?: number;
  owner?: {
    avatar?: string;
    title?: string;
    name?: string;
    username?: string;
    userid?: string;
    id?: string;
  };
  metadata?: {
    last_reviewed?: string;
    file_extension?: string;
    icon_link?: string;
    has_thumbnail?: boolean;
    thumbnail_link?: string;
    image_media_metadata?: { width?: number; height?: number };
    video_media_metadata?: { durationMillis?: string; width?: number; height?: number };
  } & Record<string, unknown>;
  last_updated_by?: {
    user?: {
      avatar?: string;
      username?: string;
      userid?: string;
    };
    timestamp?: string;
  };
} & Record<string, unknown>;

export type SearchResultsItem =
  | SearchResultListItem
  | SearchResultImageItem
  | SearchResultMediaItem
  | SearchResultImageGroupItem
  | SearchResultVideoGroupItem;

export type SearchResultsSection =
  | {
      type: "section";
      title?: string;
      titleIcon?: React.ReactNode;
      titleIconBgColor?: string;
      titleClassName?: string;
      layout: "list";
      items: SearchResultListItem[];
      footerAction?: SearchResultsAction;
      className?: string;
    }
  | {
      type: "section";
      title?: string;
      titleIcon?: React.ReactNode;
      titleIconBgColor?: string;
      titleClassName?: string;
      layout: "imageGrid";
      items: SearchResultImageItem[];
      columns?: 2 | 3 | 4;
      footerAction?: SearchResultsAction;
      className?: string;
    }
  | {
      type: "section";
      title?: string;
      titleIcon?: React.ReactNode;
      titleIconBgColor?: string;
      titleClassName?: string;
      layout: "mediaGrid";
      items: SearchResultMediaItem[];
      columns?: 2 | 3 | 4;
      footerAction?: SearchResultsAction;
      className?: string;
    };

export type SearchResultsProps = {
  section:
    | SearchResultsSection
    | SearchResultsItem
    | SearchResultsRecord
    | Array<SearchResultsItem | SearchResultsRecord>;
  className?: string;
  theme?: SearchResultsTheme;
  footerAction?: SearchResultsAction;
  onRecordClick?: (record: SearchResultsRecord, index: number) => void;
  onItemClick?: (item: SearchResultsItem) => void;
  hideHeader?: boolean;
  requestHeaders?: Record<string, string>;
};

export type SearchResultsImageGroupProps = Omit<SearchResultsProps, "section"> & {
  section: SearchResultImageGroupItem | SearchResultsRecord[] | SearchResultsSection;
};

export type SearchResultsVideoGroupProps = Omit<SearchResultsProps, "section"> & {
  section: SearchResultVideoGroupItem | SearchResultsRecord[] | SearchResultsSection;
};
