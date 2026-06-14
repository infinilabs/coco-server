# pipeline
# Note: PipelineConfigV2 is defined in the framework and has no explicit MustRegisterSchemaWithIndexName call,
# so its index name is derived from the lowercased Go type name without a version suffix (no -v2).
PUT $[[SETUP_INDEX_PREFIX]]pipelineconfigv2
{
  "mappings": {
    "properties": {
      "id":                { "type": "keyword" },
      "created":           { "type": "date" },
      "updated":           { "type": "date" },
      "_system":           { "type": "object" },
      "name":              { "type": "keyword" },
      "enabled":           { "type": "boolean" },
      "singleton":         { "type": "boolean" },
      "auto_start":        { "type": "boolean" },
      "keep_running":      { "type": "boolean" },
      "retry_delay_in_ms": { "type": "integer" },
      "max_running_in_ms": { "type": "long" },
      "logging": {
        "properties": {
          "enabled": { "type": "boolean" }
        }
      },
      "processor": { "enabled": false },
      "labels":    { "type": "object" },
      "transient": { "type": "boolean" }
    }
  }
}

POST $[[SETUP_INDEX_PREFIX]]pipelineconfigv2/$[[SETUP_DOC_TYPE]]/enrich_documents
{
  "_system": {
    "owner_id": "$[[SETUP_OWNER_ID]]"
  },
  "id": "enrich_documents",
  "name": "Enrich Documents",
  "enabled": true,
  "processor": [
      {
        "file_type_detection": {}
      },
      {
        "file_metadata": {}
      },
      {
        "generate_document_cover": {}
      },
      {
        "document_text_attachment_extraction": {
          "tika_endpoint": "http://127.0.0.1:9998",
          "tika_timeout_in_seconds": 360,
          "chunk_size": 7000,
          "extract_attachments": true
        }
      },
      {
        "face_extraction": {
          "tika_endpoint": "http://127.0.0.1:9998",
          "tika_timeout_in_seconds": 360,
          "pigo_facefinder_path": "./config/ai/facefinder"
        }
      },
      {
        "document_summarization": {
          "model_context_length": 128000,
          "ai_insights_max_length": 500
        }
      },
      {
        "extract_tags": {
          "model_context_length": 128000
        }
      },
      {
        "document_embedding": {
          "embedding_dimension": 1024
        }
      }
  ]
}

POST $[[SETUP_INDEX_PREFIX]]pipelineconfigv2/$[[SETUP_DOC_TYPE]]/enrich_attachments
{
  "_system": {
    "owner_id": "$[[SETUP_OWNER_ID]]"
  },
  "id": "enrich_attachments",
  "name": "Enrich Attachments",
  "enabled": true,
  "processor": [
      {
        "attachment_text_extraction": {
          "tika_endpoint": "http://127.0.0.1:9998",
          "tika_timeout_in_seconds": 360
        }
      }
  ]
}
