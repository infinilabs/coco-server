import { Button, Form, Input } from "antd";
import { FormInstance } from "antd/lib";
import { ReactSVG } from "react-svg";
import OllamaSvg from '@/assets/svg-icon/ollama.svg'
import OpenAISvg from '@/assets/svg-icon/openai.svg'
import DeepseekSvg from '@/assets/svg-icon/deepseek.svg'
import ButtonRadio from "@/components/button-radio";

type ModelType = 'deepseek' | 'ollama' | 'openai'

const LLMForm = memo(({ form, onSubmit, loading }: { form: FormInstance, onSubmit: (isPass?: boolean) => void; loading: boolean }) => {
    const formItemClassNames = "m-b-32px"
    const inputClassNames = "h-40px"
    const { t } = useTranslation();
    const { defaultRequiredRule, formRules } = useFormRules();
    const [type, setType] = useState<ModelType>('deepseek')

    return (
        <>
            <div className="text-32px color-#333 m-b-16px">
                {t('page.guide.llm.title')}
            </div>
            <div className="text-16px color-#999 m-b-64px">
                {t('page.guide.llm.desc')}
            </div>
            <Form
                form={form}
                layout="vertical"
                initialValues={{
                    llm: { 
                        type,
                        keepalive: '30m'
                    }
                }}
              >
                <Form.Item
                    name={['llm', 'type']}
                    label={t(`page.settings.llm.type`)}
                    className={formItemClassNames}
                    rules={[defaultRequiredRule]}
                >
                    <ButtonRadio
                        options={[
                            { value: 'deepseek', label: <span className="flex items-center deepseek-icon"><ReactSVG src={DeepseekSvg} className="m-r-4px"/>Deepseek</span>},
                            { value: 'ollama', label: <span className="flex items-center"><ReactSVG src={OllamaSvg} className="m-r-4px"/>Ollama</span>},
                            { value: 'openai', label: <span className="flex items-center"><ReactSVG src={OpenAISvg} className="m-r-4px"/>OpenAI</span>}
                        ]}
                        onChange={(value: ModelType) => {
                            setType(value)
                            form.setFieldsValue({ token: undefined })
                        }}
                    />
                </Form.Item>
                <Form.Item
                    name={['llm', 'endpoint']}
                    label={t(`page.settings.llm.endpoint`)}
                    className={formItemClassNames}
                    rules={formRules.endpoint}
                >
                    <Input className={inputClassNames}/>
                </Form.Item>
                <Form.Item
                    name={['llm', 'default_model']}
                    label={t(`page.settings.llm.defaultModel`)}
                    className={formItemClassNames}
                    rules={[defaultRequiredRule]}
                >
                    <Input />
                </Form.Item>
                <Form.Item
                    name={['llm', 'keepalive']}
                    label={t(`page.settings.llm.keepalive`)}
                    className={formItemClassNames}
                    rules={[defaultRequiredRule]}
                >
                    <Input />
                </Form.Item>
                {
                    (type === 'openai' || type === 'deepseek') && (
                        <Form.Item
                            name={['llm', 'token']}
                            label={'Token'}
                            className={formItemClassNames}
                            rules={[defaultRequiredRule]}
                        >
                            <Input.Password />
                        </Form.Item>
                    )
                }
                <div className="flex justify-between">
                    <Button type="link" size="large" className="h-56px text-14px px-0" onClick={() => onSubmit(true)}>{t('page.guide.setupLater')}</Button>
                    <Button loading={loading} type="primary" size="large" className="w-56px h-56px text-24px" onClick={() => onSubmit()}>
                      <SvgIcon icon="mdi:check" />
                    </Button>
                </div>
            </Form>
        </>
    )
})

export default LLMForm;