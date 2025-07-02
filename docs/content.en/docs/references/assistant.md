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
curl -N -H 'Content-Type: application/json'   -XPOST http://localhost:9000/chat/_create -d'{
  "message":"how are you doing?"
}'
```

This API is HTTP Streaming based API, the first line will be session creation result, as follows:
```
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
The `payload` field is the result of the first request message.

And start with the second line will be `chunk` based chat response, please refer to section [### Send a Message](### Send a Message)

Here is the full request and response:
```shell
➜ curl  -H'X-API-TOKEN: d1iah2v3edbmq3jakcqg035du63myi8es0j4vqibelhvjnwy6gaeqd0ewitu4c6h1ad73c56pqknjbui4qfw'  -N -H 'Content-Type: application/json'   -XPOST http://localhost:9000/chat/_create -d'{
  "message":"how are you doing?"
}'
{"_id":"d1iah7f3edbmq3jakcs0","_source":{"id":"d1iah7f3edbmq3jakcs0","created":"2025-07-02T11:33:49.167656+08:00","updated":"2025-07-02T11:33:49.167664+08:00","status":"active","title":"how are you doing?","visible":true},"payload":{"id":"d1iah7f3edbmq3jakcsg","created":"2025-07-02T11:33:49.191362+08:00","updated":"2025-07-02T11:33:49.191371+08:00","type":"user","session_id":"d1iah7f3edbmq3jakcs0","from":"","message":"how are you doing?","details":null,"up_vote":0,"down_vote":0,"assistant_id":"default"},"result":"created"}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iah7f3edbmq3jakct0","message_type":"assistant","reply_to_message":"d1iah7f3edbmq3jakcsg","chunk_sequence":0,"chunk_type":"response","message_chunk":""}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iah7f3edbmq3jakct0","message_type":"assistant","reply_to_message":"d1iah7f3edbmq3jakcsg","chunk_sequence":1,"chunk_type":"response","message_chunk":""}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iah7f3edbmq3jakct0","message_type":"assistant","reply_to_message":"d1iah7f3edbmq3jakcsg","chunk_sequence":2,"chunk_type":"response","message_chunk":"I"}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iah7f3edbmq3jakct0","message_type":"assistant","reply_to_message":"d1iah7f3edbmq3jakcsg","chunk_sequence":3,"chunk_type":"response","message_chunk":"'m"}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iah7f3edbmq3jakct0","message_type":"assistant","reply_to_message":"d1iah7f3edbmq3jakcsg","chunk_sequence":4,"chunk_type":"response","message_chunk":" an"}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iah7f3edbmq3jakct0","message_type":"assistant","reply_to_message":"d1iah7f3edbmq3jakcsg","chunk_sequence":5,"chunk_type":"response","message_chunk":" artificial intelligence, so"}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iah7f3edbmq3jakct0","message_type":"assistant","reply_to_message":"d1iah7f3edbmq3jakcsg","chunk_sequence":6,"chunk_type":"response","message_chunk":" I don't have"}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iah7f3edbmq3jakct0","message_type":"assistant","reply_to_message":"d1iah7f3edbmq3jakcsg","chunk_sequence":7,"chunk_type":"response","message_chunk":" feelings or physical states"}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iah7f3edbmq3jakct0","message_type":"assistant","reply_to_message":"d1iah7f3edbmq3jakcsg","chunk_sequence":8,"chunk_type":"response","message_chunk":", but I'm"}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iah7f3edbmq3jakct0","message_type":"assistant","reply_to_message":"d1iah7f3edbmq3jakcsg","chunk_sequence":9,"chunk_type":"response","message_chunk":" here and ready to"}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iah7f3edbmq3jakct0","message_type":"assistant","reply_to_message":"d1iah7f3edbmq3jakcsg","chunk_sequence":10,"chunk_type":"response","message_chunk":" help you with any"}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iah7f3edbmq3jakct0","message_type":"assistant","reply_to_message":"d1iah7f3edbmq3jakcsg","chunk_sequence":11,"chunk_type":"response","message_chunk":" questions or tasks you"}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iah7f3edbmq3jakct0","message_type":"assistant","reply_to_message":"d1iah7f3edbmq3jakcsg","chunk_sequence":12,"chunk_type":"response","message_chunk":" might have! How"}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iah7f3edbmq3jakct0","message_type":"assistant","reply_to_message":"d1iah7f3edbmq3jakcsg","chunk_sequence":13,"chunk_type":"response","message_chunk":" can I assist you"}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iah7f3edbmq3jakct0","message_type":"assistant","reply_to_message":"d1iah7f3edbmq3jakcsg","chunk_sequence":14,"chunk_type":"response","message_chunk":" today?"}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iah7f3edbmq3jakct0","message_type":"assistant","reply_to_message":"d1iah7f3edbmq3jakcsg","chunk_sequence":15,"chunk_type":"response","message_chunk":""}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iah7f3edbmq3jakct0","message_type":"system","reply_to_message":"d1iah7f3edbmq3jakcsg","chunk_sequence":0,"chunk_type":"reply_end","message_chunk":"Processing completed"}
```

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
curl -N -H 'Content-Type: application/json' -XPOST http://localhost:9000/chat/csk30fjq50k7l4akku9g/_chat -d '{"message":"Hello"}'
```

This API is HTTP Streaming based API, the first line will be request message result, as follows:

```shell
//response
{
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
}
```
Starting from the second line, the response will use a chunk-based format. Each chunk will include a `chunk_type` indicating the stage of the reply.

Here is the full request and response:

```shell
➜ curl  -H'X-API-TOKEN: d1iah2v3edbmq3jakcqg035du63myi8es0j4vqibelhvjnwy6gaeqd0ewitu4c6h1ad73c56pqknjbui4qfw'  -N -H 'Content-Type: application/json' -XPOST http://localhost:9000/chat/d1iah7f3edbmq3jakcs0/_chat -d '{"message":"Hello"}'

[{"_id":"d1iahrf3edbmq3jakd30","_source":{"id":"d1iahrf3edbmq3jakd30","created":"2025-07-02T11:35:09.015438+08:00","updated":"2025-07-02T11:35:09.015446+08:00","type":"user","session_id":"d1iah7f3edbmq3jakcs0","from":"","message":"Hello","details":null,"up_vote":0,"down_vote":0,"assistant_id":"default"},"result":"created"}]
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iahrf3edbmq3jakd3g","message_type":"assistant","reply_to_message":"d1iahrf3edbmq3jakd30","chunk_sequence":0,"chunk_type":"response","message_chunk":""}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iahrf3edbmq3jakd3g","message_type":"assistant","reply_to_message":"d1iahrf3edbmq3jakd30","chunk_sequence":1,"chunk_type":"response","message_chunk":""}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iahrf3edbmq3jakd3g","message_type":"assistant","reply_to_message":"d1iahrf3edbmq3jakd30","chunk_sequence":2,"chunk_type":"response","message_chunk":"Hello"}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iahrf3edbmq3jakd3g","message_type":"assistant","reply_to_message":"d1iahrf3edbmq3jakd30","chunk_sequence":3,"chunk_type":"response","message_chunk":"!"}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iahrf3edbmq3jakd3g","message_type":"assistant","reply_to_message":"d1iahrf3edbmq3jakd30","chunk_sequence":4,"chunk_type":"response","message_chunk":" How"}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iahrf3edbmq3jakd3g","message_type":"assistant","reply_to_message":"d1iahrf3edbmq3jakd30","chunk_sequence":5,"chunk_type":"response","message_chunk":" can I assist you"}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iahrf3edbmq3jakd3g","message_type":"assistant","reply_to_message":"d1iahrf3edbmq3jakd30","chunk_sequence":6,"chunk_type":"response","message_chunk":" today?"}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iahrf3edbmq3jakd3g","message_type":"assistant","reply_to_message":"d1iahrf3edbmq3jakd30","chunk_sequence":7,"chunk_type":"response","message_chunk":""}
{"session_id":"d1iah7f3edbmq3jakcs0","message_id":"d1iahrf3edbmq3jakd3g","message_type":"system","reply_to_message":"d1iahrf3edbmq3jakd30","chunk_sequence":0,"chunk_type":"reply_end","message_chunk":"Processing completed"}
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