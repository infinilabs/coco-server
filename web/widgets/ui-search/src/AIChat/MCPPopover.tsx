import { Hammer } from "lucide-react";
import clsx from "clsx";
import { useTranslation } from "react-i18next";
import type { TFunction } from "i18next";

import CommonPopover, { type DataSource } from "./CommonPopover";

export type { DataSource };

interface MCPPopoverProps {
  mcp_servers: { enabled?: boolean; visible?: boolean };
  selectedIds: string[];
  onSelectionChange: (ids: string[]) => void;
  isMCPActive: boolean;
  setIsMCPActive: (val: boolean) => void;
  getMCPByServer: (query?: string) => Promise<DataSource[]>;
  shortcut?: string;
  t?: TFunction;
}

export default function MCPPopover({
  mcp_servers,
  selectedIds,
  onSelectionChange,
  isMCPActive,
  setIsMCPActive,
  getMCPByServer,
  t: tProp,
}: MCPPopoverProps) {
  const { t: tOriginal } = useTranslation();
  const t = tProp || tOriginal;

  return (
    <CommonPopover
      visible={mcp_servers?.visible}
      selectedIds={selectedIds}
      onSelectionChange={onSelectionChange}
      isActive={isMCPActive}
      setIsActive={setIsMCPActive}
      fetchData={getMCPByServer}
      icon={
        <Hammer
          className={clsx("size-4", isMCPActive ? "text-[var(--ant-color-primary)]" : "text-#333 dark:text-#666")}
        />
      }
      label={t("search.input.MCP") || "MCP"}
      title={t("search.input.MCP") || "MCP"}
      showItemIcon
      t={t}
    />
  );
}
