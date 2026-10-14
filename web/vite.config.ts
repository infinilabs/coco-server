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
          // Split the previously monolithic vendor-core chunk so the browser
          // downloads large, rarely-changing dependency groups in parallel and
          // re-fetches none of them when only app code changes. Package names
          // are resolved from the segment after the LAST node_modules/ so the
          // rule works under pnpm's .pnpm/<pkg>@<ver>/node_modules/<pkg>/ paths.
          manualChunks: id => {
            if (!id.includes('node_modules')) return undefined;
            if (id.includes('@ant-design/pro-components') || id.includes('@ant-design/pro-')) {
              return 'vendor-antd-pro';
            }
            const segments = id.substring(id.lastIndexOf('node_modules/') + 'node_modules/'.length).split('/');
            const pkg = segments[0]?.startsWith('@') ? `${segments[0]}/${segments[1] ?? ''}` : (segments[0] ?? '');
            if (pkg === 'mermaid' || pkg.startsWith('@mermaid-js/') || pkg === 'cytoscape' || pkg.startsWith('cytoscape-') || pkg === 'dagre' || pkg === 'dagre-d3-es' || pkg === 'graphlib' || pkg === 'khroma') {
              return 'vendor-mermaid';
            }
            if (pkg === 'pdfjs-dist' || pkg === 'react-pdf' || pkg === 'pdfast' || pkg === 'docx-preview') {
              return 'vendor-doc';
            }
            if (pkg === 'katex') {
              return 'vendor-katex';
            }
            if (pkg === 'highlight.js' || pkg === 'lowlight') {
              return 'vendor-highlight';
            }
            if (/^(react-markdown|remark-|rehype-|unified$|micromark|mdast-util-|hast-util-|unist-util-|vfile|property-information|html-url-attributes|space-separated-tokens|comma-separated-tokens|trim-lines|markdown-table|longest-streak|decode-named-character-reference|character-entities|entities|ccount|devlop|fault$)/.test(pkg)) {
              return 'vendor-markdown';
            }
            // bridges into the markdown family; left in the core chunk they
            // re-create the core <-> markdown cycle the grouping above avoids
            if (/^(hastscript|prettier|parse5|parse-entities|stringify-entities|htmlparser2|dom-serializer|domhandler|domutils|refractor|character-entities)/.test(pkg)) {
              return 'vendor-markdown';
            }
            if (pkg === 'react-syntax-highlighter') {
              return 'vendor-highlight';
            }
            // @antv/g2 depends on pdfast (vendor-doc), which would otherwise
            // re-create the core <-> doc cycle from the chart stack
            if (pkg.startsWith('@antv/') || pkg === 'd3' || pkg.startsWith('d3-')) {
              return 'vendor-antv';
            }
            if (pkg === 'antd' || pkg.startsWith('@ant-design/') || pkg.startsWith('@rc-component/') || pkg.startsWith('rc-')) {
              return 'vendor-antd';
            }
            if (pkg === 'framer-motion' || pkg === 'motion' || pkg === 'motion-dom' || pkg === 'motion-utils') {
              return 'vendor-motion';
            }
            // @infinilabs/* UI packages import antd, so leaving them in the
            // catch-all core chunk creates a vendor-core <-> vendor-antd cycle;
            // depending on which chunk the browser evaluates first, rc-util's
            // module-init reads react before the core chunk exports it and the
            // app dies with "Cannot read properties of undefined (reading
            // 'useLayoutEffect')". A dedicated leaf chunk keeps the graph acyclic.
            if (pkg.startsWith('@infinilabs/')) {
              return 'vendor-infini';
            }
            return 'vendor-core';
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
