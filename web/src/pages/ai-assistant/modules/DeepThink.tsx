import { Form, Select, Space,Switch } from "antd";
import ModelSelect from "./ModelSelect";

interface DeepThinkProps {
  providers: any[];
  className?: string;
}

export const DeepThink = (props: DeepThinkProps) => {
  const {providers = [], className} = props;
  const {t} = useTranslation();
  return <div className={className}>
    <Space direction="vertical" className="w-100%">
      <div className="text-gray-400">{t('page.assistant.labels.intent_analysis_model')}</div>
      <Form.Item className="mb-[10px]" name={["config", "intent_analysis_model"]}>
        <ModelSelect
         providers={providers}
        />
      </Form.Item>
    </Space>
    <Space direction="vertical" className="w-100%">
      <div className="text-gray-400">{t('page.assistant.labels.picking_doc_model')}</div>
      <Form.Item className="mb-[10px]" name={["config", "picking_doc_model"]}>
       <ModelSelect
         providers={providers}
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
