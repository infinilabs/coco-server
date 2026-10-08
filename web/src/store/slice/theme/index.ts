import { createSelector, createSlice } from '@reduxjs/toolkit';
import type { PayloadAction } from '@reduxjs/toolkit';
import { getPaletteColorByNumber, getRgb } from '@sa/color';

import { DARK_MODE_MEDIA_QUERY } from '@/constants/common';
import { localStg } from '@/utils/storage';

import type { AppThunk } from '../..';

import { initThemeSettings, toggleAuxiliaryColorModes, toggleGrayscaleMode, updateDarkMode } from './shared';

interface InitialStateType {
  darkMode: boolean;
  settings: App.Theme.ThemeSetting;
}

type DeepPartial<T> = {
  [P in keyof T]?: T[P] extends (infer U)[]
    ? DeepPartial<U>[]
    : T[P] extends ReadonlyArray<infer U>
      ? ReadonlyArray<DeepPartial<U>>
      : DeepPartial<T[P]>;
};

const themeSchemes: UnionKey.ThemeScheme[] = ['light', 'dark', 'auto'];

const initTheme = initThemeSettings();

const initialState: InitialStateType = {
  darkMode: window.$wujie?.props?.theme
    ? window.$wujie?.props?.theme === 'dark'
    : initTheme.themeScheme === 'auto'
      ? window.matchMedia && window.matchMedia(DARK_MODE_MEDIA_QUERY).matches
      : initTheme.themeScheme === 'dark',
  settings: initThemeSettings()
};

export const themeSlice = createSlice({
  initialState,
  name: 'theme',
  reducers: {
    changeReverseHorizontalMix(state, { payload }: PayloadAction<boolean>) {
      state.settings.layout.reverseHorizontalMix = payload;
    },

    resetTheme() {
      return initialState;
    },
    /**
     * Set colourWeakness value
     *
     * @param isColourWeakness
     */
    setColourWeakness(state, { payload }: PayloadAction<boolean>) {
      toggleAuxiliaryColorModes(payload);
      state.settings.colourWeakness = payload;
    },
    setDarkMode(state, { payload }: PayloadAction<boolean>) {
      state.darkMode = payload;
    },
    setFixedHeaderAndTab(state, { payload }: PayloadAction<boolean>) {
      state.settings.fixedHeaderAndTab = payload;
    },
    setFooter(state, { payload }: PayloadAction<Partial<App.Theme.ThemeSetting['footer']>>) {
      Object.assign(state.settings.footer, payload);
    },
    /**
     * Set grayscale value
     *
     * @param isGrayscale
     */
    setGrayscale(state, { payload }: PayloadAction<boolean>) {
      toggleGrayscaleMode(payload);
      state.settings.grayscale = payload;
    },
    setHeader(state, { payload }: PayloadAction<DeepPartial<App.Theme.ThemeSetting['header']>>) {
      Object.assign(state.settings.header, payload);
    },
    setIsInfoFollowPrimary(state, { payload }: PayloadAction<boolean>) {
      state.settings.isInfoFollowPrimary = payload;
    },
    setIsOnlyExpandCurrentParentMenu(state, { payload }: PayloadAction<boolean>) {
      state.settings.isOnlyExpandCurrentParentMenu = payload;
    },
    setLayoutMode(state, { payload }: PayloadAction<UnionKey.ThemeLayoutMode>) {
      state.settings.layout.mode = payload;
    },
    setLayoutScrollMode(state, { payload }: PayloadAction<UnionKey.ThemeScrollMode>) {
      state.settings.layout.scrollMode = payload;
    },
    setPage(state, { payload }: PayloadAction<Partial<App.Theme.ThemeSetting['page']>>) {
      Object.assign(state.settings.page, payload);
    },
    setRecommendColor(state, { payload }: PayloadAction<boolean>) {
      state.settings.recommendColor = payload;
    },
    /**
     * Merge admin-configured surface colors (appearance settings) into one
     * mode's theme tokens. Values arrive as hex and are stored in the rgb()
     * form the CSS-var generator expects; empty/unparseable entries are
     * skipped so they keep the built-in values.
     *
     * @param mode Which theme mode's tokens to patch
     * @param colors Configured neutral colors (layout / container / base_text)
     */
    setThemeTokenColors(
      state,
      { payload: { colors, mode } }: PayloadAction<{ colors: Record<string, string>; mode: 'dark' | 'light' }>
    ) {
      const target = state.settings.tokens[mode]?.colors;
      if (!target) return;
      // hex → the `rgb(r, g, b)` string the CSS-var generator expects
      const toRgb = (hex?: string) => {
        if (!hex) return null;
        try {
          const { b, g, r } = getRgb(hex);
          return `rgb(${r}, ${g}, ${b})`;
        } catch {
          return null;
        }
      };
      const layout = toRgb(colors.layout);
      if (layout) target.layout = layout;
      const container = toRgb(colors.container);
      if (container) target.container = container;
      const baseText = toRgb(colors.base_text);
      if (baseText) target['base-text'] = baseText;
    },
    setSider(state, { payload }: PayloadAction<Partial<App.Theme.ThemeSetting['sider']>>) {
      Object.assign(state.settings.sider, payload);
    },
    setSiderInverted(state, { payload }: PayloadAction<boolean>) {
      state.settings.sider.inverted = payload;
    },
    setTab(state, { payload }: PayloadAction<Partial<App.Theme.ThemeSetting['tab']>>) {
      Object.assign(state.settings.tab, payload);
    },
    /**
     * Set theme scheme
     *
     * @param themeScheme
     */
    setThemeScheme(state, { payload }: PayloadAction<UnionKey.ThemeScheme>) {
      state.darkMode = updateDarkMode(payload);
      state.settings.themeScheme = payload;
    },
    setWatermark(state, { payload }: PayloadAction<Partial<App.Theme.ThemeSetting['watermark']>>) {
      Object.assign(state.settings.watermark, payload);
    },
    /**
     * Update theme colors
     *
     * @param key Theme color key
     * @param color Theme color
     */
    updateThemeColors(
      state,
      { payload: { color, key } }: PayloadAction<{ color: string; key: App.Theme.ThemeColorKey }>
    ) {
      let colorValue = color;

      if (state.settings.recommendColor) {
        // get a color palette by provided color and color name, and use the suitable color

        colorValue = getPaletteColorByNumber(color, 500, true);
      }

      if (key === 'primary') {
        state.settings.themeColor = colorValue;
      } else {
        state.settings.otherColor[key] = colorValue;
      }
    }
  },
  selectors: {
    getDarkMode: theme => {
      if (window.$wujie?.props?.theme) {
        return window.$wujie?.props?.theme === 'dark';
      }
      if (theme.settings.themeScheme === 'auto') {
        return (window.matchMedia && window.matchMedia(DARK_MODE_MEDIA_QUERY).matches) || theme.darkMode;
      }
      return theme.darkMode;
    },
    getThemeSettings: theme => theme.settings
  }
});

export const { getDarkMode, getThemeSettings } = themeSlice.selectors;

export const {
  changeReverseHorizontalMix,
  resetTheme,
  setColourWeakness,
  setDarkMode,
  setFixedHeaderAndTab,
  setFooter,
  setGrayscale,
  setHeader,
  setIsInfoFollowPrimary,
  setIsOnlyExpandCurrentParentMenu,
  setLayoutMode,
  setLayoutScrollMode,
  setPage,
  setRecommendColor,
  setSider,
  setSiderInverted,
  setTab,
  setThemeScheme,
  setThemeTokenColors,
  setWatermark,
  updateThemeColors
} = themeSlice.actions;

// 计算属性选择器
export const themeColors = createSelector([getThemeSettings], ({ isInfoFollowPrimary, otherColor, themeColor }) => {
  const colors: App.Theme.ThemeColor = {
    primary: themeColor,
    ...otherColor,
    info: isInfoFollowPrimary ? themeColor : otherColor.info
  };
  return colors;
});

/** Cache theme settings */
export const cacheThemeSettings = (): AppThunk => (_, getState) => {
  const isProd = import.meta.env.PROD;

  if (!isProd) return;

  localStg.set('themeSettings', getThemeSettings(getState()));
};

export const settingsJson = createSelector([getThemeSettings], settings => {
  return JSON.stringify(settings);
});

export const toggleThemeScheme = (): AppThunk<boolean> => (dispatch, getState) => {
  const themeSettings = getThemeSettings(getState());
  const index = themeSchemes.findIndex(item => item === themeSettings.themeScheme);
  const nextIndex = index === themeSchemes.length - 1 ? 0 : index + 1;
  const nextThemeScheme = themeSchemes[nextIndex];
  const darkMode = updateDarkMode(nextThemeScheme);
  const themeScheme = nextThemeScheme;
  dispatch(setDarkMode(darkMode));
  dispatch(setThemeScheme(themeScheme));
  dispatch(cacheThemeSettings());
  return darkMode;
};
