import clsx from "clsx";
import { SquareArrowOutUpRight } from "lucide-react";

import { AuthImage } from "./AuthImage";
import { AuthorDate } from "./AuthorDate";
import { BreadcrumbsLine } from "./BreadcrumbsLine";
import { ItemInteractive } from "./ItemInteractive";
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
            <div className="line-clamp-3 text-14px leading-22px text-[#666] dark:text-white/80">
              {item.summary}
            </div>
          ) : null}

          {item.breadcrumbs?.length || item.author || item.date ? (
            <div className="mt-2 flex min-w-0 items-center gap-8px text-[#666] dark:text-white/80">
              <div className="min-w-0 shrink-0">
                <BreadcrumbsLine breadcrumbs={item.breadcrumbs} />
              </div>
              {
                item.author || item.date ? (
                  <span className="h-3 w-px flex-none bg-[#666]" aria-hidden="true" />
                ) : null
              }
              <div className="flex min-w-0 flex-1 items-center gap-6px">
                <AuthorDate author={item.author} date={item.date} />
                {item.href ? (
                  <span className="flex-none text-[#007EFF] hover:bg-black/5 dark:hover:bg-white/10 p-2px rounded-2px" onClick={(e) => {
                    e.stopPropagation();
                    window.open(item.href, "_blank");
                  }}>
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
        "border-0 bg-transparent group block w-full rounded-xl p-4! text-left no-underline transition-colors",
        "hover:bg-slate-100/70 focus:outline-none focus-visible:ring-2 focus-visible:ring-slate-300",
        "dark:hover:bg-slate-800/60 dark:focus-visible:ring-slate-600",
        item.isActive ? "bg-slate-100/70 dark:bg-slate-800/60" : ''
      )}
    >
      {content}
    </ItemInteractive>
  );
}
