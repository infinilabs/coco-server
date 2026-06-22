---
title: "File Metadata"
weight: 1
---

## File Metadata Processor

Extracts metadata from supported file types and stores the results in the
document metadata.

| File type | Extracted metadata |
|---|---|
| Image | `colors` (top-3 dominant color names), `width` (px), `height` (px) |

### Configuration

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `message_field` | string | No | `messages` | Pipeline context key for the input messages |
| `output_queue` | object | No | `null` | Queue to push processed documents to |

### Example

```yaml
- file_metadata:
    output_queue:
      name: "documents_with_metadata"
```
