import { InputNumber, Radio, Select } from 'antd';
import { useCallback, useState, useEffect } from 'react';
import './index.scss';

const style = {
  display: 'flex',
  flexDirection: 'column',
  gap: 8
};
export const DataSync = ({ onChange, value = { enabled: true, interval: '1h', strategy: 'interval' } }) => {
  const { t } = useTranslation();
  const [innerV, setValue] = useState(value);

  // Update local state when value prop changes
  useEffect(() => {
    setValue(value);
  }, [value]);

  const onInnerChange = ev => {
    setValue(oldV => {
      const newV = {
        ...oldV,
        strategy: ev.target.value
      };
      if (typeof onChange === 'function') {
        onChange(newV);
      }
      return newV;
    });
  };
  const onIntervalChange = useCallback(interval => {
    setValue(oldV => {
      const newV = {
        ...oldV,
        interval
      };
      if (typeof onChange === 'function') {
        onChange(newV);
      }
      return newV;
    });
  }, []);
  return (
    <div id="data-sync-comp">
      <Radio.Group
        style={style}
        value={innerV.strategy}
        onChange={onInnerChange}
      >
        <Radio
          className="mb-10px flex"
          disabled={true}
          value="manual"
        >
          <div>
            <div className="mb-5px">{t('page.datasource.new.labels.manual_sync')}</div>
            <div className="text-gray-400">{t('page.datasource.new.labels.manual_sync_desc')}</div>
          </div>
        </Radio>
        <Radio
          className="mb-10px"
          value="interval"
        >
          <div>
            <div className="mb-5px">{t('page.datasource.new.labels.scheduled_sync')}</div>
            <div className="text-gray-400">{t('page.datasource.new.labels.scheduled_sync_desc')}</div>
          </div>
        </Radio>
        <div className="mb-10px ml-25px">
          <SyncTime
            value={innerV.interval}
            onChange={onIntervalChange}
          />
        </div>
        <Radio
          className="mb-10px"
          disabled={true}
          value="realtime"
        >
          <div>
            <div className="mb-5px">{t('page.datasource.new.labels.realtime_sync')}</div>
            <div className="text-gray-400">{t('page.datasource.new.labels.realtime_sync_desc')}</div>
          </div>
        </Radio>
      </Radio.Group>
    </div>
  );
};

const SyncTime = ({ onChange, value }) => {
  // Extract number and unit from value (default to "10s")
  const { t } = useTranslation();
  const match = value?.match(/^(\d+)([smh])$/);
  const initialNumber = match ? Number.parseInt(match[1], 10) : 10;
  const initialUnit = match ? match[2] : 's';

  const [num, setNum] = useState(initialNumber);
  const [unit, setUnit] = useState(initialUnit);

  // Update parent component when num or unit changes
  useEffect(() => {
    onChange?.(`${num}${unit}`);
  }, [num, unit, onChange]);

  return (
    <InputNumber
      addonBefore={t('page.datasource.every')}
      min={1}
      value={num}
      addonAfter={
        <Select
          getPopupContainer={triggerNode => triggerNode.parentNode}
          value={unit}
          options={[
            { label: t('page.datasource.seconds'), value: 's' },
            { label: t('page.datasource.minutes'), value: 'm' },
            { label: t('page.datasource.hours'), value: 'h' }
          ]}
          onChange={newUnit => setUnit(newUnit)}
        />
      }
      onChange={newNum => setNum(newNum || 1)}
    />
  );
};
