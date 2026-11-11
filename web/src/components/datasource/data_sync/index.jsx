import { InputNumber, Radio, Select } from 'antd';
import { useCallback, useState, useEffect } from 'react';
import './index.scss';

const style = {
  display: 'flex',
  flexDirection: 'column',
  gap: 8
};

// DataSync: pull-mode sync strategies only — manual waits for
// POST /datasource/:id/_sync, interval auto-dispatches on the configured
// cadence. Push mode (formerly the "realtime" option) is now owned by the
// datasource-level Webhook switch: when webhook is enabled the pages hide
// this block entirely and disable pull, so realtime is no longer selectable
// here. A legacy datasource stored with strategy=realtime but webhook off
// shows an inline note instead of a selectable option.
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
        {innerV.strategy === 'interval' && (
          <div className="mb-10px ml-25px">
            <SyncTime
              value={innerV.interval}
              onChange={onIntervalChange}
            />
          </div>
        )}
        {innerV.strategy === 'realtime' && (
          <div className="ml-25px" style={{ color: 'var(--ant-color-warning, #faad14)' }}>
            {t('page.datasource.new.labels.realtime_legacy_note')}
          </div>
        )}
      </Radio.Group>
    </div>
  );
};

const SyncTime = ({ onChange, value }) => {
  // Extract number and unit from value (default to "10s")
  const { t } = useTranslation();
  const match = value?.match(/^(\d+)([smh])$/);

  const [num, setNum] = useState(match ? Number.parseInt(match[1], 10) : 10);
  const [unit, setUnit] = useState(match ? match[2] : 's');

  // Follow incoming value changes: the edit page loads the stored interval
  // asynchronously after mount, so num/unit must track the prop instead of
  // freezing at the initial parse (P2-5 echo bug).
  useEffect(() => {
    const m = value?.match(/^(\d+)([smh])$/);
    if (m) {
      setNum(Number.parseInt(m[1], 10));
      setUnit(m[2]);
    }
  }, [value]);

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
