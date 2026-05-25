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
