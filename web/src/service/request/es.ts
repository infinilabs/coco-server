export function formatESSearchResult(esResp: any) {
  if (!esResp || !esResp.hits) {
    return {
      data: [],
      took: 0,
      total: 0
    };
  }
  const took = esResp.took;
  let total = esResp.hits.total;
  if (total && typeof total === 'object') {
    total = total.value;
  }
  if (total == null || total == 0) {
    return {
      data: [],
      took,
      total
    };
  }
  const dataArr = [];
  if (esResp.hits.hits) {
    for (const hit of esResp.hits.hits) {
      if (!hit._source.id) {
        hit._source.id = hit._id;
      }
      hit._source._index = hit._index;
      if (hit._type) {
        hit._source._type = hit._type;
      }
      if (hit.highlight) {
        hit._source.highlight = hit.highlight;
      }
      dataArr.push(hit._source);
    }
  }
  return {
    aggregations: esResp.aggregations,
    data: dataArr,
    took,
    total
  };
}

export function formatSearchFilter(filter: any, reverse = false) {
  if (!filter) return ''
  const keys = Object.keys(filter);
  if (keys.length === 0) return ''
  return Object.keys(filter).map((key) => `filter=${reverse ? '!' : ''}${key}:any(${filter[key].join(',')})`).join('&')
}
