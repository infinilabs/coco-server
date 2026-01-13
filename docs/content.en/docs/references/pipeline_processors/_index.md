---
title: "Pipeline Processors"
weight: 100
bookCollapseSection: true
---

# Processor

A processor performs specific actions on the input document when it flows 
through the pipeline:

```text
[Doc input] --> [Processor] --> [Doc output]
```

# Pipeline

A pipeline is basically multiple processors chained together, users can achieve
sophisticated document processing via pipelines:

```text
[Doc input] --> [Processor A] --> [Processor B] --> [Processor C] --> [Doc output]
```