import { useEffect, useMemo, useRef, useState } from 'react';
import cloneDeep from 'lodash/cloneDeep';
import { useTranslation } from 'react-i18next';

import FilterDefaultSvg from '../icons/filter-default.svg';
import { FilterCheckboxGroup, FilterColorPicker, FilterTags } from './Filter';

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
  readonly config?: AggregationConfig;
  readonly aggregations?: Aggregation[];
  readonly filter?: Record<string, any>;
  readonly onSearch?: (filters: Record<string, any>) => void;
  /** active time preset ('7d'/'90d'/'1y'), shared with the toolbar's date_range */
  readonly dateRangeValue?: string;
  /** selecting a time bucket applies that preset instead of an aggfilter */
  readonly onDateRangeChange?: (value: string) => void;
  /** resolves user ids to display entities (title/username/name) */
  readonly getUserEntities?: (ids: string[], callback?: (data: any) => void) => any;
}

// document owner facet key — the value is a user id; display names come from
// the host's getUserEntities hook, not from an aggregation payload
export const OWNER_FILTER_FIELD = '_system.owner_id';

// update-time facet key: buckets ('7d'/'90d'/'1y') mirror the toolbar's
// date_range presets, so selection routes through onDateRangeChange
export const TIME_FILTER_FIELD = 'updated_range';

// display order for the time facet — the engine does not guarantee bucket
// order for date_range, and newest-first reads best in the rail
const TIME_BUCKET_ORDER = ['7d', '90d', '1y'];

// well-known aggregation fields get a localized title; anything else falls
// back to the host-provided label, then the raw field key
export const FIELD_LABEL_KEYS: Record<string, string> = {
  'source.id': 'labels.source',
  source: 'labels.source',
  'source.connector_id': 'labels.connector',
  type: 'labels.type',
  category: 'labels.category',
  categories: 'labels.category',
  tags: 'labels.tag',
  tag: 'labels.tag',
  lang: 'labels.language',
  language: 'labels.language',
  color: 'labels.color',
  [OWNER_FILTER_FIELD]: 'labels.createdBy',
  [TIME_FILTER_FIELD]: 'labels.updateTime'
};

const entityDisplayName = (entity: any) => entity?.title ?? entity?.username ?? entity?.name ?? undefined;

export function Aggregations(props: AggregationsProps) {
  const {
    config = {},
    aggregations = [],
    filter = {},
    onSearch,
    dateRangeValue,
    onDateRangeChange,
    getUserEntities
  } = props;
  const { t } = useTranslation();

  const [currentFilters, setCurrentFilters] = useState<Record<string, any>>(filter);

  useEffect(() => {
    setCurrentFilters(filter);
  }, [JSON.stringify(filter)]);

  // user-id → display entity cache for the owner facet (ids are meaningless in
  // the rail); mirrors the result list's owner resolution
  const ownerNameCacheRef = useRef<Record<string, any>>({});
  const pendingOwnerIdsRef = useRef<Set<string>>(new Set());
  const [ownerNameVersion, setOwnerNameVersion] = useState(0);

  const ownerAgg = aggregations.find(aggregation => aggregation.key === OWNER_FILTER_FIELD);
  const ownerBucketIds = (ownerAgg?.list || []).map(item => String(item.key));

  // re-derived whenever a batch of entity lookups lands (ownerNameVersion)
  const ownerNameMap = useMemo(() => {
    const map: Record<string, string> = {};
    ownerBucketIds.forEach(id => {
      const name = entityDisplayName(ownerNameCacheRef.current[id]);
      if (name) map[id] = name;
    });
    return map;
  }, [ownerNameVersion, JSON.stringify(ownerBucketIds)]);

  useEffect(() => {
    if (typeof getUserEntities !== 'function' || ownerBucketIds.length === 0) return;

    const missingOwnerIds = ownerBucketIds.filter(
      id => ownerNameCacheRef.current[id] === undefined && !pendingOwnerIdsRef.current.has(id)
    );
    if (missingOwnerIds.length === 0) return;

    missingOwnerIds.forEach(id => pendingOwnerIdsRef.current.add(id));

    const markMissingOwnersAsLoaded = () => {
      missingOwnerIds.forEach(id => {
        ownerNameCacheRef.current[id] = null;
        pendingOwnerIdsRef.current.delete(id);
      });
      setOwnerNameVersion(prev => prev + 1);
    };

    try {
      const request = getUserEntities(missingOwnerIds, (res: any) => {
        const entities = Array.isArray(res) ? res : Array.isArray(res?.data) ? res.data : [];
        const entityMap = new Map<string, any>();
        entities.forEach((entity: any) => {
          if (entity?.id) {
            entityMap.set(entity.id, entity);
          }
        });

        missingOwnerIds.forEach(id => {
          ownerNameCacheRef.current[id] = entityMap.get(id) ?? null;
          pendingOwnerIdsRef.current.delete(id);
        });

        setOwnerNameVersion(prev => prev + 1);
      });

      Promise.resolve(request).catch(markMissingOwnersAsLoaded);
    } catch {
      markMissingOwnersAsLoaded();
    }
  }, [JSON.stringify(ownerBucketIds), getUserEntities]);

  const ownerDisplayName = (id: string) => ownerNameMap[id];

  const onChange = (value: any, aggregation: Aggregation) => {
    const newFilters = cloneDeep(currentFilters);
    newFilters[aggregation.key] = value;
    setCurrentFilters(newFilters);
    onSearch?.(newFilters);
  };

  const onClear = (aggregation: Aggregation) => {
    const newFilters = cloneDeep(currentFilters);
    delete newFilters[aggregation.key];
    setCurrentFilters(newFilters);
    onSearch?.(newFilters);
  };

  // time facet: buckets are nested presets (7d ⊂ 90d ⊂ 1y), so it behaves as
  // single-select — checking an option switches the preset, unchecking the
  // active one resets to all-time
  const onTimeChange = (values: Array<string | number>) => {
    const next = values.length > 0 ? String(values[values.length - 1]) : 'all-time';
    onDateRangeChange?.(next);
  };

  if (!aggregations || aggregations.length === 0) return null;

  // raw bucket keys like `web_page` read as internal jargon — map the known
  // ones to localized, human-friendly labels and pass unknown ones through
  const friendlyValue = (key: string) => {
    const normalized = String(key)
      .trim()
      .toLowerCase()
      .replace(/[\s-]+/g, '_');
    return t(`labels.value_${normalized}`, { defaultValue: String(key) });
  };

  return (
    <>
      {aggregations.map((aggregation, index) => {
        let count = 0;
        aggregation.list.forEach(item => (count += item.count));
        const type = config?.[aggregation.key]?.type || 'checkbox';
        const isOwnerField = aggregation.key === OWNER_FILTER_FIELD;
        const isTimeField = aggregation.key === TIME_FILTER_FIELD;
        const selectedValue = isTimeField ? undefined : currentFilters[aggregation.key];
        const activeTimeValue = dateRangeValue && dateRangeValue !== 'all-time' ? dateRangeValue : undefined;
        const labelKey = FIELD_LABEL_KEYS[aggregation.key];
        // date_range buckets report every configured range, zero-count ones
        // included — drop them, hide the group when none match, and keep the
        // rail order stable regardless of engine bucket ordering
        const bucketList = isTimeField
          ? aggregation.list
              .filter(item => item.count > 0)
              .sort((a, b) => TIME_BUCKET_ORDER.indexOf(a.key) - TIME_BUCKET_ORDER.indexOf(b.key))
          : aggregation.list;
        if (bucketList.length === 0) return null;
        const commonProps = {
          defaultExpand: index <= 2,
          title: labelKey
            ? t(labelKey, { defaultValue: config?.[aggregation.key]?.label || aggregation.key })
            : config?.[aggregation.key]?.label || aggregation.key,
          value: selectedValue,
          clearable: isTimeField
            ? Boolean(activeTimeValue)
            : Array.isArray(selectedValue)
              ? selectedValue.length > 0
              : Boolean(selectedValue),
          onChange: (value: any) => {
            onChange(value, aggregation);
          },
          onClear: () => {
            if (isTimeField) {
              onTimeChange([]);
              return;
            }
            onClear(aggregation);
          }
        };
        let content;
        if (type === 'color') {
          content = (
            <FilterColorPicker
              {...commonProps}
              onChange={value => onChange(value?.toHex(), aggregation)}
            />
          );
        } else if (type === 'tag') {
          content = (
            <FilterTags
              {...commonProps}
              value={commonProps.value || []}
              options={bucketList.map(item => ({
                label: item.name || friendlyValue(item.key),
                value: item.key
              }))}
            />
          );
        } else if (isTimeField) {
          content = (
            <FilterCheckboxGroup
              {...commonProps}
              value={activeTimeValue ? [activeTimeValue] : []}
              options={bucketList.map(item => ({
                label: item.name || friendlyValue(item.key),
                value: item.key,
                icon: item.icon || FilterDefaultSvg,
                count: item.count
              }))}
              onChange={onTimeChange}
            />
          );
        } else {
          content = (
            <FilterCheckboxGroup
              {...commonProps}
              value={commonProps.value || []}
              options={bucketList.map(item => ({
                label: isOwnerField
                  ? ownerDisplayName(item.key) || item.name || friendlyValue(item.key)
                  : item.name || friendlyValue(item.key),
                value: item.key,
                icon: item.icon || FilterDefaultSvg,
                count: item.count
              }))}
            />
          );
        }
        return (
          <div
            className='mb-24px'
            key={aggregation.key}
          >
            {content}
          </div>
        );
      })}
    </>
  );
}

export default Aggregations;
