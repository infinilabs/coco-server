---
title: "Assistant"
weight: 50
---

# Assistant API

## Assistant API Reference


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