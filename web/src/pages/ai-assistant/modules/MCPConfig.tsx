import { InputNumber, Select, Space, Switch } from "antd";
import ModelSelect from "./ModelSelect";

interface MCPConfigProps {
  value?: any;
  onChange?: (value: string) => void;
  options: any[];
  modelProviders: any[];
}

export const MCPConfig = (props: MCPConfigProps) => {
  const { t } = useTranslation();
  const { value = {}, onChange } = props;
  const onIDsChange = (newIds: string[]) => {
    if (onChange) {
      onChange({
        ...value,
        ids: newIds,
      });
    }
  };
  const onEnabledChange = (enabled: boolean) => {
    onChange?.({
      ...value,
      enabled,
      enabled_by_default: enabled === false ? false : value.enabled_by_default,
      visible: enabled === false ? false : value.visible,
    });
  };
  const onVisibleChange = (visible: boolean) => {
    onChange?.({
      ...value,
      visible,
    });
  };
  const onEnabled_by_defaultChange = (enabled_by_default: boolean) => {
    onChange?.({
      ...value,
      enabled_by_default,
    });
  };
  const onMaxIterationsChange = (maxIterations: number | null) => {
    onChange?.({
      ...value,
      max_iterations: maxIterations,
    });
  };

  const onModelChange = (model: any) => {
    onChange?.({
      ...value,
      model: model,
    });
  };

  return (
    <Space direction="vertical" className="w-600px mt-[5px]">
      <div>
        <Switch size="small" value={value.enabled} onChange={onEnabledChange} />
      </div>
      <Select
        onChange={onIDsChange}
        mode="multiple"
        allowClear
        options={props.options}
        value={value.ids}
      />
      <div>
        <Space>
          <span>{t("page.assistant.labels.show_in_chat")}</span>
          <Switch
            value={value.visible}
            size="small"
            onChange={onVisibleChange}
          />
        </Space>
      </div>
      <div>
        <Space>
          <span>{t("page.assistant.labels.enabled_by_default")}</span>
          <Switch
            value={value.enabled_by_default}
            size="small"
            onChange={onEnabled_by_defaultChange}
          />
        </Space>
      </div>
      <div>
        <Space direction="vertical">
          <p className="text-[#999] mt-10px">
            {t("page.assistant.labels.max_iterations")}
          </p>
          <InputNumber
            min={1}
            max={100}
            value={value.max_iterations || 5}
            onChange={onMaxIterationsChange}
            defaultValue={5}
          />
        </Space>
      </div>
      <div>
        <Space direction="vertical">
          <p className="text-[#999] mt-10px">
            {t("page.assistant.labels.caller_model")}
          </p>
          <ModelSelect
            value={value.model}
            onChange={onModelChange}
            width="600px"
            providers={props.modelProviders}
          />
        </Space>
      </div>
    </Space>
  );
};
