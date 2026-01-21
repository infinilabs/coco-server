---
title: "Misc File Extraction"
weight: 2
---

## Misc File Extraction Processor

Comprehensive file processing for various file types. Extracts text content, metadata, generates thumbnails, and performs face detection.

### Configuration

| Parameter | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `message_field` | string | `documents` | The field in the pipeline context containing the documents to process |
| `output_queue` | object | `null` | Optional queue configuration for sending processed documents to a output queue |
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