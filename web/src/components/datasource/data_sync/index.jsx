import { Radio } from "antd"
import { useState } from "react";
import WorkTime from "../work_time";
import { InputNumber } from "antd";
import { Select } from "antd";
import "./index.scss";

const style = {
  display: 'flex',
  flexDirection: 'column',
  gap: 8,
};
export const DataSync = ({
  value = {sync_type: "interval", interval: "60s"},
  onChange,
})=>{
  const { t } = useTranslation();
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
  return <div id="data-sync-comp">
     <Radio.Group value={innerV.sync_type} onChange={onInnerChange} style={style}>
        <Radio  className="mb-10px flex" value="manual">
          <div>
          <div className="mb-5px">{t('page.datasource.new.labels.manual_sync')}</div>
          <div className="text-gray-400">{t('page.datasource.new.labels.manual_sync_desc')}</div>
        </div>
        </Radio>
        <Radio className="mb-10px" value="interval">
          <div>
            <div className="mb-5px">{t('page.datasource.new.labels.scheduled_sync')}</div>
            <div className="text-gray-400">{t('page.datasource.new.labels.scheduled_sync_desc')}</div> 
          </div>
        </Radio>
        <div className="ml-25px mb-10px"><SyncTime value={innerV.interval}/></div>
        <Radio className="mb-10px" value="realtime">
          <div>
          <div className="mb-5px">{t('page.datasource.new.labels.realtime_sync')}</div>
          <div className="text-gray-400">{t('page.datasource.new.labels.realtime_sync_desc')}</div> 
        </div>
        </Radio>
    </Radio.Group>
   
  </div>
}

const SyncTime = ({ value, onChange }) => {
  // Extract number and unit from value (default to "10s")
  const match = value?.match(/^(\d+)([smh])$/);
  const initialNumber = match ? parseInt(match[1], 10) : 10;
  const initialUnit = match ? match[2] : "s";

  const [num, setNum] = useState(initialNumber);
  const [unit, setUnit] = useState(initialUnit);

  // Update parent component when num or unit changes
  useEffect(() => {
    onChange?.(`${num}${unit}`);
  }, [num, unit, onChange]);

  return (
    <InputNumber
      addonBefore="Every"
      value={num}
      min={1}
      onChange={(newNum) => setNum(newNum || 1)}
      addonAfter={
        <Select
          getPopupContainer={(triggerNode) => triggerNode.parentNode}
          value={unit}
          onChange={(newUnit) => setUnit(newUnit)}
          options={[
            { value: "s", label: "second" },
            { value: "m", label: "minutes" },
            { value: "h", label: "hour" },
          ]}
        />
      }
    />
  );
};