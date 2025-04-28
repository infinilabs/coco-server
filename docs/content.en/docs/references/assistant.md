---
title: "Assistant"
weight: 50
---

# Assistant

## Assistant API Reference

### Create an AI assistant

```shell
//request
curl -XPOST http://localhost:9000/assistant/ -d'{
  "name" : "deault",
  "description" : "default AI chat assistant",
  "icon" : "font_Google-video",
  "type" : "deep_think",
  "config" : {
    "intent_analysis_model" : {
      "name" : "tongyi-intent-detect-v3",
      "provider_id" : "cvuai3dath2dlgqqpc2g",
      "settings": {
        "temperature" : 0.8,
        "top_p" : 0.5,
        "presence_penalty" : 0,
        "frequency_penalty" : 0,
        "max_tokens" : 1024
      }
    },
    "picking_doc_model" : {
      "name" : "deepseek-r1-distill-qwen-32b",
      "provider_id" : "cvuai3dath2dlgqqpc2g",
      "settings": {
        "temperature" : 0.8,
        "top_p" : 0.5,
        "presence_penalty" : 0,
        "frequency_penalty" : 0,
        "max_tokens" : 1024
      }
    }
  },
  "answering_model" : {
    "provider_id" : "cvuai3dath2dlgqqpc2g",
    "name" : "deepseek-r1",
    "settings" : {
      "temperature" : 0.8,
      "top_p" : 0.5,
      "presence_penalty" : 0,
      "frequency_penalty" : 0,
      "max_tokens" : 1024
    }
  },
  "datasource" : {
    "enabled" : true,
    "ids" : [
      "d895f22ed2ff25ad8c6080af1cc23a21"
    ],
    "visible" : true,
    "filter": {"term":{"name": "test"}}
  },
  "mcp_servers" : {
    "enabled" : true,
    "ids" : [
      "*"
    ],
    "visible" : true,
    "max_iterations": 3,
    "model": {
      "temperature" : 0.8,
      "top_p" : 0.5,
      "presence_penalty" : 0,
      "frequency_penalty" : 0,
      "max_tokens" : 1024
    }
  },
  "tools": {
    "builtin": {
      "calculator": true, 
      "wikipedia: false, 
      "duckduckgo": false, 
      "scraper": false
    },
    "enabled": true
  },
  "keepalive" : "30m",
  "enabled" : true,
  "chat_settings" : {
    "greeting_message" : "Hi! I’m Coco, nice to meet you. I can help answer your questions by tapping into the internet and your data sources. How can I assist you today?",
    "suggested" : {
      "enabled" : false,
      "questions" : [ ]
    },
    "input_preprocess_tpl" : "",
    "history_message" : {
      "number" : 5,
      "compression_threshold" : 1000,
      "summary" : true
    }
  },
  "builtin" : false,
  "role_prompt" : ""
}'
//response
{
  "_id": "cvuak1lath2dlgqqpcjg",
  "result": "created"
}
```

### Update an AI assistant

```shell
//request
curl -XPUT http://localhost:9000/assistant/cvuak1lath2dlgqqpcjg -d'{
   "name" : "deault",
  "description" : "default AI chat assistant",
  "icon" : "font_Google-video",
  "type" : "deep_think",
  "config" : {
    "intent_analysis_model" : {
      "name" : "tongyi-intent-detect-v3",
      "provider_id" : "cvuai3dath2dlgqqpc2g",
      "settings": {
        "temperature" : 0.8,
        "top_p" : 0.5,
        "presence_penalty" : 0,
        "frequency_penalty" : 0,
        "max_tokens" : 1024
      }
    },
    "picking_doc_model" : {
      "name" : "deepseek-r1-distill-qwen-32b",
      "provider_id" : "cvuai3dath2dlgqqpc2g",
      "settings": {
        "temperature" : 0.8,
        "top_p" : 0.5,
        "presence_penalty" : 0,
        "frequency_penalty" : 0,
        "max_tokens" : 1024
      }
    }
  },
  "answering_model" : {
    "provider_id" : "cvuai3dath2dlgqqpc2g",
    "name" : "deepseek-r1",
    "settings" : {
      "temperature" : 0.8,
      "top_p" : 0.5,
      "presence_penalty" : 0,
      "frequency_penalty" : 0,
      "max_tokens" : 1024
    }
  },
  "datasource" : {
    "enabled" : true,
    "ids" : [
      "d895f22ed2ff25ad8c6080af1cc23a21"
    ],
    "visible" : true,
    "filter": {"term":{"name": "test"}}
  },
  "mcp_servers" : {
    "enabled" : true,
    "ids" : [
      "*"
    ],
    "visible" : true,
    "max_iterations": 3,
    "model": {
      "temperature" : 0.8,
      "top_p" : 0.5,
      "presence_penalty" : 0,
      "frequency_penalty" : 0,
      "max_tokens" : 1024
    }
  },
  "tools": {
    "builtin": {
      "calculator": true, 
      "wikipedia: false, 
      "duckduckgo": false, 
      "scraper": false
    },
    "enabled": true
  },
  "keepalive" : "30m",
  "enabled" : true,
  "chat_settings" : {
    "greeting_message" : "Hi! I’m Coco, nice to meet you. I can help answer your questions by tapping into the internet and your data sources. How can I assist you today?",
    "suggested" : {
      "enabled" : false,
      "questions" : [ ]
    },
    "input_preprocess_tpl" : "",
    "history_message" : {
      "number" : 5,
      "compression_threshold" : 1000,
      "summary" : true
    }
  },
  "builtin" : false,
  "role_prompt" : ""
}'
//response
{
  "_id": "cvuak1lath2dlgqqpcjg",
  "result": "updated"
}
```

### View an AI assistant
```shell
curl -XGET http://localhost:9000/assistant/cvuak1lath2dlgqqpcjg
```


### Delete the AI assistant

```shell
//request
curl  -H 'Content-Type: application/json'   -XDELETE http://localhost:9000/assistant/cvuak1lath2dlgqqpcjg 

//response
{
  "_id": "cvuak1lath2dlgqqpcjg",
  "result": "deleted"
}'
```

### Search AI assistant
```shell
curl -X POST "http://localhost:9000/assistant/_search" -d'
{
"from": 0,
"size": 10
}'
```

### Clone an AI assistant
```shell
//request
curl -XPOST http://localhost:9000/assistant/cvuak1lath2dlgqqpcjg/_clone
//response
{
  "_id": "d04r1gic7k812t6qg3n0",
  "result": "created"
}'
```

### Retrieve Chat History (sessions)

```shell
//request
curl -XGET http://localhost:9000/chat/_history?query={filter_keyword}

//response
{"took":997,"timed_out":false,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"hits":{"total":{"value":1,"relation":"eq"},"max_score":1.0,"hits":[{"_index":".infini_session","_type":"_doc","_id":"csk30fjq50k7l4akku9g","_score":1.0,"_source":{"id":"csk30fjq50k7l4akku9g","created":"2024-11-04T10:23:58.980669+08:00","updated":"2024-11-04T10:23:58.980678+08:00","status":"active"}}]}}
```

### Open a Existing Chat Session

```shell
//request
curl  -H 'Content-Type: application/json'   -XPOST http://localhost:9000/chat/csk30fjq50k7l4akku9g/_open

//response
{
  "_id": "csk30fjq50k7l4akku9g",
  "_source": {
    "id": "csk30fjq50k7l4akku9g",
    "created": "2024-11-04T10:23:58.980669+08:00",
    "updated": "2024-11-04T10:25:20.541856+08:00",
    "status": "active"
  },
  "found": true
}
```


### Create a Chat Session

```shell
//request
curl  -H'WEBSOCKET-SESSION-ID: csk88l3q50kb4hr5unn0'  -H 'Content-Type: application/json'   -XPOST http://localhost:9000/chat/_new -d'{
  "message":"how are you doing?"
}'

//response
{
  "_id": "csk30fjq50k7l4akku9g",
  "_source": {
    "id": "csk30fjq50k7l4akku9g",
    "created": "2024-11-04T10:23:58.980669+08:00",
    "updated": "2024-11-04T10:23:58.980678+08:00",
    "status": "active"
  },
  "result": "created"
  "payload": {
    //first chat message
  }
}
```
Tips: `WEBSOCKET-SESSION-ID` should be replaced with the actual WebSocket session ID. You will receive a message each time you connect to the Coco AI WebSocket server. For example: `ws://localhost:2900/ws` or `wss://localhost:2900/ws` if TLS is enabled. Parse the websocket session id,  save it and pass it each time you send message to Coco server.

> Note: If the Coco server doesn’t recognize your WebSocket ID, it won’t be able to process the reply, as it can’t send the response in a streaming manner.

{{% load-img "/img/websocket-on-connect.jpg?raw=true" "WebSocket ID" %}}

### Get Chat Session Info
```shell
//request
curl -XGET http://localhost:9000/chat/csk30fjq50k7l4akku9g

//response
{
  "_id": "csk30fjq50k7l4akku9g",
  "_source": {
    "id": "csk30fjq50k7l4akku9g",
    "created": "2025-04-01T10:48:38.389295+08:00",
    "updated": "2025-04-01T10:48:40.572748+08:00",
    "status": "active",
    "title": "xx"
  },
  "found": true
}
```

### Update Chat Session Info
```shell
//request
curl -XPUT http://localhost:9000/chat/csk30fjq50k7l4akku9g -d'
{
    "title":"my title",
    "context":{
        "attachments":[]
    }
}'

//response
{
  "_id": "csk30fjq50k7l4akku9g",
  "result": "updated"
}
```

### Delete Chat Session
```shell
//request
curl -DELETE http://localhost:9000/chat/csk30fjq50k7l4akku9g

//response
{
  "_id": "csk30fjq50k7l4akku9g",
  "result": "deleted"
}
```

### Retrieve a Chat History

```shell
//request
curl -XGET http://localhost:9000/chat/csk30fjq50k7l4akku9g/_history

//response
{"took":4,"timed_out":false,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"hits":{"total":{"value":0,"relation":"eq"},"max_score":null,"hits":[]}}
```

### Send a Message

```shell
//request
curl -H'WEBSOCKET-SESSION-ID: csk88l3q50kb4hr5unn0' -H 'Content-Type: application/json' -XPOST http://localhost:9000/chat/csk30fjq50k7l4akku9g/_send -d '{"message":"Hello"}'

//response
[{
  "_id": "csk325rq50k85fc5u0j0",
  "_source": {
    "id": "csk325rq50k85fc5u0j0",
    "type": "user",
    "created": "2024-11-04T10:27:35.211502+08:00",
    "updated": "2024-11-04T10:27:35.211508+08:00",
    "session_id": "csk30fjq50k7l4akku9g",
    "message": "Hello"
  },
  "result": "created"
}]
```

### Close a Chat Session

```shell
//request
curl  -H 'Content-Type: application/json'   -XPOST http://localhost:9000/chat/csk30fjq50k7l4akku9g/_close

//response
{
  "_id": "csk30fjq50k7l4akku9g",
  "_source": {
    "id": "csk30fjq50k7l4akku9g",
    "created": "2024-11-04T10:23:58.980669+08:00",
    "updated": "2024-11-04T10:28:47.461033+08:00",
    "status": "closed"
  },
  "found": true
}
```

### Cancel a Message

```shell
//request
curl   -H 'Content-Type: application/json'   -XPOST http://localhost:9000/chat/csk30fjq50k7l4akku9g/_cancel

//response
{
  "acknowledged": true
}
```

## Assistant UI Management

### Search Assistant
Log in to the Coco-Server admin dashboard, click `AI Assistant` in the left menu to view all assistant lists, as shown below:  
{{% load-img "/img/assistant/list.png" "assistant list" %}}

Enter keywords in the search box above the list and click the `Refresh` button to search for matching assistant, as shown below:  
{{% load-img "/img/assistant/filter-list.png" "assistant search" %}}


### Add Assistant
Click `Add` in the top-right corner of the list to create a new assistant, as shown below:  
{{% load-img "/img/assistant/add-1.png" "add assistant" %}}  
{{% load-img "/img/assistant/add-2.png" "add assistant" %}}
{{% load-img "/img/assistant/add-3.png" "add assistant" %}}

The system provides default values for the assistant configuration. Modify these values as needed, then click the save button to complete the creation.


### Delete Assistant
Select the target assistant in the list, click `Delete` on the right side of the entry, and confirm in the pop-up dialog to complete the deletion. As shown below:  
{{% load-img "/img/assistant/delete.png" "delete assistant" %}}

> The built-in assistant cannot be deleted, but it can be modified.


### Edit Assistant
Select the target assistant in the list, click `Edit` on the right side to enter the editing page. Modify the configuration and click save to update. As shown below:  
{{% load-img "/img/assistant/edit.png" "edit assistant" %}}


### Clone Assistant
Select the target assistant in the list, click `Clone` on the right side to clone a assistant and then you will enter the editing page. Just like the operation of `Edit Assistant`, Modify the configuration and click save to update.