---
title: "Document Cover"
weight: 2
---

## Document Cover Processor

Generates a cover image and a thumbnail for a document. The resulting URLs are
stored in `doc.Cover` and `doc.Thumbnail`.

| Supported format | Notes |
|---|---|
| PDF | Cover from the first page |
| PPTX / DOCX / XLSX and OpenDocument equivalents | Cover from the first page |
| Markdown | Rendered cover |
| Image | Thumbnail of the image itself |

### Requirements/Dependencies

| Tool | Required for |
|---|---|
| `pdftoppm` (poppler-utils) | PDF cover generation |
| LibreOffice (`soffice`) | Office document cover generation |
| Chromium (headless) | Markdown rendered cover |

### Configuration

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `message_field` | string | No | `messages` | Pipeline context key containing the `[]queue.Message` to process |
| `output_queue` | object | No | `null` | Queue to push processed documents to |

### Example

```yaml
- document_cover:
    output_queue:
      name: "documents_with_cover"
```
