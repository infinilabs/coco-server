---
title: "Logout"
weight: 60
---

# Logout

## Logout API

The Logout API securely logs the user out from the Coco Server. It destroys the current session.

Both `GET` and `POST` methods are supported.

```shell
//request
curl -XPOST http://localhost:9000/account/logout \
  -H "Authorization: Bearer <access_token>"

//response
{
  "status": "ok"
}
```