import clsx from "clsx";

import { ActionLink } from "./ActionLink";

import type { SearchResultsAction } from "../types";

export function SectionFooter({ action }: { action?: SearchResultsAction }) {
  if (!action) return null;
  return (
    <div className="mt-4 flex w-full items-center">
      <span className="h-px flex-1 bg-[#e8e8e8] dark:bg-slate-700" aria-hidden="true" />
      <ActionLink
        action={action}
        className={clsx("px-4")}
      />
      <span className="h-px flex-1 bg-[#e8e8e8] dark:bg-slate-700" aria-hidden="true" />
    </div>
  );
}
