export function formatESSearchResult(esResp: any) {
  if (!esResp || !esResp.hits) {
    return {
      took: 0,
      total: 0,
      data: [],
    };
  }
  const took = esResp.took;
  const total = esResp.hits.total;
  if (total == null || total.value == 0) {
    return {
      took: took,
      total: total,
      data: [],
    };
  }
  let dataArr = [];
  if (esResp.hits.hits) {
    for (let hit of esResp.hits.hits) {
      if (!hit._source.id) {
        hit._source["id"] = hit._id;
      }
      hit._source["_index"] = hit._index;
      if (hit["_type"]) {
        hit._source["_type"] = hit["_type"];
      }
      if (hit["highlight"]) {
        hit._source["highlight"] = hit["highlight"];
      }
      dataArr.push(hit._source);
    }
  }
  return {
    took: took,
    total: total,
    data: dataArr,
    aggregations: esResp.aggregations,
  };
}