import { useEffect, useMemo, useState } from 'react';
import FullscreenPage from './FullscreenPage';
import FullscreenModal from './FullscreenModal';
import './ui-search/index.css';
import useQueryParams from './hooks/queryParams'

export default (props) => {
    const { shadow, id, token, server } = props;
    const [settings, setSettings] = useState()

    const { payload = {}, enabled_module = {} } = settings || {}
    const [queryParams, setQueryParams] = useQueryParams();
    console.log(queryParams)

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

    const handleQueryParams = (query) => {
        const urlParams = new URLSearchParams(window.location.search);
        urlParams.set('query', query.keyword);
        urlParams.set('from', query.from);
        urlParams.set('size', query.size);
        urlParams.delete('filter'); 
        const { filters = {} } = query
        Object.keys(filters).map((key) => {
            urlParams.append('filter', `${key}:any(${filters[key].join(',')}`);
        })
        const newQuery = urlParams.toString();
        const newUrl = `${window.location.origin}/${window.location.hash}${newQuery ? `?${newQuery}` : ''}`;
        history.pushState({}, '', newUrl);
    }

    useEffect(() => {
        fetchSettings(server, id, token);
    }, [server, id, token]);

    const componentProps = {
        ...props,
        id,
        shadow,
        "logo": {
            "light": payload?.logo?.light,
            "light-mobile": payload?.logo?.light_mobile,
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
            handleQueryParams(query)
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
            <FullscreenPage {...componentProps} />
        )
    } else if (settings?.type === 'modal') {
        return <FullscreenModal {...componentProps} />
    } else {
        return null
    }
}