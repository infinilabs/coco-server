import { Button, Form, Input, List, Select, Switch, message } from 'antd';
import type { FormProps } from 'antd';
import { createModelProvider, fetchProviderModels, getLLMModels, testModelProvider } from '@/service/api/model-provider';
import { getConnectorIcons } from '@/service/api/connector';
import { MinusCircleOutlined } from '@ant-design/icons';
import { formatESSearchResult } from '@/service/request/es';
import { getUUID } from '@/utils/common';
import { ModalForm, ProFormSelect } from '@ant-design/pro-components';
import type { AnyObject } from 'antd/es/_util/type';
// @ts-ignore
import { IconSelector } from '../../connector/new/icon_selector';
import { ArrowUpDown, Boxes, Brain, Eye, Settings, Type } from 'lucide-react';
import { isArray } from 'lodash';

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

// chip-style option buttons (rounded pill + icon, neutral fill when selected)
// used for the type / capability / modality groups in the model modal
const chipCls = (active: boolean) =>
  `flex h-32px cursor-pointer select-none items-center gap-6px rounded-6px border px-12px text-14px transition-colors ${
    active
      ? 'border-[var(--ant-color-border)] bg-[var(--ant-color-fill-secondary)] font-500'
      : 'border-[var(--ant-color-border-secondary)] hover:border-[var(--ant-color-primary-border)]'
  }`;

const ChipSelect = ({ value, onChange, options }: any) => (
  <div className='flex flex-wrap gap-8px'>
    {options.map((o: any) => {
      const Icon = o.icon;

      return (
        <div
          className={chipCls(value === o.value)}
          key={o.value}
          onClick={() => onChange?.(o.value)}
        >
          {Icon && <Icon className='size-14px' />}
          {o.label}
        </div>
      );
    })}
  </div>
);

const ChipToggle = ({ value, onChange, label, icon: Icon }: any) => (
  <div className='flex flex-wrap gap-8px'>
    <div
      className={chipCls(!!value)}
      onClick={() => onChange?.(!value)}
    >
      {Icon && <Icon className='size-14px' />}
      {label}
    </div>
  </div>
);

export const ModelsComponent = ({ value = [], onChange }: any) => {  const { hasAuth } = useAuth();

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
  const [loading, setLoading] = useState(false);

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
  const [modelOpen, setModelOpen] = useState(false);
  const [editKey, setEditKey] = useState<string>();

  // the provider connection fields live on the OUTER form (new & edit pages
  // both name them api_type/base_url/api_key), so reach up through the
  // FormInstance instead of threading props through every call site
  const parentForm = Form.useFormInstance();

  // remote model list is fetched on demand (user-triggered) and merged with
  // the aggregated historical ids; manual entry stays available either way
  const [fetchingModels, setFetchingModels] = useState(false);
  const [remoteModels, setRemoteModels] = useState<string[] | null>(null);

  // infer a model's nature from its id — advisory only: applied while the
  // type field still holds the default, never overwrites an explicit choice.
  // Vision markers (qwen-vl / GLM-4.5V trailing V / *vision*) map onto the
  // language type + support_vision capability, per the compat taxonomy
  const guessModelType = (id: string): { type: string; vision: boolean } => {
    const n = (id || '').toLowerCase();
    if (!n) return { type: 'language', vision: false };
    if (n.includes('rerank')) return { type: 'rerank', vision: false };
    if (n.includes('embed') || n.includes('bge-') || n.includes('gte-')) return { type: 'embedding', vision: false };
    return { type: 'language', vision: n.includes('vision') || n.includes('vl') || /v$/.test(n) };
  };

  const modelOptions = useMemo(() => {
    const seen = new Set<string>();
    const existing = new Set(innerValue.map(v => v.value?.name));
    const opts: { label: string; value: string; disabled: boolean }[] = [];
    const push = (id: string) => {
      if (!id || seen.has(id)) return;
      seen.add(id);
      opts.push({ label: id, value: id, disabled: existing.has(id) });
    };
    (remoteModels || []).forEach(push);
    models.forEach(push);
    return opts;
  }, [remoteModels, models, innerValue]);

  const handleFetchModels = async () => {
    const conn = parentForm?.getFieldsValue(['api_type', 'base_url', 'api_key']) || {};
    if (!conn?.api_type) {
      message.warning(t('page.modelprovider.hints.needsApiType'));
      return;
    }
    if (!conn?.base_url) {
      message.warning(t('page.modelprovider.hints.testNeedsBaseUrl'));
      return;
    }
    setFetchingModels(true);
    try {
      const res: any = await fetchProviderModels({
        api_type: conn.api_type,
        base_url: conn.base_url,
        api_key: conn.api_key
      });
      const d = res?.data;
      if (d?.ok) {
        setRemoteModels(d.models || []);
        message.success(`${t('page.modelprovider.hints.fetchedModels')} (${d.models?.length ?? 0})`);
      } else {
        message.error(`${t('page.modelprovider.hints.fetchModelsFailed')}: ${d?.error || `HTTP ${d?.status ?? '-'}`}`);
      }
    } catch (e: any) {
      message.error(`${t('page.modelprovider.hints.fetchModelsFailed')}: ${e?.message || e}`);
    } finally {
      setFetchingModels(false);
    }
  };

  // the provider connection fields live on the OUTER form (new & edit pages
  // both name them api_type/base_url/api_key), so reach up through the
  // FormInstance instead of threading props through every call site
  const [testing, setTesting] = useState(false);
  const handleTestModel = async () => {
    try {
      await form.validateFields(['name']);
    } catch {
      return;
    }
    const conn = parentForm?.getFieldsValue(['api_type', 'base_url', 'api_key']) || {};
    if (!conn?.base_url) {
      message.warning(t('page.modelprovider.hints.testNeedsBaseUrl'));
      return;
    }
    const nameVal = form.getFieldValue('name');
    const model = isArray(nameVal) ? nameVal[0] : nameVal;
    setTesting(true);
    try {
      const res: any = await testModelProvider({
        api_type: conn.api_type,
        base_url: conn.base_url,
        api_key: conn.api_key,
        model,
        type: form.getFieldValue('type')
      });
      const d = res?.data;
      if (d?.ok) {
        message.success(`${t('page.modelprovider.hints.testSuccess')} (${d.latency_ms ?? '-'}ms)`);
      } else {
        message.error(`${t('page.modelprovider.hints.testFailed')}: ${d?.error || `HTTP ${d?.status ?? '-'}`}`);
      }
    } catch (e: any) {
      message.error(`${t('page.modelprovider.hints.testFailed')}: ${e?.message || e}`);
    } finally {
      setTesting(false);
    }
  };

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
                    {value?.support_reasoning ? t('page.modelprovider.labels.inferenceMode') : '-'}
                  </span>,
                  <span key='vision-input'>
                    {/* legacy type=vision counts as vision-capable too */}
                    {value?.support_vision || value?.type === 'vision' ? t('page.modelprovider.options.vision') : '-'}
                  </span>,
                  <Settings
                    className='size-4 translate-y-[2px] cursor-pointer text-[#999]'
                    key='settings'
                    onClick={() => {
                      // backend serializes support flags with omitempty:
                      // false never lands in ES, so read-back is undefined —
                      // set them explicitly or stale values resurrect.
                      // Legacy type=vision stays as-is (the compat period
                      // keeps the legacy type selectable, no data migration)
                      form.setFieldsValue({
                        ...value,
                        support_reasoning: value?.support_reasoning ?? false,
                        support_vision: value?.support_vision ?? false
                      });

                      setEditKey(key);

                      setModelOpen(true);
                    }}
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
        open={modelOpen}
        title={t('page.modelprovider.labels.addModel')}
        trigger={<Button type='primary'>{t('common.add')}</Button>}
        width={560}
        submitter={{
          // test button shares the footer row with Save/Cancel, aligned left
          render: (_props, dom) => (
            <div className='flex w-full items-center justify-between'>
              <Button
                loading={testing}
                onClick={handleTestModel}
              >
                {t('page.modelprovider.hints.testModel')}
              </Button>
              <div className='flex items-center gap-8px'>{dom}</div>
            </div>
          )
        }}
        modalProps={{
          centered: true,
          destroyOnClose: true,
          onCancel: () => {
            setEditKey(undefined);
            setModelOpen(false);
          }
        }}
        onFinish={async values => {
          const name = isArray(values.name) ? values.name[0] : values.name;

          const matched = innerValue.find(item => item.key === editKey);

          // a model name must be unique within a provider — GetModel indexes
          // by name, a duplicate would silently shadow the existing entry
          if (!matched && innerValue.some(item => item.value?.name === name)) {
            message.warning(t('page.modelprovider.hints.duplicateModel'));
            return false;
          }

          if (matched) {
            setInnerValue(prev => {
              return prev.map(item => {
                if (item.key === editKey) {
                  return {
                    ...item,
                    value: {
                      ...item.value,
                      ...values,
                      name
                    }
                  };
                }

                return item;
              });
            });

            return setModelOpen(false);
          }

          setInnerValue(prev => [...prev, { key: getUUID(), value: { ...values, name } }]);

          return true;
        }}
      >
        {/* the fetch button sits right of the model-id control: the select is
            narrowed and the button pinned to the control's top edge — anchoring
            bottom instead would slide the button down to the validation-error
            text whenever "required" fires (label height 22px + 8px margin is
            constant, so top offset tracks the input top) */}
        <div className='relative'>
          <ProFormSelect
            label={t('page.modelprovider.labels.modelID')}
            name='name'
            fieldProps={{
              loading,
              maxCount: 1,
              mode: 'tags',
              placeholder: t('page.modelprovider.hints.selectOrInputModel'),
              style: { width: 'calc(100% - 128px)' },
              options: modelOptions,
              onChange: (v: any) => {
                const id = isArray(v) ? v[0] : v;
                if (!id) return;
                // only suggest while the type still holds its default; an
                // explicit user choice is never overwritten
                const cur = form.getFieldValue('type');
                if (cur && cur !== 'language') return;
                const guessed = guessModelType(String(id));
                if (guessed.type !== 'language') {
                  form.setFieldValue('type', guessed.type);
                } else if (guessed.vision) {
                  form.setFieldValue('support_vision', true);
                }
              }
            }}
            rules={[
              {
                required: true
              }
            ]}
          />
          <Button
            className='absolute right-0 top-30px w-120px'
            loading={fetchingModels}
            onClick={handleFetchModels}
          >
            {t('page.modelprovider.hints.fetchModels')}
          </Button>
        </div>

        {/* 视觉 stays as a legacy type during the compat period (existing data
            carries it); new vision-capable language models should prefer
            Text + Vision-input modality instead */}
        <Form.Item
          colon={false}
          initialValue='language'
          label={t('page.modelprovider.labels.modelType')}
          name='type'
          rules={[{ required: true }]}
        >
          <ChipSelect
            options={[
              { label: t('page.modelprovider.options.textModel'), value: 'language', icon: Type },
              { label: t('page.modelprovider.options.vision'), value: 'vision', icon: Eye },
              { label: t('page.modelprovider.options.embedding'), value: 'embedding', icon: Boxes },
              { label: t('page.modelprovider.options.rerank'), value: 'rerank', icon: ArrowUpDown }
            ]}
          />
        </Form.Item>

        {/* capabilities & input modalities only apply to language models (an
            unset type counts as language, mirroring core.Model.Validate);
            preserve:false drops the flags from the payload when the group
            hides, so switching type never trips backend validation */}
        <Form.Item
          noStyle
          shouldUpdate={(prev, curr) => prev.type !== curr.type}
        >
          {({ getFieldValue }) =>
            getFieldValue('type') === 'language' || !getFieldValue('type') ? (
              <>
                <Form.Item
                  colon={false}
                  initialValue={true}
                  label={t('page.modelprovider.labels.modelCapabilities')}
                  name='support_reasoning'
                  preserve={false}
                >
                  <ChipToggle
                    icon={Brain}
                    label={t('page.modelprovider.options.reasoning')}
                  />
                </Form.Item>

                <Form.Item
                  colon={false}
                  label={t('page.modelprovider.labels.inputModalities')}
                  name='support_vision'
                  preserve={false}
                >
                  <ChipToggle
                    icon={Eye}
                    label={t('page.modelprovider.options.vision')}
                  />
                </Form.Item>
              </>
            ) : null
          }
        </Form.Item>
      </ModalForm>
    </>
  );
};
