import { Form, Input } from 'antd';
import { useTranslation } from 'react-i18next';

import { MultiFilePathInput } from '@/components/datasource/type/file_paths';

const createPathValidator = () => async (_: any, value: string[]) => {
  // check is empty
  if (!value || value.length === 0 || value.every(item => !item)) {
    return Promise.reject(new Error('Please input folder path!'));
  }

  // check absolute path
  //   - Unix/Linux: /...
  //   - Windows: C:\... or C:/... or \\... (UNC)
  const absolutePathRegex = /^(?:\/|[a-zA-Z]:[\\/]|\\\\)/;

  for (const path of value) {
    // only check nonempty path
    if (path && !absolutePathRegex.test(path)) {
      return Promise.reject(new Error(`'${path}' is not a valid absolute path.`));
    }

    const hasForwardSlash = path.includes('/');
    const hasBackwardSlash = path.includes('\\');

    if (hasForwardSlash && hasBackwardSlash) {
      return Promise.reject(
        new Error(`'${path}' uses mixed path separators. Please use either '/' or '\\' consistently.`)
      );
    }
  }
  return Promise.resolve();
};

const createExtensionValidator = () => async (_, value: string) => {
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
            `File extension '${ext}' is  invalid. Use formats like 'pdf' or '.pdf', with only letters and numbers.`
          )
        );
      }
    }
  }

  return Promise.resolve();
};

// eslint-disable-next-line react/display-name,react-refresh/only-export-components
export default () => {
  const { t } = useTranslation();
  const pathValidator = createPathValidator();
  const extensionValidator= createExtensionValidator();
  return (
    <>
      <Form.Item
        label={t('page.datasource.new.labels.folder_paths', 'Folder Paths')}
        name={['config', 'paths']}
        rules={[{ validator: pathValidator }]}
        tooltip={t('page.datasource.new.tooltip.folder_paths', 'Absolute paths to the folders you want to scan.')}
      >
        <MultiFilePathInput
          addButtonText={t('page.datasource.file_paths_add', 'Add File Path')}
          placeholder="/path/to/your/folder"
        />
      </Form.Item>
      <Form.Item
        label={t('page.datasource.new.labels.file_extensions', 'File Extensions (optional)')}
        name={['config', 'extensions_str']}
        rules={[{ validator: extensionValidator }]}
        tooltip={t('page.datasource.new.tooltip.file_extensions', 'Comma-separated list. e.g., pdf, docx, txt')}
      >
        <Input
          placeholder="pdf, docx, md"
          style={{ width: 500 }}
        />
      </Form.Item>
    </>
  );
};
