---
title: "Document Processors"
weight: 110
bookCollapseSection: true
---

Processors that operate on **documents** flowing through a pipeline.

Their input is a `[]queue.Message` where each message carries a serialized
`core.Document`. They are typically used in file-processing pipelines that
run after a connector fetches a document.