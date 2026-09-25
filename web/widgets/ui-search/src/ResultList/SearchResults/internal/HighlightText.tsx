import { Fragment } from "react";

import { decodeHtmlEntities } from "../../../utils/utils";

/**
 * Render an ES highlight fragment — plain text carrying literal `<em>…</em>`
 * markers around matched terms — with the hits styled as subtle pills.
 *
 * The string is split on the markers and re-assembled as React nodes, so any
 * markup embedded in the indexed content stays inert text; nothing is ever
 * injected as HTML.
 */
export function HighlightText({ text }: { text?: string }) {
  if (!text) return null;

  const decoded = decodeHtmlEntities(text) ?? text;
  const parts = decoded.split(/(<em>|<\/em>)/);
  let inHit = false;

  const nodes = parts.map((part, index) => {
    if (part === "<em>") {
      inHit = true;
      return null;
    }
    if (part === "</em>") {
      inHit = false;
      return null;
    }
    if (!part) return null;
    if (!inHit) return <Fragment key={index}>{part}</Fragment>;
    return (
      <mark
        key={index}
        className="rounded-2px bg-[#E8F0FE] px-2px text-[#174EA6] dark:bg-[#8AB4F8]/20 dark:text-[#8AB4F8]"
      >
        {part}
      </mark>
    );
  });

  return <span>{nodes}</span>;
}
