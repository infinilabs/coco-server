---
title: "Document Processors"
weight: 110
bookCollapseSection: true
---

Processors that operate on **documents** flowing through a pipeline.

Their input is a batch of pipeline messages where each message carries a
serialized document. They are typically used in file-processing pipelines that
run after a connector fetches a document.