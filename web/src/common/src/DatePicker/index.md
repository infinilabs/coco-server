## Props

### DatePicker

| Property | Description | Type | Default | Version |
| -------- | ----------- | ---- | ------- | ------- |
| className | 组件根元素的类名称 | string | - | 1.0.0 |
| popoverClassName | 弹窗元素的类名称 | string | - | 1.0.0 |
| popoverPlacement | 弹窗位置，可选 top left right bottom topLeft topRight bottomLeft bottomRight leftTop leftBottom rightTop rightBottom | string | 'bottom' | 1.0.0 |
| locale | 语言 | string | 'en-US' | 1.0.0 |
| dateFormat | 日期格式化 | string | 'YYYY-MM-DD HH:mm:ss' | 1.0.0 |
| Start | 开始日期 | string | 'now-15m' | 1.0.0 |
| End | 结束日期 | string | 'now' | 1.0.0 |
| onRangeChange | 时间范围变更的回调 | ({start: string, end: string, isAbsolute?: boolean}) => void | - | 1.0.0 |
| isRefreshPaused | 是否暂停自动刷新 | boolean | true | 1.0.0 |
| refreshInterval | 自动刷新间隔 | number | 10000 | 1.0.0 |
| onRefreshChange | 自动刷新变更的回调 | ({isRefreshPaused: boolean, refreshInterval: number}) => void | - | 1.0.0 |
| onRefresh | 自动刷新触发的操作 | ({start: string, end: string, refreshInterval: number}) => void | - | 1.0.0 |
| showTimeSetting | 是否显示时间配置 | boolean | false | 1.0.0 |
| shouldTimeField | 当显示时间配置时，是否必须设置时间字段 | boolean | true | 1.0.0 |
| showTimeField | 是否显示时间字段配置 | boolean | false | 1.0.0 |
| timeField | 时间字段  | string | - | 1.0.0 |
| timeFields | 时间字段列表  | string[] | [] | 1.0.0 |
| showTimeInterval | 是否显示时间间隔  | boolean | false | 1.0.0 |
| timeInterval | 时间间隔  | string | - | 1.0.0 |
| timeIntervalDisabled | 禁用时间间隔  | boolean | false | 1.0.0 |
| showTimeout | 是否显示超时时间  | boolean | false | 1.0.0 |
| timeout | 超时时间  | string | 10s | 1.0.0 |
| onTimeSettingChange | 时间配置变更的回调 | ({timeField: string, timeInterval: string, timeout: string}) => void | - | 1.0.0 |
| autoFitLoading | 自动适配时间载入状态  | boolean | false | 1.0.0 |
| onAutoFit | 自动适配时间的回调 | () => void | - | 1.0.0 |
| timeZone | 时区 | string | 'Asia/Shanghai' | 1.0.0 |
| onTimeZoneChange | 时区变更的回调 | (timeZone: string) => void | - | 1.0.0 |
| commonlyUsedRanges | 快速选择列表 | {start: string, end: string, label: string}[] | [] | 1.0.0 |
| recentlyUsedRangesKey | 时间范围历史字段 | string | - | 1.0.0 |