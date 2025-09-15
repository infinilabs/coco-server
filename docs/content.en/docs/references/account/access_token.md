---
title: "API Token"
weight: 20
---

# API Token

## Request Access Token

An API Token can be used in your own application to access Coco Server.

Example request:
```
curl -H "Authorization: Bearer <access_token>" -XPOST http://localhost:9000/auth/access_token
```