import { Button, Form, Input } from "antd";
import { FormInstance } from "antd/lib";

const UserForm = memo(({ form, onSubmit }: { form: FormInstance, onSubmit: () => void }) => {
    const formItemClassNames = "m-b-32px"
    const inputClassNames = "h-40px"
    const { t } = useTranslation();
    const { defaultRequiredRule, formRules } = useFormRules();

    return (
        <>
            <div className="text-32px color-#333 m-b-16px">
                {t('page.guide.user.title')}
            </div>
            <div className="text-16px color-#999 m-b-64px">
                {t('page.guide.user.desc')}
            </div>
            <Form
                form={form}
                layout="vertical"
            >
                <Form.Item
                    name="name"
                    label={t('page.guide.user.name')}
                    className={formItemClassNames}
                    rules={[defaultRequiredRule]}
                >
                    <Input className={inputClassNames}/>
                </Form.Item>
                <Form.Item
                    name="email"
                    label={t('page.guide.user.email')}
                    className={formItemClassNames}
                    rules={formRules.email}
                >
                    <Input className={inputClassNames}/>
                </Form.Item>
                <Form.Item
                    name="password"
                    label={t('page.guide.user.password')}
                    className={formItemClassNames}
                    rules={formRules.pwd}
                >
                    <Input.Password className={inputClassNames}/>
                </Form.Item>
                <div className="text-right">
                    <Button type="primary" size="large" className="w-56px h-56px text-24px" onClick={() => onSubmit()}>
                        <SvgIcon icon="mdi:arrow-right" />
                    </Button>
                </div>
            </Form>
        </>
    )
})

export default UserForm;