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
