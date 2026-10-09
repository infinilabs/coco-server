/**
 * Term highlight for the document preview (W17 terms-only mode): walk
 * the rendered content's text nodes, wrap matches of the query terms in
 * <mark> elements and scroll the first hit into view. DOM-based so it
 * works behind every renderer (text blob, markdown, HTML) without
 * touching them; the precise locator landing (chunk/page jump) rides on
 * W3's semantic_chunk payload and lands later.
 */

export interface HighlightResult {
  marks: number;
  firstElement?: HTMLElement;
}

export function highlightTerms(root: HTMLElement | null, rawQuery: string, maxMarks = 400): HighlightResult {
  if (!root || !rawQuery) return { marks: 0 };

  const terms = Array.from(
    new Set(
      rawQuery
        .split(/[\s,，]+/)
        .map(t => t.trim())
        .filter(t => t.length >= 2)
    )
  );
  if (terms.length === 0) return { marks: 0 };

  // longest first so overlapping shorter terms don't fragment the wrap
  terms.sort((a, b) => b.length - a.length);

  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
    acceptNode(node: Text) {
      const parent = node.parentElement;
      if (!parent) return NodeFilter.FILTER_REJECT;
      // never touch scripts, styles or existing marks
      const tag = parent.tagName;
      if (tag === 'SCRIPT' || tag === 'STYLE' || tag === 'MARK') return NodeFilter.FILTER_REJECT;
      if (!node.nodeValue || !node.nodeValue.trim()) return NodeFilter.FILTER_REJECT;
      return NodeFilter.FILTER_ACCEPT;
    }
  });

  const targets: Text[] = [];
  let current = walker.nextNode();
  while (current) {
    targets.push(current as Text);
    current = walker.nextNode();
  }

  let marks = 0;
  let firstElement: HTMLElement | undefined;
  for (const textNode of targets) {
    if (marks >= maxMarks) break;
    const value = textNode.nodeValue ?? '';
    const lower = value.toLowerCase();
    // find the earliest match across all terms in this node
    let bestIdx = -1;
    let bestTerm = '';
    for (const term of terms) {
      const idx = lower.indexOf(term.toLowerCase());
      if (idx >= 0 && (bestIdx < 0 || idx < bestIdx)) {
        bestIdx = idx;
        bestTerm = term;
      }
    }
    if (bestIdx < 0) continue;

    const mark = document.createElement('mark');
    mark.style.backgroundColor = 'rgba(250, 219, 20, 0.45)';
    mark.style.color = 'inherit';
    mark.style.borderRadius = '2px';
    mark.style.padding = '0 1px';
    const after = textNode.splitText(bestIdx);
    after.splitText(bestTerm.length);
    mark.appendChild(document.createTextNode(after.nodeValue ?? ''));
    after.parentNode?.replaceChild(mark, after);
    marks++;
    if (!firstElement) firstElement = mark;
  }

  return { marks, firstElement };
}
