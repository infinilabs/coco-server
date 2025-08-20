import { Form, Input, InputNumber, Switch } from 'antd';
import React from 'react';
import { useTranslation } from 'react-i18next';

import { FieldMapping } from '../modules/FieldMapping';

// eslint-disable-next-line react/display-name,react-refresh/only-export-components
export default ({ dbType }: { readonly dbType: string }) => {
  const { t } = useTranslation();

  // eslint-disable-next-line @typescript-eslint/no-shadow
  const getDbInfo = (dbType: string) => {
    switch (dbType) {
      case 'mysql':
        return {
          placeholder: 'mysql://user:password@tcp(localhost:3306)/database',
          tooltip: t(
            'page.datasource.rdbms.tooltip.connection_uri.mysql',
            'MySQL connection string. e.g., mysql://user:password@tcp(localhost:3306)/database'
          )
        };
      case 'postgresql':
      default:
        return {
          placeholder: 'postgresql://user:password@localhost:5432/database?sslmode=disable',
          tooltip: t(
            'page.datasource.rdbms.tooltip.connection_uri.postgresql',
            'PostgreSQL connection string. e.g., postgresql://user:password@localhost:5432/database?sslmode=disable'
          )
        };
    }
  };

  const dbInfo = getDbInfo(dbType);

  return (
    <>
      <Form.Item
        label={t('page.datasource.rdbms.labels.connection_uri', 'Connection URI')}
        name={['config', 'connection_uri']}
        tooltip={dbInfo.tooltip}
        rules={[
          {
            message: t('page.datasource.rdbms.error.connection_uri_required', 'Please input Connection URI!'),
            required: true
          }
        ]}
      >
        <Input
          placeholder={dbInfo.placeholder}
          style={{ width: 500 }}
        />
      </Form.Item>
      <Form.Item
        label={t('page.datasource.rdbms.labels.sql', 'SQL Query')}
        name={['config', 'sql']}
        rules={[{ message: t('page.datasource.rdbms.error.sql_required', 'Please input SQL query!'), required: true }]}
        tooltip={t('page.datasource.rdbms.tooltip.sql', 'The SQL query to execute for fetching data.')}
      >
        <Input.TextArea
          placeholder="SELECT id, title, content, updated_at FROM articles"
          rows={4}
          style={{ width: 500 }}
        />
      </Form.Item>
      <Form.Item
        label={t('page.datasource.rdbms.labels.last_modified_field', 'Last Modified Field')}
        name={['config', 'last_modified_field']}
        tooltip={t(
          'page.datasource.rdbms.tooltip.last_modified_field',
          'For incremental sync, specify a field that tracks last modification time (e.g., updated). The field type should be a timestamp or datetime.'
        )}
      >
        <Input
          placeholder="updated"
          style={{ width: 500 }}
        />
      </Form.Item>
      <Form.Item
        label={t('page.datasource.rdbms.labels.pagination', 'Enable Pagination')}
        name={['config', 'pagination']}
        valuePropName="checked"
        tooltip={t(
          'page.datasource.rdbms.tooltip.pagination',
          'Enable if the database query should be paginated. This is recommended for large tables.'
        )}
      >
        <Switch />
      </Form.Item>
      <Form.Item
        noStyle
        shouldUpdate={(prevValues, currentValues) => prevValues.config?.pagination !== currentValues.config?.pagination}
      >
        {({ getFieldValue }) =>
          getFieldValue(['config', 'pagination']) ? (
            <Form.Item
              initialValue={500}
              label={t('page.datasource.rdbms.labels.page_size', 'Page Size')}
              name={['config', 'page_size']}
              tooltip={t('page.datasource.rdbms.tooltip.page_size', 'The number of records to fetch per page.')}
              rules={[
                {
                  message: t('page.datasource.rdbms.error.page_size_required', 'Please input page size!'),
                  required: true
                }
              ]}
            >
              <InputNumber
                min={1}
                style={{ width: 500 }}
              />
            </Form.Item>
          ) : null
        }
      </Form.Item>
      <Form.Item
        label={t('page.datasource.rdbms.labels.field_mapping', 'Field Mapping')}
        name={['config', 'field_mapping', 'enabled']}
        valuePropName="checked"
      >
        <Switch />
      </Form.Item>
      <Form.Item
        noStyle
        shouldUpdate={(prevValues, currentValues) =>
          prevValues.config?.field_mapping?.enabled !== currentValues.config?.field_mapping?.enabled
        }
      >
        {({ getFieldValue }) =>
          getFieldValue(['config', 'field_mapping', 'enabled']) ? (
            <Form.Item
              colon={false}
              label=" "
            >
              <FieldMapping enabled={getFieldValue(['config', 'field_mapping', 'enabled'])} />
            </Form.Item>
          ) : null
        }
      </Form.Item>
    </>
  );
};
