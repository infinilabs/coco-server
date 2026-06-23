import { Globe } from "lucide-react";
import clsx from "clsx";
import { useTranslation } from "react-i18next";
import type { TFunction } from "i18next";

import CommonPopover, { type DataSource } from "./CommonPopover";

export type { DataSource };

interface SearchPopoverProps {
  datasource: { enabled?: boolean; visible?: boolean };
  selectedIds: string[];
  onSelectionChange: (ids: string[]) => void;
  isSearchActive: boolean;
  setIsSearchActive: (val: boolean) => void;
  getDataSources: (query?: string) => Promise<DataSource[]>;
  shortcut?: string;
  t?: TFunction;
}

export default function SearchPopover({
  datasource,
  selectedIds,
  onSelectionChange,
  isSearchActive,
  setIsSearchActive,
  getDataSources,
  t: tProp,
}: SearchPopoverProps) {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;

  return (
    <CommonPopover
      visible={datasource?.visible}
      selectedIds={selectedIds}
      onSelectionChange={onSelectionChange}
      isActive={isSearchActive}
      setIsActive={setIsSearchActive}
      fetchData={getDataSources}
      icon={
        <Globe
          className={clsx("size-4", isSearchActive ? "text-[var(--ant-color-primary)]" : "text-#333 dark:text-#666")}
        />
      }
      label={t("search.input.search") || "Search"}
      title={t("search.input.search") || "Search"}
      t={t}
    />
  );
}
