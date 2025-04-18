import { Input } from 'antd';
import Apply from './Apply';
import styles from './TimeZone.less';
import { TIMEZONES } from './utils/time_zones';
import { useMemo, useState } from 'react';
import { getDateString } from './utils/utils';

const TimeZone = props => {
  const { currentLocales, onTimeZoneChange, onCancel, dateFormat } = props;

  const [timeZone, setTimeZone] = useState();
  const [searchValue, setSearchValue] = useState();

  const handleApply = () => {
    const index = TIMEZONES.findIndex(item => item.value === timeZone);
    if (index !== -1) {
      onTimeZoneChange(timeZone);
    }
  };

  const filterList = useMemo(() => {
    if (!searchValue) return TIMEZONES;
    return TIMEZONES.filter(
      item =>
        item.label?.toLocaleLowerCase().indexOf(searchValue.toLocaleLowerCase()) !== -1
    );
  }, [searchValue]);

  const current = useMemo(() => {
    return TIMEZONES.find(item => item.value === props.timeZone);
  }, [props.timeZone]);

  return (
    <div className={styles.timeZone}>
      <div className={styles.title}>{currentLocales[`datepicker.time_zone`]}</div>
      <Input
        placeholder="Search by country, city, time zone, or GMT offset"
        style={{ width: '100%' }}
        onChange={e => setSearchValue(e.target.value)}
      />
      <div className={styles.list}>
        {filterList.map(item => (
          <div
            key={item.value}
            className={`${styles.item} ${timeZone === item.value ? styles.selected : ''}`}
            onClick={() => {
              setTimeZone(item.value);
            }}
          >
            <div className={styles.name} title={item.label}>{`${item.label}`}</div>
            <div className={styles.date}>
              {getDateString(undefined, item.value, dateFormat)}
            </div>
          </div>
        ))}
      </div>
      {current && (
        <div className={styles.current}>
          <div className={styles.title}>{currentLocales[`datepicker.time_zone.current`]}</div>
          <div className={styles.value} title={current.label}>{`${current.label}`}</div>
        </div>
      )}
      <div className={styles.apply}>
        <Apply currentLocales={currentLocales} onApply={handleApply} onCancel={onCancel} />
      </div>
    </div>
  );
};

export default TimeZone;
