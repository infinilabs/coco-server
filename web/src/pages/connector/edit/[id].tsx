import {
  Button,
  Form,
  Input,
  message,
  Select,
} from 'antd';
import type { FormProps } from 'antd';
import {updateConnector, getConnectorIcons, getConnectorCategory} from '@/service/api/connector';
 import {AssetsIcons} from '../new/assets_icons';
 import { IconSelector } from "../new/icon_selector";
 import { Tags } from '@/components/common/tags';
 import InfiniIcon from '@/components/common/icon';
 import { formatESSearchResult } from '@/service/request/es';

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();
  let {state: initialConnector} = useLocation();
  const connectorID = initialConnector?.id || '';
  initialConnector = {
    ...initialConnector,
    assets_icons: initialConnector.assets?.icons || {},
    ...(initialConnector.config || {}),
  }
  const [loading, setLoading] = useState(false);

  const onFinish: FormProps<any>['onFinish'] = (values) => {
    const category = typeof values.category === 'string' ? values.category : (values.category[0] || '');
    const sValues = {
      name: values.name,
      description: values.description,
      icon: values.icon,
      category: category,
      tags: values.tags,
      assets: {
          icons: values.assets_icons,
      },
      config: {}
    }
    if(connectorID === "google_drive"){
      sValues.config = {
        client_id: values.client_id,
        client_secret: values.client_secret,
        redirect_url: values.redirect_url,
        auth_url: values.auth_url,
        token_url: values.token_url,
      }
    }
  
    updateConnector(connectorID, sValues).then((res)=>{
      if(res.data?.result == "updated"){
        message.success(t('common.updateSuccess'))
        nav('/settings?tab=connector', {});
      }
    })
  };
  const [iconsMeta, setIconsMeta] = useState([]);
  useEffect(() => {
    getConnectorIcons().then((res)=>{
      if(res.data?.length > 0){
        setIconsMeta(res.data);
      }
    });
  }, []);

  const [categories, setCategories] = useState([]);
  useEffect(() => {
    getConnectorCategory().then(({data})=>{
      if(!data?.error){
        const newData = formatESSearchResult(data);
        const cates = newData.aggregations.categories.buckets.map((item: any)=>{
          return item.key;
        });
        setCategories(cates);
      }
    });
  }, []);
  
  const onFinishFailed: FormProps<any>['onFinishFailed'] = (errorInfo) => {
    console.log('Failed:', errorInfo);
  };
  const { defaultRequiredRule, formRules } = useFormRules();
  return <div className="bg-white pt-15px pb-15px min-h-full">
      <div
        className="flex-col-stretch sm:flex-1-hidden">
        <div>
          <div className='mb-4 flex items-center text-lg font-bold'>
            <div className="w-10px h-1.2em bg-[#1677FF] mr-20px"></div>
            <div>{t('page.connector.edit.title')}</div>
          </div>
        </div>
        <div>
         <Form
            labelCol={{ span: 4 }}
            wrapperCol={{ span: 18 }}
            layout="horizontal"
            initialValues={initialConnector}
            colon={false}
            autoComplete="off"
            onFinish={onFinish}
            onFinishFailed={onFinishFailed}
          >
            <Form.Item label={t('page.connector.new.labels.name')} rules={[{ required: true, message: 'Please input connector name!' }]} name="name">
              <Input className='max-w-600px' />
            </Form.Item>
            <Form.Item label={t('page.connector.new.labels.category')} rules={[{ required: true}]} name="category">
              <Select options={categories.map(cate=>{return{value: cate}})} mode='tags' maxTagCount={1} className='max-w-600px'/>
            </Form.Item>
            <Form.Item label={t('page.connector.new.labels.icon')} rules={[{ required: true}]} name="icon">
              {initialConnector.builtin === true ? <InfiniIcon width="2em" height="2em" className="w-2em h-2em" src={initialConnector.icon} />:
              <IconSelector icons={iconsMeta} className='max-w-200px' readonly={initialConnector.builtin === true} />}
            </Form.Item>
            <Form.Item label={t('page.connector.new.labels.assets_icons')} name="assets_icons">
            {initialConnector.builtin === true ? <AssetsIconsView/> :<AssetsIcons iconsMeta={iconsMeta} readonly={initialConnector.builtin === true}/>}
            </Form.Item>
            {connectorID === "google_drive" && <>
              <Form.Item
                name="client_id"
                label={t('page.connector.new.labels.client_id')}
                rules={[defaultRequiredRule]}
              >
              <Input className='max-w-600px'  />
              </Form.Item>
                <Form.Item
                  name="client_secret"
                  label={t('page.connector.new.labels.client_secret')}
                  rules={[defaultRequiredRule]}
              >
                  <Input className='max-w-600px'  />
              </Form.Item>
              <Form.Item
                  name="redirect_url"
                  label={t('page.connector.new.labels.redirect_url')}
                  rules={formRules.endpoint}
              >
                  <Input className='max-w-600px'  />
              </Form.Item>
              <Form.Item
                  name="auth_url"
                  label={t('page.connector.new.labels.auth_url')}
                  rules={formRules.endpoint}
              >
                  <Input className='max-w-600px'  />
              </Form.Item>
              <Form.Item
                  name="token_url"
                  label={t('page.connector.new.labels.token_url')}
                  rules={formRules.endpoint}
              >
                  <Input className='max-w-600px'  />
              </Form.Item>
            </>}
            <Form.Item label={t('page.connector.new.labels.description')} name="description">
              <Input.TextArea/>
            </Form.Item>
            <Form.Item label={t('page.connector.new.labels.tags')} name="tags">
              <Tags />
            </Form.Item>
            <Form.Item label=" ">
              <Button type='primary'  htmlType="submit">{t('common.save')}</Button>
            </Form.Item>
          </Form>

        </div>
      </div>
  </div>
}

const AssetsIconsView = ({value={}})=>{
  const { t } = useTranslation();
  const icons = Object.keys(value).map((key)=>{
    return {
      type: key,
      icon: value[key],
    }
  });
  return <div className='flex flex-col'>
    <div className='flex flex-wrap gap-10px'>
      {icons.map((icon, index)=>{
        return <div key={index} className='flex items-center gap-5px'>
          <InfiniIcon width="1em" height="1em" className="w-1em h-1em" src={icon.icon} />
          <span>{icon.type}</span>
        </div>
      })}
    </div>
  </div>
}