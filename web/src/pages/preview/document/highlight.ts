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

// W17 tier 2 (quote landing): highlight the cited chunk's exact excerpt —
// lands the reader on the precise passage rather than scattering term marks.
// Matching is whitespace-normalized (renderers insert breaks the source
// text doesn't have). Returns the first highlighted <mark>, if any.
export function highlightPhrase(root: HTMLElement | null, phrase: string): HTMLElement | undefined {
  if (!root || !phrase || phrase.trim().length < 8) return undefined;

  const norm = (s: string) => s.replace(/\s+/g, ' ').trim().toLowerCase();
  const target = norm(phrase);

  // walk text nodes building a running normalized string; remember each
  // node's [startOffset, endOffset) within the normalized stream
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
    acceptNode(node: Text) {
      const parent = node.parentElement;
      if (!parent) return NodeFilter.FILTER_REJECT;
      const tag = parent.tagName;
      if (tag === 'SCRIPT' || tag === 'STYLE' || tag === 'MARK') return NodeFilter.FILTER_REJECT;
      return NodeFilter.FILTER_ACCEPT;
    }
  });

  const nodes: Array<{ node: Text; start: number; end: number; text: string }> = [];
  let stream = '';
  let current = walker.nextNode();
  while (current) {
    const textNode = current as Text;
    const raw = textNode.nodeValue ?? '';
    // normalize this node's text: collapse runs, keep single spaces so
    // word boundaries across nodes still align
    let normed = '';
    for (const ch of raw) {
      normed += /\s/.test(ch) ? ' ' : ch.toLowerCase();
    }
    nodes.push({ node: textNode, start: stream.length, end: stream.length + normed.length, text: normed });
    stream += normed;
    current = walker.nextNode();
  }

  const found = stream.indexOf(target);
  if (found < 0) return undefined;

  // wrap the matched range: split each overlapping text node and replace
  // the matched slice with a <mark>
  let firstMark: HTMLElement | undefined;
  for (const entry of nodes) {
    const startInNode = Math.max(found, entry.start);
    const endInNode = Math.min(found + target.length, entry.end);
    if (startInNode >= endInNode) continue;

    const node = entry.node;
    const localStart = startInNode - entry.start;
    const localEnd = endInNode - entry.start;
    const raw = node.nodeValue ?? '';
    // find raw offsets for the normalized slice (walk chars, skipping spaces
    // that normalization collapsed)
    let rawStart = -1, rawEnd = raw.length, normPos = entry.start;
    for (let i = 0; i < raw.length; i++) {
      const chNorm = /\s/.test(raw[i]) ? ' ' : raw[i].toLowerCase();
      if (normPos >= localStart && rawStart < 0) rawStart = i;
      normPos += chNorm.length;
      if (normPos >= localEnd) { rawEnd = i + 1; break; }
    }
    if (rawStart < 0) continue;

    const mark = document.createElement('mark');
    mark.style.backgroundColor = 'rgba(82, 196, 26, 0.35)'; // green: precise passage
    mark.style.color = 'inherit';
    mark.style.borderRadius = '2px';
    mark.style.padding = '0 1px';
    const after = node.splitText(rawStart);
    after.splitText(rawEnd - rawStart);
    mark.appendChild(document.createTextNode(after.nodeValue ?? ''));
    after.parentNode?.replaceChild(mark, after);
    if (!firstMark) firstMark = mark;
  }
  return firstMark;
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
