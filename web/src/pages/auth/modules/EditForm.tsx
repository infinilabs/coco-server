import React, { memo, useEffect, useState } from 'react';
import { Button, Form, Select, Spin } from 'antd';
import { useLoading, useRequest } from '@sa/hooks';

import { fetchPrincipalSearch } from '@/service/api/security';

import DropdownList from '@/common/src/DropdownList';
import { formatESSearchResult } from '@/service/request/es';
import { useTranslation } from 'react-i18next';
import RoleSelect from '@/pages/security/modules/RoleSelect';
import { getLocale } from '@/store/slice/app';

const { Option } = Select;

interface EditFormProps {
  readonly actionText: string;
  readonly loading?: boolean;
  readonly onSubmit: (params: any, before?: () => void, after?: () => void) => Promise<void>;
  readonly record?: any;
}

export const EditForm = memo((props: EditFormProps) => {
  const { actionText, onSubmit, record } = props;
  const { t } = useTranslation();
  const locale = useAppSelector(getLocale);
  
  const { endLoading, loading, startLoading } = useLoading();

  const TYPE_OPTIONS = [
    { key: 'user', label: t('page.auth.labels.user') },
    // { key: 'team', label: t('page.auth.labels.team') }
  ];

  const { defaultRequiredRule } = useFormRules();
  const [form] = Form.useForm();

  const {
    data: principalRes,
    loading: principalLoading,
    run: runPrincipalSearch
  } = useRequest(fetchPrincipalSearch, { manual: true });

  const [principalQueryParams, setPrincipalQueryParams] = useState({
    query: '',
    from: 0,
    size: 10,
    type: TYPE_OPTIONS[0].key
  });

  useEffect(() => {
    runPrincipalSearch(principalQueryParams);
  }, [principalQueryParams]);

  const principalResult = React.useMemo(() => {
    const rs = formatESSearchResult(principalRes);
    return {
      data: rs.data || [],
      total: rs.total || 0
    };
  }, [principalRes]);

  const handleSubmit = async () => {
    const params = await form.validateFields();
    const { principal_type, principal, roles } = params
    onSubmit({
      principal_type,
      principal_id: principal?.id,
      display_name: principal?.name,
      roles: (roles || []).map((item) => item.name)
    }, startLoading, endLoading);
  };

  useEffect(() => {
    runPrincipalSearch(principalQueryParams);
  }, []);

  useEffect(() => {
    if (record && typeof record === 'object') {
      form.setFieldsValue({
        ...record,
        principal: {
          id: record.principal_id,
          name: record.display_name
        },
        roles: (record.roles || []).map((item) => ({
          name: item
        }))
      })
      if (record.principal_type) setPrincipalQueryParams((state) => ({...state, type: record.principal_type}))
    } else {
      form.setFieldsValue({
        principal_type: TYPE_OPTIONS[0].key
      });
    }
  }, [record]);

  const itemClassNames = '!w-496px';

  return (
    <Spin spinning={props.loading || loading || principalLoading || false}>
      <Form
        colon={false}
        form={form}
        labelAlign='left'
        layout='horizontal'
        labelCol={{
          style: { maxWidth: 200, minWidth: 200, textAlign: 'left' }
        }}
      >
        <Form.Item
          label={t('page.auth.labels.type')}
          name='principal_type'
          rules={[defaultRequiredRule]}
        >
          <Select
            className={itemClassNames} 
            onChange={(val: 'team' | 'user') => {
              setPrincipalQueryParams((state) => ({...state, type: val}))
            }}
          >
            {TYPE_OPTIONS.map(t => (
              <Option
                key={t.key}
                value={t.key}
              >
                {t.label}
              </Option>
            ))}
          </Select>
        </Form.Item>
        <Form.Item
          label={t(`page.auth.labels.${principalQueryParams.type}`)}
          name='principal'
          rules={[defaultRequiredRule]}
        >
          <DropdownList
            className={itemClassNames} 
            width="100%"
            data={principalResult.data}
            placeholder={t(`page.auth.labels.${principalQueryParams.type}`)}
            renderLabel={(item: any) => <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                {item.avatar && <img src={item.avatar} className='rounded-full w-16px h-16px' />}
                <span>{item.name}</span>
                </div>
            }
            rowKey='id'
            pagination={{
              currentPage: principalResult.total
                ? Math.floor(principalQueryParams.from / principalQueryParams.size) + 1
                : 0,
              total: principalResult.total,
              onChange: page => {
                setPrincipalQueryParams(params => ({ ...params, from: (page - 1) * params.size }));
              }
            }}
            renderItem={(item: any) => (
              <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                {item.avatar && <img src={item.avatar} className='rounded-full' style={{ width: 16, height: 16 }} />}
                <span>{item.name}</span>
                {item.description ? (
                  <span style={{ color: 'var(--ant-color-text-tertiary)' }}>{item.description}</span>
                ) : null}
              </div>
            )}
            onSearchChange={(query: string) => {
              setPrincipalQueryParams(params => ({ ...params, query, from: 0, type: row.type }));
            }}
            locale={locale}
          />
        </Form.Item>
        <Form.Item
          label={t('page.auth.labels.roles')}
          name='roles'
          rules={[defaultRequiredRule]}
        >
          <RoleSelect
            className={itemClassNames}
            mode="multiple"
            width="100%"
            allowClear
            placeholder={t('page.auth.labels.roles')}
          />
        </Form.Item>
        <Form.Item label=' '>
          <Button
            type='primary'
            onClick={() => handleSubmit()}
          >
            {actionText}
          </Button>
        </Form.Item>
      </Form>
    </Spin>
  );
});
