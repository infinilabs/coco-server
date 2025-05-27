import { Button, Form, Input, Select, message } from 'antd';
import type { FormProps } from 'antd';

import { Tags } from '@/components/common/tags';
import { createConnector, getConnectorCategory, getConnectorIcons } from '@/service/api/connector';
import { formatESSearchResult } from '@/service/request/es';

import { AssetsIcons } from './assets_icons';
import { IconSelector } from './icon_selector';

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();

  const onFinish: FormProps<any>['onFinish'] = values => {
    const sValues = {
      // "url": "http://coco.rs/connectors/google_drive",
      assets: {
        icons: values.assets_icons
      },
      category: values?.category?.[0] || '',
      description: values.description,
      icon: values.icon,
      name: values.name,
      tags: values.tags
    };
    createConnector(sValues).then(res => {
      if (res.data?.result == 'created') {
        message.success(t('common.addSuccess'));
        nav('/settings?tab=connector', {});
      }
    });
  };

  const onFinishFailed: FormProps<any>['onFinishFailed'] = errorInfo => {
    console.log('Failed:', errorInfo);
  };
  const [iconsMeta, setIconsMeta] = useState([]);
  useEffect(() => {
    getConnectorIcons().then(res => {
      if (res.data?.length > 0) {
        setIconsMeta(res.data);
      }
    });
  }, []);
  const [categories, setCategories] = useState([]);
  useEffect(() => {
    getConnectorCategory().then(({ data }) => {
      if (!data?.error) {
        const newData = formatESSearchResult(data);
        const cates = newData.aggregations.categories.buckets.map((item: any) => {
          return item.key;
        });
        setCategories(cates);
      }
    });
  }, []);
  return (
    <div className="h-full min-h-500px">
      <ACard
        bordered={false}
        className="sm:flex-1-auto min-h-full flex-col-stretch card-wrapper"
      >
        <div className="mb-30px ml--16px flex items-center text-lg font-bold">
          <div className="mr-20px h-1.2em w-10px bg-[#1677FF]" />
          <div>{t('page.connector.new.title')}</div>
        </div>
        <Form
          autoComplete="off"
          colon={false}
          initialValues={{ assets_icons: { default: 'font_Google-document' } }}
          labelCol={{ span: 4 }}
          layout="horizontal"
          wrapperCol={{ span: 18 }}
          onFinish={onFinish}
          onFinishFailed={onFinishFailed}
        >
          <Form.Item
            label={t('page.connector.new.labels.name')}
            name="name"
            rules={[{ required: true }]}
          >
            <Input className="max-w-600px" />
          </Form.Item>
          <Form.Item
            label={t('page.connector.new.labels.category')}
            name="category"
            rules={[{ required: true }]}
          >
            <Select
              className="max-w-600px"
              maxCount={1}
              mode="tags"
              placeholder="Select or input a category"
              options={categories.map(cate => {
                return { value: cate };
              })}
            />
          </Form.Item>
          <Form.Item
            label={t('page.connector.new.labels.icon')}
            name="icon"
            rules={[{ required: true }]}
          >
            <IconSelector
              className="max-w-600px"
              icons={iconsMeta}
              type="connector"
            />
          </Form.Item>
          <Form.Item
            label={t('page.connector.new.labels.assets_icons')}
            name="assets_icons"
          >
            <AssetsIcons iconsMeta={iconsMeta} />
          </Form.Item>
          <Form.Item
            label={t('page.connector.new.labels.description')}
            name="description"
          >
            <Input.TextArea />
          </Form.Item>
          <Form.Item
            label={t('page.connector.new.labels.tags')}
            name="tags"
          >
            <Tags />
          </Form.Item>
          <Form.Item label=" ">
            <Button
              htmlType="submit"
              type="primary"
            >
              {t('common.save')}
            </Button>
          </Form.Item>
        </Form>
      </ACard>
    </div>
  );
}
