---
title: "Profile"
weight: 50
---

# Profile API

## Profile API Reference

### Profile

The `Profile` endpoint retrieves information about a user profile. Below is the field description for the profile object.

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

#### Request

```bash
curl -XGET http://localhost:9000/profile
```
#### Response

```bash
{
  "id": "user123",
  "username": "jdoe",
  "email": "jdoe@example.com",
  "avatar": "https://example.com/images/avatar.jpg",
  "created": "2024-01-01T10:00:00Z",
  "updated": "2025-01-01T10:00:00Z",
  "roles": ["admin", "editor"],
  "preferences": {
    "theme": "dark",
    "language": "en"
  }
}
```