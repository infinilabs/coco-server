import { Button, Form, Select, Spin } from 'antd';
import '../index.scss';
import { fetchSettings, updateSettings } from '@/service/api/server';
import { useLoading, useRequest } from '@sa/hooks';
import { formatESSearchResult } from '@/service/request/es';
import { createPipeline, getEnablePipelines } from '@/service/api/pipeline';

const DocProcessing = memo(() => {
  const [form] = Form.useForm();
  const { t } = useTranslation();

  const { hasAuth } = useAuth()

  const permissions = {
    update: hasAuth('coco#system/update'),
  }

  const { endLoading, loading, startLoading } = useLoading();
  const [pipelineList, setPipelineList] = useState([]);

  const {
    data,
    loading: dataLoading,
    run
  } = useRequest(fetchSettings, {
    manual: true
  });

  const fetchPipelines = async () => {
    startLoading();
    const res = await getEnablePipelines()
    if (res?.data) {
      const newResult = formatESSearchResult(res?.data);
      setPipelineList(newResult.data as any);
    }
    endLoading();
  }

  useEffect(() => {
    run();
    fetchPipelines();
  }, []);

  const handleSubmit = async () => {
    const params = await form.validateFields();
    startLoading();

    const result = await updateSettings({
      document_processing: params
    });
    if (result?.data?.acknowledged) {
      window.$message?.success(t('common.updateSuccess'));
    }
    endLoading();
  };

  useMount(() => {
    run();
  });

  useEffect(() => {
    if (data?.document_processing) {
      form.setFieldsValue({
        ...data?.document_processing
      });
    }
  }, [JSON.stringify(data)]);

  return (
    <ListContainer>
      <Spin spinning={dataLoading || loading}>
        <Form
          className="settings-form py-24px"
          colon={false}
          form={form}
          labelAlign="left"
        >
          <Form.Item
            label={(
              <span className='color-[var(--ant-color-text-tertiary)]'>
                {t('page.settings.document_processing.labels.default_pipeline_for_attachment')}
              </span>
            )}
          >
            <PipelineSelectItem
              label={t('page.settings.document_processing.labels.processing_pipeline')}
              name="default_pipeline_for_attachment"
              pipelineList={pipelineList}
            />
          </Form.Item>
          <Form.Item
            label={(
              <span className='color-[var(--ant-color-text-tertiary)]'>
                {t('page.settings.document_processing.labels.default_pipeline_for_document')}
              </span>
            )}
          >
            <PipelineSelectItem
              label={t('page.settings.document_processing.labels.processing_pipeline')}
              name="default_pipeline_for_document"
              pipelineList={pipelineList}
            />
          </Form.Item>
          <Form.Item
            label={(
              <span className='color-[var(--ant-color-text-tertiary)]'>
                {t('page.settings.document_processing.labels.output_language')}
              </span>
            )}
          >
            <div className='m-b-8px color-[var(--ant-color-text-tertiary)]'>
              {t('page.settings.document_processing.labels.output_language_desc')}
            </div>
            <Form.Item noStyle name={'llm_generation_language'}>
              <Select
                options={[
                  {
                    value: 'en-US',
                    label: 'English'
                  },
                  {
                    value: 'zh-CN',
                    label: '中文'
                  }
                ]}
              />
            </Form.Item>
          </Form.Item>
          {
            permissions.update && (
              <Form.Item label=" " >
                <Button
                  type="primary"
                  onClick={() => handleSubmit()}
                >
                  {t('common.update')}
                </Button>
              </Form.Item>
            )
          }
        </Form>
      </Spin>
    </ListContainer>
  );
});

export default DocProcessing;

export const PipelineSelectItem = ({ label, desc, name, pipelineList = [] }: { label: string, desc?: string, name: string, pipelineList?: any[] }) => {

  return (
    <>
      <div className='m-b-4px'>{label}</div>
      {desc && <div className='m-b-8px color-[var(--ant-color-text-tertiary)]'>{desc}</div>}
      <Form.Item noStyle name={name}>
        <Select
          options={pipelineList.map(item => ({
            label: item.name,
            value: item.id
          }))}
        />
      </Form.Item>
    </>
  )
}