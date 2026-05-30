---
title: "Attachment Processors"
weight: 120
bookCollapseSection: true
---

Processors that operate on **attachments** flowing through a pipeline.

Their input is a `[]queue.Message` where each message carries a serialized
`core.Attachment`.

They are typically used as sub-pipelines invoked by `process_attachments`
after a file has been uploaded via the attachment API.

Available attachment processors:

- `generate_attachment_cover` generates cover and thumbnail images for an attachment.
- `attachment_text_extraction` extracts text content from an attachment into `attachment.text`.