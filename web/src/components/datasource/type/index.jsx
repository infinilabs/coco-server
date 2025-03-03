import Icon, {UploadOutlined} from '@ant-design/icons';
import { useState, useCallback } from 'react';
import { GoogleDriveSVG, HugoSVG, YuqueSVG,NotionSVG,SearchSVG,BucketSVG } from '../../icons';
import { Button, Input, Upload } from 'antd';
import {IndexingScope} from "../indexing_scope"
import {MultiURLInput} from './urls';

const Types = {
  GoogleDrive: "google_drive",
  HugoSite: "hugo_site",
  Yuque: "yuque",
  Notion: "notion",
  ObjectStorage: "object_storage",
  Search: "search"
}

export const TypeList =  ({
  value = {
    id: Types.GoogleDrive,
    config: {},
  }, 
  onChange=()=>{},
  onTestClick,
})=>{
  const { t } = useTranslation();
  const [v, setValue] = useState(value);
  const onItemClick =(type)=>{
    setValue((oldV)=>{
      const newV = {
        ...oldV,
        id: type,
      }
      if(typeof onChange === "function"){
        onChange(newV);
      }
      return newV;
    });
  }
  const  onInnerTestClick = useCallback(()=>{
    if(typeof onTestClick === "function"){
      onTestClick(v)
    }
  }, [v])
  const onTokenChange = (tv)=>{
    setValue((oldV)=>{
      const newV = {
        ...oldV,
      }
      newV.config.token = tv.target.value;
      if(typeof onChange === "function"){
        onChange(newV);
      }
      return newV;
    });
  }
  const onIndexingScopeChange = (scope)=>{
    setValue((oldV)=>{
      const newV = {
        ...oldV,
      }
      newV.config = {
        ...oldV.config,
        ...scope,
      }
      if(typeof onChange === "function"){
        onChange(newV);
      }
      return newV;
    });
  }
  const onSiteURLsChange = (urls)=>{
    setValue((oldV)=>{
      const newV = {
        ...oldV,
        config: {
          ...oldV.config,
          urls: urls
        }
      }
      if(typeof onChange === "function"){
        onChange(newV);
      }
      return newV;
    });
  }
  const scope = {
    indexing_books: !!v.config.indexing_books,
    indexing_docs: !!v.config.indexing_docs,
    indexing_groups:!!v.config.indexing_groups,
    indexing_users: !!v.config.indexing_users,
    include_private_book: !!v.config.private_book,
    include_private_doc: !!v.config.include_private_doc,
  }

  const onCredentialChange = (credential)=>{
    setValue((oldV)=>{
      const newV = {
        ...oldV,
        config: {
          ...oldV.config,
          credential: credential,
        }
      }
      if(typeof onChange === "function"){
        onChange(newV);
      }
      return newV;
    });
  }
  return <div>
    <div className='flex gap-10px'>
      <TypeComponent onChange={onItemClick} icon={GoogleDriveSVG} text="Google Drive" selected={v.id===Types.GoogleDrive}  name={Types.GoogleDrive}/>
      <TypeComponent onChange={onItemClick} icon={HugoSVG} text="HUGO Site" selected={v.id===Types.HugoSite} name={Types.HugoSite}/>
      <TypeComponent onChange={onItemClick} icon={YuqueSVG} text="Yuque" selected={v.id===Types.Yuque} name={Types.Yuque}/>
      <TypeComponent onChange={onItemClick} icon={NotionSVG} text="Notion" selected={v.id===Types.Notion} name={Types.Notion}/>
      {/* <TypeComponent onChange={onItemClick} icon={BucketSVG} text="Object Storage" selected={v===Types.ObjectStorage} name={Types.ObjectStorage}/>
      <TypeComponent onChange={onItemClick} icon={SearchSVG} text="Search" selected={v===Types.Search} name={Types.Search}/> */}
    </div>
    {(v.id === Types.Notion || v.id === Types.Yuque) &&
    <div className='my-20px'>
      <div className='pb-8px text-gray-400'>Token</div>
      <div className='flex gap-5px'>
      <Input value={v.config.token} onChange={onTokenChange} className='max-w-500px'/><Button onClick={onInnerTestClick}>{t('common.testConnection')}</Button>
      </div>
    </div>}
    { v.id === Types.Yuque && <IndexingScope value={scope} onChange={onIndexingScopeChange}/>}
    {v.id === Types.HugoSite && <div className='my-20px'><MultiURLInput onChange={onSiteURLsChange}/></div>}
    { v.id === Types.GoogleDrive &&<div className='my-20px'>
      <div className='pb-8px text-gray-400'>Credential JSON File</div>
      <div className='flex gap-5px'>
      <FileUploader onChange={onCredentialChange}/>
      <Button onClick={onInnerTestClick}>{t('common.testConnection')}</Button>
      </div>
      </div>}
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

const FileUploader = ({onChange})=>{
  const [fileList, setFileList] = useState([])
  // const [credential, setCredential] = useState({});

  const props = {
    beforeUpload: (file) => {
    const isJson = file.type === "application/json";

    if (!isJson) {
      console.error("Only JSON files are allowed!");
      return Upload.LIST_IGNORE; // Prevents adding non-JSON files
    }

    const reader = new FileReader();
    reader.onload = (e) => {
      try {
        const fileContent = JSON.parse(e.target.result);
        if(typeof onChange === "function"){
          onChange(fileContent);
        }
      } catch (error) {
        console.error("Invalid JSON file!");
      }
    };

    reader.readAsText(file); // Read the file as text

    return false; // Prevents automatic upload
    },
    onRemove: () => {
      setCredential({})
    },
    fileList,
  };
  return (<Upload {...props} maxCount={1} accept=".json">
    <Button icon={<UploadOutlined />}>Click To Upload File</Button>
  </Upload>)
}