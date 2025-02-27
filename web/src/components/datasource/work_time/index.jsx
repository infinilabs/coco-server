import {Select, TimePicker } from "antd";
import Icon from '@ant-design/icons';
import { cloneDeep } from "lodash";
import moment from "moment";
const format = "HH:mm";

export const DEFAULT_WORKTIME_VALUE = { day_of_week: 7, hour_minute: "00:00" };

export default (props) => {
  const { value = {
    type: "per_week",
    times: [DEFAULT_WORKTIME_VALUE],
  }, onChange } = props;
  const { type = "per_week", times = [] } = value;

  const handleChange = (n, v, i) => {
    const newValue = cloneDeep(value);
    if (!newValue.times) newValue.times = [];
    if (!newValue.times[i]) newValue.times[i] = {};
    newValue.times[i][n] = v;
    onChange(newValue);
  };

  const onRemove = (index) => {
    const newValue = cloneDeep(value);
    newValue.times.splice(index, 1);
    onChange(newValue);
  };

  const onAdd = () => {
    const newValue = cloneDeep(value);
    if (!newValue.times) newValue.times = [];
    newValue.times.push(DEFAULT_WORKTIME_VALUE);
    onChange(newValue);
  };

  return (
    <div>
      <div
        style={{
          display: "flex",
          alignItems: "center",
          gap: 16,
        }}
      >
        <Select value={type} style={{ width: 200, marginBottom: 16 }}>
          <Select.Option key="per_week">
            Weekly
          </Select.Option>
        </Select>
      </div>
      {times.map((item, index) => (
        <div
          key={index}
          style={{
            display: "flex",
            alignItems: "center",
            gap: 16,
            marginTop: index === 0 ? 0 : 16,
          }}
        >
          <Select
            style={{ width: 200 }}
            value={item.day_of_week}
            options={[{ value: 7, label: <span>Sunday</span> },{ value: 1, label: <span>Monday</span> }]}
            onChange={(value) => handleChange("day_of_week", value, index)}
          >
            {/* <Select.Option key={7} value={7}>
              Sunday
            </Select.Option>
            <Select.Option key={1} value={1}>
              Monday
            </Select.Option>
            <Select.Option key={2} value={2}>
              Tuesday
            </Select.Option>
            <Select.Option key={3} value={3}>
             Wednesday
            </Select.Option>
            <Select.Option key={4} value={4}>
              Thursday
            </Select.Option>
            <Select.Option key={5} value={5}>
             Friday
            </Select.Option>
            <Select.Option key={6} value={6}>
              Saturday
            </Select.Option> */}
          </Select>
          <TimePicker
            style={{ width: 200 }}
            allowClear={false}
            format={format}
            value={
              item.hour_minute ? moment(item.hour_minute, format) : undefined
            }
            onChange={(time, timeString) => {
              handleChange("hour_minute", timeString, index);
            }}
          />
          <div style={{ fontSize: 16, color: "rgb(16, 16, 16)" }}>
            {index !== 0 && (
              <Icon
                type="close-circle"
                style={{ marginRight: 8 }}
                onClick={() => onRemove(index)}
              />
            )}
            <Icon type="plus-circle" onClick={() => onAdd()} />
          </div>
        </div>
      ))}
    </div>
  );
};
