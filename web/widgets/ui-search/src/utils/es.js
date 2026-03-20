export const formatESResult = (res) => {
    const hits = {
        took: res?.took || 0,
        total: res?.hits?.total?.value || 0,
        hits: res?.hits?.hits ? res?.hits?.hits.map((item) => ({ ...item._source })) : []
    }
    const aggregations = []
    if (res?.aggregations) {
        const keys = Object.keys(res?.aggregations)
        keys.forEach((key) => {
            const buckets = res?.aggregations[key]?.buckets || []
            aggregations.push({
                key,
                list: buckets.map((b) => ({
                    count: b.doc_count,
                    key: b.key,
                    name: b.top?.hits?.hits?.[0]?._source?.source?.name || ''
                }))
            })
        })
    }
    return {
        hits,
        aggregations: aggregations.filter((item) => !!item.list && item.list.length > 0)  
    }
}