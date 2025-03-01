import { Button, Form, Input } from "antd";
import { FormInstance } from "antd/lib";

const LLMForm = memo(({ form, onSubmit }: { form: FormInstance, onSubmit: () => void }) => {
    const formItemClassNames = "m-b-32px"
    const inputClassNames = "h-40px"

    return (
        <>
            <div className="text-28px color-#333 m-b-16px">
                Connect to a Large Model
            </div>
            <div className="text-14px color-#999 m-b-64px">
              After integrating a large model, you will unlock the AI chat feature, providing intelligent search and an efficient work assistant.
            </div>
            <Form
                className=""
                form={form}
                layout="vertical"
              >
                <Form.Item
                    name="integration_method"
                    label="Integration method"
                    className={formItemClassNames}
                >
                  <div className="flex justify-between gap-24px">
                    <Button className="h-40px w-[calc((100%-24px)/2)]">Self-built model</Button>
                    <Button className="h-40px w-[calc((100%-24px)/2)]">API call</Button>
                  </div>
                </Form.Item>
                <Form.Item
                    name="ollama_host"
                    label="Endpoint"
                    className={formItemClassNames}
                >
                    <Input className={inputClassNames}/>
                </Form.Item>
                <Form.Item
                    name="ollama_model"
                    label="Default Model"
                    className={formItemClassNames}
                >
                    <Input className={inputClassNames}/>
                </Form.Item>
                <div className="flex justify-between">
                    <Button type="link" size="large" className="h-56px text-14px px-0" onClick={() => onSubmit()}>Set Up Later</Button>
                    <Button type="primary" size="large" className="w-56px h-56px text-24px" onClick={() => onSubmit()}>
                      <SvgIcon icon="mdi:check" />
                    </Button>
                </div>
            </Form>
        </>
    )
})

export default LLMForm;