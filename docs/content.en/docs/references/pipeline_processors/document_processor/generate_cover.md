---
title: "Generate Document Cover"
weight: 2
---

## Generate Document Cover Processor

Generates a cover image and a thumbnail for a document.

> **Note**: For attachment cover generation, use `generate_attachment_cover` instead.

The generated cover and thumbnail are stored with the document and can be
retrieved via the document API.

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
| `message_field` | string | No | `messages` | Pipeline context key for the input messages |
| `output_queue` | object | No | `null` | Queue to push processed documents to |

### Example

```yaml
- generate_document_cover:
    output_queue:
      name: "documents_with_cover"
```
