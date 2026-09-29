interface EsSearchResult {
  hits?: {
    hits?: Array<{
      _source?: Record<string, string | undefined>;
      [key: string]: unknown;
    }>;
    [key: string]: unknown;
  };
  [key: string]: unknown;
}

export function normalizeCoverIconUrl(data: EsSearchResult, baseUrl: string) {
  if (!data?.hits?.hits || !Array.isArray(data.hits.hits)) {
    return data;
  }

  let resolvedBaseUrl = baseUrl || '';
  if (resolvedBaseUrl && !resolvedBaseUrl.toLowerCase().startsWith('http')) {
    resolvedBaseUrl = `${window.location.origin}${window.location.pathname}/${resolvedBaseUrl}`;
  }

  const normalizeField = (value: string | undefined) => {
    if (typeof value !== 'string' || !value) return value;
    const text = value.toLowerCase();
    if (text.startsWith('/') || text.startsWith('#/')) {
      return `${resolvedBaseUrl}/${value}`.replace(/([^:]\/)\/+/g, '$1');
    }
    return value;
  };

  const normalizedHits = data.hits.hits.map((item: { _source?: Record<string, string | undefined>; [key: string]: unknown }) => {
    const source = item?._source;
    if (!source || typeof source !== 'object') {
      return item;
    }

    return {
      ...item,
      _source: {
        ...source,
        cover: normalizeField(source.cover),
        icon: normalizeField(source.icon),
        url: normalizeField(source.url),
        thumbnail: normalizeField(source.thumbnail),
      }
    };
  });

  return {
    ...data,
    hits: {
      ...data.hits,
      hits: normalizedHits
    }
  };
}

// Crawled documents routinely carry pre-escaped HTML entities in their
// indexed text (`&rsquo;`, `&quot;`, `&#39;` …). Rendering them raw leaks
// markup noise into titles and snippets, so decode the common set before
// display. Each replace pass scans its input once; replacements are not
// re-scanned, so `&amp;lt;` safely decodes to `&lt;` and stops there.
const NAMED_ENTITIES: Record<string, string> = {
  amp: "&",
  lt: "<",
  gt: ">",
  quot: '"',
  apos: "'",
  nbsp: " ",
  rsquo: "\u2019",
  lsquo: "\u2018",
  rdquo: "\u201D",
  ldquo: "\u201C",
  mdash: "\u2014",
  ndash: "\u2013",
  hellip: "\u2026",
  middot: "\u00B7",
  copy: "\u00A9",
  reg: "\u00AE",
  trade: "\u2122",
  deg: "\u00B0",
  euro: "\u20AC",
  pound: "\u00A3",
  eacute: "\u00E9",
  egrave: "\u00E8",
  agrave: "\u00E0",
  ccedil: "\u00E7",
  auml: "\u00E4",
  ouml: "\u00F6",
  uuml: "\u00FC",
  szlig: "\u00DF",
};

export function decodeHtmlEntities(input?: string): string | undefined {
  if (typeof input !== "string" || !input.includes("&")) return input;

  return input
    .replace(/&#(\d+);/g, (match, code: string) => {
      const num = Number(code);
      return num > 0 && num <= 0x10ffff ? String.fromCodePoint(num) : match;
    })
    .replace(/&#x([0-9a-fA-F]+);/g, (match, hex: string) => {
      const num = Number.parseInt(hex, 16);
      return num > 0 && num <= 0x10ffff ? String.fromCodePoint(num) : match;
    })
    .replace(/&([a-zA-Z][a-zA-Z0-9]*);/g, (match, name: string) => {
      return NAMED_ENTITIES[name.toLowerCase()] ?? match;
    });
}

/**
 * Tidy a raw document excerpt for use as a result snippet.
 *
 * Indexed content routinely starts with heading debris — markdown `#` tokens
 * or a `Page Title #` fragment — and is cut mid-word when clipped. Strip the
 * debris and land the cut on a sentence boundary when one is nearby.
 *
 * Only use on plain text: a string carrying literal `<em>` highlight markers
 * must not pass through here (the leading-fragment strip could break a
 * marker pair).
 */
export function cleanSummary(text?: string, maxLen = 240): string | undefined {
  if (typeof text !== "string" || !text) return text;

  let s = stripLeadingHeadingDebris(text);
  if (!s) return text.replace(/\s+/g, " ").trim();

  if (s.length <= maxLen) return s;

  // cut at the last sentence boundary inside the budget
  const bounded = s.slice(0, maxLen);
  const lastBreak = Math.max(
    bounded.lastIndexOf("。"),
    bounded.lastIndexOf("."),
    bounded.lastIndexOf("!"),
    bounded.lastIndexOf("?"),
    bounded.lastIndexOf(";"),
    bounded.lastIndexOf("；"),
    bounded.lastIndexOf("!")
  );
  if (lastBreak > maxLen * 0.4) {
    return bounded.slice(0, lastBreak + 1);
  }
  // no boundary nearby — cut at the last space instead of mid-word
  const lastSpace = bounded.lastIndexOf(" ");
  if (lastSpace > maxLen * 0.4) {
    return `${bounded.slice(0, lastSpace)}…`;
  }
  return `${bounded}…`;
}

/**
 * Strip leading heading debris from an ES highlight fragment (a string that
 * may carry literal `<em>…</em>` markers). Marker-aware: the prefix up to the
 * first `# ` is dropped whole (headings in crawled pages mix marked terms and
 * plain text); the term almost always recurs later in the fragment.
 */
export function cleanHighlightFragment(text?: string): string | undefined {
  if (typeof text !== "string" || !text) return text;

  let s = stripLeadingHeadingDebris(text);

  return s || text.trim();
}

// shared debris stripper: markdown heading tokens plus a chain of
// `Page Title #` anchor-heading segments (bounded so a mid-sentence `#`
// inside the first ~80 chars can eat at most one segment)
function stripLeadingHeadingDebris(text: string): string {
  let s = text.replace(/\s+/g, " ").trim();
  s = s.replace(/^(?:#+\s*)+/, "");
  for (let i = 0; i < 4; i++) {
    const next = s.replace(/^[^#]{1,80}?\s*#\s+/, "");
    if (next === s || next.length < 20) break;
    s = next;
  }
  return s;
}

export function generateRandomString(size: number) {
  const characters =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  let result = "";
  for (let i = 0; i < size; i++) {
    const randomIndex = Math.floor(Math.random() * characters.length);
    result += characters.charAt(randomIndex);
  }
  return result;
}

export function calculateCharLength(str: string) {
  if (!str) return 0;
  let totalLength = 0;
  for (let i = 0; i < str.length; i++) {
    const char = str.charAt(i);
    if (/[\u4e00-\u9fa5\u3000-\u303f\uff00-\uffef]/.test(char)) {
      totalLength += 2;
    } else {
      totalLength += 1;
    }
  }
  return totalLength;
}