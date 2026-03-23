---
title: "Authentication"
weight: 10
---

# Authentication

## Authentication Methods

The API supports three methods of authentication:

### 1. Login API

Authenticate with the Coco Server to obtain a JWT access token.

**Password-only login** (single-user mode):
```shell
curl -H 'Content-Type: application/json' -XPOST http://localhost:9000/account/login -d'{
  "password": "mypassword"
}'
```

**Email and password login** (multi-user mode):
```shell
curl -H 'Content-Type: application/json' -XPOST http://localhost:9000/account/login -d'{
  "email": "admin@example.com",
  "password": "mypassword"
}'
```

Response:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expire_in": 86400,
  "id": "coco-default-user",
  "status": "ok",
  "username": "coco-default-user"
}
```

| **Field**      | **Type** | **Description**                                         |
|----------------|----------|---------------------------------------------------------|
| `access_token` | `string` | JWT token for Bearer authentication (valid for 24 hours). |
| `expire_in`    | `int`    | Token validity duration in seconds.                      |
| `id`           | `string` | User ID.                                                 |
| `status`       | `string` | Response status (`ok`).                                  |
| `username`     | `string` | Username of the authenticated user.                      |

### 2. Bearer Authentication

Use the `Authorization` header with the `access_token` returned by the Login API.

```bash
curl -XGET http://localhost:9000/<api_endpoint> \
  -H "Authorization: Bearer <access_token>"
```

Example:
```shell
curl -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  http://localhost:9000/account/profile
```

### 3. API Token Authentication

Use the `X-API-TOKEN` header with a long-lived API token. See [API Token](./access_token) for how to create tokens.

```shell
curl -XGET http://localhost:9000/account/profile \
  -H "X-API-TOKEN: a1b2c3d4-5678-90ab-cdef-1234567890ab_randomchars..."
```

> API Tokens are valid for 365 days and are suitable for automated integrations and applications.