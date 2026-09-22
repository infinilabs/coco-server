---
title: "Extract Entities"
weight: 8
---

## Extract Entities Processor

Builds the ontology layer automatically: extracts entities and typed
relations from a document using a language model, disambiguates them against
existing wiki entities, proposes the unknown ones, and links the document to
the resolved entities.

Extracted entities are always created with `proposed` status — publishing
requires human review (entity pages go through the wiki article workflow).

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
| `entity_types` | list | No | *(built-in vocabulary)* | Allowed entity types (ontology vocabulary) |
| `max_entities` | int | No | `15` | Hard cap on extracted entities per document |
| `wiki_kb_id` | string | No | — | When set, newly-proposed entities get a draft wiki page in this knowledge base |
| `llm_generation_lang` | string | No | *(app default)* | BCP 47 language tag for generated content (e.g. `en-US`, `zh-CN`) |

### Example

```yaml
- extract_entities:
    model_provider: openai
    model: gpt-4o-mini
    entity_types: ["store", "campaign", "contract", "supplier", "person", "product"]
    max_entities: 20
    wiki_kb_id: "my-kb-id"
    output_queue:
      name: "documents_enriched"
```

### What it writes

- `entity_ids` on the document: resolved entity ids, usable as a search filter
- new `proposed` wiki entities (with aliases, properties, provenance) for
  unmatched extractions
- typed relation edges between co-occurring entities, provenance set to the
  source document id
- draft wiki article stubs when `wiki_kb_id` is configured
