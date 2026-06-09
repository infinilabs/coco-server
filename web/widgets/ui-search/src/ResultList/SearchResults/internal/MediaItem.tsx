import clsx from "clsx";
import { Play } from "lucide-react";

import { AuthImage } from "./AuthImage";
import { BreadcrumbsLine } from "./BreadcrumbsLine";
import { ItemInteractive } from "./ItemInteractive";

import type { SearchResultMediaItem, SearchResultsProps } from "../types";

export function MediaItem({
  item,
  onItemClick,
}: {
  item: SearchResultMediaItem;
  onItemClick?: SearchResultsProps["onItemClick"];
}) {
  const breadcrumbs =
    item.breadcrumbs ??
    ([item.sourceLabel, item.categoryLabel].filter(Boolean) as string[]);
  const interactiveHref = item.onClick ? undefined : item.href;
  return (
    <ItemInteractive
      href={interactiveHref}
      target={item.target}
      rel={item.rel}
      onClick={() => {
        item.onClick?.();
        onItemClick?.(item);
      }}
      className={clsx(
        "border-0 bg-transparent group block w-full rounded-xl border border-transparent text-left no-underline transition-colors",
      )}
    >
      <div className="overflow-hidden rounded-lg ring-1 ring-slate-200 dark:ring-slate-700">
        <div className="relative aspect-4/3 bg-slate-100 dark:bg-slate-800">
          <AuthImage
            src={item.cover}
            alt={item.thumbnailAlt ?? item.title}
            className="absolute inset-0 h-full w-full object-cover transform-gpu transition-transform duration-300 ease-out group-hover:scale-[1.1]"
            loading="lazy"
          />
          {item.mediaType === "video" ? (
            <div className="absolute inset-0 flex items-center justify-center">
              <span className="inline-flex h-10 w-10 items-center justify-center rounded-full bg-black/55 text-white ring-1 ring-white/30">
                <Play className="h-5 w-5 translate-x-px" />
              </span>
            </div>
          ) : null}
        </div>
      </div>

      <div className="mt-2">
        <div className="mb-1 truncate text-sm text-[#1A0CAB] dark:text-[#8AB4F8]">
          {item.title}
        </div>

        {item.matchCountText ? (
          <div className="mb-1 truncate text-xs text-[#666] dark:text-white/80">
            {item.matchCountText}
          </div>
        ) : null}

        {breadcrumbs.length ? (
          <div className="text-[#666] dark:text-white/80">
            <BreadcrumbsLine breadcrumbs={breadcrumbs} />{" "}
          </div>
        ) : null}
      </div>
    </ItemInteractive>
  );
}
