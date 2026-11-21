import { useEffect, useRef, useState } from 'react';
import { Button, Input } from 'antd';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';

// 搜索设置 · 结果 · 推荐主题: same management shape as the blocked-pattern
// rows — one input per topic + delete icon, add button underneath; order is
// preserved because it drives the home-page rotation order
interface Row {
  key: number;
  text: string;
}

const MAX = 20;

const compile = (rows: Row[]) => {
  const seen = new Set<string>();
  return rows
    .map((r) => r.text.trim())
    .filter((t) => {
      if (!t || seen.has(t)) return false;
      seen.add(t);
      return true;
    });
};

interface FieldProps {
  value?: string[];
  onChange?: (topics: string[]) => void;
  /** feature switch off → rows stay visible & editable-free (dimmed) */
  disabled?: boolean;
}

const RecommendTopicsField = ({ value = [], onChange, disabled = false }: FieldProps) => {
  const { t } = useTranslation();
  const keyRef = useRef(0);
  const nextKey = () => ++keyRef.current;
  const [rows, setRows] = useState<Row[]>(() => (value ?? []).map((v) => ({ key: nextKey(), text: v })));
  const [atMax, setAtMax] = useState(rows.length >= MAX);

  // external re-sync (initial load / discard): rebuild rows only when the
  // incoming value differs from what the current rows would compile to, so
  // operator typing is never clobbered by the echo of its own onChange
  useEffect(() => {
    const incoming = value ?? [];
    setRows((prev) => {
      if (JSON.stringify(compile(prev)) === JSON.stringify(incoming)) return prev;
      setAtMax(incoming.length >= MAX);
      return incoming.map((v) => ({ key: nextKey(), text: v }));
    });
  }, [value]);

  const emit = (next: Row[]) => {
    setRows(next);
    setAtMax(compile(next).length >= MAX);
    onChange?.(compile(next));
  };

  const update = (key: number, text: string) =>
    emit(rows.map((r) => (r.key === key ? { ...r, text } : r)));

  const remove = (key: number) => emit(rows.filter((r) => r.key !== key));

  const add = () => emit([...rows, { key: nextKey(), text: '' }]);

  return (
    <div className={`flex flex-col gap-8px ${disabled ? 'opacity-50 pointer-events-none' : ''}`}>
      <div className="flex flex-col gap-8px">
        {rows.map((row) => (
          <div key={row.key} className="flex max-w-520px items-center gap-8px">
            <Input
              allowClear
              className="flex-1"
              maxLength={100}
              placeholder={t('page.settings.search_settings.labels.recommend_topics_placeholder')}
              value={row.text}
              onChange={(e) => update(row.key, e.target.value)}
            />
            <Button
              size="small"
              type="text"
              icon={<DeleteOutlined />}
              onClick={() => remove(row.key)}
            />
          </div>
        ))}
      </div>
      <div>
        <Button size="small" icon={<PlusOutlined />} disabled={atMax} onClick={add}>
          {t('page.settings.search_settings.labels.recommend_topics_add')}
        </Button>
        {atMax && (
          <span className="ml-8px text-12px text-gray-400">
            {t('page.settings.search_settings.labels.recommend_topics_max')}
          </span>
        )}
      </div>
    </div>
  );
};

export default RecommendTopicsField;
