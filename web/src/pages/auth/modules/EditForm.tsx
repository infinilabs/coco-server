/* eslint-disable react/display-name */
/* eslint-disable react/jsx-no-comment-textnodes */
/* eslint-disable react-hooks/exhaustive-deps */
/* eslint-disable @typescript-eslint/no-shadow */
import React, { memo, useEffect, useState } from 'react';
import { Button, Form, Input, Select, Space, Spin } from 'antd';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';
import { useLoading, useRequest } from '@sa/hooks';

import { fetchRoles, fetchUserSearch } from '@/service/api/role';

import DropdownList from '@/common/src/DropdownList';
import { formatESSearchResult } from '@/service/request/es';
import { useTranslation } from 'react-i18next';

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
  const { endLoading, loading, startLoading } = useLoading();


  const TYPE_OPTIONS = [
    { key: 'user', label: t('page.auth.labels.user') },
    { key: 'team', label: t('page.auth.labels.team') }
  ];

  const { defaultRequiredRule } = useFormRules();
  const [form] = Form.useForm();

  const {
    data: principalRes,
    loading: principalLoading,
    run: runPrincipalSearch
  } = useRequest(fetchUserSearch, { manual: true });

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

  const { data: roleRes, loading: roleLoading, run: runRoleSearch } = useRequest(fetchRoles, { manual: true });

  const [roleQueryParams, setRoleQueryParams] = useState({
    query: '',
    from: 0,
    size: 10,
    sort: 'created:desc'
  });

  useEffect(() => {
    runRoleSearch(roleQueryParams);
  }, [roleQueryParams]);

  const roleResult = React.useMemo(() => {
    const rs = formatESSearchResult(roleRes);
    return {
      data: rs.data || [],
      total: rs.total || 0
    };
  }, [roleRes]);

  const handleSubmit = async () => {
    const params = await form.validateFields();
    const { principal_type, principal, role } = params
    onSubmit({
      principal_type,
      principal_id: principal?.id,
      display_name: principal?.name,
      role: (role || []).map((item) => item.name)
    }, startLoading, endLoading);
  };

  useEffect(() => {
    runPrincipalSearch(principalQueryParams);
    runRoleSearch(roleQueryParams);
  }, []);

  useEffect(() => {
    if (record && typeof record === 'object') {
      form.setFieldsValue({
        ...record,
        principal: {
          id: record.principal_id,
          name: record.display_name
        },
        role: (record.role || []).map((item) => ({
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
    <Spin spinning={props.loading || loading || principalLoading || roleLoading || false}>
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
            placeholder={t('page.auth.labels.name')}
            renderLabel={(item: any) => item?.name}
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
              <div style={{ display: 'flex', gap: 8 }}>
                <span>{item.name}</span>
                {item.description ? (
                  <span style={{ color: 'var(--ant-color-text-tertiary)' }}>{item.description}</span>
                ) : null}
              </div>
            )}
            onSearchChange={(query: string) => {
              setPrincipalQueryParams(params => ({ ...params, query, from: 0, type: row.type }));
            }}
          />
        </Form.Item>
        <Form.Item
          label={t('page.auth.labels.role')}
          name='role'
          rules={[defaultRequiredRule]}
        >
          <DropdownList
            className={itemClassNames}
            mode="multiple"
            width="100%"
            allowClear
            data={roleResult.data}
            placeholder={t('page.role.title')}
            renderItem={(item: any) => <span>{item.name}</span>}
            renderLabel={(item: any) => item?.name}
            rowKey='name'
            pagination={{
              currentPage: roleResult.total ? Math.floor(roleQueryParams.from / roleQueryParams.size) + 1 : 0,
              total: roleResult.total,
              onChange: page => {
                setRoleQueryParams(params => ({ ...params, from: (page - 1) * params.size }));
              }
            }}
            onSearchChange={(query: string) => {
              setRoleQueryParams(params => ({ ...params, query, from: 0 }));
            }}
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
