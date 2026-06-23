import { Button, DatePicker, Dropdown, Popover, Slider, Space, Tooltip } from "antd";
import { BrushCleaning, Calendar, ChevronDown, ChevronRight, Crosshair, Heading, RotateCw } from "lucide-react";
import { useTranslation } from 'react-i18next';
import { type FC, useRef, useState } from "react";
import ExchangeIcon from "../icons/Exchange";
import { ACTION_TYPE_SEARCH_HYBRID, ACTION_TYPE_SEARCH_KEYWORD, ACTION_TYPE_SEARCH_SEMANTIC, DEFAULT_SEARCH_FUZZINESS, DEFAULT_SEARCH_SORT, MAX_SEARCH_FUZZINESS, MIN_SEARCH_FUZZINESS, SORT_BEST_MATCH, SORT_CREATED_ASC, SORT_CREATED_DESC, SORT_UPDATED_DESC } from "../SearchBox/ActionBar/SearchActions";

interface ToolbarProps {
  searchType?: string;
  onSearchTypeChange?: (type: string) => void;
  fuzziness?: number;
  onFuzzinessChange?: (fuzziness: number) => void;
  sort?: string;
  onSortChange?: (sort: string) => void;
  dateRange?: string;
  onDateRangeChange?: (dateRange: string) => void;
  start?: string;
  end?: string;
  onCustomDateRangeChange?: (range: { start?: string; end?: string }) => void;
}

export const Toolbar: FC<ToolbarProps> = ({
  searchType = ACTION_TYPE_SEARCH_KEYWORD,
  onSearchTypeChange,
  fuzziness = DEFAULT_SEARCH_FUZZINESS,
  onFuzzinessChange,
  sort = DEFAULT_SEARCH_SORT,
  onSortChange,
  dateRange = "all-time",
  onDateRangeChange,
  start,
  end,
  onCustomDateRangeChange,
}) => {
  const { t } = useTranslation();
  const [sortKey] = sort.split(',');
  const [dateDropdownOpen, setDateDropdownOpen] = useState(false);
  const [filterPopoverOpen, setFilterPopoverOpen] = useState(false);
  const [filterDateRange, setFilterDateRange] = useState<any>(null);
  const wrapperRef = useRef<HTMLDivElement>(null);

  const searchTypeItems = [
    {
      key: ACTION_TYPE_SEARCH_HYBRID,
      label: t('labels.hybrid'),
    },
    {
      key: ACTION_TYPE_SEARCH_KEYWORD,
      label: t('labels.keyword'),
    },
    {
      key: ACTION_TYPE_SEARCH_SEMANTIC,
      label: t('labels.semantic'),
    },
  ];
  const sortItems = [
    {
      key: SORT_BEST_MATCH,
      label: t('labels.bestMatch'),
    },
    {
      key: SORT_CREATED_DESC,
      label: t('labels.newest'),
    },
    {
      key: SORT_CREATED_ASC,
      label: t('labels.oldest'),
    },
    {
      key: SORT_UPDATED_DESC,
      label: t('labels.recentlyUpdated'),
    },
  ];

  const handleFuzzinessChange = (value: number | number[]) => {
    if (typeof value === 'number') {
      onFuzzinessChange?.(value);
    }
  };

  const handleClearFilters = () => {
    setFilterDateRange(null);
    onCustomDateRangeChange?.({});
  };

  const getPopupContainer = () => {
    return wrapperRef.current?.closest?.('.ui-search') as HTMLElement || document.body;
  };

  const dateMenuItems = [
    {
      key: "all-time",
      label: t("labels.allTime"),
    },
    {
      key: "7d",
      label: t("labels.past7Days"),
    },
    {
      key: "90d",
      label: t("labels.past90Days"),
    },
    {
      key: "1y",
      label: t("labels.past1year"),
    },
    {
      key: "more",
      label: (
        <div className="flex items-center justify-between gap-24px">
          <span>{t("labels.more")}</span>
          <ChevronRight size={16} />
        </div>
      ),
    },
  ];

  const hasCustomDateRange = Boolean(start && end);
  const dateRangeLabel = hasCustomDateRange ? t('labels.custom') : dateMenuItems.find((item) => item.key === dateRange)?.label || t('labels.allTime');
  const dateRangeTooltip = hasCustomDateRange ? `${start} - ${end}` : undefined;

  const filterPopoverContent = (
    <div className="w-320px text-14px text-[#666] dark:text-[#999]">
      <div className="flex items-center justify-between">
        <span className="font-bold text-[#333] dark:text-[#E5E7EB]">{t("labels.filters")}</span>
        <Button
          size="small"
          color="primary"
          variant="link"
          className="!h-24px !px-8px"
          onClick={handleClearFilters}
        >
          <BrushCleaning size={14} />
        </Button>
      </div>

      <div className="pt-16px pb-8px text-[#999] dark:text-[#666]">
        {t("labels.updateTime")}
      </div>
      <DatePicker.RangePicker
        className="w-full"
        value={filterDateRange}
        placeholder={[t("labels.selectDateRange"), t("labels.selectDateRange")]}
        getPopupContainer={getPopupContainer}
        format="YYYY/MM/DD"
        onChange={(value, dateStrings) => {
          setFilterDateRange(value);
          onCustomDateRangeChange?.({
            start: dateStrings[0] || undefined,
            end: dateStrings[1] || undefined,
          });
        }}
      />
    </div>
  );

  return (
    <div ref={wrapperRef} className="flex flex-wrap items-start w-full gap-16px">
      <Dropdown getPopupContainer={getPopupContainer} trigger={['click']} menu={{
        items: searchTypeItems,
        onClick: ({ key }) => onSearchTypeChange?.(key),
      }}>
        <Button
          color="default"
          variant="link"
          className="!leading-none !h-18px !p-0 text-12px text-[#666] dark:text-[#999] flex items-center justify-center"
          onClick={() => onFuzzinessChange?.(DEFAULT_SEARCH_FUZZINESS)}
        >
          <Space size={4}>
            <Heading className="w-16px h-16px text-16px" />
            {searchTypeItems.find((item) => item.key === searchType)?.label || t('labels.keyword')}
            <ChevronDown className="w-16px h-16px text-16px" />
          </Space>
        </Button>
      </Dropdown>
      <Space size={4} className="text-12px text-[#666] dark:text-[#999]">
        <Crosshair className="w-16px h-16px text-16px" />
        {t('labels.fuzziness')}
        <Slider
          className="w-75px my-0"
          classNames={{
            track: '!bg-[var(--ant-color-primary)]',
            handle: '[&::after]:!bg-[var(--ant-color-primary)] [&::after]:!shadow-[0_0_0_1px_#fff] dark:[&::after]:!shadow-[0_0_0_1px_#000]',
          }}
          min={MIN_SEARCH_FUZZINESS}
          max={MAX_SEARCH_FUZZINESS}
          step={1}
          value={fuzziness}
          onChange={handleFuzzinessChange}
        />
        <Button
          color="default"
          variant="link"
          className="!leading-none !w-16px !h-16px !min-w-16px !p-0 text-[#999] dark:text-[#666] flex items-center justify-center"
          icon={<RotateCw size={16} />}
          onClick={() => onFuzzinessChange?.(DEFAULT_SEARCH_FUZZINESS)}
        />
      </Space>
      <Dropdown getPopupContainer={getPopupContainer} trigger={['click']} menu={{
        items: sortItems,
        onClick: ({ key }) => onSortChange?.(key),
      }}>
        <Button
          color="default"
          variant="link"
          className="!leading-none !h-18px !p-0 text-12px text-[#666] dark:text-[#999] flex items-center justify-center"
        >
          <Space size={4}>
            <ExchangeIcon className="w-16px h-16px text-16px" />
            {sortItems.find((item) => item.key === sortKey)?.label || t('labels.bestMatch')}
            <ChevronDown className="w-16px h-16px text-16px" />
          </Space>
        </Button>
      </Dropdown>
      <Dropdown
        getPopupContainer={getPopupContainer}
        trigger={['click']}
        open={dateDropdownOpen}
        onOpenChange={(open) => setDateDropdownOpen(open)}
        menu={{
          items: dateMenuItems,
          onClick: ({ key }) => {
            if (key === "more") {
              setDateDropdownOpen(false);
              setFilterPopoverOpen(true);
              return;
            }
            onDateRangeChange?.(key);
            setFilterDateRange(null);
            onCustomDateRangeChange?.({});
            setFilterPopoverOpen(false);
            setDateDropdownOpen(false);
          },
        }}
      >
        <Popover
          getPopupContainer={getPopupContainer}
          open={filterPopoverOpen}
          placement="bottomLeft"
          trigger="click"
          content={filterPopoverContent}
          onOpenChange={(open) => {
            if (!open) {
              setFilterPopoverOpen(false);
            }
          }}
          arrow={false}
          classNames={{
            container: "!p-16px"
          }}
        >
          <Tooltip getPopupContainer={getPopupContainer} title={dateRangeTooltip} open={dateRangeTooltip ? undefined : false}>
            <Button
              color="default"
              variant="link"
              className="!leading-none !h-18px !p-0 text-12px text-[#666] dark:text-[#999] flex items-center justify-center"
            >
              <Space size={4}>
                <Calendar className="w-16px h-16px text-16px" />
                {dateRangeLabel}
                <ChevronDown className="w-16px h-16px text-16px" />
              </Space>
            </Button>
          </Tooltip>
        </Popover>
      </Dropdown>
    </div>
  );
}

export default Toolbar;
