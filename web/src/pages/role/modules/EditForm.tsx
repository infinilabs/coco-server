import { Button, Form, Input, Spin } from "antd";
import { useLoading } from '@sa/hooks';
import './EditForm.css';
import Permissions from "./Permissions";

export const EditForm = memo(props => {
  const { actionText, record, onSubmit } = props;
  const [form] = Form.useForm();
  const { t } = useTranslation();
  const { defaultRequiredRule } = useFormRules();
  const { endLoading, loading, startLoading } = useLoading();

  const handleSubmit = async () => {
    const params = await form.validateFields();
    onSubmit(params, startLoading, endLoading)
  };

  const initValue = (record) => {
    form.setFieldsValue(record);
  }

  useEffect(() => {
    initValue(record)
  }, [record])

  const itemClassNames = '!w-496px';

  return (
    <Spin spinning={props.loading || loading || false}>
      <Form
        colon={false}
        form={form}
        labelAlign="left"
        layout="horizontal"
        labelCol={{
          style: { maxWidth: 200, minWidth: 200, textAlign: 'left' }
        }}
      >
        <Form.Item
          label={t('page.role.labels.name')}
          name="name"
          rules={[defaultRequiredRule]}
        >
          <Input className={itemClassNames} />
        </Form.Item>
        <Form.Item
          label={t('page.role.labels.description')}
          name="description"
        >
          <Input.TextArea
            className={itemClassNames}
            rows={4}
          />
        </Form.Item>
        <Form.Item
          label={t('page.role.labels.permission')}
          name="permission"
        >
          <Permissions />
        </Form.Item>
        <Form.Item label=" ">
          <Button
            type="primary"
            onClick={() => handleSubmit()}
          >
            {actionText}
          </Button>
        </Form.Item>
      </Form>
    </Spin>
  )
});
