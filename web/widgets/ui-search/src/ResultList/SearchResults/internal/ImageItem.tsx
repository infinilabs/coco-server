import clsx from "clsx";

import { AuthImage } from "./AuthImage";
import { BreadcrumbsLine } from "./BreadcrumbsLine";
import { ItemInteractive } from "./ItemInteractive";

import type { SearchResultImageItem, SearchResultsProps } from "../types";

export function ImageItem({
  item,
  onItemClick
}: {
  item: SearchResultImageItem;
  onItemClick?: SearchResultsProps["onItemClick"];
}) {
  const interactiveHref = item.onClick ? undefined : item.href;
  const breadcrumbs = item.breadcrumbs ?? [];
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
        "border-0 bg-transparent group block w-full rounded-xl border border-transparent text-left no-underline transition-colors"
      )}
    >
      <div className="overflow-hidden rounded-lg ring-1 ring-slate-200 dark:ring-slate-700">
        <div className="relative aspect-video bg-slate-100 dark:bg-slate-800">
          <AuthImage
            src={item.imageUrl}
            alt={item.imageAlt ?? item.title}
            className="absolute inset-0 h-full w-full object-cover transform-gpu transition-transform duration-300 ease-out group-hover:scale-[1.03]"
            loading="lazy"
          />
        </div>
      </div>
      <div className="mt-2">
        <div className="mb-1 truncate text-sm text-[#1A0CAB] dark:text-[#8AB4F8]">
          {item.title}
        </div>
        {item.subtitle ? (
          <div className="mb-1 truncate text-sm text-[#666] dark:text-white/80">
            {item.subtitle}
          </div>
        ) : null}

        {breadcrumbs.length ? (
          <div className="text-[#666] dark:text-white/80">
            <BreadcrumbsLine breadcrumbs={breadcrumbs} />
          </div>
        ) : null}
      </div>
    </ItemInteractive>
  );
}
