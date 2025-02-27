import { Button, Form, Input, InputNumber, Select, Switch, Upload } from "antd";
import "../index.scss"

const ADVANCED = [
    {
        key: '1',
        label: '随机性 (temperature)',
        desc: '值越大，回复越随机',
        input: <InputNumber min={0.1} step={0.1} defaultValue={0.1} />
    },
    {
        key: '2',
        label: '核采样 (top_p)',
        desc: '与随机性类似，但不要和随机性一起更改',
        input: <InputNumber min={0.1} step={0.1} defaultValue={0.1} />
    },
    {
        key: '3',
        label: '单次回复限制(max tokens)',
        desc: '单次交互所用的最大 Token 数',
        input: <InputNumber min={1} step={1} precision={0} defaultValue={40000} />
    },
    {
        key: '3',
        label: '话题新鲜度(presence_penalty)',
        desc: '值越大，越有可能扩展到新话题',
        input: <InputNumber min={0.1} step={0.1} defaultValue={0.1} />
    },
    {
        key: '4',
        label: '频率惩罚度 (frequency_penalty)',
        desc: '值越大，越有可能降低重复字词',
        input: <InputNumber min={0.1} step={0.1} defaultValue={0.1} />
    },
    {
        key: '5',
        label: '开启推理强度调整',
        input: <Switch size="small" defaultChecked />
    },
    {
        key: '6',
        label: '推理强度',
        desc: '值越大，推理能力越强，但可能会增加响应时间和 Token 消耗',
        input: (
            <Select
                defaultValue="低"
                style={{ width: 88 }}
                options={[
                    { value: '低', label: '低' },
                    { value: '中', label: '中' },
                    { value: '高', label: '高' },
                ]}
            />
        )
    },
]

const LLM = memo(() => {
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
                    name="access_mode"
                    label="Access Mode"
                >
                    <Button className="w-104px m-r-10px">Builtin</Button>
                    <Button className="w-104px">API</Button>
                </Form.Item>
                <Form.Item
                    name="ollama_host"
                    label="Ollama Host"
                >
                    <Input />
                </Form.Item>
                <Form.Item
                    name="ollama_model"
                    label="Full Model"
                >
                    <Input />
                </Form.Item>
                <Form.Item
                    label=" "
                >
                    <Button type="link" className="p-0">
                        Advanced <SvgIcon icon="mdi:chevron-down"/>
                    </Button>
                </Form.Item>
                <Form.Item
                    label="Request Params"
                >
                    {
                        ADVANCED.map((item, index) => (
                            <div key={item.key} className={`flex justify-between items-center ${index !== ADVANCED.length - 1 ? 'm-b-24px' : ''} `}>
                                <div>
                                    <div className="color-#333">{item.label}</div>
                                    <div className="color-#999">{item.desc}</div>
                                </div>
                                <div>
                                    {item.input}
                                </div>
                            </div>
                        ))
                    }
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

export default LLM;