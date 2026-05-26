---
title: "Generate Cover"
weight: 2
---

## Generate Cover Processor

Generates a cover image and a thumbnail for a document or an attachment.

**Document mode** — result is stored in `doc.Cover` and `doc.Thumbnail`.  
**Attachment mode** — result is stored in `attachment.metadata.cover` and
`attachment.metadata.thumbnail`.

The processor detects which mode to use automatically:
- If `attachment_meta` is present in the pipeline context → attachment mode
- Otherwise → document mode (reads from the message queue field)

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
| `message_field` | string | No | `messages` | Pipeline context key containing the `[]queue.Message` to process (document mode only) |
| `output_queue` | object | No | `null` | Queue to push processed documents to (document mode only) |

### Example — document pipeline

```yaml
- generate_cover:
    output_queue:
      name: "documents_with_cover"
```

### Example — attachment pipeline

```yaml
- generate_cover: {}
```
