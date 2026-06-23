---
title: "Document Embedding"
weight: 6
---

## Document Embedding Processor

Generates vector embeddings for a document's text chunks using an embedding
model to enable semantic search and retrieval.

### Requirements

A configured embedding model provider is required. Set `model_provider` and
`model` in the processor config, or configure a default embedding model in the
application settings.

The processor always produces vectors with a fixed dimension of `1024`, which
is the only dimension supported by Coco semantic search. For OpenAI-compatible
providers the processor requests `1024` dimensions from the API automatically.
For other providers, such as Ollama, the configured model itself must output
`1024`-dimensional vectors; otherwise the document is passed through without
embeddings and its chunks will not appear in semantic search results.

### Configuration

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `message_field` | string | No | `messages` | Pipeline context key for the input messages |
| `output_queue` | object | No | `null` | Queue to push processed documents to |
| `model_provider` | string | No | *(app default)* | Embedding model provider ID |
| `model` | string | No | *(app default)* | Embedding model name |

### Example

```yaml
- document_embedding:
    model_provider: openai
    model: text-embedding-3-small
    output_queue:
      name: "documents_embedded"
```
