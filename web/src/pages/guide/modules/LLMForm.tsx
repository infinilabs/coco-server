import { Button, Form, Input } from 'antd';
import type { FormInstance } from 'antd/lib';
import { ReactSVG } from 'react-svg';

import DeepseekSvg from '@/assets/svg-icon/deepseek.svg';
import OllamaSvg from '@/assets/svg-icon/ollama.svg';
import OpenAISvg from '@/assets/svg-icon/openai.svg';

type ModelType = 'deepseek' | 'ollama' | 'openai';

const LLMForm = memo(
  ({ form, loading, onSubmit }: { form: FormInstance; loading: boolean; onSubmit: (isPass?: boolean) => void }) => {
    const formItemClassNames = 'm-b-32px';
    const inputClassNames = 'h-40px';
    const { t } = useTranslation();
    const { defaultRequiredRule, formRules } = useFormRules();
    const [type, setType] = useState<ModelType>('deepseek');

    return (
      <>
        <div className="m-b-16px text-32px color-[var(--ant-color-text-heading)]">{t('page.guide.llm.title')}</div>
        <div className="m-b-64px text-16px color-[var(--ant-color-text)]">{t('page.guide.llm.desc')}</div>
        <Form
          form={form}
          layout="vertical"
          initialValues={{
            llm: {
              keepalive: '30m',
              type
            }
          }}
        >
          <Form.Item
            className={formItemClassNames}
            label={t(`page.settings.llm.type`)}
            name={['llm', 'type']}
            rules={[defaultRequiredRule]}
          >
            <ButtonRadio
              options={[
                {
                  label: (
                    <span className="deepseek-icon flex items-center">
                      <ReactSVG
                        className="m-r-4px"
                        src={DeepseekSvg}
                      />
                      Deepseek
                    </span>
                  ),
                  value: 'deepseek'
                },
                {
                  label: (
                    <span className="flex items-center">
                      <ReactSVG
                        className="m-r-4px"
                        src={OllamaSvg}
                      />
                      Ollama
                    </span>
                  ),
                  value: 'ollama'
                },
                {
                  label: (
                    <span className="flex items-center">
                      <ReactSVG
                        className="m-r-4px"
                        src={OpenAISvg}
                      />
                      OpenAI
                    </span>
                  ),
                  value: 'openai'
                }
              ]}
              onChange={(value: ModelType) => {
                setType(value);
                form.setFieldsValue({ token: undefined });
              }}
            />
          </Form.Item>
          <Form.Item
            className={formItemClassNames}
            label={t(`page.settings.llm.endpoint`)}
            name={['llm', 'endpoint']}
            rules={formRules.endpoint}
          >
            <Input className={inputClassNames} />
          </Form.Item>
          <Form.Item
            className={formItemClassNames}
            label={t(`page.settings.llm.defaultModel`)}
            name={['llm', 'default_model']}
            rules={[defaultRequiredRule]}
          >
            <Input />
          </Form.Item>
          <Form.Item
            className={formItemClassNames}
            label={t(`page.settings.llm.keepalive`)}
            name={['llm', 'keepalive']}
            rules={[defaultRequiredRule]}
          >
            <Input />
          </Form.Item>
          {(type === 'openai' || type === 'deepseek') && (
            <Form.Item
              className={formItemClassNames}
              label="Token"
              name={['llm', 'token']}
              rules={[defaultRequiredRule]}
            >
              <Input.Password />
            </Form.Item>
          )}
          <div className="flex justify-between">
            <Button
              className="h-56px px-0 text-14px"
              size="large"
              type="link"
              onClick={() => onSubmit(true)}
            >
              {t('page.guide.setupLater')}
            </Button>
            <Button
              className="h-56px w-56px text-24px"
              loading={loading}
              size="large"
              type="primary"
              onClick={() => onSubmit()}
            >
              <SvgIcon icon="mdi:check" />
            </Button>
          </div>
        </Form>
      </>
    );
  }
);

export default LLMForm;
