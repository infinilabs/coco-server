import { Switch } from 'antd';
import { useTranslation } from 'react-i18next';

// the aggregation dimensions the operator can switch individually (搜索设置 ·
// 聚合与筛选) — semantic keys, mirrored by the backend validator
// (modules/system/settings.go) and the leg-local agg mapping in the search
// page (pages/search/index.tsx). Keep the three in sync.
export const AGG_DIMENSIONS = ['category', 'source', 'type', 'tags', 'owner', 'time', 'color'] as const;

// value is the DISABLED list (inverted storage, mirrors
// search_settings.aggregation_fields_disabled): unset/empty = every
// dimension enabled, the historic behavior
type Props = {
  value?: string[];
  onChange?: (value: string[]) => void;
};

const AggregationFieldsField = ({ value, onChange }: Props) => {
  const { t } = useTranslation();
  const disabled = value || [];

  const toggle = (key: string, on: boolean) => {
    const next = on ? disabled.filter(k => k !== key) : [...AGG_DIMENSIONS.filter(k => disabled.includes(k) || k === key)];
    onChange?.(next);
  };

  return (
    <div className="flex flex-col gap-8px py-4px">
      {AGG_DIMENSIONS.map(key => (
        <div key={key} className="flex items-center gap-12px">
          <Switch size="small" checked={!disabled.includes(key)} onChange={on => toggle(key, on)} />
          <span className="text-13px">{t(`page.settings.search_settings.labels.agg_field_${key}`)}</span>
        </div>
      ))}
    </div>
  );
};

export default AggregationFieldsField;
