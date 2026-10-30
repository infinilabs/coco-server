import type { TooltipProps } from 'antd';
import type { CSSProperties } from 'react';

import { getThemeSettings, toggleThemeScheme } from '@/store/slice/theme';

import ButtonIcon from '../stateless/custom/ButtonIcon';

interface Props {
  className?: string;
  /** Show tooltip */
  showTooltip?: boolean;
  style?: CSSProperties;
  /** Tooltip placement */
  tooltipPlacement?: TooltipProps['placement'];
}

const icons: Record<UnionKey.ThemeScheme, string> = {
  auto: 'material-symbols:hdr-auto',
  dark: 'material-symbols:nightlight-rounded',
  light: 'material-symbols:sunny'
};

const ThemeSchemaSwitch: FC<Props> = memo(({ showTooltip = true, tooltipPlacement = 'bottom', ...props }) => {
  const { t } = useTranslation();
  const { themeScheme } = useAppSelector(getThemeSettings);
  const dispatch = useAppDispatch();

  const tooltipContent = showTooltip ? `${t('icon.themeSchema')} : ${t(`theme.themeSchema.${themeScheme}`)}` : '';

  // no view-transition wipe here: on heavy pages (wiki with layered gradient
  // backgrounds) the transition snapshot can stick and leave a half-blended
  // stale frame behind, so the scheme flips instantly instead
  const toggleDark = () => {
    dispatch(toggleThemeScheme());
  };
  return (
    <ButtonIcon
      icon={icons[themeScheme]}
      tooltipContent={tooltipContent}
      {...props}
      tooltipPlacement={tooltipPlacement}
      onClick={toggleDark}
    />
  );
});

export default ThemeSchemaSwitch;
