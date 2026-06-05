import { defineConfig, type PluginOption } from 'vite'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'
import UnoCSS from '@unocss/vite'

export default defineConfig({
  plugins: [
    react() as PluginOption,
    UnoCSS() as PluginOption
  ],
  root: '.',
  server: {
    port: 3001,
    open: true,
    host: true
  },
  optimizeDeps: {
    exclude: ['@infinilabs/ai-chat']
  },
  resolve: {
    extensions: ['.ts', '.tsx', '.js', '.jsx'],
    alias: {
      '@': resolve(__dirname, 'src'),
    }
  },
  build: {
    outDir: 'dist',
    lib: {
      entry: resolve(__dirname, 'src/index.jsx'),
      formats: ['es'],
      fileName: 'index'
    },
    rollupOptions: {
      external: ['react', 'react-dom'],
      output: {
        globals: {
          react: 'React',
          'react-dom': 'ReactDOM'
        },
        assetFileNames: 'index.[ext]'
      }
    },
    cssCodeSplit: false,
  },
  css: {
    modules: {
      generateScopedName: '[name]__[local]___[hash:base64:5]',
    },
    preprocessorOptions: {
      less: {
        javascriptEnabled: true
      }
    }
  }
})
