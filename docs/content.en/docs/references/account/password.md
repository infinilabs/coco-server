---
title: "Modify Password"
weight: 50
---

# Modify Password

Modify the current user's password. Requires authentication.

## Modify Password

```shell
//request
curl -XPUT http://localhost:9000/account/password \
  -H "Authorization: Bearer <access_token>" \
  -H 'Content-Type: application/json' \
  -d'{
  "old_password":"current_password",
  "new_password":"new_secure_password"
}'

//response
{
  "_id": "coco-default-user",
  "result": "updated"
}
```

### Parameters

| **Field**      | **Type** | **Required** | **Description**                   |
|----------------|----------|--------------|-----------------------------------|
| `old_password` | `string` | Yes          | The user's current password.      |
| `new_password` | `string` | Yes          | The new password to set.          |