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
    groups = [],
    onGroupsChange = () => {},
    groupOptions = [],
    onGroupVisibleChange = () => {},
    autoAdjustOverflow = {
      adjustX: 1,
    },
    getPopupContainer = (triggerNode) => triggerNode.parentNode,
    extraData = [],
    showListIcon = true,
  } = props;

  const [visible, setVisible] = useState(false);
  const [showGroup, setShowGroup] = useState(false);
  const [searchValue, setSearchValue] = useState();
  const [fixedDropdownWidth, setFixedDropdownWidth] = useState();
  const domRef = useRef(null);

  const isMultiple = useMemo(() => {
    return mode === "multiple" && (value === undefined || Array.isArray(value));
  }, [mode, value]);

  const handleVisible = (visible) => {
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
              style={{ backgroundColor: disabled ? "#c4c4c4" : "#fff" }}
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

  return (
    <div
      className={`${styles.dropdownList} ${className}`}
      style={{ width }}
      ref={domRef}
    >
      <Popover
        open={disabled ? false : visible}
        onOpenChange={handleVisible}
        placement={popoverPlacement}
        content={renderContent()}
        trigger={"click"}
        overlayClassName={`${styles.popover} ${popoverClassName}`}
        overlayStyle={{
          width: overlayWidth,
          maxHeight:
            16 + 32 + 8 + 18 + 12 + 40 + (pagination?.pageSize || 10) * 32,
        }}
        autoAdjustOverflow={autoAdjustOverflow}
        getPopupContainer={getPopupContainer}
      >
        {children ? (
          children
        ) : (
          <Button
            style={{ width: "100%" }}
            disabled={disabled}
            className={`${styles.button} common-ui-dropdownlist-select ${
              allowClear ? styles.allowClear : ""
            }`}
          >
            <div className={styles.content}>
              <div className={styles.label}>
                {showListIcon && <BarsOutlined className={styles.icon}/>}
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
