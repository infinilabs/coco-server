import { FilterOutlined, PlusOutlined, EllipsisOutlined, ExclamationCircleOutlined} from '@ant-design/icons';
import { Button, Dropdown, Table, message, Modal, Input, Switch } from 'antd';
import { formatESSearchResult } from '@/service/request/es';
import { deleteIntegration, fetchIntegrations, updateIntegration } from '@/service/api/integration';
import { useLoading } from '@sa/hooks';

const { confirm } = Modal;

export function Component() {
  const { t } = useTranslation();

  const { tableWrapperRef } = useTableScroll();

  const nav = useNavigate();

  const [data, setData] = useState({
    data: [],
    total: 0,
  });
  const { endLoading, loading, startLoading } = useLoading();
  

  const [reqParams, setReqParams] = useState({
    query: '',
    from: 0, 
    size: 10,
  })

  const fetchData = async (reqParams) => {
    startLoading();
    const res = await fetchIntegrations(reqParams);
    const newData = formatESSearchResult(res.data);
    setData(newData);
    endLoading();
  };

  const handleTableChange = (pagination) => {
    setReqParams((params)=>{
      return {
        ...params,
        size: pagination.pageSize,
        from: (pagination.current-1) * pagination.pageSize,
      }
    })
  };

  const onRefreshClick = (query: string)=>{
    setReqParams((oldParams)=>{
      return {
        ...oldParams,
        query: query,
        form: 0,
      }
    })
  }

  const handleDelete = async (id) => {
    startLoading()
    const res = await deleteIntegration(id)
    if(res.data?.result === "deleted"){
      message.success(t('common.deleteSuccess'))
    }
    fetchData(reqParams)
    endLoading()
  }

  const handleEnabled = async (record) => {
    startLoading()
    const { _index, _type, ...rest } = record;
    const res = await updateIntegration(rest)
    if(res.data?.result === "updated"){
      message.success(t('common.updateSuccess'))
    }
    fetchData(reqParams)
    endLoading()
  }

  const columns = [
    {
      title: t('page.integration.columns.name'),
      dataIndex: "name",
    },
    {
      title: t('page.integration.columns.type'),
      dataIndex: "type",
      render: (value) => t(`page.integration.form.labels.type_${value}`)
    },
    {
      title: t('page.integration.columns.description'),
      dataIndex: "description",
    },
    {
      title: t('page.integration.columns.datasource'),
      dataIndex: "datasource",
      render: (value, record) => {
        return value?.includes('*') ? '*' : (value?.length || 0)
      }
    },
    {
      title: t('page.integration.columns.enabled'),
      dataIndex: "enabled",
      render: (_, record) => {
        return (
          <Switch 
            size="small" 
            checked={record.enabled} 
            onChange={(checked) => {
              confirm({
                icon: <ExclamationCircleOutlined />,
                title: t('common.tip'),
                content: t(`page.integration.update.${checked ? 'enable' : 'disable'}_confirm`, { name: record.name }),
                onOk() {
                  handleEnabled({ ...record, enabled: checked})
                },
              });
            }}
          />
        )
      }
    },
    {
      title: t('common.operation'),
      fixed: 'right',
      width: "90px",
      render: (_, record) => {

        const items = [
          {
            label: t('common.edit'),
            key: "edit",
          },
          {
            label: t('common.delete'),
            key: "delete",
          },
        ];

        const onMenuClick = ({key, record}: any)=>{
          switch(key){
            case "edit":
              nav(`/integration/edit/${record.id}`, {state:record});
              break;
            case "delete":
              confirm({
                icon: <ExclamationCircleOutlined />,
                title: t('common.tip'),
                content: t('page.integration.delete.confirm', { name: record.name }),
                onOk() {
                  handleDelete(record.id)
                },
              });
              break;
          }
        }
        return <Dropdown menu={{ items, onClick:({key})=>onMenuClick({key, record}) }}>
          <EllipsisOutlined/>
        </Dropdown>
      },
    },
  ];
  // rowSelection object indicates the need for row selection
  const rowSelection = {
    onChange: (selectedRowKeys: React.Key[], selectedRows) => {
    },
    getCheckboxProps: (record) => ({
      name: record.name,
    }),
  };



  useEffect(() => {
    fetchData(reqParams)
  }, [reqParams]);

  return (
    <div className="h-full min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
      <ACard
        bordered={false}
        className="flex-col-stretch sm:flex-1-hidden card-wrapper"
        ref={tableWrapperRef}
      >
      <div className='mb-4 mt-4 flex items-center justify-between'>
        <Input.Search addonBefore={<FilterOutlined />} className='max-w-500px' onSearch={onRefreshClick} enterButton={ t('common.refresh')}></Input.Search>
        <Button type='primary' icon={<PlusOutlined/>}  onClick={() => nav(`/integration/new`)}>{t('common.add')}</Button>
      </div>
      <Table
          rowKey="id"
          loading={loading}
          size="middle"
          rowSelection={{ ...rowSelection }}
          columns={columns}
          dataSource={data.data}
          pagination={
            {
              showTotal:(total, range) => `${range[0]}-${range[1]} of ${total} items`,
              defaultPageSize: 10,
              defaultCurrent: 1,
              total: data.total?.value || data?.total,
              showSizeChanger: true,
            }
          }
          onChange={handleTableChange}
        />
      </ACard>
    </div>
  );
}
