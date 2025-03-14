---
title: "Attachment"
weight: 100
---

# Attachment

Attachment API are used to upload your local files to Coco server.

## Upload attachment in chat session

`curl -X POST http://localhost:9000/chat/[session_id]/_upload -F "files=LOCAL_FILE"`

Example
```
curl -X POST http://localhost:9000/chat/cu737jrq50kcaicgn7pg/_upload \
  -H "X-API-TOKEN: cv9pnurq50k1hii28630jy429g4b49viecrlj9529onpa6n0lti7yohioitvyotd0677rop5uszc0cnll03j" \
  -F "files=@/Users/medcl/Downloads/tmp/neurips19-diskann.pdf" \
  -F "files=@/Users/medcl/Downloads/tmp/Adaptive_searching_in_succinctly_encoded.pdf"
```

> The `session_id` need to be replaced with actual session id.

Response
```
{
  "acknowledged": true,
  "attachments": [
    "cv9q94bq50k2r0s6nob0",
    "cv9q94bq50k2r0s6nobg"
  ]
}
```

## Download attachment
```
curl -X GET http://localhost:9000/attachment/cv9q94bq50k2r0s6nobg \
  -H "X-API-TOKEN: cv9pnurq50k1hii28630jy429g4b49viecrlj9529onpa6n0lti7yohioitvyotd0677rop5uszc0cnll03j"
```

## Check attachment exists
```
 curl -I http://localhost:9000/attachment/cv9q94bq50k2r0s6nobg \
  -H "X-API-TOKEN: cv9pnurq50k1hii28630jy429g4b49viecrlj9529onpa6n0lti7yohioitvyotd0677rop5uszc0cnll03j"
```
Response
```
HTTP/1.1 200 OK
Content-Length: 2221342
Created: &{119327000 63877520584 4392528800}
Filename: neurips19-diskann.pdf
Vary: Accept-Encoding
Date: Fri, 14 Mar 2025 03:49:49 GMT
```

## Search attachments
```
curl -X GET http://localhost:9000/attachment/_search\?session\=cu737jrq50kcaicgn7pg \
  -H "X-API-TOKEN: cv9pnurq50k1hii28630jy429g4b49viecrlj9529onpa6n0lti7yohioitvyotd0677rop5uszc0cnll03j"
```
Response
```
{"took":2,"timed_out":false,"_shards":{"total":1,"successful":1,"skipped":0,"failed":0},"hits":{"total":{"value":2,"relation":"eq"},"max_score":0.87546873,"hits":[{"_index":"coco_attachment","_type":"_doc","_id":"cv9r1krq50k4qhqcqhu0","_score":0.87546873,"_source":{"id":"cv9r1krq50k4qhqcqhu0","created":"2025-03-14T12:30:11.827843+08:00","updated":"2025-03-14T12:30:11.827851+08:00","session":"cu737jrq50kcaicgn7pg","name":"neurips19-diskann.pdf","icon":"pdf","url":"/attachment/cv9r1krq50k4qhqcqhu0","size":2221342}},{"_index":"coco_attachment","_type":"_doc","_id":"cv9r1l3q50k4qhqcqhug","_score":0.87546873,"_source":{"id":"cv9r1l3q50k4qhqcqhug","created":"2025-03-14T12:30:12.857468+08:00","updated":"2025-03-14T12:30:12.857483+08:00","session":"cu737jrq50kcaicgn7pg","name":"Adaptive_searching_in_succinctly_encoded.pdf","icon":"pdf","url":"/attachment/cv9r1l3q50k4qhqcqhug","size":125382}}]}}
```