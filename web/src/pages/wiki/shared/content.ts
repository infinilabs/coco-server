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
      currentSection = 'other';
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
