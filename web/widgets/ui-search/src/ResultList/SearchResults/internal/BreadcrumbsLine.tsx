import { Tooltip } from "antd";
import { ChevronRight } from "lucide-react";

export function BreadcrumbsLine({ breadcrumbs }: { breadcrumbs?: string[] }) {
  if (!breadcrumbs?.length) return null;
  return (
    <div className="flex min-w-0 flex-1 items-center gap-x-4px text-xs">
      {breadcrumbs.map((text, index) =>
        index === breadcrumbs.length - 1 ? (
          // the last crumb is usually the long one (a path/category) — let it
          // absorb the remaining width and keep the full text on hover
          <Tooltip key={`${text}-${index}`} title={text}>
            <span className="min-w-0 flex-1 cursor-default truncate">{text}</span>
          </Tooltip>
        ) : (
          <span className="flex min-w-0 items-center gap-x-4px" key={`${text}-${index}`}>
            <span className="min-w-0 truncate">{text}</span>
            <ChevronRight className="size-3 shrink-0 opacity-45" />
          </span>
        )
      )}
    </div>
  );
}
