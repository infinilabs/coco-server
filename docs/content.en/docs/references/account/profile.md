---
title: "Profile"
weight: 50
---

# Profile API

## Profile API Reference

### Profile

Below is the field description for the profile object.

| **Field**       | **Type**        | **Description**                                                                      |
|-----------------|-----------------|--------------------------------------------------------------------------------------|
| `id`            | `string`        | Unique identifier for the user profile.                                              |
| `name`          | `string`        | User's display name.                                                                 |
| `email`         | `string`        | User's email address.                                                                |
| `permissions`   | `array[string]` | List of permission keys assigned to the user, e.g., `["coco:document:read"]`.        |

---

### Get Profile

Requires authentication. Returns the current user's profile information and permissions.

#### Request

```bash
curl -XGET http://localhost:9000/account/profile \
  -H "Authorization: Bearer <access_token>"
```
#### Response

```json
{
  "id": "user123",
  "name": "jdoe",
  "email": "jdoe@example.com",
  "permissions": [
    "coco:document:create",
    "coco:document:read",
    "coco:search:search"
  ]
}
```