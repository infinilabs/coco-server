import { Select, Image } from "antd";
import { ReactSVG } from 'react-svg';
import "./icon.css";
import FontIcon from '@/components/common/font_icon'; 

export const IconSelector = ({value, onChange, className, icons=[]})=> {
  return <Select showSearch={true} value={value} className={className} popupMatchSelectWidth={false} onChange={onChange}>
    {icons.map(icon => {
        return <Select.Option value={icon.path} item={icon} >
          <div className="flex items-center gap-3px">
            {icon.source === "fonts" ? <FontIcon name={icon.path} /> : icon.path.endsWith(".svg") ? <ReactSVG src={icon.path} className="limitw" /> : <Image preview={false} width="1em" height="1em" src={icon.path}/>}
          
            <span className="overflow-hidden text-ellipsis">{icon.name}</span>
          </div>
        </Select.Option>
    })}
  </Select>
}