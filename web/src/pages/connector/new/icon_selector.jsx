import { Select, Image } from "antd"

export const IconSelector = ({value, onChange, className, icons=[]})=> {
  return <Select showSearch={true} value={value}  className={className} onChange={onChange}>
    {icons.map(icon => {
        return <Select.Option value={icon.path} item={icon} >
          <div className="flex items-center gap-3px"><Image preview={false} width="1em" height="1em" src={icon.path}/><span>{icon.name}</span></div>
        </Select.Option>
    })}
  </Select>
}