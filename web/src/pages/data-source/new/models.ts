export const NetworkDriveConfig = (values: any) => {
  const extensions: Array<string> = values.config?.extensions_str
    ? values.config.extensions_str
        .split(',')
        .map((s: string) => s.trim())
        .filter(Boolean)
    : [];
  return {
    domain: values.config?.domain || '',
    endpoint: values.config?.endpoint || '',
    extensions,
    password: values.config?.password || '',
    paths: (values.config?.paths || []).filter(Boolean),
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

export const GithubConfig = (values: any) => {
  return {
    index_issues: values.config?.index_issues,
    index_pull_requests: values.config?.index_pull_requests,
    owner: values.config?.owner || '',
    repos: (values.config?.repos || []).filter(Boolean),
    token: values.config?.token || ''
  };
};
