import babel from '@rollup/plugin-babel';
import commonjs from '@rollup/plugin-commonjs';
import resolve from '@rollup/plugin-node-resolve';
import terser from '@rollup/plugin-terser';
import deletePlugin from 'rollup-plugin-delete';
import postcss from 'rollup-plugin-postcss';
import autoprefixer from 'autoprefixer';
import UnoCSS from '@unocss/postcss';
import url from '@rollup/plugin-url';
import replace from '@rollup/plugin-replace';
import cssnano from 'cssnano';

export default {
  input: 'src/index.jsx',
  output: [
    {
      file: 'dist/index.js',
      format: 'esm',
      sourcemap: false,
      inlineDynamicImports: true
    },
    // {
    //   file: 'dist/index.iife.js',
    //   format: 'iife',
    //   name: 'UISearch',
    //   sourcemap: true,
    //   globals: {
    //     'react': 'React',
    //     'react-dom': 'ReactDOM',
    //     'antd': 'antd',
    //   },
    // },
  ],
  external: ['react', 'react-dom'],
  context: 'window',
  plugins: [
    deletePlugin({ targets: 'dist/*' }),
    replace({
      'process.env.NODE_ENV': JSON.stringify('production'),
      preventAssignment: true,
    }),
    resolve({ extensions: ['.js', '.jsx'], browser: true }),
    commonjs({
      include: 'node_modules/**',
      transformMixedEsModules: true
    }),
    url({
      include: ['**/*.svg'],
      limit: 10000000,
    }),
    babel({
      babelHelpers: 'runtime',
      presets: [
        [
          '@babel/preset-react',
          {
            'runtime': 'automatic'
          }
        ]
      ],
      exclude: 'node_modules/**',
      plugins: [
        '@babel/plugin-transform-runtime',
      ],
    }),
    postcss({
      extensions: ['.scss', '.less', '.css'],
      modules: {
        generateScopedName: '[name]__[local]___[hash:base64:5]',
      },
      use: ['less', 'sass'],
      extract: 'index.css',
      minimize: true,
      plugins: [
        UnoCSS,
        autoprefixer,
        cssnano({ preset: 'default' }),
      ],
    }),
    terser(),
  ],
  onwarn(warning, warn) {
    if (warning.code === 'MODULE_LEVEL_DIRECTIVE') {
      return;
    }
    warn(warning);
  },
};
