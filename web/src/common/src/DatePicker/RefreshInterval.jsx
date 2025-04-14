import { Button, Icon, InputNumber, Select } from "antd";
import { useState } from "react";

import { timeUnits } from "./utils/time_units";
import { toMilliseconds, fromMilliseconds } from "./utils/utils";

import styles from "./RefreshInterval.less";

const timeUnitsOptions = Object.keys(timeUnits).map((key) => {
  return { value: key, text: `${timeUnits[key]}s` };
});

const RefreshInterval = (props) => {
  const {
    currentLocales,
    isRefreshPaused,
    refreshInterval = 10000,
    onRefreshChange,
  } = props;

  const [time, setTime] = useState(() => fromMilliseconds(refreshInterval));
  const { value, units } = time;

  const onValueChange = (value) => {
    setTime({ ...time, value });
  };

  const onUnitsChange = (units) => {
    setTime({ ...time, units });
  };

  const onPlayClick = () => {
    onRefreshChange({
      refreshInterval: toMilliseconds(units, value),
      isRefreshPaused: !isRefreshPaused,
    });
  };

  return (
    <div className={styles.refreshInterval}>
      <div className={styles.title}>
        {currentLocales["datepicker.refresh_every"]}
      </div>
      <InputNumber
        min={1}
        value={value}
        className={styles.value}
        onChange={onValueChange}
      />
      <Select value={units} className={styles.units} onChange={onUnitsChange}>
        {timeUnitsOptions.map((item) => (
          <Select.Option key={item.value} value={item.value}>
            {currentLocales[`datepicker.time.units.${item.value}`]}
          </Select.Option>
        ))}
      </Select>
      <Button className={styles.play} type="primary" onClick={onPlayClick}>
        {isRefreshPaused ? (
          <Icon type="caret-right" />
        ) : (
          <Icon type="pause" />
        )}
      </Button>
    </div>
  );
};

export default RefreshInterval;
