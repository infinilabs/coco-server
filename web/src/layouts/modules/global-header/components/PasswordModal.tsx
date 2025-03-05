import { modifyPassword } from "@/service/api";
import { Form, Input, Modal } from "antd";

const PasswordModal = ({ onClose, onSuccess, open }: { onClose: () => void; onSuccess: () => void; open: boolean }) => {
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
        onSuccess()
      }, 1000)
    }
  }

  return (
    <Modal
      open={open}
      title={t('common.modifyPassword')}
      width="560px"
      onCancel={onClose}
      onOk={() => handleSubmit()}
      destroyOnClose
    >
      <Form
        form={form}
        layout="vertical"
        className="py-24px"
      >
        <Form.Item
          name="old_password"
          label={t('common.oldPassword')}
          rules={formRules.pwd}
        >
            <Input.Password />
        </Form.Item>
        <Form.Item
          name="new_password"
          label={t('common.newPassword')}
          rules={formRules.pwd}
        >
            <Input.Password />
        </Form.Item>
      </Form>
    </Modal>
  );
};

export default PasswordModal;
