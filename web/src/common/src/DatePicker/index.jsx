import React, { useMemo, useState, useRef } from "react";
import { Button, Icon } from "antd";
import moment from "moment";
import dateMath from "@elastic/datemath";

import Range from "./Range";
import locales from "./locales";
import isRelativeToNow from "./utils/pretty_duration";
import { toMilliseconds, fromMilliseconds } from "./utils/utils";

import styles from "./index.less";

const DEFAULT_COMMONLY_USED_RANGES = [
  {
    start: "now/d",
    end: "now/d",
    label: "Today",
    key: "today",
  },
  {
    start: "now/w",
    end: "now/w",
    label: "This week",
    key: "this_week",
  },
  {
    start: "now-15m",
    end: "now",
    label: "Last 15 minutes",
    key: "last_15_minutes",
  },
  {
    start: "now-30m",
    end: "now",
    label: "Last 30 minutes",
    key: "last_30_minutes",
  },
  {
    start: "now-1h",
    end: "now",
    label: "Last 1 hour",
    key: "last_1_hour",
  },
  {
    start: "now-24h",
    end: "now",
    label: "Last 24 hours",
    key: "last_24_hours",
  },
  {
    start: "now-7d",
    end: "now",
    label: "Last 7 days",
    key: "last_7_days",
  },
  {
    start: "now-30d",
    end: "now",
    label: "Last 30 days",
    key: "last_30_days",
  },
  {
    start: "now-90d",
    end: "now",
    label: "Last 90 days",
    key: "last_90_days",
  },
];

const DEFAULT_RECENTLY_USED_RANGES_KEY = "recently-used-ranges";

const DatePicker = (props) => {
  const {
    locale = "en-US",
    className = "",
    popoverPlacement = "bottomLeft",
    dateFormat = "YYYY-MM-DD HH:mm:ss",
    start = "now-15m",
    end = "now",
    onRangeChange,
    isRefreshPaused = true,
    refreshInterval = 10000,
    showTimeSetting = false,
    shouldTimeField = true,
    showTimeField = false,
    timeField,
    timeFields = [],
    showTimeInterval = false,
    timeInterval,
    timeIntervalDisabled = false,
    showTimeout = false,
    timeout,
    autoFitLoading = false,
    timeZone = "Asia/Shanghai",
    commonlyUsedRanges = DEFAULT_COMMONLY_USED_RANGES,
    recentlyUsedRangesKey,
    onRefreshChange,
    onRefresh,
  } = props;

  const [prevQuickSelect, setPrevQuickSelect] = useState();
  const [recentlyUsedRanges, setRecentlyUsedRanges] = useState(() => {
    if (!recentlyUsedRangesKey) return [];
    const history = localStorage.getItem(
      `${recentlyUsedRangesKey}-${DEFAULT_RECENTLY_USED_RANGES_KEY}`
    );
    try {
      const ranges = JSON.parse(history);
      return Array.isArray(ranges) ? ranges : [];
    } catch (err) {
      return [];
    }
  });

  const handleRangeChange = ({ start, end, quickSelect, isAbsolute }) => {
    onRangeChange({ start, end, timeZone });
    if (isAbsolute && recentlyUsedRangesKey) {
      const newRecentlyUsedRanges = [...recentlyUsedRanges];
      newRecentlyUsedRanges.unshift({ start, end, timeZone });
      const last20 = newRecentlyUsedRanges.slice(0, 20);
      setRecentlyUsedRanges(last20);
      localStorage.setItem(
        `${recentlyUsedRangesKey}-${DEFAULT_RECENTLY_USED_RANGES_KEY}`,
        JSON.stringify(last20)
      );
    }
    if (quickSelect) {
      setPrevQuickSelect(quickSelect);
    }
  };

  const getBounds = () => {
    const startMoment = dateMath.parse(start);
    const endMoment = dateMath.parse(end, { roundUp: true });
    return {
      min:
        startMoment && startMoment.isValid()
          ? startMoment
          : moment().subtract(15, "minute"),
      max: endMoment && endMoment.isValid() ? endMoment : moment(),
    };
  };

  const stepBackward = () => {
    const { min, max } = getBounds();
    const diff = max.diff(min);
    handleRangeChange({
      start: moment(min)
        .subtract(diff + 1, "ms")
        .tz(timeZone)
        .toISOString(),
      end: moment(min)
        .subtract(1, "ms")
        .tz(timeZone)
        .toISOString(),
    });
  };

  const stepForward = () => {
    const { min, max } = getBounds();
    const diff = max.diff(min);
    const endMoment = moment(max)
      .add(diff + 1, "ms")
      .tz(timeZone);
    if (endMoment.diff(moment().tz(timeZone), "s") > 0) {
      return;
    }
    handleRangeChange({
      start: moment(max)
        .add(1, "ms")
        .tz(timeZone)
        .toISOString(),
      end: endMoment.toISOString(),
    });
  };

  const isMinimum = useMemo(() => {
    return showTimeSetting && showTimeField && shouldTimeField
      ? !timeField
      : false;
  }, [showTimeSetting, shouldTimeField, shouldTimeField, timeField]);

  const isNextDisabled = useMemo(() => {
    return isRelativeToNow(start, end);
  }, [start, end]);

  const RangeRef = React.createRef();
  const [time, setTime] = useState(() => fromMilliseconds(refreshInterval));
  const { value, units } = time;

  const onPlayClick = () => {
    if (RangeRef.current) {
      const handleRefreshChange = RangeRef.current.handleRefreshChange;
      if (handleRefreshChange) {
        handleRefreshChange({
          refreshInterval: toMilliseconds(units, value),
          isRefreshPaused: !isRefreshPaused,
        });
      }
    }
  };

  const [reloadLoading, setReloadLoading] = useState(false) 
  const onRefreshClick = () => {
    setReloadLoading(true)
    onRefresh && onRefresh({ start, end, refresh: true });
    setTimeout(()=>{
      setReloadLoading(false)
    }, 1000)
  };

  return (
    <div
      className={`${styles.datePicker} ${
        isMinimum ? styles.minimum : ""
      } ${className}`}
    >
      <Button.Group className={styles.RangeBox} style={{ width: onRefresh ? 'calc(100% - 64px)' : 'calc(100% - 32px)'}}>
        {!isMinimum && (
          <Button
            className={`${styles.iconBtn} common-ui-datepicker-backward`}
            icon="left"
            onClick={stepBackward}
          />
        )}
        <Range
          {...props}
          onRef={RangeRef}
          popoverPlacement={popoverPlacement}
          dateFormat={dateFormat}
          start={start}
          end={end}
          onRangeChange={handleRangeChange}
          isRefreshPaused={isRefreshPaused}
          refreshInterval={refreshInterval}
          showTimeSetting={showTimeSetting}
          timeFields={timeFields}
          showTimeInterval={showTimeInterval}
          timeInterval={timeInterval}
          showTimeout={showTimeout}
          timeout={timeout}
          autoFitLoading={autoFitLoading}
          timeZone={timeZone}
          commonlyUsedRanges={commonlyUsedRanges}
          recentlyUsedRanges={recentlyUsedRanges}
          isMinimum={isMinimum}
          prevQuickSelect={prevQuickSelect}
          currentLocales={locales[locale] || {}}
          onRefreshChange={onRefreshChange}
        />
        {!isMinimum && (
          <Button
            disabled={isNextDisabled}
            className={`${styles.iconBtn} common-ui-datepicker-Forward`}
            icon="right"
            onClick={stepForward}
          />
        )}
      </Button.Group>
      <Button.Group className={styles.refreshBtn}>
        <Button className={styles.play} onClick={onPlayClick}>
          {isRefreshPaused ? (
            <Icon type="caret-right" />
          ) : (
            <Icon type="pause" />
          )}
        </Button>
        {onRefresh ? (
          <Button className={styles.play} onClick={onRefreshClick}>
            {reloadLoading ? (
              <Icon type="loading" />
            ) : (
              <Icon type="reload" />
            )}
          </Button>
        ) : null}
      </Button.Group>
    </div>
  );
};

export default DatePicker;
