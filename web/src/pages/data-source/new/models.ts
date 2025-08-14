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
