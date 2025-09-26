import { Checkbox } from "antd";
import { useState } from "react";
import { cloneDeep } from "lodash";
import { BrushCleaning } from "lucide-react";

export function Aggregations(props) {
  const { config = {}, aggregations = [], filter = {}, onSearch } = props;

  const [currentFilters, setCurrentFilters] = useState(filter);

  const onAllChange = (checked, aggregation) => {
    const newFilters = cloneDeep(currentFilters);
    if (checked) {
      newFilters[aggregation.key] = aggregation.list.map((item) => item.key);
    } else {
      delete newFilters[aggregation.key];
    }
    setCurrentFilters(newFilters);
    onSearch(newFilters);
  };

  const onItemChange = (checked, aggregation, item) => {
    const newFilters = cloneDeep(currentFilters);
    if (checked) {
      if (!newFilters[aggregation.key]) newFilters[aggregation.key] = [];
      newFilters[aggregation.key].push(item.key);
    } else {
      const index = newFilters[aggregation.key]?.findIndex(
        (key) => key === item.key,
      );
      if (index !== -1) {
        newFilters[aggregation.key].splice(index, 1);
      }
      if (newFilters[aggregation.key]?.length === 0) {
        delete newFilters[aggregation.key];
      }
    }
    setCurrentFilters(newFilters);
    onSearch(newFilters);
  };

  const onClear = (aggregation) => {
    const newFilters = cloneDeep(currentFilters);
    delete newFilters[aggregation.key];
    setCurrentFilters(newFilters);
    onSearch(newFilters);
  };

  return (
    <div>
      {aggregations.map((aggregation) => {
        let count = 0;
        aggregation.list.forEach((item) => (count += item.count));
        const filterList = currentFilters[aggregation.key] || [];
        const isCheckAll =
          aggregation.list?.length > 0
            ? aggregation.list.every((item) => filterList.includes(item.key))
            : false;
        return (
          <div key={aggregation.key} className="mb-24px">
            <div className="mb-12px flex justify-between items-center">
              <div className="text-[#999] uppercase">
                {config?.[aggregation.key]?.displayName || aggregation.key}
              </div>
              <div className="cursor-pointer">
                <BrushCleaning
                  className="w-14px h-14px text-[#bbb]"
                  onClick={() => onClear(aggregation)}
                />
              </div>
            </div>
            <div>
              <div className="color-[rgba(102,102,102,1)] flex mb-16px">
                <Checkbox
                  className="mr-8px"
                  checked={isCheckAll}
                  onChange={(e) => onAllChange(e.target.checked, aggregation)}
                ></Checkbox>
                <div className="flex items-center justify-between w-[calc(100%-24px)] text-[#666]">
                  <div className="flex-1 items-center">
                    {/* <img
                      className="w-[14px] h-[14px] mr-2 vertical-align-[-2px]"
                      src={""}
                    /> */}
                    All
                  </div>
                  <div>{count || 0}</div>
                </div>
              </div>
              {aggregation.list.map((item, index) => (
                <div
                  key={index}
                  className="color-[rgba(102,102,102,1)] flex mb-16px"
                >
                  <Checkbox
                    className="mr-8px"
                    checked={filterList?.some((a) => a === item.key)}
                    onChange={(e) =>
                      onItemChange(e.target.checked, aggregation, item)
                    }
                  ></Checkbox>
                  <div className="flex items-center justify-between w-[calc(100%-24px)] gap-6px text-[#666]">
                    <div className="flex-1 items-center truncate">
                      {/* <img
                        className="w-[14px] h-[14px] mr-2 vertical-align-[-2px]"
                        src={item.icon}
                      /> */}
                      {item.name || item.key}
                    </div>
                    <div className="flex-shrink-0">{item.count}</div>
                  </div>
                </div>
              ))}
            </div>
          </div>
        );
      })}
    </div>
  );
}

export default Aggregations;
