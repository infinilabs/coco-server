---
title: "Attachment Processors"
weight: 120
bookCollapseSection: true
---

Processors that operate on **attachments** flowing through a pipeline.

Their input is a `[]queue.Message` where each message body is an attachment ID
(plain string). They are used in attachment-processing pipelines that run after
a file is uploaded via the attachment API.
