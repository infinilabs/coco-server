interface AggregationItem {
    key: string;
    list: Array<{ count: number; key: string; name: string }>;
}

export const formatESResult = (res: Record<string, unknown> = {}) => {
    const hits = {
        took: (res?.took as number) || 0,
        total: ((res?.hits as Record<string, unknown>)?.total as Record<string, unknown>)?.value || 0,
        hits: (res?.hits as Record<string, unknown>)?.hits ? ((res?.hits as Record<string, unknown>)?.hits as Array<Record<string, unknown>>).map((item) => ({ ...item._source as object })) : []
    }
    const aggregations: AggregationItem[] = []
    if (res?.aggregations) {
        const aggs = res.aggregations as Record<string, Record<string, unknown>>;
        const keys = Object.keys(aggs)
        keys.forEach((key) => {
            const buckets = ((aggs[key])?.buckets || []) as Array<Record<string, unknown>>
            aggregations.push({
                key,
                list: buckets.map((b) => ({
                    count: b.doc_count as number,
                    key: b.key as string,
                    name: ((((b.top as Record<string, unknown>)?.hits as Record<string, unknown>)?.hits as Array<Record<string, unknown>> | undefined)?.[0]?._source as Record<string, Record<string, string>> | undefined)?.source?.name || ''
                }))
            })
        })
    }
    return {
        hits,
        aggregations: aggregations.filter((item) => !!item.list && item.list.length > 0)  
    }
}