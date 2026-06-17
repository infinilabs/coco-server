import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import svgr from 'vite-plugin-svgr';
import { resolve } from 'path';

export default defineConfig({
  plugins: [
    react(),
    // @ts-ignore -- vite version mismatch between workspace packages
    svgr({
      svgrOptions: {
        exportType: 'default',
      },
    }),
  ],
  define: {
    'process.env.NODE_ENV': JSON.stringify('production'),
  },
  build: {
    lib: {
      entry: resolve(__dirname, 'src/index.tsx'),
      formats: ['es'],
      fileName: 'index',
    },
    cssCodeSplit: false,
    minify: true,
    sourcemap: false,
    rollupOptions: {
      output: {
        assetFileNames: 'index.[ext]',
        inlineDynamicImports: true,
      },
    },
  },
});
