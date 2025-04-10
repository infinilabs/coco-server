import { Form, Select, Space,Switch } from "antd"

interface DeepThinkProps {
  modelOptions: any[];
}

export const DeepThink = (props: DeepThinkProps) => {
  const {modelOptions = []} = props;
  const {t} = useTranslation();
  return <div>
    <Space direction="vertical" className="w-100%">
      <div className="text-gray-400">{t('page.assistant.labels.intent_analysis_model')}</div>
      <Form.Item className="mb-[10px]" name={["config", "intent_analysis_model", "name"]}>
        <ModelSelect
          options={modelOptions.map((item) => ({
            label: item,
            value: item,
          }))}
        />
      </Form.Item>
    </Space>
    <Space direction="vertical" className="w-100%">
      <div className="text-gray-400">{t('page.assistant.labels.picking_doc_model')}</div>
      <Form.Item className="mb-[10px]" name={["config", "picking_doc_model", "name"]}>
        <ModelSelect
          options={modelOptions.map((item) => ({
            label: item,
            value: item,
          }))}
        />
      </Form.Item>
    </Space>
    <div>
        <Space>
          <span>{t('page.assistant.labels.show_in_chat')}</span>
          <Form.Item className="my-[0px]" name={["config", "visible"]}> 
            <Switch size="small"/>
          </Form.Item>
        </Space>
      </div>
  </div>
}


export const ModelSelect = ({ value, onChange, options=[] }: any) => {
  return (<Select
    className="max-w-[600px]"
    mode="tags"
    maxCount={1}
    showSearch
    value={value}
    options={options}
    onChange={(value) => {
      const selectedValue = Array.isArray(value) ? value[0] : '';
      onChange?.(selectedValue);
    }}
  />)
}