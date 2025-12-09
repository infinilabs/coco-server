import { Flex, Form, Input, InputNumber, Switch } from 'antd';
import React from 'react';
import { useTranslation } from 'react-i18next';

import { FieldMapping } from '../modules/FieldMapping';
import IncrementalSyncFields from '../modules/IncrementalSyncFields';

const { TextArea } = Input;

const FIELD_WIDTH_CLASS = 'max-w-660px';

interface MilvusFormProps {
  readonly form?: any;
}

const MilvusForm: React.FC<MilvusFormProps> = ({ form }) => {
  const { t } = useTranslation();

  const propertyTypeOptions = [
    { label: t('page.datasource.milvus.labels.property_type_datetime', 'Datetime'), value: 'datetime' },
    { label: t('page.datasource.milvus.labels.property_type_string', 'String'), value: 'string' },
    { label: t('page.datasource.milvus.labels.property_type_int', 'Integer'), value: 'int' },
    { label: t('page.datasource.milvus.labels.property_type_float', 'Float'), value: 'float' }
  ];

  return (
    <>
      {/* Connection Section */}
      <Form.Item
        label={t('page.datasource.milvus.labels.address', 'Address')}
        name={['config', 'address']}
        rules={[
          {
            message: t('page.datasource.milvus.error.address_required', 'Please input Milvus service address!'),
            required: true
          }
        ]}
        tooltip={t(
          'page.datasource.milvus.tooltip.address',
          'Milvus service address, e.g., localhost:19530'
        )}
      >
        <Input
          className={FIELD_WIDTH_CLASS}
          placeholder="localhost:19530"
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.milvus.labels.username', 'Username')}
        name={['config', 'username']}
        tooltip={t(
          'page.datasource.milvus.tooltip.username',
          'Optional username for Milvus authentication'
        )}
      >
        <Input
          className={FIELD_WIDTH_CLASS}
          placeholder={t('page.datasource.milvus.labels.username_placeholder', 'Optional username')}
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.milvus.labels.password', 'Password')}
        name={['config', 'password']}
        tooltip={t(
          'page.datasource.milvus.tooltip.password',
          'Optional password for Milvus authentication'
        )}
      >
        <Input.Password
          className={FIELD_WIDTH_CLASS}
          placeholder={t('page.datasource.milvus.labels.password_placeholder', 'Optional password')}
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.milvus.labels.db_name', 'Database Name')}
        name={['config', 'db_name']}
        tooltip={t(
          'page.datasource.milvus.tooltip.db_name',
          'Optional database name (Milvus 2.2.0+). Leave empty for default database.'
        )}
      >
        <Input
          className={FIELD_WIDTH_CLASS}
          placeholder={t('page.datasource.milvus.labels.db_name_placeholder', 'Optional database name')}
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.milvus.labels.collection', 'Collection')}
        name={['config', 'collection']}
        rules={[
          {
            message: t('page.datasource.milvus.error.collection_required', 'Please input collection name!'),
            required: true
          }
        ]}
        tooltip={t(
          'page.datasource.milvus.tooltip.collection',
          'Name of the Milvus collection to query'
        )}
      >
        <Input
          className={FIELD_WIDTH_CLASS}
          placeholder="my_collection"
        />
      </Form.Item>

      {/* Query Section */}
      <Form.Item
        label={t('page.datasource.milvus.labels.output_fields', 'Output Fields')}
        name={['config', 'output_fields']}
        tooltip={t(
          'page.datasource.milvus.tooltip.output_fields',
          'Comma-separated list of fields to retrieve. Leave empty to retrieve all scalar fields.'
        )}
      >
        <Input
          className={FIELD_WIDTH_CLASS}
          placeholder="field1, field2, field3"
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.milvus.labels.filter', 'Filter Expression')}
        name={['config', 'filter']}
        tooltip={t(
          'page.datasource.milvus.tooltip.filter',
          'Optional scalar filtering expression, e.g., age > 10 and name like "abc%"'
        )}
      >
        <TextArea
          className={FIELD_WIDTH_CLASS}
          placeholder='age > 10 and status == "active"'
          rows={3}
        />
      </Form.Item>

      {/* Pagination Section */}
      <Form.Item
        initialValue={1000}
        label={t('page.datasource.milvus.labels.page_size', 'Page Size')}
        name={['config', 'page_size']}
        rules={[
          {
            message: t('page.datasource.milvus.error.page_size_required', 'Please input page size!'),
            required: true
          }
        ]}
        tooltip={t(
          'page.datasource.milvus.tooltip.page_size',
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

      {/* Incremental Sync Section */}
      <IncrementalSyncFields
        connectorType='milvus'
        form={form}
        propertyTypeOptions={propertyTypeOptions}
        switchLayout='flex'
        placeholders={{
          property: 'updated_at',
          resumeFrom: propertyType =>
            propertyType === 'datetime' ? '2025-01-01T00:00:00Z' : '1000',
          tieBreaker: 'id'
        }}
      />

      {/* Field Mapping Section */}
      <Form.Item label={t('page.datasource.milvus.labels.data_processing', 'Data Processing')}>
        <Flex
          align="center"
          gap="small"
        >
          <span>{t('page.datasource.milvus.labels.field_mapping', 'Field Mapping')}</span>
          <Form.Item
            initialValue={false}
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

export default MilvusForm;
