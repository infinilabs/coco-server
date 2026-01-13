---
title: "Embedding"
weight: 5
---

## Embedding Processor

Generates vector embeddings for document chunks using AI models. 

This processor enables semantic search and retrieval by converting text chunks 
into dense vector representations.


### Configuration

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `message_field` | string | `documents` | The field in the pipeline context containing the documents to process |
| `output_queue` | object | `null` | Optional queue configuration for sending processed documents to a output queue |
| `model_provider` | string | Yes | - | ID of the AI model provider for embeddings |
| `model` | string | Yes | - | Name of the embedding model (e.g., `text-embedding-3-small`) |
| `embedding_dimension` | int32 | Yes | - | Vector dimension (must match model's supported dimensions) |

### Example

```yaml
- document_embedding:
  model_provider: openai
  model: text-embedding-3-small
  embedding_dimension: 1536
```