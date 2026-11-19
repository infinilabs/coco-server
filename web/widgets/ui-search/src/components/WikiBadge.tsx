import { BookOpenCheck, Shapes } from "lucide-react";

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

// isEntityHit reports whether a search record is an ontology entity pseudo-doc
// (the entity recall leg wraps WikiEntity rows as documents).
export function isEntityHit(record?: {
  metadata?: unknown;
}): boolean {
  if (!record) return false;
  const meta = record.metadata as Record<string, unknown> | undefined;
  return meta?.entity === true;
}

// EntityTypeBadge shows an entity hit's ontology type ("产品", "技术" …) —
// the entity counterpart of the article's Wiki badge: one glance tells the
// reader WHAT kind of thing this row is about. The label comes from the
// ontology schema via the entity leg's entity_type_label metadata, falling
// back to the raw type name; violet matches the entity palette of the graph
// panel and ontology pages.
export function EntityTypeBadge({
  label,
  className,
}: {
  label: string;
  className?: string;
}) {
  return (
    <span
      className={
        "inline-flex flex-none items-center gap-3px rounded-full bg-violet-50 px-7px py-1px text-11px leading-16px font-medium text-violet-700 ring-1 ring-violet-200/80 dark:bg-violet-400/10 dark:text-violet-300 dark:ring-violet-400/25 " +
        (className ?? "")
      }
      title={label}
    >
      <Shapes size={11} strokeWidth={2.4} />
      {label}
    </span>
  );
}
