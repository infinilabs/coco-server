import { Tag, Tooltip } from 'antd';
import type { CSSProperties, MouseEvent, ReactNode } from 'react';
import { useMemo } from 'react';

import { useTranslation } from 'react-i18next';

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
}

/** An inline entity capsule for a `[[type:name]]` wikilink (ontology O4). */
export function WikiLinkTag({ type, label, resolved = false, onClick }: WikiLinkTagProps) {
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
      {type ? (
        <span className="font-mono text-11px opacity-70" title={type}>
          {type}
        </span>
      ) : null}
      <span>{label}</span>
    </Tag>
  );

  if (!resolved) {
    return <Tooltip title={t('page.wiki.wikiLink.unresolved')}>{tag}</Tooltip>;
  }
  return tag;
}

/**
 * Splits text on `[[type:name]]` wikilinks and renders each hit as a
 * capsule; everything between stays plain. Powers the article's
 * non-markdown sections (definition, lists, mentions).
 */
export function renderInlineWikiLinks(
  text: string,
  resolve: (type: string, name: string) => boolean,
  onOpen: (type: string, name: string) => void
): ReactNode[] {
  const out: ReactNode[] = [];
  const re = /\[\[([^:\[\]]+):([^\[\]]+)]]/g;
  let last = 0;
  let m: RegExpExecArray | null;
  let key = 0;
  while ((m = re.exec(text)) !== null) {
    if (m.index > last) out.push(text.slice(last, m.index));
    const type = m[1].trim();
    const label = m[2].trim();
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
  if (last < text.length) out.push(text.slice(last));
  return out;
}
