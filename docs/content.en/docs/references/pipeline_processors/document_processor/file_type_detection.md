---
title: "File Type Detection"
weight: 0
---

## File Type Detection Processor

Detects the MIME type and content category of a document based on its file
extension. Documents that already have this information set are skipped.

After processing, each document will carry:

| Field | Example value | Description |
|---|---|---|
| MIME type | `image/png` | Standard MIME type |
| Content type | `image` | The primary type portion of the MIME type |
| Content category | `image`, `document`, `text` | A high-level category |

### Configuration

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `message_field` | string | No | `messages` | Pipeline context key for the input messages |
| `output_queue` | object | No | `null` | Queue to push processed documents to |

### Example

```yaml
- file_type_detection:
    output_queue:
      name: "documents_typed"
```
