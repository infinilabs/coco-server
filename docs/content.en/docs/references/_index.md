---
title: "References"
weight: 20
bookCollapseSection: true
---

# API Reference

Welcome to the API Reference. Below are details on how to interact with the different endpoints of the coco server, including authentication and methods for working with documents.

## Authentication Methods

The API supports two methods of authentication:

### 1. Basic Authentication

Use Basic Authentication by passing a `Authorization` header with the value `Basic <base64-encoded-username:password>`.

Example request:

```bash
curl -XGET http://localhost:2900/profile \
  -H "Authorization: Basic <base64-encoded-username:password>"
```

### 2. API Token Authentication

Use the X-API-TOKEN header with your token value.

Example request:
```
curl -XGET http://localhost:2900/profile \
  -H "X-API-TOKEN: xxxxx"
```