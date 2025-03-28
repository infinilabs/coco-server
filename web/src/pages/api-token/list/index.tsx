import Search from "antd/es/input/Search";
import Icon, { FilterOutlined, PlusOutlined, ExclamationCircleOutlined, EllipsisOutlined } from "@ant-design/icons";
import { Button, Table, Modal, Dropdown, Form, Input, Spin, message } from "antd";
import type { TableColumnsType, MenuProps } from "antd";
import {getTokens, createToken, renameToken, deleteToken} from '@/service/api'
import { useForm } from "antd/es/form/Form";
import Clipboard from 'clipboard';

type APIToken = Api.APIToken.APIToken;
const { confirm } = Modal;

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();

  const items: MenuProps["items"] = [
    {
      label: t('common.rename'),
      key: "1",
    },
    {
      label: t('common.delete'),
      key: "2",
    },
  ];

  const [renameState, setRenameState] = useState({
    open: false,
    tokenInfo: {},
  });

  const onMenuClick = ({key, record}: any)=>{
    switch(key){
      case "2":
        confirm({
          icon: <ExclamationCircleOutlined />,
          title: t('common.tip'),
          content: t('page.apitoken.delete.confirm'),
          onOk() {
             //delete datasource by datasource id
            deleteToken(record.id).then((res)=>{
              if(res.data?.result === "deleted"){
                message.success(t('common.deleteSuccess'))
              }
              //reload data
              setReqParams((old)=>{
                return {
                  ...old,
                }
              })
            });
          },
          onCancel() {
          },
        });
       
        break;
      case "1":
        setRenameState({
          open: true,
          tokenInfo: record,
        });
        break;
    }
  }
  const columns: TableColumnsType<APIToken> = [
    {
      title: t('page.apitoken.columns.name'),
      dataIndex: "name",
      minWidth: 100,
    },
    {
      title: "Token",
      dataIndex: "access_token",
      minWidth: 200,
    },
    {
      title: t('page.apitoken.columns.expire_in'),
      minWidth: 100,
      dataIndex: "expire_in",
      render: (value: number)=>{
        return value ? new Date(value * 1000).toISOString() : ''
      },
    },
    {
      title: t('common.operation'),
      fixed: 'right',
      width: "90px",
      render: (_, record) => {
        return <Dropdown menu={{ items, onClick:({key})=>onMenuClick({key, record}) }}>
          <EllipsisOutlined/>
        </Dropdown>
      },
    },
  ];

  const [data, setData] = useState([]);
  const [loading, setLoading] = useState(false);
  const [reqParams, setReqParams] = useState({
    query: '',
    t: new Date().getTime(),
  })

  const fetchData = () => {
    setLoading(true);
    getTokens().then(({ data }) => {
        setData(data || []);
        setLoading(false);
      });
    };
  
  useEffect(fetchData, [reqParams]);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [modelForm] = useForm();
  const onAddClick = ()=>{
    // if(data.length >= 5){
    //   message.warning(t('page.apitoken.create.limit'));
    //   return;
    // }
    const tokenName = generateApiTokenName();
    setIsModalOpen(true);
    modelForm.setFieldsValue({name: tokenName});
  }

  const onSearchClick = (query: string)=>{
    setReqParams((old)=>{
      return {
        query: query,
        t: new Date().getTime(),
      }
    })
  }

  const initialCreateState = {
    loading: false,
    step: 1,
    token: '',
  };
  const [createState, setCreateState] = useState(initialCreateState);

  const buttonRef = useRef(null);
  useEffect(() => {
    if (!buttonRef.current) return;
    const clipboard = new Clipboard(buttonRef.current, {
      text: () => {
        return createState.token
      }
    });
    clipboard.on('success', () => {
      window.$message?.success(t('common.copySuccess'));
    });

    return () => clipboard.destroy();
  }, [createState.token, buttonRef.current, isModalOpen])

  const onModalOkClick = ()=>{
    if(createState.step === 1){
      modelForm.validateFields().then((values)=>{
        setCreateState((old)=>{
          return {
            ...old,
            loading: true,
          }
        });
        createToken(values.name).then(({data})=>{
          setCreateState((old)=>{
            return {
              ...old,
              loading: false,
              step: 2,
              token: data?.access_token || '',
            }
          });
        }).catch(()=>{
          setCreateState((old)=>{
            return {
              ...old,
              loading: false,
            }
          });
        });
      })
    }
  }
  const onModalCancel = ()=>{
    setIsModalOpen(false);
    if (createState.step === 2){
      setReqParams((old)=>{
        return {
          ...old,
        }
      })
    }
    setCreateState(initialCreateState);
  }

  return (
    <div className="h-full min-h-500px flex-col-stretch gap-16px overflow-hidden lt-sm:overflow-auto">
    <ACard
      bordered={false}
      className="flex-col-stretch sm:flex-1-hidden card-wrapper"
    >
      <div className="mb-4 mt-4 flex items-center justify-between">
        <Search
          addonBefore={<FilterOutlined />}
          className="max-w-500px"
          onSearch={onSearchClick}
          enterButton={t("common.refresh")}
        ></Search>
        <Button type='primary' icon={<PlusOutlined/>}  onClick={onAddClick}>{t('common.add')}</Button>
      </div>
      <Table<APIToken>
          rowKey="id"
          loading={loading}
          size="middle"
          columns={columns}
          dataSource={data}
          pagination={{ 
            showTotal:(total, range) => `${range[0]}-${range[1]} of ${total} items`,
            defaultPageSize: 10,
            defaultCurrent: 1,
            showSizeChanger: true,
          }}
        />
      <Modal title={t('common.create')+" API Token"}
        open={isModalOpen} 
        onOk={onModalOkClick} 
        okText={t('common.create')} 
        footer={(_, { OkBtn, CancelBtn }) => (
          <>
            {createState.step === 1 &&  <><CancelBtn /><OkBtn /></>}
            {createState.step === 2 &&  <><Button onClick={onModalCancel}>{t('common.close')}</Button>
            <Button ref={buttonRef} type="primary">{t('common.copy')}</Button></>
            }
          </>
        )}
        onCancel={onModalCancel}>
        <Spin spinning={createState.loading}>
        {createState.step === 1 && 
          <Form form={modelForm} layout="vertical" className="my-2em">
            <Form.Item label={<span className="text-gray-500">{t('page.apitoken.columns.name')}</span>} name="name">
              <Input/>
            </Form.Item>
          </Form>
        }
        {createState.step === 2 && <div>
          <div className="text-gray-500 font-size-[12px] my-[15px]">{t('page.apitoken.create.store_desc')}</div>
          <div className="text-gray-500 bg-gray-100 leading-[1.4em] py-[3px] pl-1em rounded">{createState.token}</div>
        </div>}
        </Spin>
      </Modal>
      <RenameComponent tokenInfo={renameState.tokenInfo} open={renameState.open} 
        onOkClick={()=>{
          setRenameState({
            open: false,
            tokenInfo: {},
          });
          setReqParams((old)=>{
            return {
              ...old,
            }
          })
        }}
        onCancelClick={()=>{
          setRenameState({
            open: false,
            tokenInfo: {},
          });
      }}/>
      </ACard>
      </div>
  )
}

function generateApiTokenName(prefix = "token") {
  const timestamp = Date.now(); // Current timestamp in milliseconds
  const randomString = Math.random().toString(36).substring(2, 10); // Random alphanumeric string
  return `${prefix}_${timestamp}_${randomString}`;
}


const RenameComponent = ({
  tokenInfo = {},
  onOkClick = ()=>{},
  open = false,
  onCancelClick = ()=>{},
})=>{
  const { t } = useTranslation();
  const [form] = useForm();
  const [loading, setLoading] = useState(false);

  const onModalOkClick = ()=>{
    form.validateFields().then((values)=>{
      setLoading(true);
      renameToken(tokenInfo.id, values.name).then(()=>{
        setLoading(false);
        onOkClick();
      }).catch(()=>{
        setLoading(false);
      });
    })
  }
  return (<Modal title={t('common.rename')+" API Token"}
  open={open} 
  onOk={onModalOkClick} 
  onCancel={onCancelClick}>
  <Spin spinning={loading}>
    <Form form={form} layout="vertical" className="my-2em">
      <Form.Item label={<span className="text-gray-500">{t('page.apitoken.columns.name')}</span>} name="name">
        <Input defaultValue={tokenInfo.name}/>
      </Form.Item>
    </Form>
  </Spin>
</Modal>)
}