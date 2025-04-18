## Props

### DropdownList

| Property | Description | Type | Default | Version |
| -------- | ----------- | ---- | ------- | ------- |
| className | 组件根元素的类名称 | string | - | 1.0.0 |
| popoverClassName | 弹窗元素的类名称 | string | - | 1.0.0 |
| popoverPlacement | 弹窗位置，可选 top left right bottom topLeft topRight bottomLeft bottomRight leftTop leftBottom rightTop rightBottom | string | 'bottomLeft' | 1.0.0 |
| width | 选择框宽度 | number | 300 | 1.0.0 |
| dropdownWidth | 下拉框宽度，默认与选择框宽度一样 | number | 300 | 1.0.0 |
| locale | 语言 | string | 'en-US' | 1.0.0 |
| allowClear | 可以点击清除图标删除内容 | boolean | false | 1.0.0 |
| mode | 设置模式为多选  | 'multiple' | - | 1.0.0 |
| value | 已选项 | object | - | 1.0.0 |
| onChange | 选中变更的回调 | (value: object ) => void | - | 1.0.0 |
| disabled | 是否禁用 | boolean | false | 1.0.0 |
| placeholder | 选择框默认文字 | string | - | 1.0.0 |
| loading | 是否加载中 | boolean | false | 1.0.0 |
| failed | 是否加载失败 | boolean | false | 1.0.0 |
| data | 列表数据 | [] | [] | 1.0.0 |
| rowKey | 列表行key | string | - | 1.0.0 |
| renderItem | 列表行自定义渲染 | (item: object) => ReactNode | - | 1.0.0 |
| renderTag | 列表行自定义标签 | (item: object) => ReactNode | - | 1.0.0 |
| renderLabel | 选择框文本自定义渲染 | (item: object) => ReactNode | - | 1.0.0 |
| renderEmptyList | 空列表自定义渲染 | () => ReactNode | - | 1.0.0 |
| pagination | 分页器，设为 false 时不展示和进行分页 | { currentPage: number, pageSize: number, total: number, onChange: (page: number) => void  } | { currentPage: 1, pageSize: 10 } | 1.0.0 |
| searchKey | 搜索字段，自动分页起效 | string | - | 1.0.0 |
| onSearchChange | 搜索变更的回调 | (value: string) => void | - | 1.0.0 |
| sorter | 排序，第一个元素为排序字段，第二个元素为排序方式（desc/asc） | [] | [] | 1.0.0 |
| onSorterChange | 排序变更的回调 | (sorter: []) => void | - | 1.0.0 |
| sorterOptions | 排序选项  | [{ label: string, key: string }] | [] | 1.0.0 |
| filters | 字段过滤  | object | {} | 1.0.0 |
| onFiltersChange | 字段过滤变更的回调 | (filters: object) => void | - | 1.0.0 |
| filterOptions | 字段过滤选项 | [] | [] | 1.0.0 |
| groups | 分组过滤  | [] | [] | 1.0.0 |
| onGroupsChange | 分组过滤变更的回调 | (groups: []) => void | - | 1.0.0 |
| groupOptions | 分组过滤选项 | [] | [] | 1.0.0 |
| onGroupVisibleChange | 分组显隐变更的回调 | (visible: boolean) => void | - | 1.0.0 |
| autoAdjustOverflow | 浮窗被遮挡时自动调整位置 | boolean \| { adjustX?: 0 \| 1, adjustY?: 0 \| 1 } | { adjustX: 1 } | 1.0.0 |
| getPopupContainer | 浮层渲染父节点 | (triggerNode) => element | () => document.body | 1.0.0 |
| extraData | 额外项数据，置顶显示 | object[] | [] | 1.0.0 |
| searchPlaceholder | 下拉列表搜索框默认文字 | string | - | 1.0.0 |
| showListIcon | 是否显示左侧图标 | boolean | true | 1.0.0 |
| onRefresh | 列表数据刷新的回调 | () => void | - | 1.0.0 |
| actions | 下拉列表底部左侧操作 | ReactNode[] | [] | 1.0.0 |