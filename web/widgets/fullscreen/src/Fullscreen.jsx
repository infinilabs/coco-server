import { useEffect, useState } from 'react';
import UISearch from './ui-search';
import './ui-search/index.css';

export default (props) => {
    const { shadow, id, token, server } = props;
    const [settings, setSettings] = useState()

    const { payload = {}, enabled_module = {} } = settings || {}

    async function fetchSettings(server, id, token) {
        if (!server || !id || !token) return;
        fetch(`${server}/integration/${id}`, {
            headers: {
                'APP-INTEGRATION-ID': id,
                'X-API-TOKEN': token,
                'Content-Type': 'application/json',
            },
            method: 'GET'
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
        const { filters = {} } = query
        const filterStr = Object.keys(filters).map((key) => `filter=${key}:any(${filters[key].join(',')})`).join('&')
        fetch(`${server}/query/_search?${filterStr ? filterStr + '&' : ''}query=${query.keyword}&from=${query.from}&size=${query.size}&v2=true`, {
            method: 'POST',
            headers: {
                'APP-INTEGRATION-ID': id,
                'X-API-TOKEN': token,
            },
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
                    'X-API-TOKEN': token,
                },
                method: 'POST',
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
        fetchSettings(server, id, token);
    }, [server, id, token]);

    if (settings?.type !== 'fullscreen') return null;

    return (
        <UISearch {...{
            id,
            shadow,
            "logo": {
                "light": payload?.logo?.light,
                "light-mobile": payload?.logo?.light_mobile,
            },
            "placeholder": enabled_module?.search?.placeholder,
            "welcome": payload?.welcome || "Nice to meet you. I can help answer your questions by tapping into the internet and your data sources. How can I assist you today?",
            "aiOverview": {
                "enabled": payload?.ai_overview?.enabled,
                "assistantID": 'ai_overview' || payload?.ai_overview?.assistant,
                "title": payload?.ai_overview?.title,
                "height": payload?.ai_overview?.height || "auto",
                "logo": payload?.ai_overview?.logo,
                "showActions": true,
            },
            "widgets": payload.ai_widgets?.enabled && payload.ai_widgets?.widgets ? payload.ai_widgets?.widgets.map((item) => ({
                "assistantID": item.assistant,
                "title": item.title,
                "height": item.height || "auto",
                "logo": item.logo,
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
        }} />
    )
}