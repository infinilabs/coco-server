# pipeline
# 注意：PipelineConfigV2 定义在 framework 中，没有显式调用 MustRegisterSchemaWithIndexName，
# 因此索引名直接由 Go 类型名小写推导而来，不需要 -v2 后缀。
# 字段 mapping 由 coco-search index template 通过动态 mapping 自动填充。
PUT $[[SETUP_INDEX_PREFIX]]pipelineconfigv2
{}
