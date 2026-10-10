import type { AppThunk } from '../..';

import { setRecommendColor, setThemeTokenColors, updateThemeColors } from './index';

/**
 * Apply the admin-configured appearance theme onto the theme slice. Called on
 * every boot (and every light/dark switch) so the server-side branding always
 * wins over any locally cached theme — that's what "enforced" means here.
 *
 * Empty fields are skipped so a partially configured section keeps the
 * built-in defaults for the rest.
 */
export const applyAppearanceTheme =
  (appearance: Record<string, any> | undefined | null, darkMode: boolean): AppThunk =>
  dispatch => {
    const colors = appearance?.theme_colors;
    if (!colors) return;

    dispatch(setRecommendColor(false));

    const primary = darkMode ? colors.primary_dark || colors.primary : colors.primary;
    if (primary) dispatch(updateThemeColors({ color: primary, key: 'primary' }));

    if (colors.success) dispatch(updateThemeColors({ color: colors.success, key: 'success' }));
    if (colors.warning) dispatch(updateThemeColors({ color: colors.warning, key: 'warning' }));
    if (colors.error) dispatch(updateThemeColors({ color: colors.error, key: 'error' }));

    if (colors.light) dispatch(setThemeTokenColors({ colors: colors.light, mode: 'light' }));
    if (colors.dark) dispatch(setThemeTokenColors({ colors: colors.dark, mode: 'dark' }));
  };
