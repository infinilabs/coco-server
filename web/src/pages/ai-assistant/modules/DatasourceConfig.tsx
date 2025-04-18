import { Select, Space, Switch } from "antd";

interface DatasourceConfigProps {
  value?: any;
  onChange?: (value: string) => void;
  options: any[];
}

export const DatasourceConfig = (props: DatasourceConfigProps) =>{
  const {t} = useTranslation();
  const { value={}, onChange } = props;
  const onIDsChange = (newIds: string[])=>{
    if (onChange) {
      onChange({
        ...value,
        ids: newIds,
      });
    }
  }
  const onEnabledChange = (enabled: boolean) => {
    onChange?.({
      ...value,
      enabled,
    })
  }
  const onVisibleChange = (visible: boolean) => {
    onChange?.({
      ...value,
      visible,
    })
  }
  return (
    <Space direction="vertical" className='w-600px mt-[5px]'>
      <div><Switch size="small" value={value.enabled} onChange={onEnabledChange}/></div>
      <Select
        onChange={onIDsChange}
        mode="multiple"
        allowClear
        options={props.options}
        value={value.ids}
      />
      <div>
        <Space>
          <span>{t('page.assistant.labels.show_in_chat')}</span><Switch value={value.visible} size="small" onChange={onVisibleChange}/>
        </Space>
      </div>
    </Space>
  );

}