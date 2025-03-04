import { Button, Form, Input, Upload } from "antd";
import "../index.scss"

const Server = memo(() => {
    const [form] = Form.useForm();

    return (
        <>
            <Form 
                form={form}
                labelAlign="left"
                className="settings-form"
                colon={false}
            >
                <Form.Item
                    name="name"
                    label="Server Name"
                >
                    <Input />
                </Form.Item>
                <Form.Item
                    name="desc"
                    label="Description"
                >
                    <Input.TextArea
                        autoSize={{ minRows: 3, maxRows: 5 }}
                    />
                </Form.Item>
                <Form.Item
                    name="logo"
                    label="Logo"
                >
                    <div className="flex">
                        <Upload >
                            <Button icon={<SvgIcon icon="mdi:upload"/>}>Upload File</Button>
                        </Upload>
                        <Button type="link">Reset</Button>
                    </div>
                </Form.Item>
                <Form.Item
                    name="banner"
                    label="Banner"
                >
                    <div className="flex">
                        <Upload >
                            <Button icon={<SvgIcon icon="mdi:upload"/>}>Upload File</Button>
                        </Upload>
                        <Button type="link">Reset</Button>
                    </div>
                </Form.Item>
                <Form.Item
                    name="local_host"
                    label="Local Host"
                >
                    <Input />
                </Form.Item>
                <Form.Item
                    label=" "
                >
                    <Button type="primary">Update</Button>
                </Form.Item>
            </Form>
        </>
    )
})

export default Server;