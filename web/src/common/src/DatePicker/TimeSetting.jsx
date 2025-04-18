import { InputNumber, Select, Switch } from 'antd';
import Apply from './Apply';
import styles from './TimeSetting.less';
import { useEffect, useMemo, useRef, useState } from 'react';

const timeIntervals = [
  // { label: 'Millisecond', value: 'ms' },
  { label: 'Second', value: 's' },
  { label: 'Minute', value: 'm' },
  { label: 'Hour', value: 'h' },
  { label: 'Day', value: 'd' },
  { label: 'Week', value: 'w' },
  { label: 'Month', value: 'M' },
  { label: 'Year', value: 'y' },
];

const timeOuts = [
  { label: 'Second', value: 's' },
  { label: 'Minute', value: 'm' },
];

const TimeSetting = props => {
  const { currentLocales, timeFields = [], showTimeField, showTimeInterval, timeIntervalDisabled = false, showTimeout, onTimeSettingChange, onCancel } = props;

  const [isAuto, setIsAuto] = useState(!props.timeInterval)
  const [timeField, setTimeField] = useState(props.timeField);
  const [timeInterval, setTimeInterval] = useState(props.timeInterval);
  const [timeout, setTimeout] =  useState(props.timeout);
  const timeIntervalCache = useRef('');

  const handleApply = () => {
    onTimeSettingChange({ timeField, timeInterval, timeout });
    onCancel()
  };

  const timeIntervalObject = useMemo(() => {
    if (!timeInterval) return
    const value = parseInt(timeInterval);
    return {
      value,
      unit: timeInterval.replace(`${value}`, ''),
    }
  }, [timeInterval])

  const timeoutObject = useMemo(() => {
    if (!timeout) {
      return {
        value: 10,
        unit: 's',
      }
    }
    const value = parseInt(timeout);
    return {
      value,
      unit: timeout.replace(`${value}`, ''),
    }
  }, [timeout])
  
  return (
    <div className={styles.timeSetting}>
      <div className={styles.title}>{currentLocales[`datepicker.time_setting`]}</div>
      {
        showTimeField && (
          <div className={styles.formItem}>
            <div className={styles.label}>{currentLocales[`datepicker.time_setting.time_field`]}</div>
            <Select value={timeField} onChange={setTimeField} style={{ width: '100%' }}>
              {timeFields.map(item => (
                <Select.Option key={item} value={item}>
                  {item}
                </Select.Option>
              ))}
            </Select>
          </div>
        )
      }
      {showTimeInterval && (
        <div className={styles.formItem}>
          <div className={styles.label}>
            {currentLocales[`datepicker.time_setting.time_interval`]}
            <div className={styles.auto}>
              <Switch disabled={timeIntervalDisabled} size="small" checked={isAuto} onChange={(checked) => {
                setIsAuto(checked)
                if (checked) {
                  timeIntervalCache.current = timeInterval;
                  setTimeInterval()
                } else {
                  setTimeInterval(timeIntervalCache.current || props.timeInterval || '10s')
                }
              }}/> {currentLocales[`datepicker.time_setting.time_interval.auto`]}
            </div>
          </div>
          {
            timeIntervalDisabled && isAuto && (
              <div className={styles.help}>
                {currentLocales[`datepicker.time_setting.time_interval.help`]}
              </div>
            )
          }
          <div className={styles.form}>
            {
              !isAuto && timeIntervalObject && (
                <>
                  <InputNumber
                    min={1}
                    value={timeIntervalObject.value}
                    style={{ width: '100%' }}
                    step={1}
                    precision={0}
                    onChange={(value) => {
                      if (Number.isInteger(value)) {
                        setTimeInterval(`${value}${timeIntervalObject.unit}`)
                      }
                    }}
                  />
                  <Select value={timeIntervalObject.unit} onChange={(value) => setTimeInterval(`${timeIntervalObject.value}${value}`)} style={{ width: '100%' }}>
                    {timeIntervals.map((item) => (
                        <Select.Option key={item.value} value={item.value}>
                          {currentLocales[`datepicker.time_setting.time_interval.${item.value}`]}
                        </Select.Option>
                      ))}
                  </Select>
                </>
              )
            }  
          </div>
        </div>
      )}
      {showTimeout && (
        <div className={styles.formItem}>
          <div className={styles.label}>
            {currentLocales[`datepicker.time_setting.timeout`]}
          </div>
          <div className={styles.form}>
            {
              timeoutObject && (
                <>
                  <InputNumber
                    min={10}
                    value={timeoutObject.value}
                    style={{ width: '100%' }}
                    step={1}
                    precision={0}
                    onChange={(value) => {
                      if (Number.isInteger(value)) {
                        setTimeout(`${value}${timeoutObject.unit}`)
                      }
                    }}
                  />
                  <Select value={timeoutObject.unit} onChange={(value) => setTimeout(`${timeoutObject.value}${value}`)} style={{ width: '100%' }}>
                    {timeOuts.map((item) => (
                        <Select.Option key={item.value} value={item.value}>
                          {currentLocales[`datepicker.time_setting.time_interval.${item.value}`]}
                        </Select.Option>
                      ))}
                  </Select>
                </>
              )
            }  
          </div>
        </div>
      )}
      <div className={styles.apply}>
        <Apply currentLocales={currentLocales} onApply={handleApply} onCancel={onCancel} />
      </div>
    </div>
  );
};

export default TimeSetting;