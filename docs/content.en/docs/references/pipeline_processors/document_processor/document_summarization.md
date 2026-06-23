---
title: "Document Summarization"
weight: 5
---

## Document Summarization Processor

Generates an AI summary and structured insights for a document using a language
model.

Documents shorter than `min_input_document_length` or longer than
`max_input_document_length` are skipped.

### Requirements

A configured language model provider is required. Set `model_provider` and
`model` in the processor config, or configure a default language model in the
application settings.

### Configuration

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `message_field` | string | No | `messages` | Pipeline context key for the input messages |
| `output_queue` | object | No | `null` | Queue to push processed documents to |
| `model_provider` | string | No | *(app default)* | Language model provider ID |
| `model` | string | No | *(app default)* | Language model name |
| `model_context_length` | int | No | — | Model context window size in tokens (minimum 4000) |
| `min_input_document_length` | int | No | `100` | Skip documents shorter than this (bytes) |
| `max_input_document_length` | int | No | `100000` | Skip documents longer than this (bytes) |
| `ai_insights_max_length` | int | No | `500` | Target length for the generated summary (tokens) |
| `llm_generation_lang` | string | No | *(app default)* | BCP 47 language tag for generated content (e.g. `en-US`, `zh-CN`) |

### Example

```yaml
- document_summarization:
    model_provider: openai
    model: gpt-4o-mini
    model_context_length: 8000
    output_queue:
      name: "documents_summarized"
```
