/** Create color palette vars */
function createColorPaletteVars() {
  const colors = ['primary', 'info', 'success', 'warning', 'error'];
  const colorPaletteNumbers = [50, 100, 200, 300, 400, 500, 600, 700, 800, 900, 950];

  const colorPaletteVar = {};

  colors.forEach(color => {
    colorPaletteVar[color] = `rgb(var(--${color}-color))`;
    colorPaletteNumbers.forEach(number => {
      colorPaletteVar[`${color}-${number}`] = `rgb(var(--${color}-${number}-color))`;
    });
  });

  return colorPaletteVar;
}

const colorPaletteVars = createColorPaletteVars();

/** Theme vars */
export const themeVars = {
  boxShadow: {
    header: 'var(--header-box-shadow)',
    sider: 'var(--sider-box-shadow)',
    tab: 'var(--tab-box-shadow)'
  },
  colors: {
    ...colorPaletteVars,
    'base-text': 'rgb(var(--base-text-color))',
    container: 'rgb(var(--container-bg-color))',
    inverted: 'rgb(var(--inverted-bg-color))',
    layout: 'rgb(var(--layout-bg-color))',
    nprogress: 'rgb(var(--nprogress-color))',
  }
};
