import { Form, Input, Modal } from 'antd';

import { modifyPassword } from '@/service/api';

const PasswordModal = ({
  onClose,
  onSuccess,
  open
}: {
  readonly onClose: () => void;
  readonly onSuccess: () => void;
  readonly open: boolean;
}) => {
  const { t } = useTranslation();
  const [form] = Form.useForm();
  const { formRules } = useFormRules();

  const handleSubmit = async () => {
    const params = await form.validateFields();
    const { error } = await modifyPassword(params.old_password, params.new_password);
    if (!error) {
      window.$notification?.success({
        description: t('common.loginAgain'),
        message: t('common.modifySuccess')
      });
      setTimeout(() => {
        onSuccess();
      }, 1000);
    }
  };

  return (
    <Modal
      destroyOnClose
      open={open}
      title={t('common.modifyPassword')}
      width="560px"
      onCancel={onClose}
      onOk={() => handleSubmit()}
    >
      <Form
        className="py-24px"
        form={form}
        layout="vertical"
      >
        <Form.Item
          label={t('common.oldPassword')}
          name="old_password"
          rules={formRules.pwd}
        >
          <Input.Password />
        </Form.Item>
        <Form.Item
          label={t('common.newPassword')}
          name="new_password"
          rules={formRules.pwd}
        >
          <Input.Password />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default PasswordModal;
