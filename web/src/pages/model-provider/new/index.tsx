import { Button, Form, Input, List, Select, Switch, message } from 'antd';
import type { FormProps } from 'antd';
import { createModelProvider, getLLMModels } from '@/service/api/model-provider';
import { getConnectorIcons } from '@/service/api/connector';
import { MinusCircleOutlined } from '@ant-design/icons';
import { formatESSearchResult } from '@/service/request/es';
import ModelSettings from '@/pages/ai-assistant/modules/ModelSettings';
import { getUUID } from '@/utils/common';
import { ModalForm, ProFormSelect, ProFormSwitch, ProFormText } from '@ant-design/pro-components';
import type { AnyObject } from 'antd/es/_util/type';
// @ts-ignore
import { IconSelector } from '../../connector/new/icon_selector';

export function Component() {
  const { t } = useTranslation();
  const nav = useNavigate();

  const onFinish: FormProps<any>['onFinish'] = values => {
    const newValues = {
      ...values
    };
    createModelProvider(newValues).then(res => {
      if (res.data?.result == 'created') {
        message.success(t('common.addSuccess'));
        nav('/model-provider/list', {});
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
  const { defaultRequiredRule, formRules } = useFormRules();
  const initialValues = {
    enabled: true
  };

  return (
    <div className='h-full min-h-500px'>
      <ACard
        bordered={false}
        className='min-h-full flex-col-stretch sm:flex-1-hidden card-wrapper'
      >
        <div className='mb-30px ml--16px flex items-center text-lg font-bold'>
          <div className='mr-20px h-1.2em w-10px bg-[#1677FF]' />
          <div>{t('route.model-provider_new')}</div>
        </div>
        <div className='px-30px'>
          <Form
            autoComplete='off'
            colon={false}
            initialValues={initialValues}
            labelCol={{ span: 4 }}
            layout='horizontal'
            wrapperCol={{ span: 18 }}
            onFinish={onFinish}
            onFinishFailed={onFinishFailed}
          >
            <Form.Item
              label={t('page.modelprovider.labels.name')}
              name='name'
              rules={[{ required: true }]}
            >
              <Input className='max-w-600px' />
            </Form.Item>
            <Form.Item
              label={t('page.modelprovider.labels.icon')}
              name='icon'
              rules={[{ required: true }]}
            >
              <IconSelector
                className='max-w-600px'
                icons={iconsMeta}
                type='connector'
              />
            </Form.Item>
            <Form.Item
              label={t('page.modelprovider.labels.api_type')}
              name='api_type'
              rules={[{ required: true }]}
            >
              <Select
                className='max-w-150px'
                options={[
                  { label: 'OpenAI', value: 'openai' },
                  { label: 'Ollama', value: 'ollama' }
                ]}
              />
            </Form.Item>
            <Form.Item
              label={t('page.modelprovider.labels.api_key')}
              name='api_key'
            >
              <Input.Password className='max-w-600px' />
            </Form.Item>
            <Form.Item
              label={t('page.modelprovider.labels.base_url')}
              name='base_url'
              rules={formRules.endpoint}
            >
              <Input className='max-w-600px' />
            </Form.Item>
            <Form.Item
              label={t('page.modelprovider.labels.models')}
              name='models'
              rules={[{ required: true }]}
            >
              <ModelsComponent />
            </Form.Item>
            <Form.Item
              label={t('page.modelprovider.labels.description')}
              name='description'
            >
              <Input.TextArea className='w-600px' />
            </Form.Item>
            <Form.Item
              label={t('page.modelprovider.labels.enabled')}
              name='enabled'
            >
              <Switch size='small' />
            </Form.Item>
            <Form.Item label=' '>
              <Button
                htmlType='submit'
                type='primary'
              >
                {t('common.save')}
              </Button>
            </Form.Item>
          </Form>
        </div>
      </ACard>
    </div>
  );
}

const defaultModelSettings = {
  temperature: 0.7,
  top_p: 0.9,
  presence_penalty: 0,
  frequency_penalty: 0,
  max_tokens: 4000
};
export const ModelsComponent = ({ value = [], onChange }: any) => {
  const { hasAuth } = useAuth();

  const permissions = {
    fetchModelProviders: hasAuth('coco#model_provider/search')
  };

  const initialValue = useMemo(() => {
    return (value || []).map((v: any) => ({
      value: v,
      key: getUUID()
    }));
  }, [value]);

  const [innerValue, setInnerValue] = useState<{ value: any; key: string }[]>(initialValue);
  const prevValueRef = useRef<any[]>([]);

  // Prevent unnecessary updates
  useEffect(() => {
    if (JSON.stringify(prevValueRef.current) !== JSON.stringify(value)) {
      prevValueRef.current = value;
      const iv = (value || []).map((v: any) => ({
        value: v,
        key: getUUID()
      }));

      setInnerValue(iv);
    }
  }, [value]);

  const [models, setModels] = useState<string[]>([]);
  const [loading, _setLoading] = useState(false);

  useEffect(() => {
    if (permissions.fetchModelProviders) {
      getLLMModels().then(({ data }) => {
        if (!data?.error) {
          const newData = formatESSearchResult(data);
          const models = newData.aggregations.models.buckets.map((item: any) => {
            return item.key;
          });

          setModels(models);
        }
      });
    }
  }, [permissions.fetchModelProviders]);

  const handleDelete = (key: string) => {
    const newValues = innerValue.filter(v => v.key !== key);

    console.log('newValues', newValues);

    setInnerValue(newValues);
  };

  useEffect(() => {
    onChange?.(innerValue.filter(v => v.value?.name).map(v => v.value));
  }, [innerValue]);

  const { t } = useTranslation();

  const onSettingsChange = (key: string, settings: any) => {
    const updatedValues = innerValue.map(v =>
      v.key === key
        ? {
            ...v,
            value: {
              ...(v.value || {}),
              settings
            }
          }
        : v
    );
    setInnerValue(updatedValues);
  };

  const [form] = Form.useForm<AnyObject>();

  return (
    <>
      {innerValue.length > 0 && (
        <List
          bordered
          dataSource={innerValue}
          rootClassName='mb-4 max-w-150'
          size='small'
          renderItem={item => {
            const { key, value } = item;

            return (
              <List.Item
                actions={[
                  <span key='inference-mode'>
                    {value?.settings?.reasoning ? t('page.modelprovider.labels.inferenceMode') : '-'}
                  </span>,
                  <ModelSettings
                    key='model-settings'
                    value={value?.settings}
                    onChange={settings => onSettingsChange(key, settings)}
                  />,
                  <div
                    className='cursor-pointer'
                    key='delete-model'
                    onClick={() => handleDelete(key)}
                  >
                    <MinusCircleOutlined className='text-[#999]' />
                  </div>
                ]}
              >
                {value?.name ?? '-'}
              </List.Item>
            );
          }}
        />
      )}

      <ModalForm
        form={form}
        title={t('page.modelprovider.labels.addModel')}
        trigger={<Button type='primary'>{t('common.add')}</Button>}
        width={560}
        modalProps={{
          centered: true,
          destroyOnClose: true
        }}
        onFinish={async values => {
          setInnerValue([
            ...innerValue,
            {
              key: getUUID(),
              value: {
                ...values,
                name: values.name[0]
              }
            }
          ]);

          return true;
        }}
      >
        <ProFormSelect
          label={t('page.modelprovider.labels.modelID')}
          name='name'
          fieldProps={{
            loading,
            maxCount: 1,
            mode: 'tags',
            placeholder: t('page.modelprovider.hints.selectOrInputModel'),
            options: models.map(model => ({
              label: model,
              value: model
            }))
          }}
          rules={[
            {
              required: true
            }
          ]}
        />

        <ProFormText
          initialValue={t('page.modelprovider.options.dialogModel')}
          label={t('page.modelprovider.labels.modelType')}
          name='type'
          fieldProps={{
            readOnly: true
          }}
        />

        <ProFormSwitch
          initialValue={true}
          label={t('page.modelprovider.labels.inferenceMode')}
          name={['settings', 'reasoning']}
          formItemProps={{
            layout: 'horizontal',
            className: 'mb-0!'
          }}
        />

        <span className='text-color-3'>{t('page.modelprovider.hints.inferenceMode')}</span>
      </ModalForm>
    </>
  );
};
