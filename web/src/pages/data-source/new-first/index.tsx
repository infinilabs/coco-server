import Icon, { FilterOutlined, PlusOutlined } from '@ant-design/icons';
import { Button, Image, List } from 'antd';
import Search from 'antd/es/input/Search';
import { ReactSVG } from 'react-svg';

import CloudDiskSVG from '@/assets/svg-icon/cloud_disk.svg';
import CreatorSVG from '@/assets/svg-icon/creator.svg';
import WebsiteSVG from '@/assets/svg-icon/website.svg';
import InfiniIcon from '@/components/common/icon';
import { searchConnector } from '@/service/api/connector';
import { formatESSearchResult } from '@/service/request/es';
import useQueryParams from '@/hooks/common/search';

const ConnectorCategory = {
  CloudStorage: 'cloud_storage',
  Website: 'website'
};

export function Component() {
  const [queryParams, setQueryParams] = useQueryParams();
  const { t } = useTranslation();
  const nav = useNavigate();
  const onAddClick = (key: string) => {
    nav(`/data-source/new/?type=${key}`);
  };
  const [data, setData] = useState({
    data: [],
    total: 0
  });
  const [loading, setLoading] = useState(false);

  const [keyword, setKeyword] = useState();
  
  const fetchData = () => {
    setLoading(true);
    searchConnector(queryParams)
      .then(data => {
        const newData = formatESSearchResult(data.data);
        setData(newData);
      })
      .finally(() => {
        setLoading(false);
      });
  };

  useEffect(fetchData, []);

  useEffect(() => {
    setKeyword(queryParams.query)
  }, [queryParams.query])

  const onSearchClick = (query: string) => {
    setQueryParams({
      ...queryParams,
      query,
      t: new Date().getTime()
    });
  };
  const onPageChange = (page: number, pageSize: number) => {
    setQueryParams((oldParams: any) => {
      return {
        ...oldParams,
        from: (page - 1) * pageSize,
        size: pageSize
      };
    });
  };
  return (
    <div className="min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
      <ACard
        bordered={false}
        className="flex-col-stretch sm:flex-1-hidden card-wrapper"
      >
        <div className="mb-4 mt-4 flex items-center justify-between">
          <Search
            value={keyword} 
            onChange={(e) => setKeyword(e.target.value)} 
            addonBefore={<FilterOutlined />}
            className="max-w-500px"
            enterButton={t('common.refresh')}
            onSearch={onSearchClick}
          />
        </div>
        <List
          dataSource={data.data}
          grid={{ column: 3, gutter: 16 }}
          pagination={{
            pageSize: queryParams.size,
            current: queryParams.from + 1,
            onChange: onPageChange,
            showSizeChanger: true,
            showTotal: (total, range) => `${range[0]}-${range[1]} of ${total} items`,
            total: data.total || 0
          }}
          renderItem={connector => (
            <List.Item>
              <div className="group relative border border-[var(--ant-color-border)] rounded-[8px] p-1em hover:bg-[var(--ant-control-item-bg-hover)]">
                <Button
                  className="absolute left-1/2 top-1/2 hidden transform group-hover:block -translate-x-1/2 -translate-y-1/2"
                  type="primary"
                  onClick={() => {
                    onAddClick(connector.id);
                  }}
                >
                  <PlusOutlined className="text-1.4em font-bold" />
                </Button>
                <div className="flex items-center gap-8px">
                  <IconWrapper className="w-40px h-40px">
                    <InfiniIcon
                      className="text-2em"
                      height="2em"
                      src={connector.icon}
                      width="2em"
                    />
                  </IconWrapper>
                  <span className="font-size-1.2em">{connector.name}</span>
                  {/* <Icon component={connector.icon} className="font-size-2.6em"/> <span className="font-size-1.2em">{connector.name}</span> */}
                </div>
                <div className="my-1em flex items-center gap-2em text-gray-500">
                  {connector.category === ConnectorCategory.Website && (
                    <div className="flex items-center gap-3px">
                      {' '}
                      <ReactSVG
                        className="font-size-1.2em"
                        src={WebsiteSVG}
                      />{' '}
                      <span>Website</span>
                    </div>
                  )}
                  {connector.category === ConnectorCategory.CloudStorage && (
                    <div className="flex items-center gap-3px">
                      {' '}
                      <ReactSVG
                        className="font-size-1.2em"
                        src={CloudDiskSVG}
                      />{' '}
                      <span>Cloud Storage</span>
                    </div>
                  )}
                  <div className="flex items-center gap-3px">
                    {' '}
                    <ReactSVG
                      className="font-size-1.2em"
                      src={CreatorSVG}
                    />{' '}
                    <span>{connector.author || 'INFINI Labs'}</span>
                  </div>
                </div>
                <div className="h-45px overflow-hidden text-ellipsis text-gray-500">{connector.description}</div>
                <div className="h-33px overflow-scroll">
                  <div className="mt-10px flex flex-wrap gap-5px text-12px text-gray-500">
                    {(connector.tags || []).map((tag, index) => (
                      <div
                        className="border border-[var(--ant-color-border)] rounded px-5px"
                        key={index}
                      >
                        {tag}
                      </div>
                    ))}
                  </div>
                </div>
              </div>
            </List.Item>
          )}
        />
      </ACard>
    </div>
  );
}
