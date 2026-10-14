import { Button, ColorPicker, Divider, Form, Input, Spin } from 'antd';
import '../index.scss';
import { fetchApplicationSetting, fetchSettings, updateSettings } from '@/service/api/server';
import { useLoading, useRequest } from '@sa/hooks';
import { setApplicationSetting } from '@/store/slice/server';
import { setRecommendColor, updateThemeColors } from '@/store/slice/theme';

import ImageUploadItem from './ImageUploadItem';

// built-in values shown as placeholders — anything left empty keeps these
const DEFAULT_COLORS = {
  dark: { base_text: '#E0E0E0', container: '#1C1C1C', layout: '#121212', primary_hint: '' },
  functional: { error: '#f5222d', success: '#52c41a', warning: '#faad14' },
  light: { base_text: '#1F1F1F', container: '#FFFFFF', layout: '#EEF0F3' },
  primary: '#0087FF',
  login_background: '#0087FF'
};

const COLOR_PRESETS = [
  {
    colors: ['#0087FF', '#5A54F9', '#722ED1', '#0FB981', '#F5A623', '#F5222D', '#EB2F96', '#13C2C2'],
    label: 'Brand'
  }
];

const EMPTY_APPEARANCE = {
  login: { background_color: '', background_image: '' },
  logo: { dark: '', icon: '', light: '' },
  search: { background: { dark: '', light: '' }, logo: { dark: '', light: '' } },
  slogan: '',
  theme_colors: {
    dark: { base_text: '', container: '', layout: '' },
    error: '',
    light: { base_text: '', container: '', layout: '' },
    primary: '',
    primary_dark: '',
    success: '',
    warning: ''
  },
  title: ''
};

interface ColorFieldProps {
  hint?: string;
  label: string;
  onChange: (value: string) => void;
  /** live-preview callback on drag end (optional) */
  onPreview?: (value: string) => void;
  /** shown while unset — the built-in default */
  placeholder?: string;
  value?: string;
}

/**
 * One color row: picker + current value + reset-to-default. `onChangeComplete`
 * (drag end) drives the live theme preview so dragging doesn't spam the store.
 */
const ColorField = memo(({ hint, label, onChange, onPreview, placeholder, value }: ColorFieldProps) => {
  const { t } = useTranslation();
  return (
    <div className="mb-8px flex items-center" style={{ gap: 12 }}>
      <span className="w-88px shrink-0 text-12px">{label}</span>
      <ColorPicker
        disabledAlpha
        onChange={color => onChange(color.toHexString())}
        onChangeComplete={color => onPreview?.(color.toHexString())}
        presets={COLOR_PRESETS}
        value={value || undefined}
      >
        <span className="cursor-pointer text-12px color-[var(--ant-color-text-tertiary)]">{value || placeholder}</span>
      </ColorPicker>
      {value ? (
        <Button
          className="px-0"
          size="small"
          type="link"
          onClick={() => {
            onChange('');
            onPreview?.(placeholder ?? '');
          }}
        >
          {t('common.reset')}
        </Button>
      ) : null}
      {hint ? <span className="text-12px color-[var(--ant-color-text-description)]">{hint}</span> : null}
    </div>
  );
});
const AppearanceSettings = memo(() => {
  const { t } = useTranslation();

  const { hasAuth } = useAuth();
  const permissions = {
    update: hasAuth('coco#system/update')
  };

  const dispatch = useAppDispatch();
  const { endLoading, loading, startLoading } = useLoading();

  const [values, setValues] = useState<any>(() => structuredClone(EMPTY_APPEARANCE));

  const {
    data,
    loading: dataLoading,
    run
  } = useRequest(fetchSettings, {
    manual: true
  });

  useMount(() => {
    run();
  });

  useEffect(() => {
    if (data?.appearance) {
      setValues((state: any) => {
        // deep-merge saved values over the empty shape so fields missing from
        // older saves still have their slot
        const merged: any = structuredClone(EMPTY_APPEARANCE);
        for (const key of Object.keys(merged)) {
          if (typeof merged[key] === 'string') {
            merged[key] = data.appearance[key] || '';
          } else {
            Object.assign(merged[key], data.appearance[key] || {});
            for (const sub of Object.keys(merged[key])) {
              if (typeof merged[key][sub] === 'string') continue;
              Object.assign(merged[key][sub], data.appearance[key]?.[sub] || {});
            }
          }
        }
        return merged;
      });
    }
  }, [JSON.stringify(data)]);

  const setValue = (path: string[], value: string) => {
    setValues((state: any) => {
      const next = structuredClone(state);
      let target = next;
      for (const key of path.slice(0, -1)) {
        target = target[key];
      }
      target[path[path.length - 1]] = value;
      return next;
    });
  };

  const getValue = (path: string[]): string => {
    let target = values;
    for (const key of path) {
      target = target?.[key];
    }
    return target || '';
  };

  /** live-preview the picked color on the current session (save makes it permanent) */
  const previewColor = (key: App.Theme.ThemeColorKey, color: string) => {
    if (!color) return;
    dispatch(setRecommendColor(false));
    dispatch(updateThemeColors({ color, key }));
  };

  const handleSubmit = async () => {
    startLoading();
    // only the fields this page edits — slogan and search-page branding live
    // elsewhere now (search settings), so they are omitted and preserved by
    // the backend's section merge
    const payload = {
      appearance: {
        title: getValue(['title']),
        theme_colors: values.theme_colors,
        logo: { icon: getValue(['logo', 'icon']) },
        login: values.login
      }
    };
    const result = await updateSettings(payload);
    if (result?.data?.acknowledged) {
      // refresh the public slice so branding takes effect in this session too
      const app = await fetchApplicationSetting();
      await dispatch(setApplicationSetting(app.data));
      window.$message?.success(t('common.updateSuccess'));
    }
    endLoading();
  };

  const renderModeColors = (mode: 'dark' | 'light') => {
    const dark = mode === 'dark';
    const defaults = dark ? DEFAULT_COLORS.dark : DEFAULT_COLORS.light;
    return (
      <div>
        <ColorField
          label={t('page.settings.appearance.labels.primary')}
          onChange={v => setValue(['theme_colors', dark ? 'primary_dark' : 'primary'], v)}
          onPreview={v => previewColor('primary', v)}
          placeholder={DEFAULT_COLORS.primary}
          value={getValue(['theme_colors', dark ? 'primary_dark' : 'primary'])}
        />
        <ColorField
          label={t('page.settings.appearance.labels.layout_bg')}
          onChange={v => setValue(['theme_colors', mode, 'layout'], v)}
          placeholder={defaults.layout}
          value={getValue(['theme_colors', mode, 'layout'])}
        />
        <ColorField
          label={t('page.settings.appearance.labels.container_bg')}
          onChange={v => setValue(['theme_colors', mode, 'container'], v)}
          placeholder={defaults.container}
          value={getValue(['theme_colors', mode, 'container'])}
        />
        <ColorField
          label={t('page.settings.appearance.labels.base_text')}
          onChange={v => setValue(['theme_colors', mode, 'base_text'], v)}
          placeholder={defaults.base_text}
          value={getValue(['theme_colors', mode, 'base_text'])}
        />
      </div>
    );
  };

  return (
    <ListContainer>
      <Spin spinning={dataLoading || loading}>
        <Form className="settings-form py-24px" colon={false} labelAlign="left">
          {/* ---- brand ---- */}
          <div className="color-[var(--ant-color-text)] font-medium mb-24px">
            {t('page.settings.appearance.brand')}
          </div>
          <Form.Item help={t('page.settings.appearance.labels.title_hint')} label={t('page.settings.appearance.labels.title')}>
            <Input
              allowClear
              maxLength={40}
              onChange={e => setValue(['title'], e.target.value)}
              placeholder="Coco AI"
              value={getValue(['title'])}
            />
          </Form.Item>

          <Divider />

          {/* ---- theme colors (enforced) ---- */}
          <div className="color-[var(--ant-color-text)] font-medium mb-8px">
            {t('page.settings.appearance.theme_colors')}
          </div>
          <div className="mb-24px settings-form-help">{t('page.settings.appearance.labels.theme_enforced')}</div>
          <Form.Item label={t('page.settings.appearance.labels.light_mode')}>
            {renderModeColors('light')}
          </Form.Item>
          <Form.Item label={t('page.settings.appearance.labels.dark_mode')}>
            {renderModeColors('dark')}
          </Form.Item>
          <Form.Item label={t('page.settings.appearance.labels.functional')}>
            <div>
              <ColorField
                label={t('page.settings.appearance.labels.success')}
                onChange={v => setValue(['theme_colors', 'success'], v)}
                onPreview={v => previewColor('success', v)}
                placeholder={DEFAULT_COLORS.functional.success}
                value={getValue(['theme_colors', 'success'])}
              />
              <ColorField
                label={t('page.settings.appearance.labels.warning')}
                onChange={v => setValue(['theme_colors', 'warning'], v)}
                onPreview={v => previewColor('warning', v)}
                placeholder={DEFAULT_COLORS.functional.warning}
                value={getValue(['theme_colors', 'warning'])}
              />
              <ColorField
                label={t('page.settings.appearance.labels.error')}
                onChange={v => setValue(['theme_colors', 'error'], v)}
                onPreview={v => previewColor('error', v)}
                placeholder={DEFAULT_COLORS.functional.error}
                value={getValue(['theme_colors', 'error'])}
              />
            </div>
          </Form.Item>

          <Divider />

          {/* ---- app icon ---- */}
          <div className="color-[var(--ant-color-text)] font-medium mb-24px">
            {t('page.settings.appearance.app_logo')}
          </div>
          <Form.Item label={t('page.settings.appearance.labels.logo')}>
            <ImageUploadItem
              hint={t('page.settings.appearance.labels.logo_icon_hint')}
              label={t('page.settings.appearance.labels.logo_icon')}
              onChange={v => setValue(['logo', 'icon'], v)}
              previewSize={40}
              value={getValue(['logo', 'icon'])}
            />
          </Form.Item>

          <Divider />

          {/* ---- login page ---- */}
          <div className="color-[var(--ant-color-text)] font-medium mb-24px">
            {t('page.settings.appearance.login_page')}
          </div>
          <Form.Item label={t('page.settings.appearance.labels.login_background')}>
            <div className="mb-8px flex items-center" style={{ gap: 12 }}>
              <ColorPicker
                disabledAlpha
                onChange={color => setValue(['login', 'background_color'], color.toHexString())}
                presets={COLOR_PRESETS}
                value={getValue(['login', 'background_color']) || undefined}
              >
                <span className="cursor-pointer text-12px color-[var(--ant-color-text-tertiary)]">
                  {getValue(['login', 'background_color']) || DEFAULT_COLORS.login_background}
                </span>
              </ColorPicker>
              {getValue(['login', 'background_color']) ? (
                <Button className="px-0" size="small" type="link" onClick={() => setValue(['login', 'background_color'], '')}>
                  {t('common.reset')}
                </Button>
              ) : null}
            </div>
            <ImageUploadItem
              hint={t('page.settings.appearance.labels.login_background_hint')}
              label={t('page.settings.appearance.labels.login_background_image')}
              onChange={v => setValue(['login', 'background_image'], v)}
              previewSize={40}
              value={getValue(['login', 'background_image'])}
            />
          </Form.Item>

          {permissions.update && (
            <Form.Item label=" ">
              <Button type="primary" onClick={() => handleSubmit()}>
                {t('common.update')}
              </Button>
            </Form.Item>
          )}
        </Form>
      </Spin>
    </ListContainer>
  );
});

export default AppearanceSettings;
