import { useEffect, useState } from 'react';
import queryString from 'query-string';

import FullscreenPage from './FullscreenPage';
import FullscreenModal from './FullscreenModal';

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

const AGGS = {
    all: AGGS_DEFAULT,
    image: AGGS_IMAGE,
}

function buildFilterString(filter = {}) {
    return Object.keys(filter)
        .filter((key) => filter[key] !== undefined && filter[key] !== null && filter[key] !== '')
        .map((key) => {
            const filterValue = Array.isArray(filter[key]) ? filter[key].join(',') : filter[key]
            return `filter=${key}:any(${filterValue})`
        })
        .join('&')
}


export default (props) => {
    const { shadow, id, server, enableQueryParams = true, parentTheme } = props;
    const [settings, setSettings] = useState()

    const { payload = {}, enabled_module = {} } = settings || {}

    const [theme, setTheme] = useState(window.matchMedia && window.matchMedia(DARK_MODE_MEDIA_QUERY).matches ? 'dark' : 'light')

    const apiHeaders = {
        'APP-INTEGRATION-ID': id,
    }

    async function fetchSettings(server, id) {
        if (!server || !id) return;
        fetch(`${server}/integration/${id}`, {
            headers: {
                'APP-INTEGRATION-ID': id,
                'Content-Type': 'application/json',
            },
            method: 'GET',
            credentials: 'include',
        })
        .then(response => response.json())
        .then(result => {
            if (result?._source) {
                setSettings(result?._source);
            }
        })
        .catch(error => console.log('error', error));
    }

    function search(query, callback, setLoading) {
        if (setLoading) setLoading(true)
        const { filter = {}, ...rest } = query
        const filterStr = buildFilterString(filter)
        const searchStr = `${filterStr ? filterStr + '&' : ''}${queryString.stringify(rest)}`
        fetch(`${server}/query/_search?${searchStr}`, {
            method: 'POST',
            headers: apiHeaders,
            credentials: 'include',
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

    function aggregate(query, callback, setLoading) {
        if (setLoading) setLoading(true)
        const { query: keyword, filter = {}, ...rest } = query
        const filterStr = buildFilterString(filter)
        const searchStr = `${filterStr ? filterStr + '&' : ''}${queryString.stringify({ query: keyword, ...rest })}`
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

    async function ask(assistantID, message, callback, setLoading) {
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
                throw new Error(`HTTP error! Status: ${response.status}`);
            }
            
            setLoading(false)

            const reader = response.body.getReader();
            const decoder = new TextDecoder('utf-8');
            let lineBuffer = '';

            while (true) {
            const { done, value } = await reader.read();

            if (done) {
                break;
            }

            const chunk = decoder.decode(value, { stream: true });

            lineBuffer += chunk;

            const lines = lineBuffer.split('\n');
            for (let i = 0; i < lines.length - 1; i++) {
                try {
                    const json = JSON.parse(lines[i]);
                    if (json && !(json._id && json._source && json.result)) {
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

    async function suggest(tag, params, callback) {
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

    async function recommend(tag, callback) {
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

    async function fetchProfile(callback) {
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

    async function onLogout(callback) {
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

    async function getFieldsMeta(fields, callback) {
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

    function onSystemThemeChange(e) {
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
        "widgets": payload.ai_widgets?.enabled && payload.ai_widgets?.widgets ? payload.ai_widgets?.widgets.map((item) => ({
            ...item,
            "showActions": false,
        })) : [],
        "onSearch": (query, callback, setLoading, shouldAgg = true) => {
            search(query, callback, setLoading, shouldAgg)
        },
        "onAggregation": (query, callback, setLoading) => {
            aggregate(query, callback, setLoading)
        },
        "onAsk": (assistanID, message, callback, setLoading) => {
            ask(assistanID, message, callback, setLoading)
        },
        "onSuggestion": (tag, params, callback) => {
            suggest(tag, params, callback)
        },
        "onRecommend": (tag, callback) => {
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
        "getRawContent": (item) => {
            if (item?.id && item?.title) {
                return `${server.replace(/\/$/, '')}/document/${item.id}/raw_content/${item.title}`
            }
            return ''
        },
        "apiConfig": {
            "BaseUrl": server,
            "endpoint": server,
            "headers": apiHeaders,
        },
        "getFieldsMeta": (fields, callback) => {
            getFieldsMeta(fields, callback)
        },
        "onLogoClick": () => {
            const currentUrl = new URL(window.location.href)
            currentUrl.search = ''
            history.replaceState(null, '', currentUrl.toString())
        },
        "getProfile": (callback) => {
            fetchProfile(callback)
        },
        "onLogout": (callback) => {
            onLogout(callback)
        },
        showTopAction: true
    }
    
    if (settings?.type === 'fullscreen' || settings?.type === 'page') {
        return (
            <FullscreenPage {...componentProps} enableQueryParams={enableQueryParams}/>
        )
    } else if (settings?.type === 'modal') {
        return <FullscreenModal {...componentProps} />
    } else {
        return null
    }
}
