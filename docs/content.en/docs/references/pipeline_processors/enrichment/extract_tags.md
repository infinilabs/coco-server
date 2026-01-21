---
title: "Extract Tags"
weight: 4
---

## Extract Tags Processor

Extracts structured tags from AI insights stored in document metadata using an LLM.

### Configuration

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `message_field` | string | `documents` | The field in the pipeline context containing the documents to process |
| `output_queue` | object | `null` | Optional queue configuration for sending processed documents to a output queue |
| `model_provider` | string | Yes | - | ID of the LLM provider |
| `model` | string | Yes | - | Name of the LLM model |
| `model_context_length` | uint32 | Yes | - | Minimum context length (min: 4000 tokens) |

### Example

```yaml
- extract_tags:
  model_provider: openai
  model: gpt-4o-mini
  model_context_length: 4000
```