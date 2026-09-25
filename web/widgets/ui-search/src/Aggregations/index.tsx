import { useEffect, useState } from "react";
import cloneDeep from "lodash/cloneDeep";
import { useTranslation } from "react-i18next";

import FilterDefaultSvg from "../icons/filter-default.svg"
import { FilterCheckboxGroup, FilterColorPicker, FilterTags } from "./Filter";

interface AggregationItem {
  key: string;
  name?: string;
  icon?: string;
  count: number;
}

interface Aggregation {
  key: string;
  list: AggregationItem[];
}

interface AggregationConfig {
  [key: string]: {
    type?: 'checkbox' | 'color' | 'tag';
    label?: string;
  };
}

interface AggregationsProps {
  config?: AggregationConfig;
  aggregations?: Aggregation[];
  filter?: Record<string, any>;
  onSearch?: (filters: Record<string, any>) => void;
}

// well-known aggregation fields get a localized title; anything else falls
// back to the host-provided label, then the raw field key
const FIELD_LABEL_KEYS: Record<string, string> = {
  'source.id': 'labels.source',
  source: 'labels.source',
  type: 'labels.type',
  category: 'labels.category',
  categories: 'labels.category',
  tags: 'labels.tag',
  tag: 'labels.tag',
  lang: 'labels.language',
  language: 'labels.language',
  color: 'labels.color',
};

export function Aggregations(props: AggregationsProps) {
  const { config = {}, aggregations = [], filter = {}, onSearch } = props;
  const { t } = useTranslation();

  const [currentFilters, setCurrentFilters] = useState<Record<string, any>>(filter);

  useEffect(() => {
    setCurrentFilters(filter);
  }, [JSON.stringify(filter)]);

  const onChange = (value: any, aggregation: Aggregation) => {
    const newFilters = cloneDeep(currentFilters);
    newFilters[aggregation.key] = value
    setCurrentFilters(newFilters);
    onSearch?.(newFilters);
  };

  const onClear = (aggregation: Aggregation) => {
    const newFilters = cloneDeep(currentFilters);
    delete newFilters[aggregation.key];
    setCurrentFilters(newFilters);
    onSearch?.(newFilters);
  };

  if (!aggregations || aggregations.length === 0) return null

  // raw bucket keys like `web_page` read as internal jargon — map the known
  // ones to localized, human-friendly labels and pass unknown ones through
  const friendlyValue = (key: string) => {
    const normalized = String(key).trim().toLowerCase().replace(/[\s-]+/g, '_');
    return t(`labels.value_${normalized}`, { defaultValue: String(key) });
  };

  return (
    <>
      {aggregations.map((aggregation, index) => {
        let count = 0;
        aggregation.list.forEach((item) => (count += item.count));
        const type = config?.[aggregation.key]?.type || 'checkbox';
        const selectedValue = currentFilters[aggregation.key];
        const labelKey = FIELD_LABEL_KEYS[aggregation.key];
        const commonProps = {
          defaultExpand: index <= 2,
          title: labelKey
            ? t(labelKey, { defaultValue: config?.[aggregation.key]?.label || aggregation.key })
            : (config?.[aggregation.key]?.label || aggregation.key),
          value: selectedValue,
          clearable: Array.isArray(selectedValue) ? selectedValue.length > 0 : !!selectedValue,
          onChange: (value: any) => {
            onChange(value, aggregation)
          },
          onClear: () => onClear(aggregation)
        }
        let content
        if (type === 'color') {
          content = (
            <FilterColorPicker
              {...commonProps}
              onChange={(value) => onChange(value?.toHex(), aggregation)}
            />
          )
        } else if (type === 'tag') {
          content = (
            <FilterTags
              {...commonProps}
              value={commonProps.value || []}
              options={aggregation.list.map((item) => ({
                label: item.name || friendlyValue(item.key),
                value: item.key,
              }))}
            />
          )
        } else {
          content = (
            <FilterCheckboxGroup
              {...commonProps}
              value={commonProps.value || []}
              options={aggregation.list.map((item) => ({
                label: item.name || friendlyValue(item.key),
                value: item.key,
                icon: item.icon || FilterDefaultSvg,
                count: item.count,
              }))}
            />
          )
        }
        return (
          <div key={aggregation.key} className="mb-24px">
            {content}
          </div>
        );
      })}
    </>
  );
}

export default Aggregations;
