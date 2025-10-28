import { MinusCircleOutlined, PlusCircleOutlined } from '@ant-design/icons';
import { Button, Flex, Form, Input, InputNumber, Space, Switch } from 'antd';
import React from 'react';
import { useTranslation } from 'react-i18next';

import { FieldMapping } from '../modules/FieldMapping';
import IncrementalSyncFields from '../modules/IncrementalSyncFields';

const { TextArea } = Input;

const FIELD_WIDTH_CLASS = 'max-w-660px';

interface Neo4jFormProps {
  readonly form: any;
}

const RFC3339_REGEX = /^(\d{4}-\d{2}-\d{2})T(\d{2}:\d{2}:\d{2})(?:\.\d+)?(?:Z|[+-]\d{2}:\d{2})$/;

const validateConnectionUri = (value: string) => {
  if (!value) {
    return true;
  }
  const normalized = value.trim().toLowerCase();
  const allowedSchemes = ['neo4j://', 'neo4j+s://', 'neo4j+ssc://', 'bolt://', 'bolt+s://', 'bolt+ssc://'];
  return allowedSchemes.some(scheme => normalized.startsWith(scheme));
};

const Neo4jForm: React.FC<Neo4jFormProps> = ({ form }) => {
  const { t } = useTranslation();

  const incrementalEnabled = Form.useWatch(['config', 'incremental', 'enabled'], form);
  const incrementalPropertyType = Form.useWatch(['config', 'incremental', 'property_type'], form);
  const normalizedPropertyType = (incrementalPropertyType || 'datetime')?.toLowerCase();
  const validateResumeFrom = (_rule: any, value: string) => {
    if (!incrementalEnabled || value === null || `${value}`.trim() === '') {
      return Promise.resolve();
    }
    const trimmed = `${value}`.trim();
    switch (normalizedPropertyType) {
      case 'datetime':
        return RFC3339_REGEX.test(trimmed)
          ? Promise.resolve()
          : Promise.reject(
              new Error(
                t(
                  'page.datasource.neo4j.error.incremental_resume_invalid',
                  'Please use RFC3339 timestamp, e.g., 2025-01-01T00:00:00Z.'
                )
              )
            );
      case 'int': {
        const intPattern = /^-?\d+$/;
        return intPattern.test(trimmed)
          ? Promise.resolve()
          : Promise.reject(
              new Error(
                t('page.datasource.neo4j.error.incremental_resume_invalid_int', 'Please enter a valid integer value.')
              )
            );
      }
      case 'float': {
        const numeric = Number(trimmed);
        return Number.isFinite(numeric)
          ? Promise.resolve()
          : Promise.reject(
              new Error(
                t('page.datasource.neo4j.error.incremental_resume_invalid_float', 'Please enter a valid number.')
              )
            );
      }
      default:
        return Promise.resolve();
    }
  };

  const propertyTypeOptions = [
    { label: t('page.datasource.neo4j.labels.property_type_datetime', 'Datetime'), value: 'datetime' },
    { label: t('page.datasource.neo4j.labels.property_type_string', 'String'), value: 'string' },
    { label: t('page.datasource.neo4j.labels.property_type_int', 'Integer'), value: 'int' },
    { label: t('page.datasource.neo4j.labels.property_type_float', 'Float'), value: 'float' }
  ];

  return (
    <>
      <Form.Item
        label={t('page.datasource.neo4j.labels.connection_uri', 'Connection URI')}
        name={['config', 'connection_uri']}
        rules={[
          {
            message: t('page.datasource.neo4j.error.connection_uri_required', 'Please input connection URI!'),
            required: true
          },
          {
            validator: (_rule, value) =>
              validateConnectionUri(value)
                ? Promise.resolve()
                : Promise.reject(
                    new Error(
                      t(
                        'page.datasource.neo4j.error.connection_uri_invalid',
                        'Invalid Neo4j URI format. Use bolt:// or neo4j:// host addresses.'
                      )
                    )
                  )
          }
        ]}
      >
        <Input
          className={FIELD_WIDTH_CLASS}
          placeholder="neo4j://localhost:7687"
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.neo4j.labels.database', 'Database')}
        name={['config', 'database']}
      >
        <Input
          className={FIELD_WIDTH_CLASS}
          placeholder={t('page.datasource.neo4j.labels.database_placeholder', 'Optional database name')}
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.neo4j.labels.username', 'Username')}
        name={['config', 'username']}
      >
        <Input
          className={FIELD_WIDTH_CLASS}
          placeholder="neo4j"
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.neo4j.labels.password', 'Password')}
        name={['config', 'password']}
      >
        <Input.Password
          className={FIELD_WIDTH_CLASS}
          placeholder={t('page.datasource.neo4j.labels.password_placeholder', 'Enter Neo4j password')}
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.neo4j.labels.auth_token', 'Auth Token')}
        name={['config', 'auth_token']}
        tooltip={t(
          'page.datasource.neo4j.tooltip.auth_token',
          'Optional bearer token. Overrides username and password if provided.'
        )}
      >
        <Input.Password
          className={FIELD_WIDTH_CLASS}
          placeholder={t('page.datasource.neo4j.labels.auth_token_placeholder', 'Optional Neo4j auth token')}
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.neo4j.labels.cypher', 'Cypher Query')}
        name={['config', 'cypher']}
        rules={[
          { message: t('page.datasource.neo4j.error.cypher_required', 'Please input Cypher query!'), required: true }
        ]}
      >
        <TextArea
          className={FIELD_WIDTH_CLASS}
          placeholder="MATCH (n) WHERE n.type = 'document' RETURN n"
          rows={4}
        />
      </Form.Item>

      <Form.Item
        label={t('page.datasource.neo4j.labels.parameters', 'Query Parameters')}
        tooltip={t(
          'page.datasource.neo4j.tooltip.parameters',
          'Optional key/value parameters passed to the Cypher query.'
        )}
      >
        <Form.List name={['config', 'parameters']}>
          {(fields, { add, remove }) => (
            <>
              {fields.map(({ key, name, ...restField }) => (
                <Space
                  wrap
                  align="baseline"
                  className={FIELD_WIDTH_CLASS}
                  key={key}
                  style={{ display: 'flex', marginBottom: 8, width: '100%' }}
                >
                  <Form.Item
                    {...restField}
                    name={[name, 'key']}
                    style={{ flex: 1, marginBottom: 0, minWidth: 0 }}
                    rules={[
                      {
                        message: t('page.datasource.neo4j.error.parameter_key_required', 'Please input parameter key!'),
                        required: true
                      }
                    ]}
                  >
                    <Input
                      className={FIELD_WIDTH_CLASS}
                      placeholder={t('page.datasource.neo4j.labels.parameter_key_placeholder', 'Parameter key')}
                      style={{ width: '100%' }}
                    />
                  </Form.Item>
                  <Form.Item
                    {...restField}
                    name={[name, 'value']}
                    style={{ flex: 1, marginBottom: 0, minWidth: 0 }}
                  >
                    <Input
                      className={FIELD_WIDTH_CLASS}
                      placeholder={t('page.datasource.neo4j.labels.parameter_value_placeholder', 'Parameter value')}
                      style={{ width: '100%' }}
                    />
                  </Form.Item>
                  <MinusCircleOutlined
                    style={{ color: '#ff4d4f' }}
                    onClick={() => remove(name)}
                  />
                </Space>
              ))}
              <Button
                className={FIELD_WIDTH_CLASS}
                icon={<PlusCircleOutlined />}
                style={{ width: '100%' }}
                type="dashed"
                onClick={() => add({ key: '', value: '' })}
              >
                {t('page.datasource.neo4j.labels.add_parameter', 'Add Parameter')}
              </Button>
            </>
          )}
        </Form.List>
      </Form.Item>

      <Form.Item
        label={t('page.datasource.rdbms.labels.pagination', 'Enable Pagination')}
        name={['config', 'pagination']}
        valuePropName="checked"
        tooltip={t(
          'page.datasource.neo4j.tooltip.pagination',
          'Enable if the Cypher query should be paginated. Recommended when the result set is large.'
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
                className={FIELD_WIDTH_CLASS}
                min={1}
                style={{ width: '100%' }}
              />
            </Form.Item>
          ) : null
        }
      </Form.Item>

      <IncrementalSyncFields
        connectorType="neo4j"
        form={form}
        propertyTypeOptions={propertyTypeOptions}
        switchLayout="inline"
        validateResumeFrom={validateResumeFrom}
        placeholders={{
          property: 'updated',
          resumeFrom: 'e.g. 2025-01-01T00:00:00Z',
          tieBreaker: 'elementId(n)'
        }}
      />

      <Form.Item label={t('page.datasource.rdbms.labels.data_processing', 'Data Processing')}>
        <Flex
          align="center"
          gap="small"
        >
          <span>{t('page.datasource.rdbms.labels.field_mapping', 'Field Mapping')}</span>
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

export default Neo4jForm;
