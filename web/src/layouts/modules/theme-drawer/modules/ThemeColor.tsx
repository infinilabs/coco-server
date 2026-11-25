import { Button, Switch, Tooltip } from 'antd';

import { getAppearance } from '@/store/slice/server';
import { getThemeSettings, setRecommendColor, themeColors } from '@/store/slice/theme';

import SettingItem from '../components/SettingItem';

import CustomPicker from './CustomPicker';

const ThemeColor = memo(() => {
  const { t } = useTranslation();

  const themeSettings = useAppSelector(getThemeSettings);

  // branding colors configured in the appearance settings are enforced on
  // every boot — the local pickers are display-only in that case
  const themeEnforced = Boolean(useAppSelector(getAppearance)?.theme_colors);

  const dispatch = useAppDispatch();

  const colors = useAppSelector(themeColors);

  function handleRecommendColorChange(value: boolean) {
    dispatch(setRecommendColor(value));
  }

  if (themeEnforced) {
    return (
      <Tooltip
        placement="topLeft"
        title={t('page.settings.appearance.labels.theme_enforced')}
      >
        <div className="flex-col-stretch gap-12px opacity-60">
          <SettingItem
            key="recommend-color"
            label={t('theme.recommendColor')}
            seq={4}
          >
            <Switch disabled checked={themeSettings.recommendColor} onChange={handleRecommendColorChange} />
          </SettingItem>
          {Object.entries(colors).map(([key, value], index) => (
            <CustomPicker
              disabled
              index={index}
              isInfoFollowPrimary={themeSettings.isInfoFollowPrimary}
              key={key}
              label={key}
              theme={themeSettings.themeColor}
              value={value}
            />
          ))}
        </div>
      </Tooltip>
    );
  }

  return (
    <div className="flex-col-stretch gap-12px">
      <Tooltip
        placement="topLeft"
        title={
          <p>
            <span className="pr-12px">{t('theme.recommendColorDesc')}</span>
            <br />
            <Button
              className="text-gray"
              href="https://uicolors.app/create"
              rel="noopener noreferrer"
              target="_blank"
              type="link"
            >
              https://uicolors.app/create
            </Button>
          </p>
        }
      >
        <div>
          <SettingItem
            key="recommend-color"
            label={t('theme.recommendColor')}
            seq={4}
          >
            <Switch
              defaultChecked={themeSettings.recommendColor}
              onChange={handleRecommendColorChange}
            />
          </SettingItem>
        </div>
      </Tooltip>
      {Object.entries(colors).map(([key, value], index) => (
        <CustomPicker
          index={index}
          isInfoFollowPrimary={themeSettings.isInfoFollowPrimary}
          key={key}
          label={key}
          theme={themeSettings.themeColor}
          value={value}
        />
      ))}
    </div>
  );
});

export default ThemeColor;
