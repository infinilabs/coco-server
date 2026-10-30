import { BookOpenCheck } from "lucide-react";

import { useTranslation } from "react-i18next";

// WikiBadge marks hits that come from the curated knowledge base (W16a
// projection rows / wiki pseudo-docs) so readers can tell compiled,
// reviewed content apart from raw datasource documents at a glance.
export function WikiBadge({ className }: { className?: string }) {
  const { t } = useTranslation();
  return (
    <span
      className={
        "inline-flex flex-none items-center gap-3px rounded-full bg-emerald-50 px-7px py-1px text-11px leading-16px font-medium text-emerald-700 ring-1 ring-emerald-200/80 dark:bg-emerald-400/10 dark:text-emerald-300 dark:ring-emerald-400/25 " +
        (className ?? "")
      }
      title={t("labels.wikiBadgeTip")}
    >
      <BookOpenCheck size={11} strokeWidth={2.4} />
      {t("labels.wikiBadge")}
    </span>
  );
}

// isWikiHit reports whether a search record originates from the curated
// knowledge-base layer: projection rows carry source.type "wiki" and
// metadata.wiki_article; wiki pseudo-docs (non-projection leg) set
// metadata.wiki_article too.
export function isWikiHit(record?: {
  source?: { type?: string; id?: string };
  metadata?: unknown;
}): boolean {
  if (!record) return false;
  if (record.source?.type === "wiki") return true;
  const meta = record.metadata as Record<string, unknown> | undefined;
  return meta?.wiki_article === true;
}
