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

### Configuration

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `message_field` | string | No | `messages` | Pipeline context key for the input messages |
| `output_queue` | object | No | `null` | Queue to push processed documents to |
| `model_provider` | string | No | *(app default)* | Embedding model provider ID |
| `model` | string | No | *(app default)* | Embedding model name |
| `embedding_dimension` | int | **Yes** | — | Vector dimension; must match the model's output dimension |

### Example

```yaml
- document_embedding:
    model_provider: openai
    model: text-embedding-3-small
    embedding_dimension: 1536
    output_queue:
      name: "documents_embedded"
```
