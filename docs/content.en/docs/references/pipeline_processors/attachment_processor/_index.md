---
title: "Attachment Processors"
weight: 120
bookCollapseSection: true
---

Processors that operate on **attachments** flowing through a pipeline.

Their input is a batch of pipeline messages where each message carries a
serialized attachment.

They are typically used as sub-pipelines invoked by `process_attachments`
after a file has been uploaded via the attachment API.

Available attachment processors:

- `generate_attachment_cover` generates cover and thumbnail images for an attachment.
- `attachment_text_extraction` extracts searchable text content from an attachment.