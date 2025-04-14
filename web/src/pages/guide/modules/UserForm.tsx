import { Button, Form, Input } from 'antd';
import type { FormInstance } from 'antd/lib';

const UserForm = memo(({ form, onSubmit }: { form: FormInstance; onSubmit: () => void }) => {
  const formItemClassNames = 'm-b-32px';
  const inputClassNames = 'h-40px';
  const { t } = useTranslation();
  const { defaultRequiredRule, formRules } = useFormRules();

  return (
    <>
      <div className="m-b-16px text-32px color-[var(--ant-color-text-heading)]">{t('page.guide.user.title')}</div>
      <div className="m-b-64px text-16px color-[var(--ant-color-text)]">{t('page.guide.user.desc')}</div>
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
        <div className="text-right">
          <Button
            className="h-56px w-56px text-24px"
            size="large"
            type="primary"
            onClick={() => onSubmit()}
          >
            <SvgIcon icon="mdi:arrow-right" />
          </Button>
        </div>
      </Form>
    </>
  );
});

export default UserForm;
