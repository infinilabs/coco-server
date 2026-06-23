---
title: "Generate Attachment Cover"
weight: 1
---

## Generate Attachment Cover Processor

Generates a cover image and a thumbnail for an attachment.

The generated cover and thumbnail are stored in the attachment metadata and can
be retrieved via the attachment API.

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

### Example

```yaml
- generate_attachment_cover: {}
```