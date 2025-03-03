import { Button, Form, Input, InputNumber, Radio, Select, Spin, Switch } from "antd";
import "../index.scss"
import OllamaSvg from '@/assets/svg-icon/ollama.svg'
import OpenAISvg from '@/assets/svg-icon/openai.svg'
import { ReactSVG } from 'react-svg';
import { useLoading } from '@sa/hooks';
import { updateSettings } from "@/service/api/server";

const ADVANCED = [
    {
        key: 'temperature',
        input: <InputNumber min={0.1} step={0.1} defaultValue={0.1} />
    },
    {
        key: 'topP',
        input: <InputNumber min={0.1} step={0.1} defaultValue={0.1} />
    },
    {
        key: 'maxTokens',
        input: <InputNumber min={1} step={1} precision={0} defaultValue={40000} />
    },
    {
        key: 'presencePenalty',
        input: <InputNumber min={0.1} step={0.1} defaultValue={0.1} />
    },
    {
        key: 'frequencyPenalty',
        input: <InputNumber min={0.1} step={0.1} defaultValue={0.1} />
    },
    {
        key: 'enhancedInference',
        input: <Switch size="small" defaultChecked />,
        hideDesc: true
    },
    // {
    //     key: '7',
    //     label: '推理强度',
    //     desc: '值越大，推理能力越强，但可能会增加响应时间和 Token 消耗',
    //     input: (
    //         <Select
    //             defaultValue="低"
    //             style={{ width: 88 }}
    //             options={[
    //                 { value: '低', label: '低' },
    //                 { value: '中', label: '中' },
    //                 { value: '高', label: '高' },
    //             ]}
    //         />
    //     )
    // },
]

const LLM = memo(() => {
    const [form] = Form.useForm();
    const { t } = useTranslation();

    const [type, setType] = useState<'ollama' | 'openai'>('ollama')
    const [showAdvanced, setShowAdvanced] = useState(false)

    const { endLoading, loading, startLoading } = useLoading();
    const { defaultRequiredRule } = useFormRules();

    const models = {
        'ollama': [
            { 'value': 'deepseek_r1', 'label': 'DeepSeek-R1'}
        ],
        'openai': [
            { 'value': 'openai', 'label': 'OpenAI'}
        ]
    }

    const handleSubmit = async () => {
        const params = await form.validateFields();
        startLoading()
        await updateSettings({
            llm: params
        });
        endLoading()
    }

    return (
        <Spin spinning={loading}>
            <Form 
                form={form}
                labelAlign="left"
                className="settings-form"
                colon={false}
                initialValues={{ 
                    type,
                    "endpoint":"http://127.0.0.1:8000",
                    "default_model":"deepseek_r1",
                    "parameters":{
                            "top_p":111,
                            "max_tokens":32000,
                            "presence_penalty":0.9,
                            "frequency_penalty":0.9,
                            "enhanced_inference":true,
                    } 
                }}
            >
                <Form.Item
                    name="type"
                    label={t(`page.settings.llm.type`)}
                    rules={[defaultRequiredRule]}
                >
                    <Radio.Group onChange={(e) => {
                        setType(e.target.value)
                        form.setFieldsValue({ default_model: '' })
                    }}>
                        <Radio.Button value="ollama">
                            <span className="flex items-center"><ReactSVG src={OllamaSvg} className="m-r-4px"/>Ollama</span>
                        </Radio.Button>
                        <Radio.Button value="openai">
                            <span className="flex items-center"><ReactSVG src={OpenAISvg} className="m-r-4px"/>OpenAI</span>
                        </Radio.Button>
                    </Radio.Group>
                </Form.Item>
                <Form.Item
                    name="endpoint"
                    label={t(`page.settings.llm.endpoint`)}
                    rules={[defaultRequiredRule]}
                >
                    <Input />
                </Form.Item>
                <Form.Item
                    name="default_model"
                    label={t(`page.settings.llm.defaultModel`)}
                    rules={[defaultRequiredRule]}
                >
                    <Select 
                        options={models[type] as never[]}
                    />
                </Form.Item>
                <Form.Item
                    label=" "
                >
                    <Button type="link" className="p-0" onClick={() => setShowAdvanced(!showAdvanced)}>
                        {t('common.advanced')} <SvgIcon icon={`${showAdvanced ? "mdi:chevron-down" : "mdi:chevron-up"}`}/>
                    </Button>
                </Form.Item>
                <Form.Item
                    label={t(`page.settings.llm.requestParams`)}
                    className={`${showAdvanced ? '' : 'h-0px m-0px overflow-hidden'}`}
                >
                    {
                        ADVANCED.map((item) => (
                            <div key={item.key} className={`flex justify-between items-center`}>
                                <div className="[flex:1]">
                                    <div className="color-#333">{t(`page.settings.llm.${item.key}`)}</div>
                                    {!item.hideDesc && <div className="color-#999">{t(`page.settings.llm.${item.key}Desc`)}</div>}
                                </div>
                                <div >
                                    <Form.Item
                                        name={['parameters', item.key]}
                                        label=""
                                    >
                                        {item.input}
                                    </Form.Item>
                                </div>
                            </div>
                        ))
                    }
                </Form.Item>
                <Form.Item
                    label=" "
                >
                    <Button type="primary" onClick={() => handleSubmit()}>{t('common.update')}</Button>
                </Form.Item>
            </Form>
        </Spin>
    )
})

export default LLM;