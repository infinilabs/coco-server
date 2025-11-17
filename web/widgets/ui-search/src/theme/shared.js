import { getColorPalette, getRgb, transformColorWithOpacity } from './color';
import { theme as antdTheme } from 'antd';

import { overrideThemeSettings, themeSettings } from './settings';
import { themeVars } from './vars';

export const DARK_MODE_MEDIA_QUERY = '(prefers-color-scheme: dark)';
export const DARK_CLASS = 'dark';
export const VAR_PREFIX = 'ui-search'

/** Init theme settings */
export function initThemeSettings() {
  return Object.assign(themeSettings, overrideThemeSettings);
}

/**
 * Get css var by tokens
 *
 * @param tokens Theme base tokens
 */
function getCssVarByTokens(tokens, prefix = '') {
  const styles = [];

  function removeVarPrefix(value) {
    return value.replace('var(', '').replace(')', '');
  }

  function removeRgbPrefix(value) {
    return value.replace('rgb(', '').replace(')', '');
  }

  for (const [key, tokenValues] of Object.entries(themeVars)) {
    for (const [tokenKey, tokenValue] of Object.entries(tokenValues)) {
      let cssVarsKey = removeVarPrefix(tokenValue);
      let cssValue = tokens[key][tokenKey];

      if (key === 'colors') {
        cssVarsKey = removeRgbPrefix(cssVarsKey);
        const { b, g, r } = getRgb(cssValue);
        cssValue = `${r} ${g} ${b}`;
      }
      styles.push(`--${prefix + cssVarsKey}: ${cssValue}`);
    }
  }

  const styleStr = styles.join(';');

  return styleStr;
}

/**
 * Add theme vars to global
 *
 * @param tokens
 */
export function addThemeVarsToGlobal(tokens, darkTokens, prefix = '', container) {
  const cssVarStr = getCssVarByTokens(tokens, prefix);
  const darkCssVarStr = getCssVarByTokens(darkTokens, prefix);

  const css = `
   .${VAR_PREFIX} {
      ${cssVarStr}
    }
  `;

  const darkCss = `
    .${VAR_PREFIX}.${DARK_CLASS} {
      ${darkCssVarStr}
    }
  `;

  const styleId = `${VAR_PREFIX}-theme-vars`;

  const style = (container || document).querySelector(`#${styleId}`) || document.createElement('style');

  style.id = styleId;

  style.textContent = css + darkCss;

  (container || document.head).appendChild(style);
}
/**
 * Create theme palette colors
 *
 * @param colors Theme colors
 * @param [recommended=false] Use recommended color. Default is `false`
 */
function createThemePaletteColors(colors, recommended = false) {
  const colorKeys = Object.keys(colors);
  const colorPaletteVar = {};

  colorKeys.forEach(key => {
    const colorMap = getColorPalette(colors[key], recommended);

    colorPaletteVar[key] = colorMap.get(500);

    colorMap.forEach((hex, number) => {
      colorPaletteVar[`${key}-${number}`] = hex;
    });
  });

  return colorPaletteVar;
}

/**
 * create theme token css vars value by theme settings
 *
 * @param colors Theme colors
 * @param tokens Theme setting tokens
 * @param [recommended=false] Use recommended color. Default is `false`
 */
export function createThemeToken(
  colors,
  tokens,
  recommended = false
) {
  const paletteColors = createThemePaletteColors(colors, recommended);

  const { dark, light } = tokens || themeSettings.tokens;

  const themeTokens = {
    boxShadow: {
      ...light.boxShadow
    },
    colors: {
      ...paletteColors,
      nprogress: paletteColors.primary,
      ...light.colors
    }
  };

  const darkThemeTokens = {
    boxShadow: {
      ...themeTokens.boxShadow,
      ...dark?.boxShadow
    },
    colors: {
      ...themeTokens.colors,
      ...dark?.colors
    }
  };

  return {
    darkThemeTokens,
    themeTokens
  };
}

/**
 * Get antd theme
 *
 * @param colors Theme colors
 * @param darkMode Is dark mode
 */
export function getAntdTheme(
  colors,
  darkMode,
  tokens
) {
  const { darkAlgorithm, defaultAlgorithm } = antdTheme;

  const { error, info, primary, success, warning } = colors;

  const bgColor = transformColorWithOpacity(primary, darkMode ? 0.3 : 0.1, darkMode ? '#000000' : '#fff');
  const containerBgColor = darkMode ? tokens.dark?.colors?.container : tokens.light?.colors.container;

  const theme = {
    algorithm: [darkMode ? darkAlgorithm : defaultAlgorithm],
    components: {
      Button: {
        controlHeightSM: 28
      },
      Collapse: {
        contentPadding: '16px 16px 24px 16px',
        headerBg: containerBgColor
      },
      Menu: {
        darkItemBg: 'transparent',
        darkSubMenuItemBg: 'transparent',
        itemMarginInline: 8,
        itemSelectedBg: bgColor,
        subMenuItemBg: 'transparent'
      }
    },
    cssVar: {
      prefix: `${VAR_PREFIX}-antd`,
    },
    token: {
      colorBgContainer: containerBgColor,
      colorError: error,
      colorInfo: info,
      colorPrimary: primary,
      colorSuccess: success,
      colorWarning: warning
    }
  };

  return theme;
}
/**
 * Toggle css dark mode
 *
 * @param darkMode Is dark mode
 */
export function toggleCssDarkMode(darkMode = false) {
  function addDarkClass() {
    document.documentElement.classList.add(DARK_CLASS);
  }

  function removeDarkClass() {
    document.documentElement.classList.remove(DARK_CLASS);
  }

  if (darkMode) {
    addDarkClass();
  } else {
    removeDarkClass();
  }
}

/**
 * Toggle auxiliary color modes
 *
 * @param grayscaleMode
 * @param colourWeakness
 */
export function toggleAuxiliaryColorModes(colourWeakness = false) {
  const htmlElement = document.documentElement;
  htmlElement.style.filter = colourWeakness ? 'invert(80%)' : '';
}

/** Setup theme vars to html */
export function setupThemeVarsToHtml(
  themeColors,
  tokens,
  recommended = false,
  container
) {
  const { darkThemeTokens, themeTokens } = createThemeToken(themeColors, tokens, recommended);
  addThemeVarsToGlobal(themeTokens, darkThemeTokens, VAR_PREFIX, container);
}

export function updateDarkMode(themeScheme) {
  if (themeScheme === 'dark') {
    return true;
  } else if (themeScheme === 'light') {
    return false;
  }
  return window.matchMedia(DARK_MODE_MEDIA_QUERY).matches;
}

export function toggleHtmlClass(className) {
  function add() {
    document.documentElement.classList.add(className);
  }

  function remove() {
    document.documentElement.classList.remove(className);
  }

  return {
    add,
    remove
  };
}

export function toggleGrayscaleMode(grayscaleMode = false) {
  const GRAYSCALE_CLASS = 'grayscale';

  const { add, remove } = toggleHtmlClass(GRAYSCALE_CLASS);

  if (grayscaleMode) {
    add();
  } else {
    remove();
  }
}
