export const createExtensionValidator = (t: (k: string, d: string) => string) => async (_: any, value: string) => {
  if (!value) {
    // field is optional
    return Promise.resolve();
  }

  const extensions = value.split(',').map(ext => ext.trim());
  const validExtensionRegex = /^\.?[a-zA-Z0-9]+$/;

  for (const ext of extensions) {
    // check nonempty parts
    if (ext) {
      if (!validExtensionRegex.test(ext)) {
        return Promise.reject(
          new Error(
            t(
              'page.datasource.commons.error.extensions_format',
              "Invalid file extensions. Use formats like 'pdf' or '.pdf', with only letters and numbers."
            )
          )
        );
      }
    }
  }

  return Promise.resolve();
};
