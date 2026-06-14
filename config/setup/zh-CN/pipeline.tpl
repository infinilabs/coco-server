# pipeline
# 注意：PipelineConfigV2 定义在 framework 中，没有显式调用 MustRegisterSchemaWithIndexName，
# 因此索引名直接由 Go 类型名小写推导而来，不需要 -v2 后缀。
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
  "name": "文档增强",
  "enabled": true,
  "processor": [
      {
        "file_type_detection": {}
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
  "name": "附件增强",
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
