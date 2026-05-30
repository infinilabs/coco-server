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
    esbuild: {
      drop: configEnv.command === 'build' ? ['console', 'debugger'] : []
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
        '~': fileURLToPath(new URL('./', import.meta.url)),
        // -------------------------------------------------------------------
        // TODO(临时方案 / TEMPORARY — 上生产前务必移除):
        // 本地通过 pnpm link 引入 ../../ui-common/packages/AIChat 进行联调时，
        // AIChat 的 dist 产物在被 rollup 解析时，会从它自身目录向上查找
        // react-router-dom，而 ui-common 那边没装这个 peer dep，导致构建失败
        //   "Rollup failed to resolve import 'react-router-dom' from
        //    .../AIChat/dist/ganttDiagram-*.js"
        // 这里通过 alias 强制把它指向 coco/web 自己装的副本，让本地联调能直接
        // 构建通过。
        //
        // 注意：不要把 react / react-dom 也用目录形式 alias，这样会绕过
        // 包自身 package.json 的 exports 条件，导致 react-dom/cjs 下的
        // server.node.* 被打进浏览器 bundle，运行时报
        // "Cannot read properties of undefined (reading 'prototype')"
        // （util.inherits 找不到）。
        //
        // 正式发布前一定要：
        //   1) 撤掉下面这一行 alias；
        //   2) 让 web/package.json 改回依赖发布版的 @infinilabs/ai-chat
        //      （即移除当前的 pnpm.overrides link 指向）。
        // -------------------------------------------------------------------
        'react-router-dom': fileURLToPath(new URL('./node_modules/react-router-dom', import.meta.url))
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
