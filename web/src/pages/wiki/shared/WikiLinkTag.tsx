import { Popover, Tag, Tooltip } from 'antd';
import type { CSSProperties, MouseEvent, ReactNode } from 'react';
import { useMemo } from 'react';

import { useTranslation } from 'react-i18next';

import { parseWikilinkInner } from './content';

// Same palette and hash as KbGraph so a type keeps its color everywhere —
// capsule in an article, node on the canvas.
const TYPE_PALETTE = ['#722ed1', '#13c2c2', '#fa8c16', '#52c41a', '#eb2f96', '#2f54eb', '#faad14', '#a0d911'];

export function wikiTypeColor(type?: string): string {
  if (!type) return TYPE_PALETTE[0];
  let hash = 0;
  for (let i = 0; i < type.length; i += 1) hash = (hash * 31 + type.charCodeAt(i)) >>> 0;
  return TYPE_PALETTE[hash % TYPE_PALETTE.length];
}

interface WikiLinkTagProps {
  type?: string;
  label: string;
  /** set when the link resolved to an entity at save time */
  resolved?: boolean;
  onClick?: (e: MouseEvent<HTMLSpanElement>) => void;
  children?: ReactNode;
  /** resolved links may carry a hover card (entity preview) — the type
   * itself never renders inline: the capsule color codes it and the card
   * spells it out */
  popover?: ReactNode;
}

/** An inline entity capsule for a `[[type:name]]` wikilink (ontology O4). */
export function WikiLinkTag({ type, label, resolved = false, onClick, popover }: WikiLinkTagProps) {
  const { t } = useTranslation();
  const style = useMemo<CSSProperties>(() => {
    if (!resolved) {
      return {
        border: '1px dashed #bfbfbf',
        color: '#8c8c8c',
        background: 'transparent',
        cursor: 'pointer'
      };
    }
    const color = wikiTypeColor(type);
    return {
      border: `1px solid ${color}66`,
      color,
      background: `${color}14`,
      cursor: 'pointer'
    };
  }, [type, resolved]);

  const tag = (
    <Tag className="m-0 inline-flex items-center gap-2px" style={style} onClick={onClick}>
      <span>{label}</span>
    </Tag>
  );

  if (!resolved) {
    return <Tooltip title={t('page.wiki.wikiLink.unresolved')}>{tag}</Tooltip>;
  }
  if (popover) {
    return (
      <Popover content={popover} mouseEnterDelay={0.35} placement="top" trigger="hover">
        {tag}
      </Popover>
    );
  }
  return tag;
}

/**
 * Splits text on `[[type:name]]` wikilinks and renders each hit as a
 * capsule; markdown links in the remaining text render as real anchors
 * (citation entries carry `[file](#/preview/...)` links — leaving them
 * literal is what made the raw and rendered views drift). Powers the
 * article's non-markdown sections (definition, lists, mentions).
 */
export function renderInlineWikiLinks(
  text: string,
  resolve: (type: string, name: string) => boolean,
  onOpen: (type: string, name: string) => void
): ReactNode[] {
  const out: ReactNode[] = [];
  const re = /\[\[([^\[\]]+)]]/g;
  let last = 0;
  let m: RegExpExecArray | null;
  let key = 0;
  while ((m = re.exec(text)) !== null) {
    const link = parseWikilinkInner(m[1]);
    // an inner that doesn't parse (e.g. "[:x]") stays literal: skip without
    // updating `last` so the raw text rides the next segment
    if (!link) continue;
    if (m.index > last) renderMarkdownLinks(text.slice(last, m.index), `md-${key++}`, out);
    const { type, label } = link;
    out.push(
      <WikiLinkTag
        key={`wl-${key++}`}
        type={type}
        label={label}
        resolved={resolve(type, label)}
        onClick={() => onOpen(type, label)}
      />
    );
    last = re.lastIndex;
  }
  if (last < text.length) renderMarkdownLinks(text.slice(last), `md-${key++}`, out);
  return out;
}

/** render `[label](href)` markdown links inside a plain-text segment */
function renderMarkdownLinks(segment: string, keyPrefix: string, out: ReactNode[]) {
  // href may carry a markdown title: [label](href "title")
  const re = /\[([^\]\[]+)\]\(([^)\s]+)(?:\s+"[^"]*")?\)/g;
  let last = 0;
  let m: RegExpExecArray | null;
  let key = 0;
  while ((m = re.exec(segment)) !== null) {
    if (m.index > last) out.push(segment.slice(last, m.index));
    const href = m[2];
    out.push(
      <a
        key={`${keyPrefix}-${key++}`}
        href={href}
        rel='noreferrer'
        target={/^https?:/i.test(href) ? '_blank' : undefined}
      >
        {m[1]}
      </a>
    );
    last = re.lastIndex;
  }
  if (last < segment.length) out.push(segment.slice(last));
}
