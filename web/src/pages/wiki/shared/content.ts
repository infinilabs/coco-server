/* Structured-markdown parsing for wiki articles — ported from the coco-wiki
 * prototype (parseStructuredContent) and generalized to accept both the zh-CN
 * and en-US section headers of the Obsidian wiki format (see knowledge-hub
 * design doc §3.3). Rendering stays in the page components; this module only
 * parses. */

export interface WikiLink {
  type: string;
  label: string;
}

export interface StructuredContent {
  /** plain content before the first section header — the page lead paragraph */
  intro: string;
  definition: string;
  keyCharacteristics: string[];
  applications: string[];
  relatedConcepts: string[];
  relatedEntities: string[];
  mentions: string[];
  /** 正文内容 body plus any unrecognized sections — rendered as markdown */
  mainContent: string;
}

const SECTION_HEADERS: Record<string, string[]> = {
  definition: ['## 定义', '## Definition'],
  keyCharacteristics: ['## 关键特征', '## Key Characteristics'],
  applications: ['## 应用场景', '## Applications'],
  relatedConcepts: ['## 关联概念', '## Related Concepts'],
  relatedEntities: ['## 关联实体', '## Related Entities'],
  mentions: ['## 来源引用', '## Mentions in Source'],
  mainContent: ['## 正文内容', '## Content']
};

function matchHeader(line: string): string | null {
  for (const [section, headers] of Object.entries(SECTION_HEADERS)) {
    if (headers.includes(line)) return section;
  }
  return null;
}

export function parseStructuredContent(content: string): StructuredContent {
  const result: StructuredContent = {
    intro: '',
    definition: '',
    keyCharacteristics: [],
    applications: [],
    relatedConcepts: [],
    relatedEntities: [],
    mentions: [],
    mainContent: content
  };

  const lines = content.split('\n');
  const firstHeaderIdx = lines.findIndex(line => line.trim().startsWith('## '));
  if (firstHeaderIdx === -1) {
    // free-form content with no sections at all — render as-is
    return result;
  }

  // everything before the first header is the lead paragraph; the structured
  // sections render themselves, so the markdown body only carries the
  // 正文内容 section and any unrecognized sections (dropping them here — or
  // re-rendering the known ones — is what made raw and rendered views drift)
  result.intro = lines
    .slice(0, firstHeaderIdx)
    .join('\n')
    .trim();
  result.mainContent = '';

  let currentSection = '';
  const body: string[] = [];
  for (let i = firstHeaderIdx; i < lines.length; i += 1) {
    const line = lines[i].trim();
    const matched = line.startsWith('## ') ? matchHeader(line) : null;

    if (matched) {
      currentSection = matched;
      continue;
    }
    if (line.startsWith('## ')) {
      // unrecognized section: keep the heading in the rendered markdown so
      // the section still shows its title (dropping it made raw and rendered
      // views disagree — content floated out with no header)
      currentSection = 'other';
      body.push(lines[i]);
      continue;
    }

    switch (currentSection) {
      case 'definition':
        if (line) result.definition += `${result.definition ? ' ' : ''}${line}`;
        break;
      case 'keyCharacteristics':
      case 'applications':
      case 'relatedConcepts':
      case 'relatedEntities':
      case 'mentions':
        if (line.startsWith('- ')) result[currentSection].push(line.slice(2).trim());
        break;
      case 'mainContent':
      case 'other':
        body.push(lines[i]);
        break;
      default:
        break;
    }
  }
  result.mainContent = body.join('\n').trim();
  return result;
}

/** parse `[[type:label]]` wiki links out of a related-list entry */
export function parseWikiLink(entry: string): WikiLink | null {
  const m = entry.match(/^\[\[([^:]+):(.+)]]$/);
  if (!m) return null;
  return { type: m[1].trim(), label: m[2].trim() };
}

/* ---------------- compiled-topic citations ---------------- */

/** one numbered source claim parsed out of a compiled page's 来源主张 list */
export interface CitationRef {
  n: number;
  claim: string;
  docId?: string;
  /** encoded deep-link query for the document preview (highlight the evidence) */
  q?: string;
  excerpt?: string;
}

const CLAIMS_HEADING = /\n?##\s*来源主张\s*\n/;
const SOURCE_LINE = /^> 来源：.*$/gm;
/* boilerplate leads compiled pages used to open with — the header's AI
   badge already says all of it; pages compiled before the leads were
   dropped still carry them in stored content */
const BOILERPLATE_LEAD =
  /^>\s*本页由(?:知识编译|治理队列).*$\n?|^\*\*[^*]+\*\*\s*是知识管道从.*$\n?/gm;
const CLAIM_ITEM = /^(\d+)\.\s+(.+)$/;
/* trailing `[源](#/preview/document/<id>?q=<encoded> "verbatim excerpt")`:
   the excerpt runs from the first `"` to the line's final `"` — stopping the
   match at the first `)` broke on excerpts containing parens (shop codes
   like "(3373)"), left the whole raw link in the claim text and lost the
   excerpt; excerpts may also carry inner quotes (JSON evidence) */
const SOURCE_LINK =
  /\[源\]\(#\/preview\/document\/([A-Za-z0-9_-]+)(?:\?q=([^"\s]*))?[^"]*"(.*)"\s*\)\s*$/;
const SOURCE_LINK_BARE = /\[源\]\(#\/preview\/document\/([A-Za-z0-9_-]+)/;

/**
 * Split a compiled page's content into its reading body and the numbered
 * source-claim list trailing it: the per-section `> 来源：` lines and the
 * compile banner go away (provenance lives on the interactive [n] chips,
 * AI status on the header badge) and the 来源主张 section becomes data
 * for the hover popovers instead of layout bulk.
 */
export function splitCitationBlock(content: string): { body: string; claims: CitationRef[] } {
  if (!content) return { body: '', claims: [] };
  let body = content.replace(BOILERPLATE_LEAD, '').replace(SOURCE_LINE, '');
  const claims: CitationRef[] = [];
  const split = body.split(CLAIMS_HEADING);
  if (split.length > 1) {
    body = split.slice(0, -1).join('\n');
    for (const raw of split[split.length - 1].split('\n')) {
      const m = raw.match(CLAIM_ITEM);
      if (!m) continue;
      const src = raw.match(SOURCE_LINK) ?? raw.match(SOURCE_LINK_BARE);
      claims.push({
        n: Number(m[1]),
        // strip the whole trailing source link — greedy to the line's last `)`
        // so excerpts containing parens don't defeat the strip
        claim: m[2].replace(/\s*\[源\]\(.*\)\s*$/, '').trim(),
        docId: src?.[1],
        q: src?.[2],
        excerpt: src?.[3]
      });
    }
  }
  return { body: body.trim(), claims };
}

/**
 * Turn bare citation markers `[12]` in compiled prose into markdown links
 * (`[12](#cite-12)`) so the reader renders them as interactive chips —
 * wikilinks `[[..]]`, existing markdown links `[x](..)` and code fences
 * are left alone.
 *
 * The prefix guard is a zero-width lookbehind (not a consumed char): consecutive
 * markers like `[14][15]` must EACH match — a consuming prefix leaves the second
 * marker unconverted, rendering it as body-size raw text next to the 10.5px chips.
 */
export function linkifyCitations(body: string): string {
  if (!body) return body;
  // transform only outside fenced code blocks (even-indexed segments)
  return body
    .split('```')
    .map((seg, i) =>
      i % 2 === 0
        ? seg.replace(/(?<![`\[\\])\[(\d{1,3})\](?!\()/g, (_m, n: string) => `[${n}](#cite-${n})`)
        : seg
    )
    .join('```');
}

/* ---------------- evidence URL rendering ---------------- */

export type TextToken = { text: string } | { url: string };

/** absolute URLs plus bare domain paths (cdn.maximscakes.com.hk/x.jpg);
    CJK punctuation and closing brackets end a token, trailing ASCII
    sentence punctuation is trimmed by the splitter */
const URL_TOKEN =
  /(https?:\/\/[^\s，。；：？！“”‘’『』「」()[\]<>"']+|(?:[a-z0-9-]+\.)+(?:com|net|org|hk|cn|io|dev|app)(?:\/[^\s，。；：？！“”‘’『』「」()[\]<>"']*)?)/gi;

/**
 * Split evidence text into plain segments and URL tokens so the UI renders
 * long links as short clickable anchors instead of wrapped raw text — the
 * verbatim URL adds nothing to the reading surface.
 */
export function splitUrlTokens(text: string): TextToken[] {
  if (!text) return [];
  const tokens: TextToken[] = [];
  let last = 0;
  for (const m of text.matchAll(URL_TOKEN)) {
    const idx = m.index ?? 0;
    if (idx > last) tokens.push({ text: text.slice(last, idx) });
    // a URL never ends a sentence: drop trailing . , ; : ! ? kept only for paths like .jpg
    let url = m[0];
    const trimmed = url.replace(/[.,;:!?]+$/, '');
    if (trimmed) tokens.push({ url: trimmed });
    last = idx + url.length;
  }
  if (last < text.length) tokens.push({ text: text.slice(last) });
  return tokens.length ? tokens : [{ text }];
}

/** short display label for a URL token: hostname plus a clipped path */
export function shortUrlLabel(url: string): string {
  const full = /^https?:\/\//i.test(url) ? url : `https://${url}`;
  try {
    const u = new URL(full);
    const host = u.hostname.replace(/^www\./, '');
    const path = u.pathname === '/' ? '' : u.pathname;
    const label = host + path;
    return label.length > 30 ? `${label.slice(0, 30)}…` : label;
  } catch {
    return url.length > 30 ? `${url.slice(0, 30)}…` : url;
  }
}
