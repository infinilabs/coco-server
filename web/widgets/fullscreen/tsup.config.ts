import type { Options } from 'tsup';
import { defineConfig } from 'tsup';
import svgr from 'esbuild-plugin-svgr';

export default defineConfig(
  config =>
    [
      {
        clean: true,
        entry: ['src/index.jsx'],
        format: ['esm'],
        minify: true,
        noExternal: ['react', 'react-dom', 'antd', '@ant-design/cssinjs', 'query-string'],
        platform: 'browser',
        sourcemap: true,
        splitting: false,
        esbuildPlugins: [svgr()],
      },
      {
        clean: true,
        entry: ['src/styles/index.css'],
        minify: true
      }
    ] as Options[]
);
