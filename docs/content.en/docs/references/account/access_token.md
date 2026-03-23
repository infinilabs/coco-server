---
title: "API Token"
weight: 20
---

# API Token

API Tokens are long-lived tokens that can be used in your own applications to access Coco Server APIs.
Tokens are valid for 365 days from creation. Use the `X-API-TOKEN` header to authenticate requests.

## API Token Fields

| **Field**      | **Type**        | **Description**                                                              |
|----------------|-----------------|------------------------------------------------------------------------------|
| `name`         | `string`        | Custom name for the token. Auto-generated if not provided.                   |
| `permissions`  | `array[object]` | List of permission keys the token is authorized for. Defaults to the creating user's full permissions. |
| `access_token` | `string`        | The actual token string (only returned on creation).                         |
| `expire_in`    | `int64`         | Unix timestamp when the token expires (365 days from creation).              |

## Request Access Token

Create a new API token. The token string is only returned once during creation.

```shell
//request
curl -H "Authorization: Bearer <access_token>" \
  -H 'Content-Type: application/json' \
  -XPOST http://localhost:9000/auth/access_token -d'{
  "name": "my-app-token",
  "permissions": []
}'

//response
{
  "access_token": "a1b2c3d4-5678-90ab-cdef-1234567890ab_randomchars...",
  "expire_in": 1756684800,
  "status": "ok"
}
```

> **Note:** Save the `access_token` value securely. It cannot be retrieved again after creation.

## Search Access Tokens

List and search existing API tokens. The actual token values are masked in search results.

```shell
//request
curl -H "Authorization: Bearer <access_token>" \
  -XGET http://localhost:9000/auth/access_token/_search

//response
{
  "took": 2,
  "hits": {
    "total": { "value": 1, "relation": "eq" },
    "hits": [
      {
        "_id": "token_id_123",
        "_source": {
          "id": "token_id_123",
          "name": "my-app-token",
          "type": "general",
          "expire_in": 1756684800,
          "created": "2025-01-01T00:00:00Z",
          "updated": "2025-01-01T00:00:00Z"
        }
      }
    ]
  }
}
```

## Update Access Token

Update the name or permissions of an existing API token.

```shell
//request
curl -H "Authorization: Bearer <access_token>" \
  -H 'Content-Type: application/json' \
  -XPUT http://localhost:9000/auth/access_token/token_id_123 -d'{
  "name": "renamed-token"
}'

//response
{
  "_id": "token_id_123",
  "result": "updated"
}
```

### Parameters

| **Field**     | **Type**        | **Required** | **Description**                                     |
|---------------|-----------------|--------------|-----------------------------------------------------|
| `name`        | `string`        | Yes          | Updated name for the token.                         |
| `permissions` | `array[object]` | No           | Updated permission set (must be subset of user's permissions). |

## Delete Access Token

Delete an existing API token. The token will be immediately invalidated.

```shell
//request
curl -H "Authorization: Bearer <access_token>" \
  -XDELETE http://localhost:9000/auth/access_token/token_id_123

//response
{
  "_id": "token_id_123",
  "result": "deleted"
}
```

## Using an API Token

Once you have a token, use the `X-API-TOKEN` header to authenticate API requests:

```shell
curl -XGET http://localhost:9000/account/profile \
  -H "X-API-TOKEN: a1b2c3d4-5678-90ab-cdef-1234567890ab_randomchars..."
```