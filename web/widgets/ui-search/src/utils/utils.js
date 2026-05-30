export function normalizeCoverIconUrl(data, baseUrl) {
  if (!data?.hits?.hits || !Array.isArray(data.hits.hits)) {
    return data;
  }

  let resolvedBaseUrl = baseUrl || '';
  if (resolvedBaseUrl && !resolvedBaseUrl.toLowerCase().startsWith('http')) {
    resolvedBaseUrl = `${window.location.origin}${window.location.pathname}/${resolvedBaseUrl}`;
  }

  const normalizeField = (value) => {
    if (typeof value !== 'string' || !value) return value;
    const text = value.toLowerCase();
    if (text.startsWith('/') || text.startsWith('#/')) {
      return `${resolvedBaseUrl}/${value}`.replace(/([^:]\/)\/+/g, '$1');
    }
    return value;
  };

  const normalizedHits = data.hits.hits.map((item) => {
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

export function generateRandomString(size) {
  const characters =
    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789";
  let result = "";
  for (let i = 0; i < size; i++) {
    const randomIndex = Math.floor(Math.random() * characters.length);
    result += characters.charAt(randomIndex);
  }
  return result;
}

export function calculateCharLength(str) {
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