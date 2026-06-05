import { ChevronDown, ChevronUp } from "lucide-react";

export function AIExpandToggle({
  collapsed,
  expandText = "展开更多",
  collapseText = "收起",
  onToggle
}: {
  collapsed: boolean;
  expandText?: string;
  collapseText?: string;
  onToggle: () => void;
}) {
  return (
    <div className="mt-2 flex justify-center">
      <button
        type="button"
        className="inline-flex items-center gap-1 text-sm cursor-pointer text-[#666] dark:text-slate-300 hover:text-[#1677ff] focus:outline-none"
        onClick={onToggle}
      >
        {collapsed ? expandText : collapseText}
        {collapsed ? (
          <ChevronDown className="h-4 w-4" />
        ) : (
          <ChevronUp className="h-4 w-4" />
        )}
      </button>
    </div>
  );
}
