---
title: "Profile"
weight: 50
---

# Profile API

## Profile API Reference

### Profile

Below is the field description for the profile object.

| **Field**            | **Type**            | **Description**                                                                                      |
|----------------------|---------------------|------------------------------------------------------------------------------------------------------|
| `id`                 | `string`            | Unique identifier for the user profile.                                                              |
| `username`           | `string`            | User's display name or username.                                                                     |
| `email`              | `string`            | User's email address.                                                                               |
| `avatar`             | `string` (URL)      | URL to the user's avatar image.                                                                      |
| `created`            | `string` (datetime) | Timestamp when the profile was created.                                                              |
| `updated`            | `string` (datetime) | Timestamp when the profile was last updated.                                                         |
| `roles`              | `array[string]`     | List of roles assigned to the user, e.g., `["admin", "editor"]`.                                     |
| `preferences`        | `object`            | User-specific preferences or settings.                                                               |
| `preferences.theme`  | `string`            | Preferred theme, e.g., `dark` or `light`.                                                            |
| `preferences.language` | `string`          | Preferred language, e.g., `en`, `fr`.                                                                |

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