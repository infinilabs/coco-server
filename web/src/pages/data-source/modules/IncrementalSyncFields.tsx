import { Flex, Form, Input, Select, Switch } from 'antd';
import React from 'react';
import { useTranslation } from 'react-i18next';

const FIELD_WIDTH_CLASS = 'max-w-660px';

interface PropertyTypeOption {
  label: string;
  value: string;
}

interface IncrementalSyncFieldsProps {
  readonly connectorType: string;
  readonly form: any; // 'mongodb', 'neo4j', etc.
  readonly namePrefix?: string[];
  readonly placeholders: {
    property: string;
    resumeFrom?: string | ((propertyType: string) => string);
    tieBreaker: string;
  };
  readonly propertyTypeOptions: PropertyTypeOption[];
  readonly switchLayout?: 'flex' | 'inline';
  readonly validateResumeFrom?: (rule: any, value: string) => Promise<void>;
}

const IncrementalSyncFields: React.FC<IncrementalSyncFieldsProps> = ({
  connectorType,
  form,
  namePrefix = ['config'],
  placeholders,
  propertyTypeOptions,
  switchLayout = 'inline',
  validateResumeFrom
}) => {
  const { t } = useTranslation();

  // Generate translation keys dynamically based on connector type
  const getTranslationKey = (category: string, key: string) => {
    return `page.datasource.${connectorType}.${category}.${key}`;
  };

  const incrementalEnabled = Form.useWatch([...namePrefix, 'incremental', 'enabled'], form);
  const propertyType = Form.useWatch([...namePrefix, 'incremental', 'property_type'], form);

  const getResumePlaceholder = () => {
    if (typeof placeholders.resumeFrom === 'function') {
      return placeholders.resumeFrom(propertyType || 'datetime');
    }
    return placeholders.resumeFrom || '';
  };

  return (
    <>
      {/* Incremental Sync Toggle */}
      {switchLayout === 'flex' ? (
        <Form.Item
          label={t(getTranslationKey('labels', 'incremental_sync'), 'Incremental Sync')}
          tooltip={t(
            getTranslationKey('tooltip', 'incremental_sync'),
            'Enable to resume scans from the last seen property value (watermark).'
          )}
        >
          <Flex
            align="center"
            gap="small"
          >
            <span>{t(getTranslationKey('labels', 'incremental_sync_enable'), 'Enable Incremental Sync')}</span>
            <Form.Item
              initialValue={false}
              noStyle
              name={[...namePrefix, 'incremental', 'enabled']}
              valuePropName="checked"
            >
              <Switch />
            </Form.Item>
          </Flex>
        </Form.Item>
      ) : (
        <Form.Item
          initialValue={false}
          label={t(getTranslationKey('labels', 'incremental_sync'), 'Incremental Sync')}
          name={[...namePrefix, 'incremental', 'enabled']}
          tooltip={t(
            getTranslationKey('tooltip', 'incremental_sync'),
            'Enable to resume scans from the last seen property value (watermark).'
          )}
          valuePropName="checked"
        >
          <Switch />
        </Form.Item>
      )}

      {/* Incremental Fields (shown when enabled) */}
      {incrementalEnabled ? (
        <>
          {/* Property Field */}
          <Form.Item
            label={t(getTranslationKey('labels', 'property'), 'Tracking Property')}
            name={[...namePrefix, 'incremental', 'property']}
            tooltip={t(getTranslationKey('tooltip', 'property'), 'Field name used to track changes')}
            rules={[
              {
                message: t(getTranslationKey('error', 'property_required'), 'Please input tracking property name!'),
                required: true
              }
            ]}
          >
            <Input
              className={FIELD_WIDTH_CLASS}
              placeholder={placeholders.property}
            />
          </Form.Item>

          {/* Property Type */}
          <Form.Item
            initialValue="datetime"
            label={t(getTranslationKey('labels', 'property_type'), 'Property Type')}
            name={[...namePrefix, 'incremental', 'property_type']}
            tooltip={t(getTranslationKey('tooltip', 'property_type'), 'Data type of the tracking property')}
          >
            <Select
              className={FIELD_WIDTH_CLASS}
              options={propertyTypeOptions}
              style={{ width: '100%' }}
            />
          </Form.Item>

          {/* Tie-breaker Field */}
          <Form.Item
            label={t(getTranslationKey('labels', 'tie_breaker'), 'Tie-breaker Field')}
            name={[...namePrefix, 'incremental', 'tie_breaker']}
            rules={[
              {
                message: t(getTranslationKey('error', 'tie_breaker_required'), 'Please input tie-breaker field!'),
                required: true
              }
            ]}
            tooltip={t(
              getTranslationKey('tooltip', 'tie_breaker'),
              'Secondary field to break ties when multiple documents have the same property value'
            )}
          >
            <Input
              className={FIELD_WIDTH_CLASS}
              placeholder={placeholders.tieBreaker}
            />
          </Form.Item>

          {/* Resume From */}
          <Form.Item
            label={t(getTranslationKey('labels', 'resume_from'), 'Resume From Value')}
            name={[...namePrefix, 'incremental', 'resume_from']}
            rules={validateResumeFrom ? [{ validator: validateResumeFrom }] : undefined}
            tooltip={t(
              getTranslationKey('tooltip', 'resume_from'),
              'Optional manual starting point for the first sync'
            )}
          >
            <Input
              className={FIELD_WIDTH_CLASS}
              placeholder={getResumePlaceholder()}
              style={{ width: '100%' }}
            />
          </Form.Item>
        </>
      ) : null}
    </>
  );
};

export default IncrementalSyncFields;
