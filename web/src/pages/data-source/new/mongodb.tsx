import { Form, Input, InputNumber, Switch, Select, Space, Button } from 'antd';
import React, { useState } from 'react';
import { useTranslation } from 'react-i18next';
import { MinusCircleOutlined, PlusOutlined } from '@ant-design/icons';

import { FieldMapping } from './FieldMapping';

const { Option } = Select;

export default function MongoDB() {
  const { t } = useTranslation();
  const [showAdvanced, setShowAdvanced] = useState(false);

  return (
    <>
      {/* 基本连接配置 */}
      <Form.Item
        label={t('page.datasource.mongodb.labels.connection_uri', 'Connection URI')}
        name={['config', 'connection_uri']}
        tooltip={t(
          'page.datasource.mongodb.tooltip.connection_uri',
          'MongoDB connection string. e.g., mongodb://username:password@localhost:27017/database'
        )}
        rules={[
          {
            message: t('page.datasource.mongodb.error.connection_uri_required', 'Please input Connection URI!'),
            required: true
          }
        ]}
      >
        <Input
          placeholder="mongodb://username:password@localhost:27017/database"
          style={{ width: 500 }}
        />
      </Form.Item>

      {/* 数据库名称 */}
      <Form.Item
        label={t('page.datasource.mongodb.labels.database', 'Database')}
        name={['config', 'database']}
        rules={[
          {
            message: t('page.datasource.mongodb.error.database_required', 'Please input Database name!'),
            required: true
          }
        ]}
      >
        <Input
          placeholder="database_name"
          style={{ width: 300 }}
        />
      </Form.Item>

      {/* 集合配置 */}
      <Form.Item
        label={t('page.datasource.mongodb.labels.collections', 'Collections')}
        required
      >
        <Form.List name={['config', 'collections']}>
          {(fields, { add, remove }) => (
            <>
              {fields.map(({ key, name, ...restField }) => (
                <div key={key} style={{ border: '1px solid #d9d9d9', borderRadius: '6px', padding: '16px', marginBottom: '16px' }}>
                  <Space align="baseline">
                    <Form.Item
                      {...restField}
                      name={[name, 'name']}
                      label={t('page.datasource.mongodb.labels.collection_name', 'Collection Name')}
                      rules={[
                        { message: t('page.datasource.mongodb.error.collection_name_required'), required: true }
                      ]}
                    >
                      <Input placeholder="collection_name" style={{ width: 200 }} />
                    </Form.Item>
                    
                    <Form.Item
                      {...restField}
                      name={[name, 'filter']}
                      label={t('page.datasource.mongodb.labels.filter', 'Filter (JSON)')}
                    >
                      <Input.TextArea 
                        placeholder='{"status": "published"}'
                        style={{ width: 200 }}
                        rows={2}
                      />
                    </Form.Item>

                    <MinusCircleOutlined
                      style={{ color: 'red' }}
                      onClick={() => remove(name)}
                    />
                  </Space>

                  {/* 字段映射配置 */}
                  <div style={{ marginTop: '16px' }}>
                    <Form.Item
                      {...restField}
                      name={[name, 'title_field']}
                      label={t('page.datasource.mongodb.labels.title_field', 'Title Field')}
                    >
                      <Input placeholder="title" style={{ width: 150 }} />
                    </Form.Item>
                    
                    <Form.Item
                      {...restField}
                      name={[name, 'content_field']}
                      label={t('page.datasource.mongodb.labels.content_field', 'Content Field')}
                    >
                      <Input placeholder="content" style={{ width: 150 }} />
                    </Form.Item>

                    <Form.Item
                      {...restField}
                      name={[name, 'category_field']}
                      label={t('page.datasource.mongodb.labels.category_field', 'Category Field')}
                    >
                      <Input placeholder="category" style={{ width: 150 }} />
                    </Form.Item>

                    <Form.Item
                      {...restField}
                      name={[name, 'tags_field']}
                      label={t('page.datasource.mongodb.labels.tags_field', 'Tags Field')}
                    >
                      <Input placeholder="tags" style={{ width: 150 }} />
                    </Form.Item>

                    <Form.Item
                      {...restField}
                      name={[name, 'url_field']}
                      label={t('page.datasource.mongodb.labels.url_field', 'URL Field')}
                    >
                      <Input placeholder="url" style={{ width: 150 }} />
                    </Form.Item>

                    <Form.Item
                      {...restField}
                      name={[name, 'timestamp_field']}
                      label={t('page.datasource.mongodb.labels.timestamp_field', 'Timestamp Field')}
                    >
                      <Input placeholder="updated_at" style={{ width: 150 }} />
                    </Form.Item>
                  </div>
                </div>
              ))}
              
              <Form.Item>
                <Button
                  type="dashed"
                  onClick={() => add()}
                  block
                  icon={<PlusOutlined />}
                >
                  {t('page.datasource.mongodb.labels.add_collection', 'Add Collection')}
                </Button>
              </Form.Item>
            </>
          )}
        </Form.List>
      </Form.Item>

      {/* 高级配置 */}
      <Form.Item
        label={t('page.datasource.mongodb.labels.advanced_config', 'Advanced Configuration')}
        name={['config', 'advanced_enabled']}
        valuePropName="checked"
      >
        <Switch onChange={(checked) => setShowAdvanced(checked)} />
      </Form.Item>

      {showAdvanced && (
        <div style={{ border: '1px solid #d9d9d9', borderRadius: '6px', padding: '16px', marginBottom: '16px' }}>
          <Form.Item
            label={t('page.datasource.mongodb.labels.batch_size', 'Batch Size')}
            name={['config', 'batch_size']}
            tooltip={t('page.datasource.mongodb.tooltip.batch_size', 'Number of documents to process in each batch')}
          >
            <InputNumber
              min={1}
              max={10000}
              defaultValue={1000}
              style={{ width: 200 }}
            />
          </Form.Item>

          <Form.Item
            label={t('page.datasource.mongodb.labels.max_pool_size', 'Max Pool Size')}
            name={['config', 'max_pool_size']}
            tooltip={t('page.datasource.mongodb.tooltip.max_pool_size', 'Maximum number of connections in the connection pool')}
          >
            <InputNumber
              min={1}
              max={100}
              defaultValue={10}
              style={{ width: 200 }}
            />
          </Form.Item>

          <Form.Item
            label={t('page.datasource.mongodb.labels.timeout', 'Timeout')}
            name={['config', 'timeout']}
            tooltip={t('page.datasource.mongodb.tooltip.timeout', 'Connection timeout in seconds')}
          >
            <Input
              placeholder="30s"
              style={{ width: 200 }}
            />
          </Form.Item>

          <Form.Item
            label={t('page.datasource.mongodb.labels.sync_strategy', 'Sync Strategy')}
            name={['config', 'sync_strategy']}
            tooltip={t('page.datasource.mongodb.tooltip.sync_strategy', 'Choose between full sync or incremental sync')}
          >
            <Select defaultValue="full" style={{ width: 200 }}>
              <Option value="full">{t('page.datasource.mongodb.options.full_sync', 'Full Sync')}</Option>
              <Option value="incremental">{t('page.datasource.mongodb.options.incremental_sync', 'Incremental Sync')}</Option>
            </Select>
          </Form.Item>
        </div>
      )}

      {/* 字段映射 */}
      <Form.Item
        label={t('page.datasource.mongodb.labels.field_mapping', 'Field Mapping')}
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
        {({ getFieldValue }) => <FieldMapping enabled={getFieldValue(['config', 'field_mapping', 'enabled'])} />}
      </Form.Item>
    </>
  );
}
