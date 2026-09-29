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
  definition: string;
  keyCharacteristics: string[];
  applications: string[];
  relatedConcepts: string[];
  relatedEntities: string[];
  mentions: string[];
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
    definition: '',
    keyCharacteristics: [],
    applications: [],
    relatedConcepts: [],
    relatedEntities: [],
    mentions: [],
    mainContent: content
  };

  const lines = content.split('\n');
  let currentSection = '';
  let mainContentStart = 0;

  for (let i = 0; i < lines.length; i += 1) {
    const line = lines[i].trim();
    const matched = line.startsWith('## ') ? matchHeader(line) : null;

    if (matched) {
      currentSection = matched;
      if (matched === 'mainContent') {
        mainContentStart = i + 2; // skip the header and the blank line
      } else if (mainContentStart === 0) {
        mainContentStart = i;
      }
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
      default:
        break;
    }
  }

  if (mainContentStart > 0 && mainContentStart < lines.length) {
    result.mainContent = lines.slice(mainContentStart).join('\n');
  }
  return result;
}

/** parse `[[type:label]]` wiki links out of a related-list entry */
export function parseWikiLink(entry: string): WikiLink | null {
  const m = entry.match(/^\[\[([^:]+):(.+)]]$/);
  if (!m) return null;
  return { type: m[1].trim(), label: m[2].trim() };
}
