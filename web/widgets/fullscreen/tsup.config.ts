import type { Options } from 'tsup';
import { defineConfig } from 'tsup';
import svgr from 'esbuild-plugin-svgr';
import path from 'path';

const uiSearchDistDir = path.resolve(__dirname, '../ui-search/dist');

export default defineConfig(
  config =>
    [
      {
        clean: true,
        entry: ['src/index.jsx'],
        format: ['esm'],
        minify: true,
        noExternal: ['react', 'react-dom', 'antd', '@ant-design/cssinjs', 'query-string', 'ui-search'],
        platform: 'browser',
        sourcemap: false,
        splitting: false,
        esbuildPlugins: [svgr()],
        esbuildOptions(options) {
          options.alias = {
            ...options.alias,
            'ui-search': path.join(uiSearchDistDir, 'index.js'),
            'ui-search/css': path.join(uiSearchDistDir, 'index.css'),
          };
        },
      },
      {
        clean: true,
        entry: ['src/styles/index.css'],
        minify: true
      }
    ] as Options[]
);
