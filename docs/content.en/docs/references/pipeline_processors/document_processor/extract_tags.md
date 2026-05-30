---
title: "Extract Tags"
weight: 7
---

## Extract Tags Processor

Extracts structured keyword tags from a document's content using a language
model. Tags are stored in `doc.Tags`.

Documents that have not yet been summarized (no `AiInsights`) are skipped.

### Requirements

A configured language model provider is required. Set `model_provider` and
`model` in the processor config, or configure a default language model in the
application settings.

### Configuration

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `message_field` | string | No | `messages` | Pipeline context key containing the `[]queue.Message` to process |
| `output_queue` | object | No | `null` | Queue to push processed documents to |
| `model_provider` | string | No | *(app default)* | Language model provider ID |
| `model` | string | No | *(app default)* | Language model name |
| `model_context_length` | int | No | — | Model context window size in tokens (minimum 4000) |
| `llm_generation_lang` | string | No | *(app default)* | BCP 47 language tag for generated content (e.g. `en-US`, `zh-CN`) |

### Example

```yaml
- extract_tags:
    model_provider: openai
    model: gpt-4o-mini
    output_queue:
      name: "documents_tagged"
```
