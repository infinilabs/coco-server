import Icon, {
  CheckCircleFilled,
  ClockCircleFilled,
  CloseCircleFilled,
  EllipsisOutlined,
  FilterOutlined,
  PlusOutlined
} from '@ant-design/icons';
import type { TableColumnsType } from 'antd';
import { Button, Dropdown, Input, Table, Tag, Typography } from 'antd';
import type { AnyObject } from 'antd/es/_util/type';
import dayjs from 'dayjs';

import useQueryParams from '@/hooks/common/queryParams';

const Runs = () => {
  const { t } = useTranslation();
  const [keyword, setKeyword] = useState<string>();
  const [queryParams, setQueryParams] = useQueryParams();
  const [loading, setLoading] = useState(false);
  const [data, setData] = useState([]);

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
      title: 'ID'
    },
    {
      dataIndex: 'category',
      render(value) {
        return <Typography.Link href="">{value}</Typography.Link>;
      },
      title: 'Pipeline'
    },
    {
      dataIndex: 'description',
      render() {
        return '长期任务';
      },
      title: '类型'
    },
    {
      dataIndex: 'tags',
      render(value) {
        return dayjs(value).format('YYYY-MM-DD HH:mm');
      },
      title: '最近执行时间'
    },
    {
      dataIndex: 'tags',
      title: '耗时'
    },
    {
      dataIndex: 'status',
      render() {
        // 进行中
        return <ClockCircleFilled className="text-4 text-primary" />;
        // 成功
        return <CheckCircleFilled className="text-4 text-success" />;
        // 失败
        return <CloseCircleFilled className="text-4 text-error" />;
      },
      title: '状态'
    },
    {
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
          type="primary"
          // onClick={onAddClick}
        >
          {t('common.operate')}
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

export default Runs;
