import { Button, Form, Input, Upload } from 'antd';
import '../index.scss';

const Server = memo(() => {
  const [form] = Form.useForm();

  return (
    <Form
      className="settings-form"
      colon={false}
      form={form}
      labelAlign="left"
    >
      <Form.Item
        label="Server Name"
        name="name"
      >
        <Input />
      </Form.Item>
      <Form.Item
        label="Description"
        name="desc"
      >
        <Input.TextArea autoSize={{ maxRows: 5, minRows: 3 }} />
      </Form.Item>
      <Form.Item
        label="Logo"
        name="logo"
      >
        <div className="flex">
          <Upload>
            <Button icon={<SvgIcon icon="mdi:upload" />}>Upload File</Button>
          </Upload>
          <Button type="link">Reset</Button>
        </div>
      </Form.Item>
      <Form.Item
        label="Banner"
        name="banner"
      >
        <div className="flex">
          <Upload>
            <Button icon={<SvgIcon icon="mdi:upload" />}>Upload File</Button>
          </Upload>
          <Button type="link">Reset</Button>
        </div>
      </Form.Item>
      <Form.Item
        label="Local Host"
        name="local_host"
      >
        <Input />
      </Form.Item>
      <Form.Item label=" ">
        <Button type="primary">Update</Button>
      </Form.Item>
    </Form>
  );
});

export default Server;
