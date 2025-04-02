import {
    Button,
    Form,
    Input,
    message,
    Select,
  } from 'antd';
  import type { FormProps } from 'antd';
  import {createConnector, getConnectorIcons, getConnectorCategory} from '@/service/api/connector';
 import {AssetsIcons} from './assets_icons';
 import { IconSelector } from "./icon_selector";
 import { Tags } from '@/components/common/tags';
import { formatESSearchResult } from '@/service/request/es';
  
  export function Component() {
    const { t } = useTranslation();
    const nav = useNavigate();
  
    const onFinish: FormProps<any>['onFinish'] = (values) => {
      const sValues = {
        name: values.name,
        description: values.description,
        icon: values.icon,
        category: values?.category[0] || '',
        tags: values.tags,
        // "url": "http://coco.rs/connectors/google_drive", 
        assets: {
            icons: values.assets_icons,
        },
      }
      createConnector(sValues).then((res)=>{
        if(res.data?.result == "created"){
          message.success(t('common.addSuccess'))
          nav('/settings?tab=connector', {});
        }
      })
    };
    
    const onFinishFailed: FormProps<any>['onFinishFailed'] = (errorInfo) => {
      console.log('Failed:', errorInfo);
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
    return <div className="bg-white pt-15px pb-15px min-h-full">
        <div
          className="flex-col-stretch sm:flex-1-hidden">
          <div>
            <div className='mb-4 flex items-center text-lg font-bold'>
              <div className="w-10px h-1.2em bg-[#1677FF] mr-20px"></div>
              <div>{t('page.connector.new.title')}</div>
            </div>
          </div>
          <div>
           <Form
              labelCol={{ span: 4 }}
              wrapperCol={{ span: 18 }}
              layout="horizontal"
              initialValues={{assets_icons:{"default":"Google-document"}}}
              colon={false}
              autoComplete="off"
              onFinish={onFinish}
              onFinishFailed={onFinishFailed}
            >
              <Form.Item label={t('page.connector.new.labels.name')} rules={[{ required: true}]} name="name">
                <Input className='max-w-600px' />
              </Form.Item>
              <Form.Item label={t('page.connector.new.labels.category')} rules={[{ required: true}]} name="category">
               <Select options={categories.map(cate=>{return{value: cate}})} placeholder="Select or input a category" mode='tags' maxCount={1} className='max-w-600px'/>
              </Form.Item>
              <Form.Item label={t('page.connector.new.labels.icon')} name="icon" rules={[{ required: true}]}>
                <IconSelector type="connector" icons={iconsMeta} className='max-w-600px' />
              </Form.Item>
              <Form.Item label={t('page.connector.new.labels.assets_icons')} name="assets_icons">
                <AssetsIcons iconsMeta={iconsMeta}/>
              </Form.Item>
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