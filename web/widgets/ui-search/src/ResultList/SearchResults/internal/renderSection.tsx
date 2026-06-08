import clsx from "clsx";

import { ImageItem } from "./ImageItem";
import { ListItem } from "./ListItem";
import { MediaItem } from "./MediaItem";
import { SectionFooter } from "./SectionFooter";
import { SectionHeader } from "./SectionHeader";

import type { SearchResultsProps, SearchResultsSection } from "../types";

export function renderSection(
  section: SearchResultsSection,
  onItemClick?: SearchResultsProps["onItemClick"],
  hideHeader?: boolean
) {
  if (section.layout === "list") {
    return (
      <div className={clsx("space-y-6", section.className)}>
        {!hideHeader && (
          <SectionHeader
            title={section.title}
            titleIcon={section.titleIcon}
            titleIconBgColor={section.titleIconBgColor}
            titleClassName={section.titleClassName}
          />
        )}
        {section.items.map((item) => (
          <ListItem key={item.id} item={item} onItemClick={onItemClick} />
        ))}
        <SectionFooter action={section.footerAction} />
      </div>
    );
  }

  if (section.layout === "mediaGrid") {
    const columns = section.columns ?? 3;
    const gridColsClass =
      columns === 2 ? "grid-cols-2" : columns === 4 ? "grid-cols-4" : "grid-cols-3";

    return (
      <div className={clsx("px-6 py-3", section.className)}>
        {!hideHeader && (
          <SectionHeader
            title={section.title}
            titleIcon={section.titleIcon}
            titleIconBgColor={section.titleIconBgColor}
            titleClassName={section.titleClassName}
            titleAction={section.footerAction}
          />
        )}
        <div className={clsx("grid gap-3", gridColsClass)}>
          {section.items.map((item) => (
            <MediaItem key={item.id} item={item} onItemClick={onItemClick} />
          ))}
        </div>
        <SectionFooter action={section.footerAction} />
      </div>
    );
  }

  const columns = section.columns ?? 3;
  const gridColsClass =
    columns === 2 ? "grid-cols-2" : columns === 4 ? "grid-cols-4" : "grid-cols-3";

  return (
    <div className={clsx("px-6 py-3",section.className)}>
      {!hideHeader && (
        <SectionHeader
          title={section.title}
          titleIcon={section.titleIcon}
          titleIconBgColor={section.titleIconBgColor}
          titleClassName={section.titleClassName}
          titleAction={section.footerAction}
        />
      )}
      <div className={clsx("grid gap-3", gridColsClass)}>
        {section.items.map((item) => (
          <ImageItem key={item.id} item={item} onItemClick={onItemClick} />
        ))}
      </div>
      <SectionFooter action={section.footerAction} />
    </div>
  );
}
