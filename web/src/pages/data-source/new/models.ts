export const NetworkDriveConfig = (values: any) => {
  return {
    domain: values.config?.domain || '',
    endpoint: values.config?.endpoint || '',
    folder_paths: values.config?.folder_paths || [],
    password: values.config?.password || '',
    share: values.config?.share || '',
    username: values.config?.username || ''
  };
};

export const RdbmsConfig = (values: any) => {
  return {
    connection_uri: values.config?.connection_uri || '',
    field_mapping: values.config?.field_mapping || {
      enabled: false,
      mapping: {}
    },
    last_modified_field: values.config?.last_modified_field || '',
    page_size: values.config?.page_size || 500,
    pagination: values.config?.pagination || false,
    sql: values.config?.sql || ''
  };
};

export const MongoDBConfig = (values: any) => {
  // 首先获取RdbmsConfig的基础配置，确保兼容性
  const baseConfig = RdbmsConfig(values);
  
  // 然后添加MongoDB特有的配置参数
  return {
    ...baseConfig, // 包含RdbmsConfig的所有基础参数
    // MongoDB特有的连接参数
    database: values.config?.database || '',
    auth_database: values.config?.auth_database || 'admin',
    cluster_type: values.config?.cluster_type || 'standalone',
    collections: values.config?.collections || [],
    // MongoDB特有的性能优化参数
    batch_size: values.config?.batch_size || 1000,
    max_pool_size: values.config?.max_pool_size || 10,
    timeout: values.config?.timeout || '30s',
    sync_strategy: values.config?.sync_strategy || 'full',
    // MongoDB特有的查询优化参数
    enable_projection: values.config?.enable_projection !== false,
    enable_index_hint: values.config?.enable_index_hint !== false
  };
};
