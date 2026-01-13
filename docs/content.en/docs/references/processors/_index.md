---
title: "Processors"
weight: 100
bookCollapseSection: true
---

# Processors

Processors transform and enrich documents in the pipeline. Each processor 
can be configured independently and chained together to create sophisticated
document processing workflows.

## Common Configuration

All processors support the following common configuration options:

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `message_field` | string | `documents` | The field in the pipeline context containing the documents to process |
| `output_queue` | object | `null` | Optional queue configuration for sending processed documents to a output queue |

---

## Embedding Processor

Generates vector embeddings for document chunks using AI models. 

This processor enables semantic search and retrieval by converting text chunks 
into dense vector representations.


### Configuration

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
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

---

## Extract Tags Processor

Extracts structured tags from AI insights stored in document metadata using an LLM.

### Configuration

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
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

---

## File Extraction Processor

Comprehensive file processing for various file types. Extracts text content, metadata, generates thumbnails, and performs face detection.

### Configuration

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `tika_endpoint` | string | No | `http://127.0.0.1:9998` | Apache Tika server URL for content extraction |
| `tika_timeout_in_seconds` | int | No | 120 | Tika processing timeout for each file |
| `vision_model_provider` | string | Yes | - | AI provider for image analysis |
| `vision_model` | string | Yes | - | Model name for image analysis |
| `pigo_facefinder_path` | string | Yes | - | Path to Pigo face detection binary |
| `chunk_size` | int | Yes | - | Text chunking size for extracted content |
| `image_content_format` | string | No | "data_uri" | Could be "data_uri" or "binary". The format that an image will be encoded in in order to be sent to a vision model |

### Example

```yaml
- file_extraction:
  tika_endpoint: http://127.0.0.1:9998
  tika_timeout_in_seconds: 120
  chunk_size: 7000
  vision_model_provider: openai
  vision_model: gpt-4o
  pigo_facefinder_path: /path/to/pigo/facefinder
```

---

## File Type Detection Processor

Detects file types based on file extensions and sets appropriate metadata.

### Configuration

No additional configuration required.

### Example

```yaml
- file_type_detection: {}
```

---

## Summary Processor

Generates AI-powered document summaries and insights with structured analysis.

### Configuration

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
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
