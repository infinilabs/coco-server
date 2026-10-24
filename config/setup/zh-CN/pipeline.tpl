# pipeline
# 注意：PipelineConfigV2 由 framework 的 pipeline 模块注册到 ORM 索引名 "pipeline-configs"（无 -v2 后缀）。
# ORM 的 schema 注册会在启动时创建索引并写入 mapping，因此本模板只负责写入默认的 pipeline 文档。
POST $[[SETUP_INDEX_PREFIX]]pipeline-configs/$[[SETUP_DOC_TYPE]]/enrich_documents
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
        "file_metadata": {}
      },
      {
        "generate_document_cover": {}
      },
      {
        "document_text_attachment_extraction": {
          "chunk_size": 7000,
          "extract_attachments": true
        }
      },
      {
        "face_extraction": {
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
        }
      },
      {
        "extract_entities": {
          "model_context_length": 128000
        }
      }
  ]
}

POST $[[SETUP_INDEX_PREFIX]]pipeline-configs/$[[SETUP_DOC_TYPE]]/enrich_attachments
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
        }
      }
  ]
}
