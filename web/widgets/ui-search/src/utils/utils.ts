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