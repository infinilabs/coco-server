import { Form, Input } from 'antd';

export function Component() {
  return (
    <div className="h-full min-h-500px">
      <ACard
        bordered={false}
        className="min-h-full flex-col-stretch sm:flex-1-hidden card-wrapper"
      >
        <div className="mb-30px ml--16px flex items-center text-lg font-bold">
          <div className="mr-20px h-1.2em w-10px bg-[#1677FF]" />
          <div>新建 Pipeline</div>
        </div>
        <div className="px-30px">
          <Form
            labelAlign="left"
            labelCol={{ span: 4 }}
          >
            <Form.Item label="名称">
              <Input />
            </Form.Item>
            <Form.Item label="配置">
              <Input.TextArea />
            </Form.Item>
          </Form>
        </div>
      </ACard>
    </div>
  );
}
