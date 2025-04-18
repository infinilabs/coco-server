import { Button, Checkbox, Divider, Input, Popover, Select } from "antd";
import Icon, { CheckOutlined, FilterOutlined, LayoutOutlined, LeftOutlined, ReloadOutlined, RightOutlined, SortAscendingOutlined } from "@ant-design/icons";
import {
  Fragment,
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
} from "react";
import { cloneDeep } from "lodash";

import Loading from "./Loading";
import Error from "./Error";
import Empty from "./Empty";

import styles from "./List.module.less";

const List = (props) => {
  const {
    visible,
    value,
    onChange,
    loading = false,
    failed = false,
    rowKey,
    renderItem,
    renderTag,
    renderEmptyList,
    onSearchChange,
    searchValue,
    sorterOptions,
    pagination,
    data,
    sorter,
    onSorterChange,
    filters,
    onFiltersChange,
    filterOptions,
    groups,
    onGroupsChange,
    groupOptions,
    currentLocales,
    showGroup,
    onShowGroupChange,
    isMultiple,
    extraData = [],
    searchPlaceholder,
    onRefresh,
    actions = [],
  } = props;

  const [currentSorter, setCurrentSorter] = useState([]);
  const [sorterVisible, setSorterVisible] = useState(false);
  const inputRef = useRef(null);

  const handleClose = useCallback(() => {
    setTimeout(() => {
      onShowGroupChange(false);
      onSearchChange();
    }, 500);
  }, [onShowGroupChange, onSearchChange]);

  const handleChange = (item) => {
    if (item.disabled) return;
    if (isMultiple) {
      const newValue = cloneDeep(value) || [];
      const index = newValue.findIndex((v) => v[rowKey] === item[rowKey]);
      if (index === -1) {
        newValue.push(item);
      } else {
        newValue.splice(index, 1);
      }
      onChange(newValue);
    } else {
      onChange(item);
    }
  };

  const handleModeChange = () => {
    onShowGroupChange(!showGroup);
  };

  const handleGroupChange = (level, group) => {
    const newGroups = [...groups];
    if (newGroups[level]?.value === group.value) {
      newGroups.splice(level, 1);
    } else {
      newGroups[level] = {
        key: group.key,
        value: group.value,
      };
    }
    newGroups.splice(level + 1, newGroups.length);
    onGroupsChange(newGroups);
  };

  const handleSearch = (value) => {
    onSearchChange(value.trim());
  };

  const handleSortChange = (value, index) => {
    const newSorter = [...currentSorter];
    newSorter[index] = value;
    setCurrentSorter(newSorter);
  };

  const handleFiltersChange = (field, value, checked) => {
    const newFilters = cloneDeep(filters);
    const item = newFilters[field] || [];
    const index = item.indexOf(value);
    if (index === -1) {
      if (checked) item.push(value);
    } else {
      if (!checked) item.splice(index, 1);
    }
    newFilters[field] = item;
    onFiltersChange(newFilters);
  };

  const handleCheckAllChange = (checked) => {
    const newValue = cloneDeep(value || []);
    data.forEach((item) => {
      if (item.disabled) return;
      const index = newValue.findIndex((v) => v[rowKey] === item[rowKey]);
      if (index === -1 && checked) {
        newValue.push(item);
      } else if (index !== -1 && !checked) {
        newValue.splice(index, 1);
      }
    });
    onChange(newValue);
  };

  useEffect(() => {
    if (inputRef.current && visible) {
      inputRef.current.focus();
    }
  }, [visible, inputRef]);

  useEffect(() => {
    if (!visible) {
      handleClose();
    }
  }, [visible, handleClose]);

  useEffect(() => {
    setCurrentSorter(sorter);
  }, [sorter]);

  const { currentPage, pageSize, total, pages } = pagination;

  const groupChildOptions = useMemo(() => {
    if (groups[0] === undefined) return [];
    const child = groupOptions.find((item) => item.value === groups[0].value);
    return child?.list || [];
  }, [groupOptions, groups]);

  const isCheckAll = useMemo(() => {
    if (!isMultiple) return false;
    if (isMultiple) {
      if (value?.length === 0) {
        return false;
      }
      if (
        data.every(
          (item) => value.findIndex((v) => v[rowKey] === item[rowKey]) !== -1
        )
      ) {
        return true;
      }
      if (
        data.some(
          (item) => value.findIndex((v) => v[rowKey] === item[rowKey]) !== -1
        )
      ) {
        return "indeterminate";
      }
      return false;
    }
  }, [isMultiple, JSON.stringify(data), JSON.stringify(value)]);

  const renderGroupOptions = (options, level) => {
    if (!showGroup || options.length === 0) return null;
    return (
      <div className={styles.group}>
        {options.map((item) => (
          <div
            key={item.value}
            className={`${styles.item} ${
              groups[level]?.value === item.value ? styles.selected : ""
            }`}
            onClick={() => handleGroupChange(level, item)}
            title={item.label}
          >
            <span className={styles.label}>{item.label}</span>
            <Icon component={RightOutlined} className={styles.icon} />
          </div>
        ))}
      </div>
    );
  };

  const newActions = actions.concat(
    onRefresh
      ? [
          <a
            disabled={loading}
            onClick={() => {
              if (!loading) {
                onRefresh();
              }
            }}
          >
            {currentLocales["dropdownlist.refresh"]}
          </a>,
        ]
      : []
  );

  return (
    <div className={styles.container}>
      {renderGroupOptions(groupOptions, 0)}
      {renderGroupOptions(groupChildOptions, 1)}
      <div className={styles.content}>
        <Input
          ref={inputRef}
          value={searchValue}
          onChange={(e) => {
            if (!loading) {
              const newValue = e.target.value;
              handleSearch(newValue);
            }
          }}
          allowClear
          className={styles.search}
          placeholder={
            searchPlaceholder ||
            currentLocales["dropdownlist.search.placeholder"]
          }
        />
        <div className={styles.tools}>
          <div className={styles.result}>
            {currentLocales["dropdownlist.result.found"]}{" "}
            {pagination === false ? data.length : total}{" "}
            {currentLocales["dropdownlist.result.records"]}
          </div>
          <div className={styles.actions}>
            {sorterOptions.length > 0 && (
              <Popover
                open={sorterVisible}
                onOpenChange={setSorterVisible}
                overlayClassName={styles.filterPopover}
                placement="bottom"
                trigger={"click"}
                content={
                  <div className={styles.sorter}>
                    <div className={styles.title}>
                      {currentLocales["dropdownlist.sort.by"]}
                    </div>
                    <div className={styles.form}>
                      <Select
                        value={currentSorter[0]}
                        style={{ width: "65%" }}
                        onChange={(value) => handleSortChange(value, 0)}
                      >
                        {sorterOptions.map((item) => (
                          <Select.Option key={item.key} value={item.key}>
                            {item.label}
                          </Select.Option>
                        ))}
                      </Select>
                      <Select
                        value={currentSorter[1]}
                        style={{ width: "35%" }}
                        onChange={(value) => handleSortChange(value, 1)}
                      >
                        <Select.Option value="desc">
                          {currentLocales["dropdownlist.sort.by.desc"]}
                        </Select.Option>
                        <Select.Option value="asc">
                          {currentLocales["dropdownlist.sort.by.asc"]}
                        </Select.Option>
                      </Select>
                    </div>
                    <div className={styles.actions}>
                      <Button
                        type="link"
                        onClick={() => {
                          onSorterChange([]);
                          setSorterVisible(false);
                        }}
                      >
                        {currentLocales["dropdownlist.sort.by.clear"]}
                      </Button>
                      <Button
                        style={{ width: 80 }}
                        type="primary"
                        onClick={() => {
                          onSorterChange(currentSorter);
                          setSorterVisible(false);
                        }}
                      >
                        {currentLocales["dropdownlist.apply"]}
                      </Button>
                    </div>
                  </div>
                }
              >
                <Icon style={{ cursor: "pointer" }} component={SortAscendingOutlined} />
              </Popover>
            )}
            {filterOptions.length > 0 && (
              <Popover
                overlayClassName={styles.filterPopover}
                placement="bottom"
                trigger={"click"}
                content={
                  <div className={styles.filters}>
                    <div className={styles.title}>
                      {currentLocales["dropdownlist.filters"]}
                      <Icon
                        className={styles.clear}
                        component={ReloadOutlined}
                        onClick={() => {
                          onFiltersChange({});
                        }}
                      />
                    </div>
                    {filterOptions.map((item) => (
                      <div key={item.key} className={styles.content}>
                        <div className={styles.label}>{item.label}</div>
                        <div className={styles.options}>
                          {(item.list || []).map((c) => (
                            <div key={c.value} className={styles.option}>
                              <Checkbox
                                onChange={(e) =>
                                  handleFiltersChange(
                                    item.key,
                                    c.value,
                                    e.target.checked
                                  )
                                }
                                checked={
                                  (filters[item.key] || []).indexOf(c.value) !==
                                  -1
                                }
                              >
                                {c.label || c.value}
                              </Checkbox>
                              {c.count !== undefined && <div>{c.count}</div>}
                            </div>
                          ))}
                        </div>
                      </div>
                    ))}
                  </div>
                }
              >
                <Icon style={{ cursor: "pointer" }} component={FilterOutlined} />
              </Popover>
            )}
            {groupOptions.length > 0 && (
              <Icon
                style={{ cursor: "pointer" }}
                component={LayoutOutlined}
                onClick={handleModeChange}
              />
            )}
            {isMultiple && (
              <Checkbox
                indeterminate={isCheckAll === "indeterminate"}
                onChange={(e) => handleCheckAllChange(e.target.checked)}
                checked={isCheckAll === true}
              />
            )}
          </div>
        </div>
        <div className={styles.listWrapper}>
          <div
            className={styles.list}
            style={{ maxHeight: `${32 * (pageSize + extraData.length)}px` }}
          >
            {loading ? (
              <Loading currentLocales={currentLocales} />
            ) : failed ? (
              <Error currentLocales={currentLocales} />
            ) : data.length === 0 && extraData.length === 0 ? (
              renderEmptyList ? (
                renderEmptyList()
              ) : (
                <Empty />
              )
            ) : (
              extraData.concat(data).map((item) => {
                const isSelected = isMultiple
                  ? (value || []).findIndex(
                      (v) => v[rowKey] === item[rowKey]
                    ) !== -1
                  : value?.[rowKey] === item[rowKey];
                return (
                  <div
                    key={item[rowKey]}
                    className={`${styles.item} ${
                      isSelected ? styles.selected : ""
                    } ${
                      item.disabled ? styles.disabled : ""
                    }`}
                    onClick={() => handleChange(item)}
                  >
                    <div className={styles.label}>
                      {renderItem ? renderItem(item) : item[rowKey]}
                    </div>
                    {renderTag && !showGroup ? (
                      <div className={styles.tag}>
                        {renderItem ? renderTag(item) : item[rowKey]}
                      </div>
                    ) : null}
                    {isSelected && <Icon component={CheckOutlined}/>}
                  </div>
                );
              })
            )}
          </div>
        </div>
        {pagination === false ? null : (
          <div className={styles.footerWrapper}>
            <div className={styles.footer}>
              <div className={styles.actions}>
                {newActions.map((item, index) => (
                  <Fragment key={index}>
                    {item}
                    {index !== newActions.length - 1 && (
                      <Divider type="vertical" />
                    )}
                  </Fragment>
                ))}
              </div>
              <div className={styles.pager}>
                <Icon
                  onClick={() => pagination.onChange(currentPage - 1)}
                  component={LeftOutlined}
                  className={`${styles.icon} ${
                    currentPage <= 1 ? styles.disabled : ""
                  }`}
                />
                <span className={styles.pageNum}>
                  {currentPage}/{pages}
                </span>
                <Icon
                  onClick={() => pagination.onChange(currentPage + 1)}
                  component={RightOutlined}
                  className={`${styles.icon} ${
                    currentPage >= pages ? styles.disabled : ""
                  }`}
                />
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
};

export default List;
