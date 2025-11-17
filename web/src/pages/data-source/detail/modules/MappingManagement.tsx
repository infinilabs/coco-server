import { FilterOutlined } from '@ant-design/icons';
import { useLoading } from '@sa/hooks';
import { Button, Input, Space, Switch, Table } from 'antd';

import { formatESSearchResult } from '@/service/request/es';
import useQueryParams from '@/hooks/common/queryParams';
import { fetchRoles } from '@/service/api/security';
import PrincipalSelect from '@/components/Resource/PrincipalSelect';

const MappingManagement = (props) => {
  const [queryParams, setQueryParams] = useQueryParams();
  const { t } = useTranslation();

  const { hasAuth } = useAuth()

  const permissions = {
    update: hasAuth('coco#datasource/update'),
  }

  const [editRow, setEditRow] = useState()

  const [data, setData] = useState({
    data: [],
    total: 0
  });
  const { endLoading, loading, startLoading } = useLoading();
  const [keyword, setKeyword] = useState();

  const fetchData = async (params) => {
    startLoading();
    const res = await fetchRoles(params);
    const newData = formatESSearchResult(res.data);
    setData(newData);
    endLoading();
  };

  const handleTableChange = pagination => {
    setQueryParams(params => {
      return {
        ...params,
        from: (pagination.current - 1) * pagination.pageSize,
        size: pagination.pageSize
      };
    });
  };

  const onRefreshClick = (query: string) => {
    setQueryParams(oldParams => {
      return {
        ...oldParams,
        from: 0,
        query,
        t: new Date().valueOf()
      };
    });
  };

  const columns = [
    {
      dataIndex: 'name',
      title: t('page.datasource.labels.externalAccount')
    },
    {
      dataIndex: 'coco',
      title: t('page.datasource.labels.cocoAccount'),
      render: (value, record) => {
        return editRow?.id === record.id ? (
          <PrincipalSelect ></PrincipalSelect>
        ) : value
      }
    },
    {
      dataIndex: 'mapping_status',
      title: t('page.datasource.labels.mappingStatus'),
      render: (value: boolean)=>{
       return value ? t('page.datasource.labels.mapped') : t('page.datasource.labels.unmapped')
      }
    },
    {
      dataIndex: 'enabled',
      title: t('page.datasource.labels.enabled'),
      width: 80,
      render: (value: boolean)=>{
       return <Switch size="small" value={value} onChange={(v)=> {}} />
      }
    },
    {
      fixed: 'right',
      hidden: !permissions.update,
      render: (_, record) => {
        return editRow?.id === record.id ? (
          <Space>
            <Button type="link" className="px-0" onClick={() => setEditRow(record)}>{t('common.save')}</Button>
            <Button type="link" className="px-0" onClick={() => setEditRow()}>{t('common.cancel')}</Button>
          </Space>
        ) : (
          <Button type="link" className="px-0" onClick={() => setEditRow(record)}>{t('common.edit')}</Button>
        )
      },
      title: t('common.operation'),
      width: '90px'
    }
  ];
  // rowSelection object indicates the need for row selection
  const rowSelection = {
    getCheckboxProps: record => ({
      name: record.name
    }),
    onChange: (selectedRowKeys: React.Key[], selectedRows) => {}
  };

  useEffect(() => {
    fetchData(queryParams);
  }, [queryParams]);

  useEffect(() => {
    setKeyword(queryParams.query)
  }, [queryParams.query])

  return (
    <ListContainer>
      <div className="mb-4 mt-4 flex items-center justify-between">
        <Input.Search
          addonBefore={<FilterOutlined />}
          className="max-w-500px"
          enterButton={t('common.refresh')}
          onSearch={onRefreshClick}
          value={keyword}
          onChange={(e) => setKeyword(e.target.value)}
        />
      </div>
      <Table
        columns={columns}
        dataSource={data.data}
        loading={loading}
        rowKey="id"
        rowSelection={{ ...rowSelection }}
        size="middle"
        pagination={{
          showTotal:(total, range) => `${range[0]}-${range[1]} of ${total} items`,
          pageSize: queryParams.size,
          current: Math.floor(queryParams.from / queryParams.size) + 1,
          total: data.total?.value || data?.total,
          showSizeChanger: true,
        }}
        onChange={handleTableChange}
      />
    </ListContainer>
  );
}

export default MappingManagement