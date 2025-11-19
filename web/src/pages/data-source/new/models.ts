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

export const GitlabConfig = (values: any) => {
  return {
    base_url: values.config?.base_url || '',
    index_issues: values.config?.index_issues,
    index_merge_requests: values.config?.index_merge_requests,
    index_snippets: values.config?.index_snippets,
    index_wikis: values.config?.index_wikis,
    owner: values.config?.owner || '',
    repos: (values.config?.repos || []).filter(Boolean),
    token: values.config?.token || ''
  };
};

export const GiteaConfig = (values: any) => {
  return {
    base_url: values.config?.base_url || '',
    index_issues: values.config?.index_issues,
    index_pull_requests: values.config?.index_pull_requests,
    owner: values.config?.owner || '',
    repos: (values.config?.repos || []).filter(Boolean),
    token: values.config?.token || ''
  };
};
const defaultFieldMapping = () => ({
  enabled: false,
  mapping: {}
});

const parseParameterValue = (value: unknown) => {
  if (typeof value !== 'string') {
    return value;
  }
  const trimmed = value.trim();
  if (trimmed === '') {
    return '';
  }
  if (trimmed === 'true' || trimmed === 'false') {
    return trimmed === 'true';
  }
  if (trimmed === 'null') {
    return null;
  }
  if (!Number.isNaN(Number(trimmed)) && /^-?\d+(?:\.\d+)?$/.test(trimmed)) {
    return Number(trimmed);
  }
  if ((trimmed.startsWith('{') && trimmed.endsWith('}')) || (trimmed.startsWith('[') && trimmed.endsWith(']'))) {
    try {
      return JSON.parse(trimmed);
    } catch (_err) {
      return value;
    }
  }
  return value;
};

const stringifyParameterValue = (value: unknown) => {
  if (value === undefined || value === null) {
    return '';
  }
  if (typeof value === 'object') {
    try {
      return JSON.stringify(value);
    } catch (_err) {
      return '';
    }
  }
  return String(value);
};

// eslint-disable-next-line complexity
export const Neo4jConfig = (values: any) => {
  const incremental = values.config?.incremental || {};
  const parametersList = Array.isArray(values.config?.parameters) ? values.config?.parameters : [];

  const parameters = parametersList.reduce((acc: Record<string, unknown>, current: any) => {
    const key = (current?.key || '').trim();
    if (!key) {
      return acc;
    }
    acc[key] = parseParameterValue(current?.value);
    return acc;
  }, {} as Record<string, unknown>);

  const incrementalConfig = incremental?.enabled
    ? {
        enabled: true,
        mode: incremental.mode || 'property_watermark',
        property: incremental.property || '',
        property_type: incremental.property_type || 'datetime',
        tie_breaker: incremental.tie_breaker || '',
        resume_from: incremental.resume_from || ''
      }
    : {
        enabled: false,
        mode: '',
        property: '',
        property_type: 'datetime',
        tie_breaker: '',
        resume_from: ''
      };

  return {
    auth_token: values.config?.auth_token || '',
    connection_uri: values.config?.connection_uri || '',
    cypher: values.config?.cypher || '',
    database: values.config?.database || '',
    field_mapping: values.config?.field_mapping || defaultFieldMapping(),
    incremental: incrementalConfig,
    page_size: values.config?.page_size || 500,
    pagination: Boolean(values.config?.pagination),
    parameters,
    password: values.config?.password || '',
    username: values.config?.username || ''
  };
};

export const Neo4jFormConfig = (values: any) => {
  const config = values.config || {};
  const incremental = config.incremental || {};
  const parameterEntries = Array.isArray(config.parameters)
    ? config.parameters
    : Object.entries(config.parameters || {}).map(([key, value]) => ({
        key,
        value: stringifyParameterValue(value)
      }));

  return {
    auth_token: config.auth_token || '',
    connection_uri: config.connection_uri || '',
    cypher: config.cypher || '',
    database: config.database || '',
    field_mapping: config.field_mapping || defaultFieldMapping(),
    incremental: {
      enabled: Boolean(incremental.enabled),
      mode: incremental.mode || 'property_watermark',
      property: incremental.property || '',
      property_type: incremental.property_type || 'datetime',
      tie_breaker: incremental.tie_breaker || '',
      resume_from: incremental.resume_from || ''
    },
    page_size: config.page_size || 500,
    pagination: Boolean(config.pagination),
    parameters: parameterEntries,
    password: config.password || '',
    username: config.username || ''
  };
};

// eslint-disable-next-line complexity
export const MongoDBConfig = (values: any) => {
  const config: any = {
    connection_uri: values.config?.connection_uri || '',
    database: values.config?.database || '',
    collection: values.config?.collection || ''
  };

  if (values.config?.query) {
    config.query = values.config.query;
  }

  if (values.config?.sort) {
    config.sort = values.config.sort;
  }

  // Pagination config - always include
  config.pagination = Boolean(values.config?.pagination);
  if (config.pagination) {
    config.page_size = values.config?.page_size || 500;
  }

  // New nested incremental structure
  const incremental = values.config?.incremental || {};
  config.incremental = incremental.enabled
    ? {
        enabled: true,
        mode: 'property_watermark',
        property: incremental.property || '',
        property_type: incremental.property_type || 'datetime',
        tie_breaker: incremental.tie_breaker || '',
        resume_from: incremental.resume_from || ''
      }
    : {
        enabled: false,
        mode: '',
        property: '',
        property_type: 'datetime',
        tie_breaker: '',
        resume_from: ''
      };

  // Field mapping config - always include
  if (values.config?.field_mapping?.enabled) {
    config.field_mapping = values.config.field_mapping;
  } else {
    config.field_mapping = defaultFieldMapping();
  }

  return config;
};

export const MongoDBFormConfig = (values: any) => {
  const config = values.config || {};

  return {
    connection_uri: config.connection_uri || '',
    database: config.database || '',
    collection: config.collection || '',
    query: config.query || '',
    sort: config.sort || '',
    pagination: Boolean(config.pagination),
    page_size: config.page_size || 500,
    incremental: {
      enabled: Boolean(config.incremental?.enabled),
      property: config.incremental?.property || '',
      property_type: config.incremental?.property_type || 'datetime',
      tie_breaker: config.incremental?.tie_breaker || '',
      resume_from: config.incremental?.resume_from || ''
    },
    field_mapping: config.field_mapping || defaultFieldMapping()
  };
};
