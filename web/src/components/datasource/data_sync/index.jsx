import { Cascader, InputNumber, Radio, Select } from 'antd';
import { useCallback, useState, useEffect } from 'react';
import './index.scss';

const style = {
  display: 'flex',
  flexDirection: 'column',
  gap: 8
};

// The cron cascader expresses "frequency + concrete slot(s)". Hour values are
// zero-padded strings ('00'..'23') so paths round-trip exactly. Only whole
// hours are offered (minute fixed at 0). Monthly day is capped at 28 — the
// only day number present in every month, so no skip semantics needed.
const MONTHLY_MAX_DAY = 28;

// Parse a 6-field cron back into a cascader path, or ['custom'] when the
// expression doesn't fit any frequency/slot combination.
const parseCronPath = expr => {
  const parts = (expr || '').trim().split(/\s+/);
  if (parts.length !== 6) return ['custom'];
  const [s, m, H, dom, mon, dow] = parts;
  if (s !== '0' || m !== '0') return ['custom'];
  const isNum = v => /^\d+$/.test(v);
  if (isNum(H) && Number(H) <= 23) {
    const h = String(Number(H)).padStart(2, '0');
    if (dom === '*' && mon === '*' && dow === '*') return ['daily', h];
    if (dom === '*' && mon === '*' && isNum(dow) && Number(dow) >= 1 && Number(dow) <= 7) {
      return ['weekly', String(Number(dow)), h];
    }
    if (isNum(dom) && Number(dom) >= 1 && Number(dom) <= MONTHLY_MAX_DAY && mon === '*' && dow === '*') {
      return ['monthly', String(Number(dom)), h];
    }
  }
  return ['custom'];
};

const buildCron = path => {
  const [freq, a, b] = path;
  switch (freq) {
    case 'daily':
      return `0 0 ${Number(a)} * * *`;
    case 'weekly':
      return `0 0 ${Number(b)} * * ${Number(a)}`;
    case 'monthly':
      return `0 0 ${Number(b)} ${Number(a)} * *`;
    default:
      return '';
  }
};

// DataSync: pull-mode sync strategies only — manual waits for
// POST /datasource/:id/_sync, interval auto-dispatches on the configured
// cadence, cron auto-dispatches when a schedule slot passes since the last
// run (e.g. daily 00:00). Push mode (formerly the "realtime" option) is now
// owned by the datasource-level Webhook switch: when webhook is enabled the
// pages hide this block entirely and disable pull, so realtime is no longer
// selectable here. A legacy datasource stored with strategy=realtime but
// webhook off shows an inline note instead of a selectable option.
export const DataSync = ({ onChange, value = { enabled: true, interval: '1h', strategy: 'interval' } }) => {
  const { t } = useTranslation();
  const [innerV, setValue] = useState(value);

  // Update local state when value prop changes
  useEffect(() => {
    setValue(value);
  }, [value]);

  const onInnerChange = ev => {
    setValue(oldV => {
      const newV = {
        ...oldV,
        strategy: ev.target.value
      };
      if (typeof onChange === 'function') {
        onChange(newV);
      }
      return newV;
    });
  };
  const onIntervalChange = useCallback(interval => {
    setValue(oldV => {
      const newV = {
        ...oldV,
        interval
      };
      if (typeof onChange === 'function') {
        onChange(newV);
      }
      return newV;
    });
  }, []);
  const onCrontabChange = useCallback(crontab => {
    setValue(oldV => {
      const newV = {
        ...oldV,
        crontab
      };
      if (typeof onChange === 'function') {
        onChange(newV);
      }
      return newV;
    });
  }, []);
  return (
    <div id="data-sync-comp">
      <Radio.Group
        style={style}
        value={innerV.strategy}
        onChange={onInnerChange}
      >
        <Radio
          className="mb-10px flex"
          value="manual"
        >
          <div>
            <div className="mb-5px">{t('page.datasource.new.labels.manual_sync')}</div>
            <div className="text-gray-400">{t('page.datasource.new.labels.manual_sync_desc')}</div>
          </div>
        </Radio>
        <Radio
          className="mb-10px"
          value="interval"
        >
          <div>
            <div className="mb-5px">{t('page.datasource.new.labels.scheduled_sync')}</div>
            <div className="text-gray-400">{t('page.datasource.new.labels.scheduled_sync_desc')}</div>
          </div>
        </Radio>
        {innerV.strategy === 'interval' && (
          <div className="mb-10px ml-25px">
            <SyncTime
              value={innerV.interval}
              onChange={onIntervalChange}
            />
          </div>
        )}
        <Radio
          className="mb-10px"
          value="cron"
        >
          <div>
            <div className="mb-5px">{t('page.datasource.new.labels.cron_sync')}</div>
            <div className="text-gray-400">{t('page.datasource.new.labels.cron_sync_desc')}</div>
          </div>
        </Radio>
        {innerV.strategy === 'cron' && (
          <div className="mb-10px ml-25px">
            <CronTime
              value={innerV.crontab}
              onChange={onCrontabChange}
            />
          </div>
        )}
        {innerV.strategy === 'realtime' && (
          <div className="ml-25px" style={{ color: 'var(--ant-color-warning, #faad14)' }}>
            {t('page.datasource.new.labels.realtime_legacy_note')}
          </div>
        )}
      </Radio.Group>
    </div>
  );
};

const SyncTime = ({ onChange, value }) => {
  // Extract number and unit from value (default to "10s")
  const { t } = useTranslation();
  const match = value?.match(/^(\d+)([smh])$/);

  const [num, setNum] = useState(match ? Number.parseInt(match[1], 10) : 10);
  const [unit, setUnit] = useState(match ? match[2] : 's');

  // Follow incoming value changes: the edit page loads the stored interval
  // asynchronously after mount, so num/unit must track the prop instead of
  // freezing at the initial parse (P2-5 echo bug).
  useEffect(() => {
    const m = value?.match(/^(\d+)([smh])$/);
    if (m) {
      setNum(Number.parseInt(m[1], 10));
      setUnit(m[2]);
    }
  }, [value]);

  // Update parent component when num or unit changes
  useEffect(() => {
    onChange?.(`${num}${unit}`);
  }, [num, unit, onChange]);

  return (
    <InputNumber
      addonBefore={t('page.datasource.every')}
      min={1}
      value={num}
      addonAfter={
        <Select
          getPopupContainer={triggerNode => triggerNode.parentNode}
          value={unit}
          options={[
            { label: t('page.datasource.seconds'), value: 's' },
            { label: t('page.datasource.minutes'), value: 'm' },
            { label: t('page.datasource.hours'), value: 'h' }
          ]}
          onChange={newUnit => setUnit(newUnit)}
        />
      }
      onChange={newNum => setNum(newNum || 1)}
    />
  );
};

// Hour slots '00'..'23' labeled 'HH:00'; reused across every frequency.
const buildHourOptions = () =>
  Array.from({ length: 24 }, (_, h) => {
    const v = String(h).padStart(2, '0');
    return { value: v, label: `${v}:00` };
  });

// Weekday labels localized via Intl (dow 1..7 = Monday..Sunday); Jan 1+n 2023
// starts on Sunday so +n lands each dow on its own weekday.
const weekdayLabel = (n, lang) =>
  new Intl.DateTimeFormat(lang, { weekday: 'short' }).format(new Date(2023, 0, 1 + n));

const CronTime = ({ onChange, value }) => {
  const { t, i18n } = useTranslation();
  const effective = value || '0 0 0 * * *';

  // Explicit mode state: deriving "custom" from the value alone is a trap —
  // a custom-mode value can legitimately equal a preset expression, which
  // would re-hide the input. Mode flips to custom only on user action or when
  // an incoming value doesn't match any cascader path (edit page async load).
  const [mode, setMode] = useState(parseCronPath(effective)[0] === 'custom' ? 'custom' : 'preset');

  useEffect(() => {
    if (parseCronPath(value || '0 0 0 * * *')[0] === 'custom') {
      setMode('custom');
    }
  }, [value]);

  const hourOptions = buildHourOptions();
  const options = [
    { value: 'daily', label: t('page.datasource.new.labels.cron_freq_daily'), children: hourOptions },
    {
      value: 'weekly',
      label: t('page.datasource.new.labels.cron_freq_weekly'),
      children: [1, 2, 3, 4, 5, 6, 7].map(d => ({
        value: String(d),
        label: weekdayLabel(d, i18n.language),
        children: hourOptions
      }))
    },
    {
      value: 'monthly',
      label: t('page.datasource.new.labels.cron_freq_monthly'),
      children: Array.from({ length: MONTHLY_MAX_DAY }, (_, i) => i + 1).map(d => ({
        value: String(d),
        label: t('page.datasource.new.labels.cron_preset_monthly_day', { day: d }),
        children: hourOptions
      }))
    }
  ];

  const path = parseCronPath(effective);
  const onCascaderChange = v => {
    onChange?.(buildCron(v));
  };

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 8 }}>
      <Cascader
        style={{ width: 260 }}
        options={options}
        value={path}
        onChange={onCascaderChange}
        placeholder="0 0 0 * * *"
      />
      {path[0] === 'custom' && (
        <div style={{ color: 'var(--ant-color-warning, #faad14)' }}>
          {t('page.datasource.new.labels.cron_unmapped_hint')}
        </div>
      )}
    </div>
  );
};
