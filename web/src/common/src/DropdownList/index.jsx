import { Button, Popover } from "antd";
import Icon, {BarsOutlined, CloseCircleOutlined, CloseOutlined, DownOutlined, UpOutlined} from "@ant-design/icons";
import { useEffect, useMemo, useRef, useState } from "react";
import { cloneDeep } from "lodash";

import AutoList from "./AutoList";
import CustomList from "./CustomList";
import locales from "./locales";

import styles from "./index.module.less";

export const DEFAULT_PAGE_SIZE = 10;
export const ORDER_DESC = "desc";
export const ORDER_ASC = "asc";

const DropdownList = (props) => {
  const {
    className = "",
    popoverClassName = "",
    popoverPlacement = "bottomLeft",
    children,
    width = 300,
    dropdownWidth,
    locale = "en-US",
    allowClear = false,
    mode = "",
    value,
    disabled = false,
    onChange,
    placeholder = "",
    loading = false,
    data = [],
    rowKey,
    renderItem,
    renderLabel,
    pagination,
    onSearchChange = () => {},
    sorter = [],
    onSorterChange = () => {},
    sorterOptions = [],
    filters = {},
    onFiltersChange = () => {},
    filterOptions = [],
    defaultGroupVisible = false,
    groups = [],
    onGroupsChange = () => {},
    groupOptions = [],
    onGroupVisibleChange = () => {},
    autoAdjustOverflow = {
      adjustX: 1,
      // Y 向自适应关闭：上下选边与高度封顶由 handleVisible 的预判定统一决定，
      // 避免 antd 量出溢出后自行翻转造成开箱跳动。
      adjustY: 0,
    },
    getPopupContainer = (triggerNode) => triggerNode.parentNode,
    extraData = [],
    showListIcon = true,
    onDropdownVisibleChange
  } = props;

  const [visible, setVisible] = useState(false);
  const [showGroup, setShowGroup] = useState(defaultGroupVisible);
  const [searchValue, setSearchValue] = useState();
  const [fixedDropdownWidth, setFixedDropdownWidth] = useState();
  // 空间不足时的稳定定位：打开前先量视口剩余空间，决定放上方还是下方，
  // 并把弹层高度封顶到该空间（内部滚动）。判定与首帧同批完成，避免
  // antd 默认的「先按 placement 渲染、量出溢出再翻转」造成的开箱跳动。
  const [popupFit, setPopupFit] = useState(null); // { side: 'bottom' | 'top', cap: number }
  const domRef = useRef(null);

  const isMultiple = useMemo(() => {
    return mode === "multiple" && (value === undefined || Array.isArray(value));
  }, [mode, value]);

  // 打开前先量视口剩余空间：决定放上方还是下方，并把弹层高度封顶到该空间
  // （内部滚动）。
  const computeFit = () => {
    if (!domRef.current) return;
    const rect = domRef.current.getBoundingClientRect();
    const vh = window.innerHeight || document.documentElement.clientHeight;
    // antd Popover 定位自带 12px 箭头间距（触发器与弹层之间），量空间必须扣除，
    // 否则 cap 后的弹层仍溢出几像素、antd 会自行翻转到另一侧（跳动根源）。
    const GAP = 14;
    const spaceBelow = vh - rect.bottom - GAP;
    const spaceAbove = rect.top - GAP;
    // 任一侧有基本可用空间（≥160px）就选它；都不够时选空间较大的一侧。
    const MIN_USABLE = 160;
    const side = spaceBelow >= MIN_USABLE || spaceBelow >= spaceAbove ? "bottom" : "top";
    const cap = Math.max(side === "bottom" ? spaceBelow : spaceAbove, 120);
    setPopupFit({ side, cap });
  };

  // 打开状态下窗口尺寸变化时重算，保证弹层不溢出、不翻转。
  useEffect(() => {
    if (!visible) return;
    const onResize = () => computeFit();
    window.addEventListener("resize", onResize);
    return () => window.removeEventListener("resize", onResize);
  }, [visible]);

  const handleVisible = (visible) => {
    if (visible) {
      computeFit();
    }
    onDropdownVisibleChange && onDropdownVisibleChange(visible)
    setVisible(visible);
  };

  const handleChange = (item) => {
    onChange(item);
    if (!isMultiple) {
      setVisible(false);
    }
  };

  const handleSearch = (value) => {
    setSearchValue(value);
    onSearchChange(value);
  };

  const handleMultipleRemove = (index) => {
    const newValue = cloneDeep(value) || [];
    newValue.splice(index, 1);
    onChange(newValue);
  };

  const handleClear = () => {
    onChange(isMultiple ? [] : undefined);
  };

  const renderContent = () => {
    const newProps = {
      ...props,
      value,
      loading,
      data,
      onSearchChange: handleSearch,
      searchValue,
      sorter,
      onSorterChange,
      sorterOptions,
      filters,
      onFiltersChange,
      filterOptions,
      groups,
      onGroupsChange,
      groupOptions,
      visible,
      onChange: handleChange,
      currentLocales: locales[locale] || {},
      showGroup,
      onShowGroupChange: (visible) => {
        setShowGroup(visible);
        onGroupVisibleChange(visible);
      },
      isMultiple,
    };
    if (!pagination || !pagination.currentPage || !pagination.total) {
      return <AutoList {...newProps} />;
    }
    return <CustomList {...newProps} />;
  };

  const renderSingleValue = (value) => {
    return renderLabel
      ? renderLabel(value)
      : renderItem
      ? renderItem(value)
      : value[rowKey];
  };

  const renderValue = () => {
    const defaultLabel = <span style={{ color: "#999" }}>{placeholder}</span>;
    if (!isMultiple) {
      return value && value[rowKey] ? renderSingleValue(value) : defaultLabel;
    }
    return value?.length > 0 ? (
      <>
        {value
          .filter((item) => !!(item && item[rowKey]))
          .map((item, index) => (
            <span
              key={index}
              className={`${styles.multipleItem} common-ui-dropdownlist-multiple-item`}
            >
              {renderSingleValue(item)}
              {
                !item.disabled ? (
                  <Icon
                    component={CloseOutlined}
                    style={{ marginLeft: 4, fontSize: 10 }}
                    onClick={(e) => {
                      e.stopPropagation();
                      handleMultipleRemove(index);
                    }}
                  />
                ) : null
              }
              
            </span>
          ))}
      </>
    ) : (
      defaultLabel
    );
  };

  useEffect(() => {
    if (!dropdownWidth && isNaN(Number(width)) && domRef.current) {
      setFixedDropdownWidth(domRef.current.offsetWidth);
    }
  }, [dropdownWidth, width, domRef]);

  let overlayWidth = dropdownWidth || width;

  if (isNaN(Number(overlayWidth))) {
    if (fixedDropdownWidth) {
      overlayWidth = `${fixedDropdownWidth}px`;
    }
  } else {
    overlayWidth = `${overlayWidth}px`;
  }

  if (showGroup) {
    overlayWidth = `calc(${overlayWidth} + ${120 * groups.length || 120}px)`;
  }

  const formatValue = renderValue();

  const defaultMaxHeight = 16 + 32 + 8 + 18 + 12 + 40 + (pagination?.pageSize || 10) * 32;
  const maxHeight = popupFit ? Math.min(defaultMaxHeight, popupFit.cap) : defaultMaxHeight;
  let placement = popoverPlacement;
  if (popupFit) {
    const wantsTop = popupFit.side === "top";
    const isTop = popoverPlacement.startsWith("top");
    if (wantsTop && !isTop) placement = popoverPlacement.replace("bottom", "top");
    if (!wantsTop && isTop) placement = popoverPlacement.replace("top", "bottom");
  }

  return (
    <div
      className={`${styles.dropdownList} ${className}`}
      style={{ width }}
      ref={domRef}
    >
      <Popover
        open={disabled ? false : visible}
        onOpenChange={handleVisible}
        placement={placement}
        content={renderContent()}
        trigger={"click"}
        overlayClassName={`${styles.popover} ${popoverClassName}`}
        overlayStyle={{
          width: overlayWidth,
          maxHeight,
        }}
        autoAdjustOverflow={autoAdjustOverflow}
        getPopupContainer={getPopupContainer}
        destroyOnHidden
      >
        {children ? (
          children
        ) : (
          <Button
            style={{ width: "100%" }}
            disabled={disabled}
            className={`${styles.button} css-var-r0 common-ui-dropdownlist-select ${
              allowClear ? styles.allowClear : ""
            }`}
          >
            <div className={styles.content}>
              {showListIcon && <BarsOutlined className={styles.icon}/>}
              <div className={styles.label}>
                <span>{formatValue}</span>
              </div>
              <Icon component={visible ? UpOutlined: DownOutlined} className={styles.down} />
              {allowClear && (
                <Icon
                  onClick={(e) => {
                    e.stopPropagation();
                    handleClear();
                  }}
                  theme="filled"
                  component={CloseCircleOutlined}
                  className={styles.close}
                />
              )}
            </div>
          </Button>
        )}
      </Popover>
    </div>
  );
};

export default DropdownList;
