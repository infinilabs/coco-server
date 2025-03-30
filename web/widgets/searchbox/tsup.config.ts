import { defineConfig, Options } from "tsup";

export default defineConfig(
  (config) =>
    [
      {
        entry: ["src/index.jsx"],
        format: ["esm"],
        clean: true,
        minify: true,
        sourcemap: config.watch,
        splitting: false,
        platform: 'browser',
        noExternal: ['react', 'react-dom', '@infinilabs/search-chat'],
      },
      {
        entry: [
          "src/styles/index.css",
        ],
        clean: true,
        minify: true,
      },
    ] as Options[],
);