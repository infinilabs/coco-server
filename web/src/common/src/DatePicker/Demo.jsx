import { useState } from 'react';
import DatePicker from '.';

const Demo = () => {
  const [range, setRange] = useState({ start: 'now-15m', end: 'now' });

  const [refresh, setRefresh] = useState({ isRefreshPaused: true, refreshInterval: 10000 });

  const [autoFitLoading, setAutoFitLoading] = useState(false);

  const [timeSetting, setTimeSetting] = useState({
    showTimeSetting: true,
    showTimeField: true,
    timeField: 'timestamp',
    timeFields: [
      'payload.elasticsearch.index_routing_table.shards.0.unassigned_info.at',
      'timestamp',
    ],
    showTimeInterval: true,
    timeInterval: '15s',
  });

  const [timeZone, setTimeZone] = useState('Asia/Shanghai');

  const onAutoFit = () => {
    setAutoFitLoading(true);
    setTimeout(() => {
      setRange({ start: '2023-09-09 09:09:09', end: '2023-10-10 10:10:10' });
      setAutoFitLoading(false);
    }, 3000);
  };

  return (
    <div style={{ display: 'flex', maxWidth: 480 }}>
      <div style={{ flex: 1 }}>
        <DatePicker
          {...range}
          onRangeChange={({ start, end }) => {
            setRange({ start, end })
          }}
          {...refresh}
          onRefreshChange={setRefresh}
          {...timeSetting}
          onTimeSettingChange={newTimeSetting => {
            setTimeSetting({ ...timeSetting, ...newTimeSetting });
          }}
          autoFitLoading={autoFitLoading}
          onAutoFit={onAutoFit}
          timeZone={timeZone}
          onTimeZoneChange={setTimeZone}
          recentlyUsedRangesKey={"demo-recently-used-ranges"}
        />
      </div>
    </div>
  );
};

export default Demo;
