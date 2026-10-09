import clsx from "clsx";
import { SquareArrowOutUpRight } from "lucide-react";

import { AuthImage } from "./AuthImage";
import { AuthorDate } from "./AuthorDate";
import { BreadcrumbsLine } from "./BreadcrumbsLine";
import { HighlightText } from "./HighlightText";
import { ItemInteractive } from "./ItemInteractive";
import { MetaDot } from "./MetaDot";
import { MetaLine } from "./MetaLine";
import { SectionHeader } from "./SectionHeader";
import { TypeBadge } from "./TypeBadge";

import type { SearchResultListItem, SearchResultsProps } from "../types";

export function ListItem({
  item,
  onItemClick
}: {
  item: SearchResultListItem;
  onItemClick?: SearchResultsProps["onItemClick"];
}) {
  const titleIcon = item.typeIcon ? (
    <TypeBadge typeIcon={item.typeIcon} />
  ) : item.fileType ? (
    <TypeBadge fileType={item.fileType} />
  ) : null;

  const handleClick =
    item.onClick || onItemClick
      ? () => {
          item.onClick?.();
          onItemClick?.(item);
        }
      : undefined;

  const interactiveHref = item.onClick ? undefined : item.href;

  const content = (
    <div className="min-w-0 w-full">
      <div className="flex min-w-0 items-center gap-2">
        <SectionHeader
          className="mb-0 w-full"
          title={item.title}
          titleIcon={titleIcon}
          source={item.source}
          titleClassName="truncate text-[#1A0CAB] dark:text-[#8AB4F8]"
        />
      </div>

      <div className="flex min-w-0 gap-3">
        {item.cover ? (
          <AuthImage
            src={item.cover}
            alt={item.thumbnailAlt ?? item.title}
            className="h-[90px]! w-[160px]! flex-none rounded-lg object-cover ring-1 ring-slate-200 dark:ring-slate-700"
            loading="lazy"
          />
        ) : null}

        <div className="min-w-0 flex-1 flex flex-col justify-between overflow-hidden">
          {item.summary ? (
            <div className="line-clamp-3 text-14px leading-22px text-[#4B5563] dark:text-[#C4C9CF]">
              <HighlightText text={item.summary} />
            </div>
          ) : null}

          {item.breadcrumbs?.length || item.author || item.date ? (
            <div className="mt-2 flex min-w-0 items-center gap-x-6px text-xs text-[#6B7280] dark:text-white/60">
              {item.breadcrumbs?.length ? (
                <BreadcrumbsLine breadcrumbs={item.breadcrumbs} />
              ) : null}
              {item.breadcrumbs?.length && (item.author || item.date) ? <MetaDot /> : null}
              <div
                className={clsx(
                  "flex min-w-0 items-center gap-x-6px",
                  item.breadcrumbs?.length ? "shrink-0" : "flex-1"
                )}
              >
                <AuthorDate author={item.author} date={item.date} />
                {item.href ? (
                  <span
                    className="flex-none text-[#6B7280] dark:text-white/50 opacity-0 transition-opacity group-hover:opacity-100 hover:opacity-100 hover:text-[var(--ant-color-primary)] p-2px rounded-2px"
                    title={item.href}
                    onClick={(e) => {
                      e.stopPropagation();
                      // W17: in-app preview links carry the live query so
                      // the preview can land on and highlight the terms
                      window.open(withHighlightQuery(item.href), "_blank");
                    }}
                  >
                    <SquareArrowOutUpRight size={12}/>
                  </span>
                ) : null}
              </div>
            </div>
          ) : (
            <MetaLine meta={item.meta} />
          )}
        </div>
      </div>
    </div>
  );

  if (!interactiveHref && !handleClick) return content;

  return (
    <ItemInteractive
      href={interactiveHref}
      target={item.target}
      rel={item.rel}
      onClick={handleClick}
      className={clsx(
        "relative border-0 bg-transparent group block w-full rounded-xl px-16px! py-14px! text-left no-underline transition-colors",
        "hover:bg-slate-100/70 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-300",
        "dark:hover:bg-slate-800/60 dark:focus-visible:ring-slate-600",
        // the selected row must read as selected, not as a stuck hover: the
        // hover-equivalent wash is kept but anchored by a primary accent bar
        item.isActive &&
          "bg-slate-100/70 dark:bg-slate-800/60 before:absolute before:bottom-10px before:left-0 before:top-10px before:w-3px before:rounded-full before:bg-[var(--ant-color-primary)] before:content-['']"
      )}
    >
      {content}
    </ItemInteractive>
  );
}

// withHighlightQuery appends ?q=<current search> to in-app preview URLs
// (term highlight in the preview, W17); external links pass untouched.
export function withHighlightQuery(href: string): string {
  try {
    if (!href.includes("/#/preview/")) return href;
    // the app search page mirrors the live query into the URL
    // (enableQueryParams) — reading it survives rerenders and needs no
    // DOM probing
    const q = new URLSearchParams(window.location.search).get("query")?.trim();
    if (!q) return href;
    const sep = href.includes("?") ? "&" : "?";
    return href + sep + "q=" + encodeURIComponent(q);
  } catch {
    return href;
  }
}
