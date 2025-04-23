import { InputNumber, Select } from 'antd';
import { timeUnits } from './utils/time_units';
import { parseTimeParts } from './utils/quick_select_utils';
import styles from './QuickSelect.less';
import RefreshInterval from './RefreshInterval';
import Apply from './Apply';
import { useState } from 'react';

const LAST = 'last';

const timeTenseOptions = [
  { value: LAST, text: 'Last' },
];
const timeUnitsOptions = Object.keys(timeUnits).map(key => {
  return { value: key, text: `${timeUnits[key]}s` };
});

const QuickSelect = props => {
  const { currentLocales, start, end, prevQuickSelect, onRangeChange, onCancel } = props;

  const [time, setTime] = useState(() => {
    const {
      timeTense: timeTenseDefault,
      timeUnits: timeUnitsDefault,
      timeValue: timeValueDefault,
    } = parseTimeParts(start, end);
    return {
      timeTense: prevQuickSelect?.timeTense || timeTenseDefault,
      timeValue: prevQuickSelect?.timeValue || timeValueDefault,
      timeUnits: prevQuickSelect?.timeUnits || timeUnitsDefault,
    };
  });

  const { timeTense, timeUnits, timeValue } = time;

  const onTimeTenseChange = value => {
    setTime({
      ...time,
      timeTense: value,
    });
  };

  const onTimeValueChange = value => {
    setTime({
      ...time,
      timeValue: value,
    });
  };

  const onTimeUnitsChange = value => {
    setTime({
      ...time,
      timeUnits: value,
    });
  };

  const handleApply = () => {
    const { timeValue, timeUnits } = time;

    onRangeChange({
      start: `now-${timeValue}${timeUnits}`,
      end: 'now',
      quickSelect: time,
    });
    onCancel()
  };

  return (
    <div className={styles.quickSelect}>
      <div className={styles.title}>{currentLocales[`datepicker.quick_select`]}</div>
      <div className={styles.form}>
        <Select value={timeTense} onChange={onTimeTenseChange} style={{ width: '100%' }}>
          {timeTenseOptions.map(item => (
            <Select.Option key={item.value} value={item.value}>
              {currentLocales[`datepicker.quick_select.${item.value}`]}
            </Select.Option>
          ))}
        </Select>
        <InputNumber
          min={1}
          value={timeValue}
          style={{ width: '100%' }}
          onChange={onTimeValueChange}
        />
        <Select value={timeUnits} onChange={onTimeUnitsChange} style={{ width: '100%' }}>
          {timeUnitsOptions.map(item => (
            <Select.Option key={item.value} value={item.value}>
              {currentLocales[`datepicker.time.units.${item.value}`]}
            </Select.Option>
          ))}
        </Select>
      </div>
      <div className={styles.apply}>
        <Apply currentLocales={currentLocales} onApply={handleApply} onCancel={onCancel} />
      </div>
      <div className={styles.refreshInterval}>
        <RefreshInterval 
          currentLocales={currentLocales} 
          isRefreshPaused={props.isRefreshPaused}
          refreshInterval={props.refreshInterval}
          onRefreshChange={props.onRefreshChange}
        />
      </div>
    </div>
  );
};

export default QuickSelect;
