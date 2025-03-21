import {
  Button,
  Form,
  Input,
  message,
  Spin,
  Switch,
} from 'antd';
import type { FormProps } from 'antd';
import {DataSync} from '@/components/datasource/data_sync';
import {Types} from '@/components/datasource/type';
import {updateDatasource, getDatasource} from '@/service/api/data-source'
import Yuque from '../new/yuque';
import Notion from '../new/notion';
import HugoSite from '../new/hugo_site';
import { useLoaderData } from 'react-router-dom';
import Clipboard from 'clipboard';
import { ReactSVG } from "react-svg";
import LinkSVG from '@/assets/svg-icon/link.svg'

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  const loaderData = useLoaderData();
  const datasourceID = loaderData?.id || '';
  const [loading, setLoading] = useState(false);
  const [datasource, setDatasource] = useState<any>({
    id: datasourceID,
  });
  useEffect(() => {
    if (!datasourceID) return;
    getDatasource(datasourceID).then((res)=>{
      if(res.data?.found === true){
        setDatasource(res.data._source || {});
      }
    });
  }, [datasourceID]);
  const copyRef = useRef<HTMLSpanElement | null>(null);
  const insertDocCmd = `curl -H'X-API-TOKEN: REPLACE_YOUR_API_TOKEN_HERE'  -H 'Content-Type: application/json' -XPOST ${location.origin}/datasource/${datasourceID}/_doc -d'
  {
    "title": "I am just a Coco doc that you can search",
    "summary": "Nothing but great start",
    "content": "Coco is a unified private search engien that you can trust."
  }'`;
  const [copyRefUpdated, setCopyRefUpdated] = useState(false);
  useEffect(() => {
    if (!copyRef.current) return;
    const clipboard = new Clipboard(copyRef.current as any, {
      text: () => {
        return insertDocCmd;
      }
    });
    clipboard.on('success', function(e) {
      message.success(t('common.copySuccess'));
    });
    return ()=>{
      clipboard.destroy();
    }
  }, [copyRefUpdated, insertDocCmd]);

  const onFinish: FormProps<any>['onFinish'] = (values) => {
    let config: any = {};
    switch (type) {
      case Types.Yuque:
        config = {
          ...(values.indexing_scope || {}),
          token: values.token || '',
        }
        break;
      case Types.Notion:
        config = {
          token: values.token || '',
        };
        break;
      case Types.HugoSite:
        config = {
          urls: values.urls || [],
        };
        break;
    }
    const sValues = {
      name: values.name,
      type: "connector",
      sync_enabled: !!values.sync_enabled,
      enabled: !!values.enabled,
      connector: {
        id: type,
        config: {
          ...(datasource?.connector?.config || {}),
          ...config,
        }
      }
    }
    if(values.sync_config){
      sValues.connector.config.interval = values.sync_config.interval;
      sValues.connector.config.sync_type = values.sync_config.sync_type || '';
    }
    updateDatasource(datasourceID, sValues).then((res)=>{
      if(res.data?.result == "updated"){
        setLoading(false);
        message.success(t('common.modifySuccess'))
        nav('/data-source/list', {});
      }
    })
  };
  datasource.sync_config = {
    interval: datasource?.connector?.config?.interval || '1h',
    sync_type: datasource?.connector?.config?.sync_type || ''
  } 
  const type = datasource?.connector?.id;
  if(!type){
    return null;
  }
  let isCustom  = false;
  switch (type) {
    case Types.Yuque:
      datasource.indexing_scope = datasource?.connector?.config || {};
      datasource.token = datasource?.connector?.config?.token || '';
      break;
    case Types.Notion:
      datasource.token = datasource?.connector?.config?.token || '';
      break;
    case Types.HugoSite:
      datasource.urls = datasource?.connector?.config?.urls || [];
      break;
    case Types.GoogleDrive:
      break;
    default:
      isCustom = true;
  }
  const onFinishFailed: FormProps<any>['onFinishFailed'] = (errorInfo) => {
    console.log('Failed:', errorInfo);
    setLoading(false);
  };
 
  return <div className="bg-white pt-15px pb-15px">
      <div
        className="flex-col-stretch sm:flex-1-hidden">
        <div>
          <div className='mb-4 flex items-center text-lg font-bold'>
            <div className="w-10px h-1.2em bg-[#1677FF] mr-20px"></div>
            {t('page.datasource.edit.title')}
          </div>
        </div>
        <div>
          <Spin spinning={loading}>
            <Form
                labelCol={{ span: 4 }}
                wrapperCol={{ span: 18 }}
                layout="horizontal"
                initialValues={datasource || {}}
                colon={false}
                autoComplete="off"
                onFinish={onFinish}
                onFinishFailed={onFinishFailed}
              >
                <Form.Item label={t('page.datasource.new.labels.name')} rules={[{ required: true, message: 'Please input datasource name!' }]} name="name">
                  <Input className='max-w-660px' />
                </Form.Item>
                {type === Types.Yuque && <Yuque />}
                {type === Types.Notion && <Notion />}
                {type === Types.HugoSite && <HugoSite />}
                {!isCustom ? <>
                  <Form.Item label={t('page.datasource.new.labels.data_sync')} name="sync_config">
                  <DataSync/>
                  </Form.Item>
                  <Form.Item label={t('page.datasource.new.labels.sync_enabled')} name="sync_enabled">
                  <Switch />
                </Form.Item>
                </>:<Form.Item label={"插入文档"} name="">
                    <div className='bg-gray-100 p-1em max-w-660px rounded'>
                      <div>
                        <pre className="whitespace-pre-wrap break-words" dangerouslySetInnerHTML={{__html: insertDocCmd}}></pre>
                      </div>
                      <div className='flex justify-end'><span ref={(inst)=>{copyRef.current=inst;setCopyRefUpdated(true)}} className='text-blue-500 flex items-center gap-1 cursor-pointer'><SvgIcon className="text-18px" icon="mdi:content-copy" />Copy</span></div>
                    </div>
                    <div>
                      <a href='https://docs.infinilabs.com/coco-server/main/docs/tutorials/howto_create_your_own_datasource/' target='_blank'
                       className='inline-flex items-center text-blue-500 my-10px'>
                        <span>How to create a data source</span><ReactSVG src={LinkSVG} className="m-l-4px"/>
                      </a>
                    </div>
                  </Form.Item>
                }
                <Form.Item label={t('page.datasource.new.labels.enabled')} name="enabled">
                  <Switch />
                </Form.Item>
                <Form.Item label=" ">
                  <Button type='primary' loading={loading}  htmlType="submit">{t('common.save')}</Button>
                </Form.Item>
              </Form>
          </Spin>
        </div>
      </div>
  </div>
}

export async function loader({ params }: LoaderFunctionArgs) {
 return params;
}