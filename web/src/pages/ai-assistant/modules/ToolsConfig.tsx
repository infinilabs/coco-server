import { InputNumber, Select, Space, Switch } from "antd";
import ModelSelect from './ModelSelect';

interface ToolsConfigProps {
  value?: any;
  onChange?: (value: string) => void;
}

export const ToolsConfig = (props: ToolsConfigProps) =>{
  const {t} = useTranslation();
  const { value={}, onChange } = props;
  const onEnabledChange = (enabled: boolean) => {
    onChange?.({
      ...value,
      enabled,
    })
  }

  const onBuiltinToolsChange = (toolKey:string, v: boolean) => {
    onChange?.({
      ...value,
      builtin: {
        ...value.builtin,
        [toolKey]: v,
      }
    })
  }
  if(!value.builtin){
    value.builtin = {};
  }

  return (
    <Space direction="vertical" className='w-600px mt-5px'>
      <div><Switch size="small" value={value.enabled} onChange={onEnabledChange}/></div>
      <div>
        <Space direction="vertical">
          <p className="text-[#999] mt-10px w-600px flex items-center" ><span>{t('page.assistant.labels.builtin_tools')}</span> </p>
          <div>
            <Space><span>Calculator</span><Switch onChange={(v)=>{onBuiltinToolsChange("calculator", v)}} value={value.builtin?.calculator} size="small" /></Space>
          </div>
          <div>
            <Space><span>Wikipedia</span><Switch onChange={(v)=>{onBuiltinToolsChange("wikipedia", v)}} value={value.builtin?.wikipedia} size="small" /></Space>
          </div>
          <div>
            <Space><span>Duckduckgo</span><Switch onChange={(v)=>{onBuiltinToolsChange("duckduckgo", v)}} value={value.builtin?.duckduckgo} size="small" /></Space>
          </div>
          <div>
            <Space><span>Scraper</span><Switch onChange={(v)=>{onBuiltinToolsChange("scraper", v)}} value={value.builtin?.scraper} size="small" /></Space>
          </div>
        </Space>
      </div>
    </Space>
  );

}