import { DownOutlined, MinusCircleOutlined, PlusCircleOutlined, SwapOutlined, UpOutlined } from '@ant-design/icons';
import { Button, Form, Input, Space, Switch, Typography } from 'antd';
import React from 'react';
import { useTranslation } from 'react-i18next';

// eslint-disable-next-line max-params
const renderMapping = (name: string[], config: string, required = false, enabled = true) => {
  // eslint-disable-next-line react-hooks/rules-of-hooks
  const { t } = useTranslation();
  const rules =
    required && enabled
      ? [{ message: t('page.datasource.rdbms.validation.required', { field: name[name.length - 1] }), required: true }]
      : [];
  return (
    <div style={{ width: 300 }}>
      <Space.Compact block>
        <Input
          readOnly
          prefix={required ? <span style={{ color: 'red' }}>*</span> : null}
          style={{ backgroundColor: '#f5f5f5', textAlign: 'center', width: '45%' }}
          value={config}
        />
        <div
          style={{
            alignItems: 'center',
            border: '1px solid #d9d9d9',
            borderRadius: 2,
            display: 'flex',
            justifyContent: 'center',
            width: '10%'
          }}
        >
          <SwapOutlined />
        </div>
        <Form.Item
          noStyle
          name={name}
          rules={rules}
        >
          <Input
            placeholder={config}
            style={{ textAlign: 'center', width: '45%' }}
          />
        </Form.Item>
      </Space.Compact>
    </div>
  );
};

const CollapsibleFieldMapping = ({
  children,
  title
}: {
  readonly children: React.ReactNode;
  readonly title: string;
}) => {
  const [isOpen, setIsOpen] = React.useState(true);

  return (
    <div>
      <div style={{ alignItems: 'center', display: 'flex', marginBottom: 8 }}>
        <Input
          readOnly
          style={{ backgroundColor: '#f5f5f5', width: 300 }}
          value={title}
        />
        <Button
          icon={isOpen ? <UpOutlined /> : <DownOutlined />}
          type="text"
          onClick={() => setIsOpen(!isOpen)}
        />
      </div>
      {isOpen && <div style={{ paddingLeft: 24 }}>{children}</div>}
    </div>
  );
};

export const FieldMapping = ({ enabled }: { readonly enabled: boolean }) => {
  const { t } = useTranslation();
  const [showMore, setShowMore] = React.useState(false);

  return (
    <div
      style={{ border: '1px solid #d9d9d9', borderRadius: '2px', display: enabled ? 'block' : 'none', padding: '16px' }}
    >
      <div style={{ display: 'flex', marginBottom: 8, width: 300 }}>
        <div style={{ textAlign: 'center', width: '45%' }}>
          <Typography.Text strong>{t('page.datasource.rdbms.labels.dest_field', 'Destination Field')}</Typography.Text>
        </div>
        <div style={{ width: '10%' }} />
        <div style={{ textAlign: 'center', width: '45%' }}>
          <Typography.Text strong>{t('page.datasource.rdbms.labels.src_field', 'Source Field')}</Typography.Text>
        </div>
      </div>
      <Form.Item>
        <Space>
          {renderMapping(['config', 'field_mapping', 'mapping', 'id'], 'id', true, enabled)}
          <Space>
            <span>MD5 Hash</span>
            <Form.Item
              noStyle
              initialValue={true}
              name={['config', 'field_mapping', 'mapping', 'hashed']}
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>
          </Space>
        </Space>
      </Form.Item>
      <Form.Item>{renderMapping(['config', 'field_mapping', 'mapping', 'title'], 'title', true, enabled)}</Form.Item>
      <Form.Item>{renderMapping(['config', 'field_mapping', 'mapping', 'url'], 'url', true, enabled)}</Form.Item>
      <Form.Item>
        {renderMapping(['config', 'field_mapping', 'mapping', 'summary'], 'summary', false, enabled)}
      </Form.Item>
      <Form.Item>
        {renderMapping(['config', 'field_mapping', 'mapping', 'content'], 'content', false, enabled)}
      </Form.Item>
      <Form.Item>
        {renderMapping(['config', 'field_mapping', 'mapping', 'created'], 'created', false, enabled)}
      </Form.Item>
      <Form.Item>
        {renderMapping(['config', 'field_mapping', 'mapping', 'updated'], 'updated', false, enabled)}
      </Form.Item>

      <Button
        type="link"
        onClick={() => setShowMore(!showMore)}
      >
        {showMore ? <UpOutlined /> : <DownOutlined />} {showMore ? 'Hide More' : 'Show More'}
      </Button>

      <div style={{ display: showMore ? 'block' : 'none' }}>
        <Form.Item>{renderMapping(['config', 'field_mapping', 'mapping', 'icon'], 'icon', false, enabled)}</Form.Item>
        <Form.Item>
          {renderMapping(['config', 'field_mapping', 'mapping', 'category'], 'category', false, enabled)}
        </Form.Item>
        <Form.Item>
          {renderMapping(['config', 'field_mapping', 'mapping', 'subcategory'], 'subcategory', false, enabled)}
        </Form.Item>
        <Form.Item>{renderMapping(['config', 'field_mapping', 'mapping', 'cover'], 'cover', false, enabled)}</Form.Item>
        <Form.Item>{renderMapping(['config', 'field_mapping', 'mapping', 'type'], 'type', false, enabled)}</Form.Item>
        <Form.Item>{renderMapping(['config', 'field_mapping', 'mapping', 'lang'], 'lang', false, enabled)}</Form.Item>
        <Form.Item>
          {renderMapping(['config', 'field_mapping', 'mapping', 'thumbnail'], 'thumbnail', false, enabled)}
        </Form.Item>
        <Form.Item>{renderMapping(['config', 'field_mapping', 'mapping', 'tags'], 'tags', false, enabled)}</Form.Item>
        <Form.Item>{renderMapping(['config', 'field_mapping', 'mapping', 'size'], 'size', false, enabled)}</Form.Item>
        <CollapsibleFieldMapping title="owner">
          <Form.Item>
            {renderMapping(['config', 'field_mapping', 'mapping', 'owner', 'avatar'], 'avatar', false, enabled)}
          </Form.Item>
          <Form.Item>
            {renderMapping(['config', 'field_mapping', 'mapping', 'owner', 'username'], 'username', false, enabled)}
          </Form.Item>
          <Form.Item>
            {renderMapping(['config', 'field_mapping', 'mapping', 'owner', 'userid'], 'userid', false, enabled)}
          </Form.Item>
        </CollapsibleFieldMapping>
        <CollapsibleFieldMapping title="last_updated_by">
          <CollapsibleFieldMapping title="user">
            <Form.Item>
              {renderMapping(
                ['config', 'field_mapping', 'mapping', 'last_updated_by', 'user', 'avatar'],
                'avatar',
                false,
                enabled
              )}
            </Form.Item>
            <Form.Item>
              {renderMapping(
                ['config', 'field_mapping', 'mapping', 'last_updated_by', 'user', 'username'],
                'username',
                false,
                enabled
              )}
            </Form.Item>
            <Form.Item>
              {renderMapping(
                ['config', 'field_mapping', 'mapping', 'last_updated_by', 'user', 'userid'],
                'userid',
                false,
                enabled
              )}
            </Form.Item>
          </CollapsibleFieldMapping>
          <Form.Item>
            {renderMapping(
              ['config', 'field_mapping', 'mapping', 'last_updated_by', 'timestamp'],
              'timestamp',
              false,
              enabled
            )}
          </Form.Item>
        </CollapsibleFieldMapping>
        <Form.List name={['config', 'field_mapping', 'mapping', 'metadata']}>
          {(fields, { add, remove }) => (
            <div style={{ alignItems: 'start', display: 'flex' }}>
              <CollapsibleFieldMapping title="metadata">
                {fields.map(({ key, name, ...restField }, index) => (
                  <Space
                    align="baseline"
                    key={key}
                    style={{ display: 'flex', marginBottom: 8 }}
                  >
                    <Input
                      readOnly
                      style={{ backgroundColor: '#f5f5f5', width: 80 }}
                      value="name"
                    />
                    <Form.Item
                      {...restField}
                      name={[name, 'name']}
                      rules={[
                        { message: t('page.datasource.rdbms.validation.metadata_name_required'), required: true }
                      ]}
                    >
                      <Input placeholder={t('page.datasource.rdbms.placeholder.metadata_name', 'Metadata Name')} />
                    </Form.Item>
                    <Input
                      readOnly
                      style={{ backgroundColor: '#f5f5f5', width: 80 }}
                      value="value"
                    />
                    <Form.Item
                      {...restField}
                      name={[name, 'value']}
                      rules={[{ message: t('page.datasource.rdbms.validation.column_name_required'), required: true }]}
                    >
                      <Input placeholder={t('page.datasource.rdbms.placeholder.column_name', 'Column Name')} />
                    </Form.Item>
                    <MinusCircleOutlined
                      style={{ color: 'red' }}
                      onClick={() => remove(name)}
                    />
                    {index === fields.length - 1 && (
                      <PlusCircleOutlined
                        style={{ color: 'blue' }}
                        onClick={() => add()}
                      />
                    )}
                  </Space>
                ))}
              </CollapsibleFieldMapping>
              {fields.length === 0 && (
                <PlusCircleOutlined
                  style={{ color: 'blue', marginLeft: 8, marginTop: 8 }}
                  onClick={() => add()}
                />
              )}
            </div>
          )}
        </Form.List>
        <Form.List name={['config', 'field_mapping', 'mapping', 'payload']}>
          {(fields, { add, remove }) => (
            <div style={{ alignItems: 'start', display: 'flex' }}>
              <CollapsibleFieldMapping title="payload">
                {fields.map(({ key, name, ...restField }, index) => (
                  <Space
                    align="baseline"
                    key={key}
                    style={{ display: 'flex', marginBottom: 8 }}
                  >
                    <Input
                      readOnly
                      style={{ backgroundColor: '#f5f5f5', width: 80 }}
                      value="name"
                    />
                    <Form.Item
                      {...restField}
                      name={[name, 'name']}
                      rules={[{ message: t('page.datasource.rdbms.validation.payload_name_required'), required: true }]}
                    >
                      <Input placeholder={t('page.datasource.rdbms.placeholder.payload_name', 'Payload Name')} />
                    </Form.Item>
                    <Input
                      readOnly
                      style={{ backgroundColor: '#f5f5f5', width: 80 }}
                      value="value"
                    />
                    <Form.Item
                      {...restField}
                      name={[name, 'value']}
                      rules={[{ message: t('page.datasource.rdbms.validation.column_name_required'), required: true }]}
                    >
                      <Input placeholder={t('page.datasource.rdbms.placeholder.column_name', 'Column Name')} />
                    </Form.Item>
                    <MinusCircleOutlined
                      style={{ color: 'red' }}
                      onClick={() => remove(name)}
                    />
                    {index === fields.length - 1 && (
                      <PlusCircleOutlined
                        style={{ color: 'blue' }}
                        onClick={() => add()}
                      />
                    )}
                  </Space>
                ))}
              </CollapsibleFieldMapping>
              {fields.length === 0 && (
                <PlusCircleOutlined
                  style={{ color: 'blue', marginLeft: 8, marginTop: 8 }}
                  onClick={() => add()}
                />
              )}
            </div>
          )}
        </Form.List>
      </div>
    </div>
  );
};