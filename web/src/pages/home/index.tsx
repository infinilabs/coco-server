import { selectUserInfo } from "@/store/slice/auth";
import { Button, Card, Form, Input, Spin } from "antd";
import { useLoading, useRequest } from '@sa/hooks';
import { fetchServer, updateSettings } from "@/service/api/server";
import Clipboard from 'clipboard';

const SETTINGS = [
  {
    key: 'llm',
    icon: <SvgIcon icon="mdi:settings-outline"/>,
    link: '/settings?tab=llm'
  },
  {
    key: 'dataSource',
    icon: <SvgIcon icon="mdi:plus-thick"/>,
    link: '/data-source'
  },
  {
    key: 'aiAssistant',
    icon: <SvgIcon icon="mdi:plus-thick"/>,
    link: ''
  }
]

export function Component() {
    
    const userInfo = useAppSelector(selectUserInfo);
    const routerPush = useRouterPush();
    const { t } = useTranslation();
    const domRef = useRef<HTMLDivElement | null>(null);
    const [form] = Form.useForm();
    const { endLoading, loading, startLoading } = useLoading();

    const {
      defaultRequiredRule
    } = useFormRules();

    const [isNameEditing, setIsNameEditing] = useState(false)
    const [isEndpointEditing, setIsEndpointEditing] = useState(false)

    const { data, run, loading: dataLoading } = useRequest(fetchServer, {
      manual: true
    });

    const handleSubmit = async (field: 'name' | 'endpoint', callback?: () => void) => {
      const params = await form.validateFields([field]);
      startLoading()
      const { error } = await updateSettings({
        server: {
          [field]: params[field]
        }
      });
      if (error) {
        form.setFieldsValue({ [field]: data?.[field]})
      }
      endLoading()
      if (callback) callback()
    }

    const initClipboard = (text?: string) => {
      if (!domRef.current || !text) return;
  
      const clipboard = new Clipboard(domRef.current, {
        text: () => text
      });
  
      clipboard.on('success', () => {
        window.$message?.success(t('common.copySuccess'));
      });
    }

    useMount(() => {
      run();
    });

    useEffect(() => {
      initClipboard(data?.endpoint)
    }, [data?.endpoint])

    useEffect(() => {
      form.setFieldsValue({ name: data?.name, endpoint: data?.endpoint })
    }, [JSON.stringify(data)])

    return (
      <Spin spinning={dataLoading || loading}>
        <Card className="m-b-12px px-32px py-40px" classNames={{ body: "!p-0" }}>
          <div className={`flex ${isNameEditing ? '[align-items:self-end]' : 'items-center'} m-b-48px`}>
            <div className={`h-40px leading-40px m-r-16px text-32px color-#333 relative ${isNameEditing ? 'w-344px' : ''}`}>
              {
                isNameEditing && (
                  <Form
                    className={`w-100% z-1 absolute top-2px left-0`}
                    form={form}
                  >
                    <Form.Item className="m-b-0" name="name" rules={[defaultRequiredRule]}>
                      <Input autoFocus className="w-100% h-40px [min-width:344px] m-r-16px "/>
                    </Form.Item>
                  </Form>
                )
              }
              {
                data?.name ? data?.name : <span>{t('page.home.server.title',  { user: userInfo.name })}</span>
              }
            </div>
            <Button 
              onClick={() => {
                if (isNameEditing) {
                  handleSubmit('name', () => setIsNameEditing(!isNameEditing))
                } else {
                  setIsNameEditing(!isNameEditing)
                }
              }} 
              type="link" 
              className="w-40px h-40px bg-#F7F9FC !hover:bg-#F7F9FC rounded-12px p-0">
              <SvgIcon className="text-24px" icon={isNameEditing ? "mdi:content-save" : "mdi:square-edit-outline"}/>
            </Button>
          </div>
          <div className="m-b-16px text-20px color-#333">
            {t('page.home.server.address')}
          </div>
          <div className="flex m-b-16px">
            <div className="w-400px h-48px color-#333 m-r-8px bg-#F7F9FC leading-48px rounded-4px relative p-r-30px"> 
              {
                isEndpointEditing ? (
                  <Form
                    form={form}
                  >
                    <Form.Item name="endpoint" rules={[defaultRequiredRule]}>
                      <Input autoFocus className="h-48px p-r-32px w-[calc(100%+30px)]" onBlur={(e) => {
                        if (e.relatedTarget?.id !== 'endpoint-save') {
                          form.setFieldsValue({ endpoint: data?.endpoint})
                        }
                      }}/>
                    </Form.Item>
                  </Form>
                ) : <div className="p-l-11px">{data?.endpoint}</div>
              }
              <Button 
                id="endpoint-save"
                onClick={() => {
                  if (isEndpointEditing) {
                    handleSubmit('endpoint', () => setIsEndpointEditing(!isEndpointEditing))
                  } else {
                    setIsEndpointEditing(!isEndpointEditing)
                  }
                }} 
                type="link" 
                className="w-30px h-48px rounded-12px p-0 absolute top-0 right-0 z-1">
                <SvgIcon className="text-24px" icon={isEndpointEditing ? "mdi:content-save" : "mdi:square-edit-outline"}/>
              </Button>
            </div>
            <div ref={domRef} >
              <Button className="w-100px h-48px" type="primary"><SvgIcon className="text-24px" icon="mdi:content-copy" /></Button>
            </div>
          </div>
          <div className="m-b-16px color-#888">
            {t('page.home.server.addressDesc')}
          </div>
          <Button type="link" className="px-0" onClick={() => window.open('https://coco.rs/#install', '_blank')}>
            {t('page.home.server.downloadCocoAI')} <SvgIcon className="text-16px" icon="mdi:external-link"/>
          </Button>
        </Card>
        <Card className="p-32px" classNames={{ body: "flex gap-32px justify-start !p-0 -mx-32px" }}>
          {
            SETTINGS.map((item) => (
              <div key={item.key} className="basis-1/3">
                <div className="text-20px color-#333 m-b-16px">{t(`page.home.settings.${item.key}`)}</div>
                <div className="color-#888 m-b-45px">{t(`page.home.settings.${item.key}Desc`)}</div>
                <Button onClick={() => item.link && routerPush.routerPush(item.link)} type="primary" className="w-40px h-40px rounded-12px text-24px p-0">{item.icon}</Button>
              </div>
            ))
          }
        </Card>
      </Spin>
    );
}
