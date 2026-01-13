---
title: "Summary"
weight: 3
---


## Summary Processor

Generates AI-powered document summaries and insights with structured analysis.

### Configuration

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `message_field` | string | `documents` | The field in the pipeline context containing the documents to process |
| `output_queue` | object | `null` | Optional queue configuration for sending processed documents to a output queue |
| `model_provider` | string | Yes | - | ID of the LLM provider |
| `model` | string | Yes | - | Name of the LLM model |
| `model_context_length` | uint32 | Yes | - | Minimum context length (min: 4000 tokens) |
| `min_input_document_length` | uint32 | No | 100 | Minimum bytes to process a document |
| `max_input_document_length` | uint32 | No | 100000 | Maximum document size to process |
| `ai_insights_max_length` | uint32 | No | 500 | Target length for AI insights (in tokens) |

### Example

```yaml
- document_summarization:
  model_provider: openai
  model: gpt-4o-mini
  model_context_length: 4000
  min_input_document_length: 100
  max_input_document_length: 100000
  ai_insights_max_length: 500
```