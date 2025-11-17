import { useEffect, useState } from 'react';

import FullscreenPage from './FullscreenPage';
import FullscreenModal from './FullscreenModal';

import './ui-search/index.css';

const DARK_MODE_MEDIA_QUERY = '(prefers-color-scheme: dark)'

export default (props) => {
    const { shadow, id, server, enableQueryParams = true, parentTheme } = props;
    const [settings, setSettings] = useState()

    const { payload = {}, enabled_module = {} } = settings || {}

    const [theme, setTheme] = useState(window.matchMedia && window.matchMedia(DARK_MODE_MEDIA_QUERY).matches ? 'dark' : 'light')

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

    function search(query, callback, setLoading, shouldAgg) {
        if (setLoading) setLoading(true)
        const { filter = {} } = query
        const filterStr = Object.keys(filter).map((key) => `filter=${key}:any(${filter[key].join(',')})`).join('&')
        fetch(`${server}/query/_search?${filterStr ? filterStr + '&' : ''}query=${query.query}&from=${query.from}&size=${query.size}&v2=true`, {
            method: 'POST',
            headers: {
                'APP-INTEGRATION-ID': id,
            },
            credentials: 'include',
            body: shouldAgg ? JSON.stringify({
                "aggs": {
                    "category": { "terms": { "field": "category" } },
                    "lang": { "terms": { "field": "lang" } },
                    "source.id": {
                    "terms":  {
                        "field": "source.id"
                    },
                    "aggs": {
                        "top": {
                        "top_hits": {
                            "size": 1,
                            "_source": ["source.name"]
                        }
                        }
                    }
                    },
                    "type": { "terms": { "field": "type" } }
                }
            }) : undefined
        })
        .then(response => {
            if (!response.ok) throw new Error('response was not ok');
            return response.json();
        })
        .then(data => {
            callback(data)
        })
        .catch(error => {
            console.error('error:', error);
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
                    'APP-INTEGRATION-ID': id,
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

            const reader = response.body.getReader();
            const decoder = new TextDecoder('utf-8');
            let lineBuffer = '';

            while (true) {
            const { done, value } = await reader.read();

            if (done) {
                setLoading(false)
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
            console.error('error:', error);
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
        "onAsk": (assistanID, message, callback, setLoading) => {
            ask(assistanID, message, callback, setLoading)
        },
        "config": {
            "aggregations": {
                "source.id": {
                    "displayName": "source"
                },
                "lang": {
                    "displayName": "language"
                }
            }
        }
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
