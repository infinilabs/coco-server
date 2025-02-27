import { Suspense, lazy } from 'react';

import TableHeaderOperation from '@/components/advanced/TableHeaderOperation';
import { enableStatusRecord, userGenderRecord } from '@/constants/business';
import { ATG_MAP } from '@/constants/common';
import { fetchDataSourceList } from '@/service/api';
import Search from 'antd/es/input/Search';
import { FilterOutlined, PlusOutlined } from '@ant-design/icons';
import { Button } from 'antd';

// import UserSearch from './modules/UserSearch';
//
// const UserOperateDrawer = lazy(() => import('./modules/UserOperateDrawer'));

const tagUserGenderMap: Record<Api.SystemManage.UserGender, string> = {
  1: 'processing',
  2: 'error'
};

export function Component() {
  const { t } = useTranslation();

  const { scrollConfig, tableWrapperRef } = useTableScroll();

  const nav = useNavigate();

  const isMobile = useMobile();

  const { columnChecks, data, run, searchProps, setColumnChecks, tableProps } = useTable(
    {
      apiFn: fetchDataSourceList,
      apiParams: {
        current: 1,
        size: 10,
      },
      columns: () => [
        {
          align: 'center',
          dataIndex: 'name',
          key: 'name',
          title: t('page.datasource.columns.name'),
          width: 64
        },
        {
          align: 'center',
          dataIndex: 'type',
          key: 'type',
          minWidth: 100,
          title: t('page.datasource.columns.type')
        },
        {
          align: 'center',
          dataIndex: 'sync_policy',
          key: 'sync_policy',
          minWidth: 100,
          title: t('page.datasource.columns.sync_policy')
        },
        {
          align: 'center',
          dataIndex: 'latest_sync_time',
          key: 'latest_sync_time',
          title: t('page.datasource.columns.latest_sync_time'),
          width: 200
        },
        {
          align: 'center',
          dataIndex: 'sync_status',
          key: 'sync_status',
          minWidth: 200,
          title: t('page.datasource.columns.sync_status')
        },
        {
          align: 'center',
          dataIndex: 'enabled',
          key: 'enabled',
          minWidth: 200,
          title: t('page.datasource.columns.enabled')
        },
        {
          align: 'center',
          key: 'operate',
          render: (_, record) => (
            <div className="flex-center gap-8px">
              <AButton
                ghost
                size="small"
                type="primary"
                onClick={() => edit(record.id)}
              >
                {t('common.edit')}
              </AButton>
              <AButton
                size="small"
                onClick={() => nav(`/manage/user-detail/${record.id}`)}
              >
                详情
              </AButton>
              <APopconfirm
                title={t('common.confirmDelete')}
                onConfirm={() => handleDelete(record.id)}
              >
                <AButton
                  danger
                  size="small"
                >
                  {t('common.delete')}
                </AButton>
              </APopconfirm>
            </div>
          ),
          title: t('common.operate'),
          width: 195
        }
      ]
    },
    { showQuickJumper: true }
  );

  const { checkedRowKeys, generalPopupOperation, handleAdd, handleEdit, onBatchDeleted, onDeleted, rowSelection } =
    useTableOperate(data, run, async (res, type) => {
      if (type === 'add') {
        // add request 调用新增的接口
        console.log(res);
      } else {
        // edit request 调用编辑的接口
        console.log(res);
      }
    });

  async function handleBatchDelete() {
    // request
    console.log(checkedRowKeys);
    onBatchDeleted();
  }

  function handleDelete(id: number) {
    // request
    console.log(id);

    onDeleted();
  }

  function edit(id: number) {
    handleEdit(id);
  }
  return (
    <div className="h-full min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
      <ACard
        bordered={false}
        className="flex-col-stretch sm:flex-1-hidden card-wrapper"
        ref={tableWrapperRef}
        // title={t('page.manage.user.title')}
        // extra={
        //   <TableHeaderOperation
        //     add={handleAdd}
        //     columns={columnChecks}
        //     disabledDelete={checkedRowKeys.length === 0}
        //     loading={tableProps.loading}
        //     refresh={run}
        //     setColumnChecks={setColumnChecks}
        //     onDelete={handleBatchDelete}
        //   />
        // }
      >
      <div className='mb-4 mt-4 flex items-center justify-between'>
        <Search addonBefore={<FilterOutlined />} className='max-w-500px' placeholder="input search text" enterButton="Refresh"></Search>
        <Button type='primary' icon={<PlusOutlined/>}  onClick={() => nav(`/data-source/new`)}>New</Button>
      </div>
        <ATable
          rowSelection={rowSelection}
          scroll={scrollConfig}
          size="small"
          {...tableProps}
        />
        <Suspense>
          {/*<UserOperateDrawer {...generalPopupOperation} />*/}
        </Suspense>
      </ACard>
    </div>
  );
}
