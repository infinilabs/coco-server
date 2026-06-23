import { Image as ImageIcon, Video as VideoIcon } from "lucide-react";

import type {
  SearchResultImageItem,
  SearchResultMediaItem,
  SearchResultsItem,
  SearchResultsSection
} from "../types";

export function itemsToSections(items: SearchResultsItem[], imageGridColumns?: 2 | 3 | 4) {
  const sections: SearchResultsSection[] = [];

  for (const item of items) {
    const last = sections.length ? sections[sections.length - 1] : undefined;

    if (item.type === "imageGroup") {
      const first = item.items[0];
      if (first?.type === "media") {
        sections.push({
          type: "section",
          title: item.title,
          titleIcon: <ImageIcon className="h-4 w-4" />,
          titleIconBgColor: "#FFAF36",
          titleClassName: "text-[#1A0CAB] dark:text-[#8AB4F8]",
          layout: "mediaGrid",
          items: item.items.filter((child): child is SearchResultMediaItem => child.type === "media"),
          columns: item.columns ?? imageGridColumns,
          footerAction: item.footerAction,
          className: item.className
        });
      } else {
        sections.push({
          type: "section",
          title: item.title,
          titleIcon: <ImageIcon className="h-4 w-4" />,
          titleIconBgColor: "#FFAF36",
          titleClassName: "text-[#1A0CAB] dark:text-[#8AB4F8]",
          layout: "imageGrid",
          items: item.items.filter((child): child is SearchResultImageItem => child.type === "image"),
          columns: item.columns ?? imageGridColumns,
          footerAction: item.footerAction,
          className: item.className
        });
      }
      continue;
    }

    if (item.type === "videoGroup") {
      sections.push({
        type: "section",
        title: item.title,
        titleIcon: <VideoIcon className="h-4 w-4" />,
        titleIconBgColor: "#1784FC",
        titleClassName: "text-[#1A0CAB] dark:text-[#8AB4F8]",
        layout: "mediaGrid",
        items: item.items,
        columns: item.columns ?? imageGridColumns,
        footerAction: item.footerAction,
        className: item.className
      });
      continue;
    }

    if (item.type === "result") {
      if (last?.layout === "list") {
        last.items.push(item);
        continue;
      }

      sections.push({
        type: "section",
        layout: "list",
        items: [item]
      });

      continue;
    }

    if (item.type === "media") {
      if (last?.layout === "mediaGrid") {
        last.items.push(item);
        continue;
      }

      sections.push({
        type: "section",
        layout: "mediaGrid",
        ...(imageGridColumns ? { columns: imageGridColumns } : {}),
        items: [item]
      });

      continue;
    }

    if (last?.layout === "imageGrid") {
      last.items.push(item);
      continue;
    }

    sections.push({
      type: "section",
      layout: "imageGrid",
      ...(imageGridColumns ? { columns: imageGridColumns } : {}),
      items: [item]
    });
  }

  return sections;
}
