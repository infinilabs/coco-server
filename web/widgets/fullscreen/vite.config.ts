import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import svgr from 'vite-plugin-svgr'
import path from 'path'

export default defineConfig({
    plugins: [
        react(),
        svgr()
    ],
    define: {
        'process.env': {},
    },
    build: {
        lib: {
            entry: path.resolve(__dirname, 'src/index.jsx'),
            name: "fullscreen",
            fileName: "index",
            formats: ["es"],
        },
        outDir: 'dist',
        sourcemap: false,
        minify: 'esbuild',
        rollupOptions: {
            external: ['antd', '@ant-design/cssinjs'],
        },
    },
})
