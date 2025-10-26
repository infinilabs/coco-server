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
import { getUUID } from '@/utils/common';
import { useTranslation } from 'react-i18next';

const { Option } = Select;

interface EditFormProps {
  readonly actionText: string;
  readonly loading?: boolean;
  readonly onSubmit: (params: any, before?: () => void, after?: () => void) => Promise<void>;
  readonly record?: any;
}

export const EditForm = memo((props: EditFormProps) => {
  const { t } = useTranslation();
  const { endLoading, loading, startLoading } = useLoading();

  const [rows, setRows] = useState<
    Array<{
      id: string;
      type: 'team' | 'user';
      principal?: any;
      role?: any;
    }>
  >([{ id: getUUID(), type: 'user' }]);

  const TYPE_OPTIONS = [
    { key: 'user', label: t('page.auth.labels.user') },
    { key: 'team', label: t('page.auth.labels.team') }
  ];

  const {
    data: principalRes,
    loading: principalLoading,
    run: runPrincipalSearch
  } = useRequest(fetchUserSearch, { manual: true });

  const [principalQueryParams, setPrincipalQueryParams] = useState({
    query: '',
    from: 0,
    size: 10,
    type: 'user' as 'team' | 'user'
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

  const addRow = () => {
    setRows(prev => [...prev, { id: getUUID(), type: 'user' }]);
  };

  const removeRow = (id: string) => {
    setRows(prev => {
      if (prev.length === 1) return prev;
      return prev.filter(r => r.id !== id);
    });
  };

  const updateRow = <K extends keyof (typeof rows)[number]>(id: string, field: K, value: (typeof rows)[number][K]) => {
    setRows(prev => prev.map(r => (r.id === id ? { ...r, [field]: value } : r)));
  };

  useEffect(() => {
    runPrincipalSearch(principalQueryParams);
    runRoleSearch(roleQueryParams);
  }, []);

  return (
    <Spin spinning={props.loading || loading || principalLoading || roleLoading || false}>
      <h3>{t('page.auth.labels.object')}</h3>
      <Form layout='vertical'>
        {rows.map((row, index) => {
          const isLast = index === rows.length - 1;
          return (
            <Space.Compact
              block
              key={row.id}
              style={{ display: 'flex', marginBottom: 8, width: '100%' }}
            >
              <Select
                style={{ width: 100 }}
                value={row.type}
                onChange={(val: 'team' | 'user') => {
                  updateRow(row.id, 'type', val);
                  updateRow(row.id, 'principal', undefined);
                  setPrincipalQueryParams(params => ({ ...params, type: val, query: '', from: 0 }));
                }}
              >
                // eslint-disable-next-line @typescript-eslint/no-shadow
                {TYPE_OPTIONS.map(t => (
                  <Option
                    key={t.key}
                    value={t.key}
                  >
                    {t.label}
                  </Option>
                ))}
              </Select>

              <DropdownList
                allowClear
                data={principalResult.data}
                dropdownWidth={300}
                placeholder={t('page.auth.labels.name')}
                renderLabel={(item: any) => item?.name}
                rowKey='id'
                value={row.principal}
                width={240}
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
                onChange={val => updateRow(row.id, 'principal', val)}
                onSearchChange={(query: string) => {
                  setPrincipalQueryParams(params => ({ ...params, query, from: 0, type: row.type }));
                }}
              />

              <Input
                disabled
                className='site-input-split'
                placeholder={t('page.auth.labels.role')}
                style={{
                  width: 60,
                  borderInlineStart: 0,
                  borderInlineEnd: 0,
                  pointerEvents: 'none'
                }}
              />

              <DropdownList
                allowClear
                data={roleResult.data}
                dropdownWidth={300}
                placeholder={t('page.role.title')}
                renderItem={(item: any) => <span>{item.name}</span>}
                renderLabel={(item: any) => item?.name}
                rowKey='id'
                value={row.role}
                width={240}
                pagination={{
                  currentPage: roleResult.total ? Math.floor(roleQueryParams.from / roleQueryParams.size) + 1 : 0,
                  total: roleResult.total,
                  onChange: page => {
                    setRoleQueryParams(params => ({ ...params, from: (page - 1) * params.size }));
                  }
                }}
                onChange={val => updateRow(row.id, 'role', val)}
                onSearchChange={(query: string) => {
                  setRoleQueryParams(params => ({ ...params, query, from: 0 }));
                }}
              />

              <Button
                disabled={rows.length === 1}
                icon={<DeleteOutlined />}
                onClick={() => removeRow(row.id)}
              />
              {isLast && (
                <Button
                  icon={<PlusOutlined />}
                  onClick={addRow}
                />
              )}
            </Space.Compact>
          );
        })}
      </Form>
    </Spin>
  );
});
