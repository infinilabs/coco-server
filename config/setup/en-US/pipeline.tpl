# pipeline
# Note: PipelineConfigV2 is defined in the framework and has no explicit MustRegisterSchemaWithIndexName call,
# so its index name is derived from the lowercased Go type name without a version suffix (no -v2).
# Field mapping is handled by the coco-search index template via dynamic mapping.
PUT $[[SETUP_INDEX_PREFIX]]pipelineconfigv2
{}
