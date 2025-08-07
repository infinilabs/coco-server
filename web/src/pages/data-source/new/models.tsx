export const NetworkDriveConfig = (values: any): object => {
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
