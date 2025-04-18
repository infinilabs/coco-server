import { DatePicker } from 'antd';
import Apply from './Apply';
import RefreshInterval from './RefreshInterval';
import styles from './StartAndEndTimes.less';
import moment from 'moment-timezone';
import dateMath from '@elastic/datemath';
import { useState } from 'react';
import { getDateString, getDateStringWithGMT } from './utils/utils';

function isRangeInvalid(start, end) {
  const startMoment = dateMath.parse(start);
  const endMoment = dateMath.parse(end, { roundUp: true });
  if (!startMoment || !endMoment || !startMoment.isValid() || !endMoment.isValid()) {
    return true;
  }
  if (startMoment.isAfter(endMoment)) {
    return true;
  }

  return false;
}

const StartAndEndTimes = props => {
  const {
    currentLocales,
    start,
    end,
    onRangeChange,
    onCancel,
    dateFormat,
    timeZone,
    recentlyUsedRanges = [],
  } = props;

  const [startAsMoment, setStartAsMoment] = useState(() => {
    const value = dateMath.parse(start);
    return value && value.isValid() ? value.tz(timeZone) : moment().tz(timeZone);
  });

  const [isInvalid, setIsInvalid] = useState(false)

  const [endAsMoment, setEndAsMoment] = useState(() => {
    const value = dateMath.parse(end);
    return value && value.isValid() ? value.tz(timeZone) : moment().tz(timeZone);
  });

  const handleApply = () => {
    const startValue = startAsMoment.toISOString(true);
    const endValue = endAsMoment.toISOString(true);
    const isInvalid = isRangeInvalid(startValue, endValue);
    if (!isInvalid) {
      onRangeChange({ start: startValue, end: endValue, isAbsolute: true });
      onCancel()
    } else {
      setIsInvalid(true)
    }
  };

  return (
    <div className={styles.startAndEndTimes}>
      <div className={styles.title}>{currentLocales[`datepicker.start_and_end_times`]}{` ${startAsMoment.format('(G[M]TZ)')}`}</div>
      <div className={styles.formItem}>
        <div className={styles.label}>{currentLocales[`datepicker.start_and_end_times.start_time`]}</div>
        <DatePicker
          format={dateFormat}
          value={startAsMoment}
          className={styles.datePicker}
          showTime
          allowClear={false}
          onChange={date => setStartAsMoment(date)}
        />
      </div>
      <div className={styles.formItem}>
        <div className={styles.label}>{currentLocales[`datepicker.start_and_end_times.end_time`]}</div>
        <DatePicker
          format={dateFormat}
          value={endAsMoment}
          className={styles.datePicker}
          showTime
          allowClear={false}
          onChange={date => setEndAsMoment(date)}
        />
        {isInvalid && (
          <div className={styles.error}>
            {currentLocales[`datepicker.start_and_end_times.end_time`]}
          </div>
        )}
      </div>
      <div className={styles.apply}>
        <Apply currentLocales={currentLocales} onApply={handleApply} onCancel={onCancel} />
      </div>
      {
        recentlyUsedRanges.length > 0 && (
          <div className={styles.recent}>
            <div className={styles.title}>{currentLocales[`datepicker.start_and_end_times.recent`]}</div>
            {recentlyUsedRanges.map((item, index) => (
              <div
                key={index}
                className={styles.item}
                onClick={() => {
                  onRangeChange({ 
                    start: moment(item.start).tz(item.timeZone).tz(timeZone).toISOString(true), 
                    end: moment(item.end).tz(item.timeZone).tz(timeZone).toISOString(true)
                  })
                  onCancel()
                }}
              >
                {getDateString(item.start, item.timeZone, dateFormat)} ~{' '}
                {getDateStringWithGMT(item.end, item.timeZone, dateFormat)}
              </div>
            ))}
          </div>
        )
      }
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

export default StartAndEndTimes;
