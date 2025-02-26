import { Button, Form, Input } from "antd";
import { FormInstance } from "antd/lib";

const UserForm = memo(({ form, onSubmit }: { form: FormInstance, onSubmit: () => void }) => {
    const formItemClassNames = "m-b-32px"
    const inputClassNames = "h-40px"

    return (
        <>
            <div className="text-28px color-#333 m-b-16px">
                Create a user account
            </div>
            <div className="text-14px color-#999 m-b-64px">
                Set up a new user account to manage access and permissions.
            </div>
            <Form
                form={form}
                layout="vertical"
            >
                <Form.Item
                    name="full_name"
                    label="Full Name"
                    className={formItemClassNames}
                >
                    <Input className={inputClassNames}/>
                </Form.Item>
                <Form.Item
                    name="email"
                    label="Email"
                    className={formItemClassNames}
                >
                    <Input className={inputClassNames}/>
                </Form.Item>
                <Form.Item
                    name="password"
                    label="Password"
                    className={formItemClassNames}
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