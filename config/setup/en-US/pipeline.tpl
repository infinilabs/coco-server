# pipeline
# Note: PipelineConfigV2 is registered by the framework's pipeline module under the ORM index
# name "pipeline-configs" (no schema suffix). The ORM schema registration creates the index and
# applies its mappings at startup, so this template only seeds the default pipeline documents.
POST $[[SETUP_INDEX_PREFIX]]pipeline-configs/$[[SETUP_DOC_TYPE]]/enrich_documents
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
  "name": "Enrich Attachments",
  "enabled": true,
  "processor": [
      {
        "attachment_text_extraction": {
        }
      }
  ]
}
