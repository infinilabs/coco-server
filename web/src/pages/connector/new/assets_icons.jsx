import { Input,Row, Col, Button } from "antd";
import { IconSelector } from "./icon_selector";
import {DeleteOutlined} from "@ant-design/icons";
import { useTranslation } from "react-i18next";
import { useState, useCallback } from "react";

export const AssetsIcons = ({value={}, onChange, iconsMeta=[]}) => {
  const {t} = useTranslation();
  const initialIcons = Object.keys(value).map((k) => {
    return {
      type: k,
      icon: value[k],
      key: new Date().getTime(),
    };
  });
  if(!initialIcons.length) {
    initialIcons.push({
      type: '',
      icon: '',
      key: new Date().getTime(),
    });
  }
  const transformIcons = (icons) => {
    return (icons || []).reduce((acc, icon)=>{
      if(!icon.type || !icon.icon){
        return acc;
      }
      acc[icon.type] = icon.icon;
      return acc;
    }, {});
  }
  const [icons, setIcons] = useState(initialIcons);
  const onIconChange = (oldKey, v) => {
    setIcons((oldIcons)=>{
      const idx = oldIcons.findIndex((icon)=> icon.key === oldKey);
      if(idx > -1) {
        oldIcons[idx] = v;
      }
      typeof onChange === "function" && onChange(transformIcons(oldIcons));
      return [
        ...oldIcons
      ];
    });
  }
  const onIconRemove = (v) => {
    setIcons((oldIcons)=>{
      oldIcons = oldIcons.filter((icon)=>{
        return icon.key !== v.key;
      })
      typeof onChange === "function" && onChange(transformIcons(oldIcons));
      return [
        ...oldIcons
      ];
    });
  }

  const onAddIcon = () => {
    setIcons((oldIcons)=>{
      return [...oldIcons, {
        type: '',
        icon: '',
        key : new Date().getTime(),
      }]
    });
  }
  return (
    <div className="assets-icons mt-3px">
      <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 32 }} className="w-100%">
        <Col key="icon" span={10}>{t('page.connector.new.labels.asset_icon')}</Col>
        <Col key="type" span={5}>{t('page.connector.new.labels.asset_type')}</Col>
        <Col key="oper" span={8}></Col>
      </Row>
      {icons.map((item) => {
        return <AssetsIcon icons={iconsMeta} key={item.key} value={item} onChange={onIconChange} onRemove={onIconRemove}/>
      })}
      <Button className="mt-10px" type="primary" onClick={onAddIcon}>{t('common.add')}</Button>
    </div>
  );
};

const AssetsIcon = ({value = {}, onChange, onRemove, icons=[]}) => {
  const [innerValue, setInnerValue] = useState(value);
  const onDeleteClick = () => {
   typeof onRemove === "function" && onRemove(value);
  }

  const onTypeChange = (e) => {
    setInnerValue(v=>{
      const newV = {
        ...v,
        type: e.target.value
      }
      onChange?.(value.key, newV);
      return newV;
    })
  }
  const onIconChange = (icon, option) => {
    setInnerValue(v=>{
      const newV = {
        ...v,
        icon
      }
      if(!newV.type) {
        newV.type = option?.item?.name || '';
      }
      onChange?.(value.key, newV);
      return newV;
    })
  }

  return <Row gutter={{ xs: 8, sm: 16, md: 24, lg: 32 }} className="w-100% my-5px">
        <Col span={10}>
          <IconSelector icons={icons} type="file" value={innerValue.icon || ''} onChange={onIconChange} />
        </Col>
        <Col span={5}>
            <Input value={innerValue.type || ''} onChange={onTypeChange} />
        </Col>
        <Col span={8} className="flex items-center" >
          <div className="cursor-pointer" onClick={onDeleteClick}><DeleteOutlined /></div>
        </Col>
  </Row>
};
