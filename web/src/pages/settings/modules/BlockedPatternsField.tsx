import { useEffect, useRef, useState } from 'react';
import { Button, Input, Select } from 'antd';
import { DeleteOutlined, PlusOutlined } from '@ant-design/icons';

type Mode = 'contains' | 'exact' | 'regex';

// storage stays the operator regex array (blocked_query_patterns); each row
// pairs a mode dropdown with a keyword input, mirroring the pinned-rule rows
const escapeRe = (s: string) => s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
const unescapeRe = (s: string) => s.replace(/\\(.)/g, '$1');

// per-pattern classification: plain literal → contains, ^literal$ → exact,
// everything else → regex (a real regex never classifies as exact because its
// body keeps meta characters after the unescape round-trip)
const classify = (p?: string): Mode => {
  if (!p) return 'regex';
  if (escapeRe(unescapeRe(p)) === p) return 'contains';
  const m = p.match(/^\^([\s\S]+)\$$/);
  if (m && escapeRe(unescapeRe(m[1])) === m[1]) return 'exact';
  return 'regex';
};

// display text for a stored pattern: unwrap/unescape word modes, raw regex as-is
const prettify = (p?: string): string => {
  if (!p) return '';
  const kind = classify(p);
  if (kind === 'contains') return unescapeRe(p);
  if (kind === 'exact') return unescapeRe(p.replace(/^\^([\s\S]+)\$$/, '$1'));
  return p;
};

// how the row keyword is stored, per the row's mode dropdown
const applyMode = (word: string, mode: Mode) => {
  if (mode === 'contains') return escapeRe(word);
  if (mode === 'exact') return `^${escapeRe(word)}$`;
  return word;
};

const isValidRe = (p: string) => {
  try {
    new RegExp(p);
    return true;
  } catch {
    return false;
  }
};

// live preview shares the backend semantics: case-insensitive substring match
// (RE2 vs JS regex edge cases exist but are fine for a save-free preview)
const matchesAny = (query: string, patterns: string[]) =>
  patterns.some((p) => {
    try {
      return new RegExp(p, 'i').test(query);
    } catch {
      return false;
    }
  });

interface Row {
  key: number;
  mode: Mode;
  text: string;
}

const compile = (rows: Row[]) =>
  rows
    .filter((r) => r.text.trim())
    .map((r) => applyMode(r.text.trim(), r.mode));

const toRows = (patterns: string[], keyOf: () => number): Row[] =>
  patterns.map((p) => ({ key: keyOf(), mode: classify(p), text: prettify(p) }));

interface FieldProps {
  value?: string[];
  onChange?: (patterns: string[]) => void;
}

// 搜索设置 · 安全 · 查询屏蔽词: one row per pattern — a mode dropdown
// (contains / exact / regex) + a keyword input + a delete icon, add button
// underneath, same management shape as the sibling intervention fields;
// everything compiles down to the same regex array the backend consumes
const BlockedPatternsField = ({ value = [], onChange }: FieldProps) => {
  const { t } = useTranslation();
  const keyRef = useRef(0);
  const nextKey = () => ++keyRef.current;
  const [rows, setRows] = useState<Row[]>(() => toRows(value, nextKey));
  const [testText, setTestText] = useState('');

  // external re-sync (initial load / discard): rebuild rows only when the
  // incoming value differs from what the current rows would compile to, so
  // operator typing is never clobbered by the echo of its own onChange
  useEffect(() => {
    const incoming = value ?? [];
    setRows((prev) => {
      if (JSON.stringify(compile(prev)) === JSON.stringify(incoming)) return prev;
      return toRows(incoming, nextKey);
    });
  }, [value]);

  const verdict = (() => {
    const query = testText.trim();
    if (!query) return null;
    try {
      return matchesAny(query, value) ? 'block' : 'allow';
    } catch {
      return 'invalid';
    }
  })();

  const update = (key: number, patch: Partial<Row>) => {
    const next = rows.map((r) => (r.key === key ? { ...r, ...patch } : r));
    setRows(next);
    onChange?.(compile(next));
  };

  const remove = (key: number) => {
    const next = rows.filter((r) => r.key !== key);
    setRows(next);
    onChange?.(compile(next));
  };

  const add = () => setRows((prev) => [...prev, { key: nextKey(), mode: 'contains', text: '' }]);

  const modeOptions = [
    { label: t('page.settings.search_settings.labels.blocked_mode_contains'), value: 'contains' },
    { label: t('page.settings.search_settings.labels.blocked_mode_exact'), value: 'exact' },
    { label: t('page.settings.search_settings.labels.blocked_mode_regex'), value: 'regex' }
  ];

  return (
    <div className="flex flex-col gap-8px">
      <div className="flex flex-col gap-8px">
        {rows.map((row) => {
          const badRegex = row.mode === 'regex' && row.text.trim() !== '' && !isValidRe(row.text.trim());
          return (
            <div key={row.key} className="flex flex-col gap-2px">
              <div className="flex max-w-520px items-center gap-8px">
                <Select
                  className="blocked-patterns-select !w-170px shrink-0"
                  value={row.mode}
                  options={modeOptions}
                  onChange={(mode) => update(row.key, { mode })}
                />
                <Input
                  allowClear
                  className="flex-1"
                  placeholder={
                    row.mode === 'regex' ? '^internal.*$' : t('page.settings.search_settings.labels.blocked_words_placeholder')
                  }
                  value={row.text}
                  status={badRegex ? 'error' : undefined}
                  onChange={(e) => update(row.key, { text: e.target.value })}
                />
                <Button
                  size="small"
                  type="text"
                  icon={<DeleteOutlined />}
                  onClick={() => remove(row.key)}
                />
              </div>
              {badRegex && (
                <div className="pl-178px text-12px" style={{ color: '#a32d2d' }}>
                  {t('page.settings.search_settings.labels.blocked_invalid_regex')}
                </div>
              )}
            </div>
          );
        })}
      </div>
      <div>
        <Button size="small" icon={<PlusOutlined />} onClick={add}>
          {t('page.settings.search_settings.labels.blocked_add')}
        </Button>
      </div>
      {value.length > 0 && (
        <div className="text-12px text-gray-400">
          {t('page.settings.search_settings.labels.blocked_effective')}
          <span className="ml-4px font-mono">{value.join(' · ')}</span>
        </div>
      )}
      <div className="mt-4px text-12px text-gray-400">
        {t('page.settings.search_settings.labels.blocked_test_label')}
      </div>
      <Input
        allowClear
        className="max-w-520px"
        placeholder={t('page.settings.search_settings.labels.blocked_test_placeholder')}
        value={testText}
        onChange={(e) => setTestText(e.target.value)}
        suffix={
          verdict ? (
            <span
              className="text-12px font-500"
              style={{ color: verdict === 'allow' ? '#3b6d11' : '#a32d2d' }}
            >
              {verdict === 'block'
                ? t('page.settings.search_settings.labels.blocked_verdict_block')
                : verdict === 'allow'
                  ? t('page.settings.search_settings.labels.blocked_verdict_allow')
                  : t('page.settings.search_settings.labels.blocked_invalid_regex')}
            </span>
          ) : null
        }
      />
    </div>
  );
};

export default BlockedPatternsField;
