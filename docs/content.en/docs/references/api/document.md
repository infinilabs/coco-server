---
title: "Document"
weight: 50
---

# Document API

## Document API Reference

### Documents

Below is the field description for the document.

| **Field**              | **Type**           | **Description**                                                                                     |
|-------------------------|--------------------|-----------------------------------------------------------------------------------------------------|
| `source`               | `string`          | The source of the document, e.g., `google_drive`.                                                  |
| `category`             | `string`          | Primary category of the document, e.g., `report`.                                                  |
| `categories`           | `array[string]`   | List of categories the document belongs to, e.g., `["business", "quarterly_reports"]`.             |
| `cover`                | `string` (URL)    | URL to the cover image of the document.                                                            |
| `title`                | `string`          | Title of the document, e.g., `Q3 Business Report`.                                                 |
| `summary`              | `string`          | A brief summary of the document content.                                                           |
| `type`                 | `string`          | Type of the document, e.g., `PDF`.                                                                 |
| `lang`                 | `string`          | Language of the document, e.g., `en`.                                                              |
| `content`              | `string`          | The main content or body of the document.                                                          |
| `icon`                 | `string` (URL)    | URL to the document's icon.                                                                        |
| `thumbnail`            | `string` (URL)    | URL to the document's thumbnail image.                                                             |
| `tags`                 | `array[string]`   | Tags associated with the document, e.g., `["finance", "quarterly", "business", "report"]`.         |
| `url`                  | `string` (URL)    | URL link to the document, e.g., a Google Drive link.                                               |
| `size`                 | `integer`         | Size of the document in bytes, e.g., `1048576` for 1 MB.                                           |
| `owner`                | `object`          | Information about the document's owner.                                                           |
| `owner.avatar`         | `string` (URL)    | URL to the owner's avatar image.                                                                   |
| `owner.username`       | `string`          | Username of the owner, e.g., `jdoe`.                                                               |
| `owner.userid`         | `string`          | User ID of the owner, e.g., `user123`.                                                             |
| `metadata`             | `object`          | Additional accessible metadata (e.g., file version, permissions).                                    |
| `payload`              | `object`          | Additional store-only metadata (e.g., file binary data).                                             |
| `last_updated_by`      | `object`          | Information about the last user who updated the document.                                          |
| `last_updated_by.user` | `object`          | Details about the user who last updated the document.                                              |
| `last_updated_by.user.avatar` | `string` (URL) | URL to the avatar of the last editor.                                                              |
| `last_updated_by.user.username` | `string` | Username of the last editor, e.g., `editor123`.                                                    |
| `last_updated_by.user.userid` | `string`   | User ID of the last editor, e.g., `editor123@example.com`.                                          |
| `last_updated_by.timestamp` | `string` (datetime) | Timestamp of the last update, e.g., `2024-11-01T15:30:00Z`.                                       |


### Index a Document

```shell
//request
curl -H 'Content-Type: application/json' -XPOST http://localhost:2900/document/ -d '{
  "source": "google_drive",
  "category": "report",
  "categories": ["business", "quarterly_reports"],
  "cover": "https://example.com/images/report_cover.jpg",
  "title": "Q3 Business Report",
  "summary": "An overview of the company financial performance for Q3.",
  "type": "PDF",
  "lang": "en",
  "content": "This quarters revenue increased by 15%, driven by strong sales in the APAC region...",
  "icon": "https://example.com/images/icon.png",
  "thumbnail": "https://example.com/images/report_thumbnail.jpg",
  "tags": ["finance", "quarterly", "business", "report"],
  "url": "https://drive.google.com/file/d/abc123/view",
  "size": 1048576,
  "owner": {
    "avatar": "https://example.com/images/user_avatar.jpg",
    "username": "jdoe",
    "userid": "user123"
  },
  "metadata": {
    "version": "1.2",
    "department": "Finance",
    "last_reviewed": "2024-10-20",
    "file_extension": "pdf",
    "icon_link": "https://example.com/images/file_icon.png",
    "has_thumbnail": true,
    "kind": "drive#file",
    "parents": ["folder123"],
    "properties": { "shared": "true" },
    "spaces": ["drive"],
    "starred": false,
    "driveId": "drive123",
    "thumbnail_link": "https://example.com/images/file_thumbnail.jpg",
    "video_media_metadata": { "durationMillis": "60000", "width": 1920, "height": 1080 },
    "image_media_metadata": { "width": 1024, "height": 768 }
  },
  "last_updated_by": {
    "user": {
      "avatar": "https://example.com/images/editor_avatar.jpg",
      "username": "editor123",
      "userid": "editor123@example.com"
    },
    "timestamp": "2024-11-01T15:30:00Z"
  }
}'

//response
{
  "_id": "cso9vr3q50k38nobvmcg",
  "result": "created"
}
```

### Get a Document

```shell
//request
curl   -XGET http://localhost:2900/document/cso9vr3q50k38nobvmcg

//response
{
  "_id": "cso9vr3q50k38nobvmcg",
  "_source": {
    "id": "cso9vr3q50k38nobvmcg",
    "created": "2024-11-10T19:58:36.009086+08:00",
    "updated": "2024-11-10T19:58:36.009092+08:00",
    "source": "google_drive",
    ...OMITTED...
   ,
  "found": true
}
```

### Update a Document

```shell
//request
curl  -H 'Content-Type: application/json'   -XPUT http://localhost:2900/document/cso9vr3q50k38nobvmcg -d'{ "source": "google_drive", ...OMITTED... , "timestamp": "2024-11-01T15:30:00Z" } }'

//response
{
  "_id": "cso9vr3q50k38nobvmcg",
  "result": "updated"
}
```

### Delete a Document

```shell
//request
curl  -XDELETE http://localhost:2900/document/cso9vr3q50k38nobvmcg

//response
{
  "_id": "cso9vr3q50k38nobvmcg",
  "result": "deleted"
}
```