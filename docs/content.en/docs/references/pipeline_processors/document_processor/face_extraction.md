---
title: "Face Extraction"
weight: 4
---

## Face Extraction Processor

Detects human faces in a document and attempts to identify each person using a
vision model. Results are stored in `doc.Metadata["faces"]`.

### Requirements/Dependencies

| Dependency | Required for |
|---|---|
| Pigo `facefinder` cascade binary | Face detection (set via `pigo_facefinder_path`) |
| [Apache Tika](https://tika.apache.org/) server | Extracting embedded images from PDF, PPTX, DOCX |

### Supported formats

| Supported format | Notes |
|---|---|
| Image | Faces detected directly |
| PDF, PPTX, DOCX | Faces detected in embedded images |

> **Depends on** `text_extraction` — this processor must run after
> `text_extraction` so that `doc.Chunks` is available for building context
> for the vision model prompt.

> **Note:** Face detection requires `pigo_facefinder_path` to be set. If
> omitted, the processor exits silently without producing any output.

### Configuration

| Parameter | Type | Required | Default | Description |
|---|---|---|---|---|
| `message_field` | string | No | `messages` | Pipeline context key containing the `[]queue.Message` to process |
| `output_queue` | object | No | `null` | Queue to push processed documents to |
| `tika_endpoint` | string | No | `http://127.0.0.1:9998` | Apache Tika server URL |
| `tika_timeout_in_seconds` | int | No | `120` | Per-file Tika timeout |
| `pigo_facefinder_path` | string | No | — | Path to the Pigo `facefinder` cascade binary |
| `vision_model_provider` | string | No | — | Provider ID for the vision model used for face recognition |
| `vision_model` | string | No | — | Model name for face recognition |
| `image_content_format` | string | No | `data_uri` | How face crop images are sent to the vision model: `data_uri` or `binary` |
| `llm_generation_lang` | string | No | *(app default)* | BCP 47 language tag for LLM-generated content (e.g. `en-US`, `zh-CN`) |

### Example

```yaml
- face_extraction:
    pigo_facefinder_path: /usr/local/share/pigo/facefinder
    vision_model_provider: openai
    vision_model: gpt-4o
    output_queue:
      name: "documents_with_faces"
```
