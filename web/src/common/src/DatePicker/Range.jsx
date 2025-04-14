import { Button, Icon, Popover, Spin } from "antd";
import {
  useEffect,
  useMemo,
  useRef,
  useState,
  useImperativeHandle,
} from "react";
import moment from "moment-timezone";

import QuickSelect from "./QuickSelect";
import StartAndEndTimes from "./StartAndEndTimes";
import TimeSetting from "./TimeSetting";
import TimeZone from "./TimeZone";
import {
  getDateString,
  getDateStringWithGMT,
  getGMTString,
} from "./utils/utils";
import { AsyncInterval } from "./utils/async_interval";
import { prettyDuration, showPrettyDuration } from "./utils/pretty_duration";

import styles from "./Range.less";

const TIME_ZONE_KEY = "time_zone";
const TIME_SETTING_KEY = "time_setting";

const SETTING = [
  {
    key: "quick_select",
    label: "Quick select",
    icon: <Icon className={styles.icon} type="thunderbolt" />,
    component: QuickSelect,
  },
  {
    key: "start_and_end_times",
    label: "Start and end times",
    icon: <Icon className={styles.icon} type="calendar" />,
    component: StartAndEndTimes,
  },
  {
    key: TIME_SETTING_KEY,
    label: "Time setting",
    icon: <Icon className={styles.icon} type="setting" />,
    component: TimeSetting,
  },
  {
    key: TIME_ZONE_KEY,
    label: "Time zone",
    render: (item, { timeZone }, locales) =>
      `${locales[`datepicker.${item.key}`]} | ${getGMTString(timeZone)}`,
    icon: <Icon className={styles.icon} type="global" />,
    component: TimeZone,
  },
];

const Range = (props) => {
  const {
    currentLocales,
    popoverClassName,
    popoverPlacement,
    dateFormat,
    start,
    end,
    onRangeChange,
    commonlyUsedRanges,
    isRefreshPaused,
    refreshInterval,
    onRefreshChange,
    onRefresh,
    showTimeSetting,
    autoFitLoading,
    onAutoFit,
    timeZone,
    isMinimum,
  } = props;

  useImperativeHandle(props.onRef, () => {
    return {
      handleRefreshChange: handleRefreshChange,
    };
  });

  const refreshIntervalRef = useRef(null);
  const [visible, setVisible] = useState(false);
  const [showMenuIcon, setShowMenuIcon] = useState(false);
  const [selectedItem, setSelectedItem] = useState();
  const [height, setHeight] = useState();
  const contentRef = useRef(null);
  const rangeRef = useRef({ start, end });

  const handleSettingClick = (item) => {
    setSelectedItem(item);
    setShowMenuIcon(true);
  };

  const handleVisible = (visible) => {
    if (!visible) {
      handleClose();
    } else {
      setVisible(true);
      setTimeout(() => {
        if (contentRef.current) {
          setHeight(contentRef.current.offsetHeight);
        }
      }, 100);
    }
  };

  const handleClose = () => {
    setVisible(false);
    setTimeout(() => {
      setSelectedItem();
      setShowMenuIcon(false);
    }, 100);
  };

  const handleRefreshChange = ({ isRefreshPaused, refreshInterval }) => {
    stopInterval();
    if (onRefreshChange) {
      onRefreshChange({ refreshInterval, isRefreshPaused });
    }
  };

  const stopInterval = () => {
    if (refreshIntervalRef.current) {
      refreshIntervalRef.current.stop();
    }
  };

  const startInterval = (refreshInterval, onRefresh) => {
    stopInterval();
    if (onRefresh && rangeRef.current) {
      const handler = () => {
        onRefresh({ ...rangeRef.current, refreshInterval });
      };
      refreshIntervalRef.current = new AsyncInterval(handler, refreshInterval);
    }
  };

  useEffect(() => {
    return () => {
      stopInterval();
    };
  }, []);

  useEffect(() => {
    if (!isRefreshPaused) {
      startInterval(refreshInterval, onRefresh);
    } else {
      stopInterval();
    }
  }, [refreshInterval, isRefreshPaused, onRefresh]);

  useEffect(() => {
    rangeRef.current = { start, end };
  }, [start, end]);

  const content = (
    <div ref={contentRef} className={styles.rangeSetting}>
      <div className={styles.menu}>
        <div className={styles.quickSelect}>
          {onAutoFit && (
            <Spin size="small" spinning={!!autoFitLoading}>
              <div
                className={`${styles.item} ${isMinimum ? styles.disabled : ""}`}
                onClick={() => {
                  onAutoFit();
                  handleClose();
                }}
              >
                {currentLocales[`datepicker.quick_select.auto_fit`]}
              </div>
            </Spin>
          )}
          {commonlyUsedRanges.map((item, index) => (
            <div
              key={index}
              className={`${styles.item} ${isMinimum ? styles.disabled : ""}`}
              onClick={() => {
                onRangeChange({ start: item.start, end: item.end });
                handleClose();
              }}
            >
              {currentLocales[`datepicker.quick_select.${item.key}`]}
            </div>
          ))}
        </div>
        <div className={styles.setting}>
          {SETTING.filter(
            (item) => item.key !== TIME_SETTING_KEY || !!showTimeSetting
          ).map((item) => (
            <div
              key={item.key}
              className={`${styles.item} ${
                selectedItem?.key === item.key ? styles.selected : ""
              } ${
                isMinimum &&
                ![TIME_ZONE_KEY, TIME_SETTING_KEY].includes(item.key)
                  ? styles.disabled
                  : ""
              }`}
              onClick={() => handleSettingClick(item)}
            >
              <div>
                {showMenuIcon && item.icon && (
                  <span className={styles.icon}>{item.icon}</span>
                )}
                {item.render
                  ? item.render(item, props, currentLocales)
                  : currentLocales[`datepicker.${item.key}`]}
              </div>
              <Icon className={styles.right} type="right" />
            </div>
          ))}
        </div>
      </div>
      {selectedItem?.component && (
        <div className={styles.content} style={{ height: height || "100%" }}>
          <selectedItem.component
            {...props}
            onRefreshChange={handleRefreshChange}
            onCancel={handleClose}
          />
        </div>
      )}
    </div>
  );

  const fullRangeText = useMemo(() => {
    if (isMinimum || !start || !end) return "";
    if (showPrettyDuration(start, end, commonlyUsedRanges)) {
      return prettyDuration(
        start,
        end,
        commonlyUsedRanges,
        dateFormat,
        currentLocales
      );
    } else {
      return `${getDateString(
        start,
        timeZone,
        dateFormat
      )} ~ ${getDateStringWithGMT(end, timeZone, dateFormat)}`;
    }
  }, [isMinimum, start, end, commonlyUsedRanges, dateFormat, timeZone]);

  const rangeText = useMemo(() => {
    const now = moment().tz(timeZone);
    const dateString = now.format("YYYY-MM-DD");
    const yearString = now.format("YYYY");
    return typeof fullRangeText === "string"
      ? fullRangeText
          .replaceAll(`${dateString}`, "")
          .replaceAll(`${yearString}-`, "")
      : "";
  }, [fullRangeText, timeZone]);

  return (
    <Popover
      visible={visible}
      onVisibleChange={handleVisible}
      placement={popoverPlacement}
      content={content}
      trigger={"click"}
      overlayClassName={`${styles.popover} ${popoverClassName}`}
    >
      <Button
        title={fullRangeText}
        className={`${styles.rangeBtn} ${
          isMinimum ? styles.minimum : ""
        } common-ui-datepicker-range`}
      >
        <div className={styles.rangeContent}>
          <Icon className={styles.clock} type="clock-circle" />
          <span className={styles.label}>{rangeText}</span>
          <Icon className={styles.down} type="down" />
        </div>
      </Button>
    </Popover>
  );
};

export default Range;
