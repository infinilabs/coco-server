import Icon, { EllipsisOutlined, FilterOutlined, PlusOutlined } from '@ant-design/icons';
import type { TableColumnsType } from 'antd';
import { Button, Dropdown, Input, Switch, Table, Tag, Typography } from 'antd';
import type { AnyObject } from 'antd/es/_util/type';

import useQueryParams from '@/hooks/common/queryParams';

const Pipeline = () => {
  const { t } = useTranslation();
  const [keyword, setKeyword] = useState<string>();
  const [queryParams, setQueryParams] = useQueryParams();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState([]);
  const navigate = useNavigate();

  const onSearchClick = (query: string) => {
    setQueryParams(old => {
      return {
        ...old,
        query,
        t: new Date().getTime()
      };
    });
  };

  const columns: TableColumnsType<AnyObject> = [
    {
      dataIndex: 'id',
      render(value) {
        return <Typography.Link href="">{value}</Typography.Link>;
      },
      title: '名称'
    },
    {
      dataIndex: 'category',
      title: '类型'
    },
    {
      dataIndex: 'description',
      title: '描述'
    },
    {
      dataIndex: 'tags',
      render() {
        return <Switch />;
      },
      title: '启用'
    },
    {
      dataIndex: 'tags',
      title: '操作'
    }
  ];

  return (
    <ListContainer>
      <div className="mb-4 mt-4 flex items-center justify-between">
        <Input.Search
          addonBefore={<FilterOutlined />}
          className="max-w-500px"
          enterButton={t('common.refresh')}
          value={keyword}
          onChange={e => setKeyword(e.target.value)}
          onSearch={onSearchClick}
        />
        <Button
          icon={<PlusOutlined />}
          type="primary"
          onClick={() => {
            navigate('/pipeline/new');
          }}
        >
          {t('common.add')}
        </Button>
      </div>
      <Table
        columns={columns}
        dataSource={data}
        loading={loading}
        rowKey="id"
        size="middle"
        // pagination={{
        //   current: Math.floor(queryParams.from / queryParams.size) + 1,
        //   pageSize: queryParams.size,
        //   showSizeChanger: true,
        //   showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} items`,
        //   total: data?.total?.value || data?.total
        // }}
        // onChange={handleTableChange}
      />
    </ListContainer>
  );
};

export default Pipeline;
