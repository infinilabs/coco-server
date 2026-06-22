import { setup } from '@/service/api/guide';
import { Button, Form, Input, Select } from 'antd';
import { useLoading } from '@sa/hooks';

const UserForm = memo(() => {
  const [form] = Form.useForm();
  const formItemClassNames = 'm-b-32px';
  const inputClassNames = 'h-40px';
  const { t } = useTranslation();
  const { defaultRequiredRule, formRules } = useFormRules();
  const { endLoading, loading, startLoading } = useLoading();
  const router = useRouterPush();

  const handleSubmit = async () => {
    const params = await form.validateFields();
    const { confirm_password, ...rest } = params
    startLoading();
    const { error } = await setup(rest);
    endLoading();
    if (!error) {
      router.routerPushByKey('login');
    }
  }

  return (
    <>
      <div className="m-b-16px text-28px color-[var(--ant-color-text-heading)]">{t('page.guide.user.title')}</div>
      <div className="m-b-64px text-14px color-[var(--ant-color-text-tertiary)]">{t('page.guide.user.desc')}</div>
      <Form
        form={form}
        layout="vertical"
      >
        <Form.Item
          className={formItemClassNames}
          label={t('page.guide.user.name')}
          name="name"
          rules={[defaultRequiredRule]}
        >
          <Input className={inputClassNames} />
        </Form.Item>
        <Form.Item
          className={formItemClassNames}
          label={t('page.guide.user.email')}
          name="email"
          rules={formRules.email}
        >
          <Input className={inputClassNames} />
        </Form.Item>
        <Form.Item
          className={formItemClassNames}
          label={t('page.guide.user.password')}
          name="password"
          rules={formRules.pwd}
        >
          <Input.Password className={inputClassNames} />
        </Form.Item>
        <Form.Item
          className={formItemClassNames}
          label={t('common.confirmPassword')}
          name="confirm_password"
          rules={[
            defaultRequiredRule,
            {
              validator: async (rule, value) => {
                const password = await form.getFieldValue('password');
                if (value && password !== value) {
                  throw new Error(t('form.pwdConfirm.invalid'));
                }
              },
            }
          ]}
        >
          <Input.Password className={inputClassNames} />
        </Form.Item>
        <Form.Item
          className={formItemClassNames}
          label={t('page.guide.user.language')}
          name="language"
          initialValue={"zh-CN"}
        >
          <Select className={inputClassNames} options={[{label:t('common.language.zh'), value:"zh-CN"}, {label: t('common.language.en'), value:"en-US"}]} />
        </Form.Item>
        <div className="text-right">
          <Button
            className="h-56px w-56px text-24px"
            size="large"
            type="primary"
            loading={loading}
            onClick={() => handleSubmit()}
          >
            <SvgIcon icon="mdi:arrow-right" />
          </Button>
        </div>
      </Form>
    </>
  );
});

export default UserForm;
