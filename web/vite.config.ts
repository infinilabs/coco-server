import process from 'node:process';
import { URL, fileURLToPath } from 'node:url';

import { defineConfig, loadEnv } from 'vite';

import { createViteProxy, getBuildTime } from './build/config';
import { include } from './build/optimize';
import { setupVitePlugins } from './build/plugins';

// https://vitejs.dev/config/
export default defineConfig(configEnv => {
  const viteEnv = loadEnv(configEnv.mode, process.cwd()) as unknown as Env.ImportMeta;

  const buildTime = getBuildTime();

  const enableProxy = configEnv.command === 'serve' && !configEnv.isPreview;
  return {
    base: viteEnv.VITE_BASE_URL,
    build: {
      outDir: '../.public',
      emptyOutDir: true,
      rollupOptions: {
        output: {
          manualChunks: id => {
            if (id.includes('node_modules')) {
              if (id.includes('@ant-design/pro-components') || id.includes('@ant-design/pro-')) {
                return 'vendor-antd-pro';
              }
              return 'vendor-core';
            }
          }
        }
      }
    },
    css: {
      preprocessorOptions: {
        scss: {
          additionalData: `@use "@/styles/scss/global.scss" as *;`,
          api: 'modern-compiler'
        },
        less: {
          // 启用JavaScript表达式的解析功能
          javascriptEnabled: true,
          // 自定义修改默认的Less变量
          modifyVars: {}
        }
      }
    },
    define: {
      BUILD_TIME: JSON.stringify(buildTime)
    },
    optimizeDeps: {
      include,
      exclude: ['@infinilabs/ai-chat']
    },
    plugins: setupVitePlugins(viteEnv, buildTime),
    preview: {
      port: 9725
    },
    resolve: {
      alias: {
        '@': fileURLToPath(new URL('./src', import.meta.url)),
        '~': fileURLToPath(new URL('./', import.meta.url))
      }
    },
    server: {
      fs: {
        cachedChecks: false
      },
      host: '0.0.0.0',
      open: true,
      port: 9527,
      proxy: createViteProxy(viteEnv, enableProxy),
      warmup: {
        clientFiles: ['./index.html', './src/{pages,components}/*']
      }
    }
  };
});
