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
const SOURCE_LINK = /\[源\]\(#\/preview\/document\/([A-Za-z0-9_-]+)(?:\?q=([^)\s]*))?[^)]*\)/;

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
      const src = raw.match(SOURCE_LINK);
      const excerptDecoded = src?.[2] ? safeDecodeURIComponent(src[2]) : undefined;
      claims.push({
        n: Number(m[1]),
        claim: m[2].replace(/\s*\[源\]\([^)]*\)\s*$/, '').trim(),
        docId: src?.[1],
        excerpt: excerptDecoded
      });
    }
  }
  return { body: body.trim(), claims };
}

function safeDecodeURIComponent(s: string): string {
  try {
    return decodeURIComponent(s);
  } catch {
    return s;
  }
}

/**
 * Turn bare citation markers `[12]` in compiled prose into markdown links
 * (`[12](#cite-12)`) so the reader renders them as interactive chips —
 * wikilinks `[[..]]`, existing markdown links `[x](..)` and code fences
 * are left alone.
 */
export function linkifyCitations(body: string): string {
  if (!body) return body;
  // transform only outside fenced code blocks (even-indexed segments)
  return body
    .split('```')
    .map((seg, i) =>
      i % 2 === 0
        ? seg.replace(/(^|[^\[`\\])\[(\d{1,3})\](?!\()/g, (_m, pre: string, n: string) => `${pre}[${n}](#cite-${n})`)
        : seg
    )
    .join('```');
}
