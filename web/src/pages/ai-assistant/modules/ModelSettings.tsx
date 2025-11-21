import { Form, Input, InputNumber, Modal, Switch } from 'antd';

interface ModelSettingsProps {
  readonly value?: any;
  readonly onChange?: (value: any) => void;
}

const PARAMETERS = [
  {
    key: 'temperature',
    input: (props: any) => (
      <InputNumber
        {...props}
        max={1.0}
        min={0}
        step={0.1}
      />
    )
  },
  {
    key: 'top_p',
    input: (props: any) => (
      <InputNumber
        {...props}
        max={1.0}
        min={0}
        step={0.1}
      />
    )
  },
  {
    key: 'presence_penalty',
    input: (props: any) => (
      <InputNumber
        {...props}
        max={2.0}
        min={-2.0}
        step={0.1}
      />
    )
  },
  {
    key: 'frequency_penalty',
    input: (props: any) => (
      <InputNumber
        {...props}
        max={2.0}
        min={-2.0}
        step={0.1}
      />
    )
  }
];

export default (props: ModelSettingsProps) => {
  const { t } = useTranslation();
  const [visible, setVisible] = useState(false);
  const onClose = () => {
    setVisible(false);
  };
  const [form] = Form.useForm();
  const onOKClick = () => {
    form
      .validateFields()
      .then(values => {
        props.onChange?.(values);
        setVisible(false);
      })
      .catch(error => {
        console.log('error', error);
      });
  };
  return (
    <div className='inline-block'>
      <div
        className='cursor-pointer'
        onClick={() => {
          setVisible(true);
        }}
      >
        <SvgIcon
          className='text-[#999]'
          localIcon='list-settings'
        />
      </div>
      <Modal
        open={visible}
        title={null}
        onCancel={onClose}
        onClose={onClose}
        onOk={onOKClick}
      >
        <Form
          form={form}
          initialValues={props.value}
        >
          <div className='ant-modal-header'>
            <div className='ant-modal-title'>{t('page.assistant.labels.model_settings')}</div>
          </div>
          {PARAMETERS.map(item => (
            <div
              className='mb-8px flex items-center justify-between'
              key={item.key}
            >
              <div className='[flex:1]'>
                <div className='mb-5px color-[var(--ant-form-label-color)]'>
                  {t(`page.assistant.labels.${item.key}`)}
                </div>
                <div className='mb-10px text-[12px] text-gray-400'>{t(`page.assistant.labels.${item.key}_desc`)}</div>
              </div>
              <div>
                <Form.Item
                  label=''
                  name={['settings', item.key]}
                >
                  <item.input />
                </Form.Item>
              </div>
            </div>
          ))}
        </Form>
      </Modal>
    </div>
  );
};
