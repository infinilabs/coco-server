import { Modal, InputNumber, Form, Input } from 'antd';

interface ModelSettingsProps {
  value?: any;
  onChange?: (value: any) => void;
}

const PARAMETERS = [
  {
      key: 'temperature',
      input: (props: any)=> <InputNumber {...props} min={0} step={0.1} max={1.0}/>
  },
  {
      key: 'top_p',
      input: (props: any) => <InputNumber {...props} min={0} step={0.1}  max={1.0}/>
  },
  {
    key: 'presence_penalty',
    input: (props: any) => <InputNumber {...props} min={-2.0} step={0.1} max={2.0} />
  },
  {
      key: 'frequency_penalty',
      input: (props: any) => <InputNumber {...props} min={-2.0} step={0.1} max={2.0} />
  },
  {
      key: 'max_tokens',
      input: (props: any) => <InputNumber {...props} min={1} step={1} precision={0} max={16385}/>
  },
  
]

export default (props: ModelSettingsProps) => {
  const { t } = useTranslation();
  const [visible, setVisible] = useState(false);
  const onClose = ()=>{
    setVisible(false)
  }
  const [form] = Form.useForm();
  const onOKClick = ()=>{
    form.validateFields().then((values)=>{
      props.onChange?.(values);
      setVisible(false);
    }).catch((error)=>{
      console.log('error', error);
    })
  }
  return <div className="inline-block">
    <div  className='cursor-pointer' onClick={()=>{setVisible(true)}} >
      <SvgIcon className='text-[#999]' localIcon='list-settings'/>
    </div>
      <Modal onCancel={onClose} onClose={onClose} open={visible} title={null} onOk={onOKClick}>
          <Form initialValues={props.value} form={form}>
              <div className="ant-modal-header">
                <div className="ant-modal-title">{t('page.assistant.labels.model_settings')}</div>
              </div>
              {
                  PARAMETERS.map((item) => (
                      <div key={item.key} className={`flex justify-between items-center mb-8px`}>
                          <div className="[flex:1]">
                              <div className="color-[var(--ant-form-label-color)] mb-5px">{t(`page.assistant.labels.${item.key}`)}</div>
                              <div className="text-gray-400 mb-10px text-[12px]">{t(`page.assistant.labels.${item.key}_desc`)}</div>
                          </div>
                          <div >
                              <Form.Item
                                  name={['settings', item.key]}
                                  label=""
                              >
                                  <item.input/>
                              </Form.Item>
                          </div>
                      </div>
                  ))
              }
              <div className="ant-modal-header">
                <div className="ant-modal-title">{t('page.assistant.labels.prompt_settings')}</div>
              </div>
              <div className={`flex flex-col items-top mb-8px`}>
                  <div className="">
                      <div className="color-[var(--ant-form-label-color)] mb-5px">{t(`page.assistant.labels.prompt_settings_template`)}</div>
                  </div>
                  <div className="">
                      <Form.Item
                          name={['prompt', 'template']}
                          label=""
                      >
                          <Input.TextArea rows={5}/>
                      </Form.Item>
                  </div>
              </div>
          </Form>
      </Modal>
  </div>

}