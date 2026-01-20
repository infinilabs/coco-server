import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import { resolve } from 'path'
import UnoCSS from '@unocss/vite'

export default defineConfig({
  plugins: [
    react(),
    UnoCSS({
      configFile: resolve(__dirname, '../../uno.config.ts')
    })
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
    alias: {
      '@': resolve(__dirname, 'src'),
    }
  },
  build: {
    outDir: 'dist',
    lib: {
      entry: resolve(__dirname, 'src/index.jsx'),
      name: 'UISearch',
      fileName: 'index'
    },
    rollupOptions: {
      external: ['react', 'react-dom'],
      output: {
        globals: {
          react: 'React',
          'react-dom': 'ReactDOM'
        }
      }
    }
  },
  css: {
    preprocessorOptions: {
      less: {
        javascriptEnabled: true
      }
    }
  }
})
