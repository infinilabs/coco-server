---
title: "Attachment"
weight: 100
---

# Attachment

Attachment API are used to upload your local files to Coco server.

## Upload attachment in chat session
```
curl -X POST http://localhost:9000/chat/session_id/_upload \
  -H "X-API-TOKEN: cv9pnurq50k1hii28630jy429g4b49viecrlj9529onpa6n0lti7yohioitvyotd0677rop5uszc0cnll03j" \
  -F "files=@/Users/medcl/Downloads/tmp/neurips19-diskann.pdf" \
  -F "files=@/Users/medcl/Downloads/tmp/Adaptive_searching_in_succinctly_encoded.pdf"
```
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