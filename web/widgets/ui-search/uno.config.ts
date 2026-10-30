import { defineConfig, presetWind3 } from 'unocss'

// dark: variants must key off the layout container's `dark` class — the
// widget resolves its theme explicitly (theme prop first, OS preference
// only for 'system'). The media-query default re-applies the OS dark
// palette on top of a light-configured widget whenever macOS runs dark,
// the same double-source-of-truth bug as the markdown theme vars.
export default defineConfig({
  content: {
    filesystem: [
      'src/**/*.{html,js,ts,jsx,tsx,vue,svelte,astro}',
    ],
  },
  presets: [
    presetWind3({ dark: 'class' }),
  ],
})