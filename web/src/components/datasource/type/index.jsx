import Icon from '@ant-design/icons';
import { useState, useCallback } from 'react';
import { GoogleDriveSVG, HugoSVG, YuqueSVG,NotionSVG,SearchSVG,BucketSVG } from '../../icons';
import { Button, Input } from 'antd';

const Types = {
  GoogleDrive: "google_drive",
  HugoSite: "hugo_site",
  Yuque: "yuque",
  Notion: "notion",
  ObjectStorage: "object_storage",
  Search: "search"
}

export const TypeList =  ({
  value, 
  onChange=()=>{},
  showTest = false,
  onTestClick,
})=>{
  const [v, setValue] = useState(value);
  const onItemClick =(newV)=>{
    setValue(newV);
    onChange(newV);
  }
  const [token, setToken] = useState("");
  const  onInnerTestClick = useCallback(()=>{
    if(typeof onTestClick === "function"){
      onTestClick(v, token)
    }
  }, [v, token])
  return <div>
    <div className='flex gap-10px'>
      <TypeComponent onChange={onItemClick} icon={GoogleDriveSVG} text="Google Drive" selected={v===Types.GoogleDrive}  name={Types.GoogleDrive}/>
      <TypeComponent onChange={onItemClick} icon={HugoSVG} text="HUGO Site" selected={v===Types.HugoSite} name={Types.HugoSite}/>
      <TypeComponent onChange={onItemClick} icon={YuqueSVG} text="Yuque" selected={v===Types.Yuque} name={Types.Yuque}/>
      <TypeComponent onChange={onItemClick} icon={NotionSVG} text="Notion" selected={v===Types.Notion} name={Types.Notion}/>
      <TypeComponent onChange={onItemClick} icon={BucketSVG} text="Object Storage" selected={v===Types.ObjectStorage} name={Types.ObjectStorage}/>
      <TypeComponent onChange={onItemClick} icon={SearchSVG} text="Search" selected={v===Types.Search} name={Types.Search}/>
    </div>
    {showTest? <div className='my-20px'>
      <div className='pb-8px text-gray-400'>Token</div>
      <div className='flex gap-5px'>
      <Input value={token} onChange={setToken} className='max-w-500px'/><Button onClick={onInnerTestClick}>连接测试</Button>
      </div>
    </div>:null}
  </div>
  
}

const TypeComponent = ({
  icon,
  text,
  selected = false,
  name,
  onChange=()=>{}
})=>{

  return <div onClick={()=>{
    onChange(name)
  }} className={"border flex items-center px-10px py-5px rounded-md min-w-120px justify-center hover:border-blue-500 hover:text-blue-500 cursor-pointer"+(selected? " border-blue-500 text-blue-500": "")}>
  <Icon component={icon}/>
  <span className="ml-2">{text}</span>
</div>
}