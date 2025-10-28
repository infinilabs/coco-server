import { Flex, Form, Input, InputNumber, Switch } from 'antd';
import React from 'react';
import { useTranslation } from 'react-i18next';

import { FieldMapping } from '../modules/FieldMapping';
import IncrementalSyncFields from '../modules/IncrementalSyncFields';

const { TextArea } = Input;

const FIELD_WIDTH_CLASS = 'max-w-660px';

interface MongoDBFormProps {
  readonly form?: any;
}

const validateConnectionUri = (value: string) => {
  if (!value) {
    return true;
  }
  const normalized = value.trim().toLowerCase();
  const allowedSchemes = ['mongodb://', 'mongodb+srv://'];
  return allowedSchemes.some(scheme => normalized.startsWith(scheme));
};

const validateJSON = (value: string) => {
  if (!value || value.trim() === '') {
    return true;
  }
  try {
    JSON.parse(value);
    return true;
    // eslint-disable-next-line @typescript-eslint/no-unused-vars
  } catch (e) {
    return false;
  }
};

const MongoDBForm: React.FC<MongoDBFormProps> = ({ form }) => {
  const { t } = useTranslation();

  const propertyTypeOptions = [
    { label: t('page.datasource.mongodb.labels.property_type_datetime', 'Datetime'), value: 'datetime' },
    { label: t('page.datasource.mongodb.labels.property_type_string', 'String'), value: 'string' },
    { label: t('page.datasource.mongodb.labels.property_type_int', 'Integer'), value: 'int' }
  ];

  return (
    <>
      {/* Connection Section */}
      <Form.Item
        label={t('page.datasource.mongodb.labels.connection_uri', 'Connection URI')}
        name={['config', 'connection_uri']}
        rules={[
          {
            message: t('page.datasource.mongodb.error.connection_uri_required', 'Please input connection URI!'),
            required: true
          },
          {
            validator: (_rule, value) =>
              validateConnectionUri(value)
                ? Promise.resolve()
                : Promise.reject(
                    new Error(
                      t(
                        'page.datasource.mongodb.error.connection_uri_invalid',
                        'Invalid MongoDB URI. Must start with mongodb:// or mongodb+srv://'
                      )
                    )
                  )
          }
        ]}
        tooltip={t(
          'page.datasource.mongodb.tooltip.connection_uri',
          'MongoDB connection string, e.g., mongodb://localhost:27017 or mongodb+srv://cluster.mongodb.net'
        )}
      >
        <Input
          className={FIELD_WIDTH_CLASS}
          placeholder="mongodb://localhost:27017"
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.mongodb.labels.database', 'Database')}
        name={['config', 'database']}
        rules={[
          {
            message: t('page.datasource.mongodb.error.database_required', 'Please input database name!'),
            required: true
          }
        ]}
      >
        <Input
          className={FIELD_WIDTH_CLASS}
          placeholder="mydb"
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.mongodb.labels.collection', 'Collection')}
        name={['config', 'collection']}
        rules={[
          {
            message: t('page.datasource.mongodb.error.collection_required', 'Please input collection name!'),
            required: true
          }
        ]}
      >
        <Input
          className={FIELD_WIDTH_CLASS}
          placeholder="documents"
        />
      </Form.Item>

      {/* Query Section */}
      <Form.Item
        label={t('page.datasource.mongodb.labels.query', 'Query Filter')}
        name={['config', 'query']}
        rules={[
          {
            validator: (_rule, value) =>
              validateJSON(value)
                ? Promise.resolve()
                : Promise.reject(
                    new Error(t('page.datasource.mongodb.error.query_invalid', 'Invalid JSON format for query'))
                  )
          }
        ]}
        tooltip={t(
          'page.datasource.mongodb.tooltip.query',
          'Optional BSON query filter in JSON format, e.g., {"status": "published"}'
        )}
      >
        <TextArea
          className={FIELD_WIDTH_CLASS}
          placeholder='{"status": "active"}'
          rows={3}
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.mongodb.labels.sort', 'Sort Specification')}
        name={['config', 'sort']}
        rules={[
          {
            validator: (_rule, value) =>
              validateJSON(value)
                ? Promise.resolve()
                : Promise.reject(
                    new Error(t('page.datasource.mongodb.error.sort_invalid', 'Invalid JSON format for sort'))
                  )
          }
        ]}
        tooltip={t(
          'page.datasource.mongodb.tooltip.sort',
          'Optional sort specification in JSON format, e.g., {"updated_at": 1, "_id": 1}'
        )}
      >
        <TextArea
          className={FIELD_WIDTH_CLASS}
          placeholder='{"updated_at": 1, "_id": 1}'
          rows={2}
        />
      </Form.Item>

      {/* Pagination Section */}
      <Form.Item
        label={t('page.datasource.mongodb.labels.pagination', 'Enable Pagination')}
        name={['config', 'pagination']}
        tooltip={t('page.datasource.mongodb.tooltip.pagination', 'Enable pagination for large collections')}
        valuePropName="checked"
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
              label={t('page.datasource.mongodb.labels.page_size', 'Page Size')}
              name={['config', 'page_size']}
              rules={[
                {
                  message: t('page.datasource.mongodb.error.page_size_required', 'Please input page size!'),
                  required: true
                }
              ]}
              tooltip={t(
                'page.datasource.mongodb.tooltip.page_size',
                'Number of documents to fetch per page (1-10000)'
              )}
            >
              <InputNumber
                className={FIELD_WIDTH_CLASS}
                max={10000}
                min={1}
                style={{ width: '100%' }}
              />
            </Form.Item>
          ) : null
        }
      </Form.Item>

      {/* Incremental Sync Section */}
      <IncrementalSyncFields
        connectorType="mongodb"
        form={form}
        propertyTypeOptions={propertyTypeOptions}
        switchLayout="flex"
        placeholders={{
          property: 'updated_at',
          resumeFrom: propertyType =>
            propertyType === 'datetime' ? '2025-01-01T00:00:00Z' : '507f1f77bcf86cd799439011',
          tieBreaker: '_id'
        }}
      />

      {/* Field Mapping Section */}
      <Form.Item label={t('page.datasource.mongodb.labels.data_processing', 'Data Processing')}>
        <Flex
          align="center"
          gap="small"
        >
          <span>{t('page.datasource.mongodb.labels.field_mapping', 'Field Mapping')}</span>
          <Form.Item
            noStyle
            name={['config', 'field_mapping', 'enabled']}
            valuePropName="checked"
          >
            <Switch />
          </Form.Item>
        </Flex>
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

export default MongoDBForm;
