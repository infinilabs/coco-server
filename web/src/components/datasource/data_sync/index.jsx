import { Radio } from "antd"
import { useState } from "react";
import WorkTime from "../work_time";

const style = {
  display: 'flex',
  flexDirection: 'column',
  gap: 8,
};
export const DataSync = ({
  value = {sync_type: "manual",},
  onChange,
})=>{
  const [innerV, setValue] = useState(value)
  const onInnerChange = (ev)=>{
    setValue(oldV=>{
      const newV = {
        ...oldV,
        sync_type: ev.target.value,
      }
      if(typeof onChange === "function"){
        onChange(newV);
      }
      return newV;
    })
  }
  return <div>
     <Radio.Group value={innerV.sync_type} onChange={onInnerChange} style={style}>
        <Radio className="mb-10px" value="manual">
          <div>
          <div className="mb-5px">手动同步</div>
          <div className="text-gray-400">仅在用户点击 "同步" 按钮时同步</div>
        </div>
        </Radio>
        <Radio className="mb-10px" value="schedule">
          <div>
            <div className="mb-5px">定时同步</div>
            <div className="text-gray-400 mb-15px">每隔固定时间同步一次</div> 
            <WorkTime/>
          </div>
        </Radio>
        <Radio className="mb-10px" value="realtime">
          <div>
          <div className="mb-5px">实时同步</div>
          <div className="text-gray-400">文件修改立即同步</div> 
        </div>
        </Radio>
    </Radio.Group>
   
  </div>
}