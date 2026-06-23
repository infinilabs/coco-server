import { useEffect, useState } from 'react';
import queryString from 'query-string';

import FullscreenPage from './FullscreenPage';
import FullscreenModal from './FullscreenModal';

import 'ui-search/css';

type AnyRecord = Record<string, any>;
type DataCallback = (data?: any) => void;
type LoadingSetter = (loading: boolean) => void;

type FullscreenProps = AnyRecord & {
    shadow?: ShadowRoot | HTMLElement;
    id?: string;
    server?: string;
    enableQueryParams?: boolean;
    parentTheme?: string;
}

const DARK_MODE_MEDIA_QUERY = '(prefers-color-scheme: dark)'

const AGGS_DEFAULT = {
    aggs: {
        category: { terms: { field: 'category' } },
        'source.id': {
            terms: {
                field: 'source.id',
            },
            aggs: {
                top: {
                    top_hits: {
                        size: 1,
                        _source: ['source.name'],
                    },
                },
            },
        },
        type: { terms: { field: 'type' } },
        tags: { terms: { field: 'tags' } },
    },
}

const AGGS_IMAGE = {
    aggs: {
        category: { terms: { field: 'category' } },
        'source.id': {
            terms: {
                field: 'source.id',
            },
            aggs: {
                top: {
                    top_hits: {
                        size: 1,
                        _source: ['source.name'],
                    },
                },
            },
        },
        type: { terms: { field: 'type' } },
        tag: { terms: { field: 'tags' } },
        color: { terms: { field: 'metadata.colors' } },
    },
}

const AGGS: Record<string, any> = {
    all: AGGS_DEFAULT,
    image: AGGS_IMAGE,
}

function buildFilterString(filter: AnyRecord = {}) {
    return Object.keys(filter)
        .filter((key) => filter[key] !== undefined && filter[key] !== null && filter[key] !== '')
        .map((key) => {
            const filterValue = Array.isArray(filter[key]) ? filter[key].join(',') : filter[key]
            return `filter=${key}:any(${filterValue})`
        })
        .join('&')
}


export default function Fullscreen(props: FullscreenProps) {
    const { shadow, id, server, enableQueryParams = true, parentTheme } = props;
    const [settings, setSettings] = useState<AnyRecord>()

    const { payload = {}, enabled_module = {} } = settings || {}

    const [theme, setTheme] = useState(window.matchMedia && window.matchMedia(DARK_MODE_MEDIA_QUERY).matches ? 'dark' : 'light')

    const apiHeaders: Record<string, string> = {
        'APP-INTEGRATION-ID': id ?? '',
    }

    async function fetchSettings(server?: string, id?: string) {
        if (!server || !id) return;
        try {
            const response = await fetch(`${server}/integration/${id}`, {
                headers: {
                    'APP-INTEGRATION-ID': id,
                    'Content-Type': 'application/json',
                },
                method: 'GET',
                credentials: 'include',
            });
            const result = await response.json();
            if (result?._source) {
                const integrationData = { ...result._source };
                const { deep_research_assistant, deep_think_assistant } = integrationData;
                if (deep_research_assistant || deep_think_assistant) {
                    const ids = [deep_research_assistant, deep_think_assistant].filter((id) => !!id);
                    const filterStr = `filter=id:any(${ids.join(',')})`;
                    const assistantRes = await fetch(`${server}/assistant/_search?from=0&size=10000&${filterStr}`, {
                        headers: {
                            'APP-INTEGRATION-ID': id,
                            'Content-Type': 'application/json',
                        },
                        method: 'GET',
                        credentials: 'include',
                    });
                    const assistantData = await assistantRes.json();
                    if (assistantData?.hits?.hits?.length) {
                        assistantData.hits.hits.forEach((item: AnyRecord) => {
                            if (item._id === deep_research_assistant) {
                                integrationData.deep_research_assistant_entity = item._source;
                            }
                            if (item._id === deep_think_assistant) {
                                integrationData.deep_think_assistant_entity = item._source;
                            }
                        });
                    }
                }
                setSettings(integrationData);
            }
        } catch (error) {
            console.log('error', error);
        }
    }

    function search(query: AnyRecord, callback: DataCallback, setLoading?: LoadingSetter) {
        if (setLoading) setLoading(true)
        const { filter = {}, start, end, ...rest } = query
        const filterStr = buildFilterString(filter)
        const dateFilterStr = [
            start ? `filter=updated>=${start}` : '',
            end ? `filter=updated<=${end}` : '',
        ].filter(Boolean).join('&')
        const searchStr = [filterStr, dateFilterStr, queryString.stringify(rest)].filter(Boolean).join('&')
        fetch(`${server}/query/_search?${searchStr}`, {
            method: 'POST',
            headers: apiHeaders,
            credentials: 'include',
            body: JSON.stringify({
                "aggs": {
                    "counts": {
                        "auto_date_histogram": {
                            "field": "updated",
                            "buckets": 120,
                            "time_zone": "Asia/Shanghai"
                        }
                    }
                }
            })
        })
            .then(response => {
                if (!response.ok) throw new Error('response was not ok');
                return response.json();
            })
            .then(data => {
                callback(data)
            })
            .catch(error => {
                callback({ error })
            }).finally(() => {
                if (setLoading) setLoading(false)
            })
    }

    function aggregate(query: AnyRecord, callback: DataCallback, setLoading?: LoadingSetter) {
        if (setLoading) setLoading(true)
        const { query: keyword, filter = {}, search_type, fuzziness, start, end, ...rest } = query
        const filterStr = buildFilterString(filter)
        const dateFilterStr = [
            start ? `filter=updated>=${start}` : '',
            end ? `filter=updated<=${end}` : '',
        ].filter(Boolean).join('&')
        const searchStr = [filterStr, dateFilterStr, queryString.stringify({ query: keyword, search_type, fuzziness })].filter(Boolean).join('&')
        fetch(`${server}/query/_search?${searchStr}`, {
            method: 'POST',
            headers: {
                ...apiHeaders,
                'Content-Type': 'application/json',
            },
            credentials: 'include',
            body: JSON.stringify(AGGS[query['metadata.content_category']] || AGGS.all),
        })
            .then(response => {
                if (!response.ok) throw new Error('response was not ok');
                return response.json();
            })
            .then(data => {
                callback(data)
            })
            .catch(error => {
                callback({ error })
            }).finally(() => {
                if (setLoading) setLoading(false)
            })
    }

    async function ask(assistantID: string, message: any, callback: DataCallback, setLoading: LoadingSetter) {
        setLoading(true)
        try {
            const response = await fetch(`${server}/assistant/${assistantID}/_ask`, {
                headers: {
                    ...apiHeaders,
                    'content-type': 'text/plain',
                },
                method: 'POST',
                credentials: 'include',
                body: JSON.stringify({
                    message: JSON.stringify(message),
                })
            });

            if (!response.ok) {
                setLoading(false)
                throw new Error(`HTTP error! Status: ${response.status}`);
            }

            // Keep the loading indicator on until the first non-heartbeat
            // chunk arrives (e.g. the assistant's real response). The backend
            // may stream `attachment_waiting` / system chunks for some time
            // before producing actual content; clearing loading on
            // `response.ok` would prematurely end the thinking state.
            let loadingCleared = false
            const clearLoadingOnce = () => {
                if (!loadingCleared) {
                    loadingCleared = true
                    setLoading(false)
                }
            }

            if (!response.body) {
                throw new Error('response body is null')
            }

            const reader = response.body.getReader();
            const decoder = new TextDecoder('utf-8');
            let lineBuffer = '';

            while (true) {
                const { done, value } = await reader.read();

                if (done) {
                    clearLoadingOnce()
                    break;
                }

                const chunk = decoder.decode(value, { stream: true });

                lineBuffer += chunk;

                const lines = lineBuffer.split('\n');
                for (let i = 0; i < lines.length - 1; i++) {
                    try {
                        const json = JSON.parse(lines[i]);
                        if (json && !(json._id && json._source && json.result)) {
                            // The first "response" chunk means content is flowing;
                            // we can safely hand off the visual state to the
                            // typing indicator owned by the chunk consumer.
                            if (json?.chunk_type === 'response') {
                                clearLoadingOnce()
                            }
                            callback(json)
                        }
                    } catch (error) {
                        console.log("error:", lines[i])
                    }
                }

                lineBuffer = lines[lines.length - 1];
            }
        } catch (error) {
            setLoading(false)
        }
    }

    async function suggest(tag: string | undefined, params: AnyRecord | undefined, callback?: DataCallback) {
        try {
            const search = queryString.stringify(params || {})
            const response = await fetch(`${server}/query/_suggest${tag ? `/${tag}` : ''}${search ? `?${search}` : ''}`, {
                method: 'GET',
                headers: apiHeaders,
                credentials: 'include',
            })

            if (!response.ok) {
                throw new Error(`HTTP error! Status: ${response.status}`)
            }

            const data = await response.json()
            callback?.(data)
        } catch (error) {
            callback?.()
        }
    }

    async function recommend(tag: string | undefined, callback?: DataCallback) {
        try {
            const response = await fetch(`${server}/query/_recommend${tag ? `/${tag}` : ''}`, {
                method: 'GET',
                headers: apiHeaders,
                credentials: 'include',
            })

            if (!response.ok) {
                throw new Error(`HTTP error! Status: ${response.status}`)
            }

            const data = await response.json()
            callback?.(data)
        } catch (error) {
            callback?.()
        }
    }

    async function fetchProfile(callback?: DataCallback) {
        try {
            const response = await fetch(`${server}/account/profile`, {
                method: 'GET',
                headers: apiHeaders,
                credentials: 'include',
            })

            if (!response.ok) {
                throw new Error(`HTTP error! Status: ${response.status}`)
            }

            const data = await response.json()
            callback?.(data)
        } catch (error) {
            callback?.()
        }
    }

    async function onLogout(callback?: DataCallback) {
        try {
            const response = await fetch(`${server}/account/logout`, {
                method: 'POST',
                headers: apiHeaders,
                credentials: 'include',
            })

            if (!response.ok) {
                throw new Error(`HTTP error! Status: ${response.status}`)
            }

            const data = await response.json()
            callback?.(data)
        } catch (error) {
            callback?.()
        }
    }

    async function onUpload(files: File[], callback?: DataCallback) {
        try {
            const formData = new FormData();
            for (const f of files) {
                formData.append('files', f, f.name);
            }
            const response = await fetch(`${server}/attachment/_upload`, {
                method: 'POST',
                headers: apiHeaders,
                body: formData,
                credentials: 'include',
            })

            if (!response.ok) {
                throw new Error(`HTTP error! Status: ${response.status}`)
            }

            const data = await response.json()
            callback?.(data)
        } catch (error) {
            callback?.()
        }
    }

    async function getUserEntities(ids: string[], callback?: DataCallback) {
        try {
            const response = await fetch(`${server}/entity/label/_batch_get`, {
                method: 'POST',
                headers: {
                    ...apiHeaders,
                    'Content-Type': 'application/json',
                },
                credentials: 'include',
                body: JSON.stringify([{
                    type: 'user',
                    id: ids,
                }]),
            })

            if (!response.ok) {
                throw new Error(`HTTP error! Status: ${response.status}`)
            }

            const data = await response.json()
            callback?.(data)
        } catch (error) {
            callback?.({})
        }
    }

    async function getFieldsMeta(fields: string[], callback?: DataCallback) {
        if (!Array.isArray(fields) || fields.length === 0) {
            callback?.()
            return
        }

        try {
            const response = await fetch(`${server}/field_meta/${fields.join(',')}`, {
                method: 'GET',
                headers: apiHeaders,
                credentials: 'include',
            })

            if (!response.ok) {
                throw new Error(`HTTP error! Status: ${response.status}`)
            }

            const data = await response.json()
            callback?.(data)
        } catch (error) {
            callback?.()
        }
    }

    useEffect(() => {
        fetchSettings(server, id);
    }, [server, id]);

    function onSystemThemeChange(e: MediaQueryListEvent) {
        setTheme(e.matches ? 'dark' : 'light')
    }

    useEffect(() => {
        const currentTheme = parentTheme || settings?.appearance?.theme
        if (currentTheme === 'auto') {
            setTheme(window.matchMedia && window.matchMedia(DARK_MODE_MEDIA_QUERY).matches ? 'dark' : 'light')
            window.matchMedia(DARK_MODE_MEDIA_QUERY).addEventListener('change', onSystemThemeChange);
        } else {
            setTheme(currentTheme)
        }
        return () => {
            if (currentTheme === 'auto') {
                window.matchMedia(DARK_MODE_MEDIA_QUERY).removeEventListener('change', onSystemThemeChange)
            }
        }
    }, [settings?.appearance?.theme, parentTheme])

    const componentProps = {
        ...props,
        settings,
        id,
        shadow,
        theme,
        language: settings?.appearance?.language || 'zh-CN',
        "logo": {
            "light": payload?.logo?.light,
            "light_mobile": payload?.logo?.light_mobile,
            "dark": payload?.logo?.dark,
            "dark_mobile": payload?.logo?.dark_mobile,
        },
        "placeholder": enabled_module?.search?.placeholder,
        "welcome": payload?.welcome || "",
        "aiOverview": {
            ...(payload?.ai_overview || {}),
            "showActions": true,
        },
        "onSearch": (query: AnyRecord, callback: DataCallback, setLoading?: LoadingSetter) => {
            search(query, callback, setLoading)
        },
        "onAggregation": (query: AnyRecord, callback: DataCallback, setLoading?: LoadingSetter) => {
            aggregate(query, callback, setLoading)
        },
        "onAsk": (assistanID: string, message: any, callback: DataCallback, setLoading: LoadingSetter) => {
            ask(assistanID, message, callback, setLoading)
        },
        "onSuggestion": (tag: string | undefined, params: AnyRecord | undefined, callback?: DataCallback) => {
            suggest(tag, params, callback)
        },
        "onRecommend": (tag: string | undefined, callback?: DataCallback) => {
            recommend(tag, callback)
        },
        "config": {
            "aggregations": {
                "source.id": {
                    "label": "source",
                    "payload": { field_name: 'source.id', field_data_type: 'keyword', support_multi_select: true }
                },
                "lang": {
                    "label": "language",
                    "payload": { field_name: 'lang', field_data_type: 'keyword', support_multi_select: true }
                },
                "color": {
                    "label": "color",
                    "type": "color",
                    "payload": { field_name: 'color', field_data_type: 'keyword', support_multi_select: true }
                },
                "tags": {
                    "label": "tag",
                    "type": "tag",
                    "payload": { field_name: 'tags', field_data_type: 'keyword', support_multi_select: true }
                },
                "category": {
                    "label": "category",
                    "payload": { field_name: 'category', field_data_type: 'keyword', support_multi_select: true }
                },
                "type": {
                    "label": "type",
                    "payload": { field_name: 'type', field_data_type: 'keyword', support_multi_select: true }
                }
            }
        },
        "apiConfig": {
            "BaseUrl": server,
            "endpoint": server,
            "headers": apiHeaders,
        },
        "getFieldsMeta": getFieldsMeta,
        "onLogoClick": () => {
            const currentUrl = new URL(window.location.href)
            currentUrl.search = ''
            history.replaceState(null, '', currentUrl.toString())
        },
        "getProfile": fetchProfile,
        "onLogout": onLogout,
        showTopAction: true,
        onUpload,
        getUserEntities
    }

    if (settings?.type === 'fullscreen' || settings?.type === 'page') {
        return (
            <FullscreenPage {...componentProps} enableQueryParams={enableQueryParams} />
        )
    } else if (settings?.type === 'modal') {
        return <FullscreenModal {...componentProps} />
    } else {
        return null
    }
}
