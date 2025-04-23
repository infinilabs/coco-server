import { Select, Space, Switch, Input } from "antd";

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

  const [showFilter, setShowFilter] = useState(!!value.filter);
  const onFilterToggle = () => {
    setShowFilter(!showFilter);
  }

  const onFilterChange = (evt: any) => {
    const filter = evt.target.value;
    onChange?.({
      ...value,
      filter,
    })
  }

  if(isObject(value.filter)){
    value.filter = JSON.stringify(value.filter);
  }

  const filterPlaceHolder = `{
  "term": {
     "name": "test"
  }
}`;

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
      <div>
         <Space direction="vertical">
          <p className="text-blue-500 mt-10px w-600px flex cursor-pointer items-center" onClick={onFilterToggle}><span>{t('page.assistant.labels.filter')}</span> <SvgIcon className="font-size-20px" icon={`${showFilter ? "mdi:chevron-up" : "mdi:chevron-down"}`}/></p>
          {showFilter && <Input.TextArea placeholder={filterPlaceHolder} value={value.filter} onChange={onFilterChange} style={{height:150}}/>}
        </Space>
      </div>
    </Space>
  );

}

function isObject(obj: any) {
  return Object.prototype.toString.call(obj) === '[object Object]';
}