import js from '@eslint/js';

/**
 * Flat ESLint config for the native WeChat mini program.
 * The framework injects a set of globals (wx, Page, Component ...) that we
 * declare here so `no-undef` does not flag them.
 */
export default [
  {
    ignores: ['node_modules/**', 'miniprogram_npm/**', 'typings/**'],
  },
  js.configs.recommended,
  {
    files: ['miniprogram/**/*.js', 'scripts/**/*.js'],
    languageOptions: {
      ecmaVersion: 2021,
      sourceType: 'commonjs',
      globals: {
        wx: 'readonly',
        App: 'readonly',
        Page: 'readonly',
        Component: 'readonly',
        Behavior: 'readonly',
        getApp: 'readonly',
        getCurrentPages: 'readonly',
        requirePlugin: 'readonly',
        getRegExp: 'readonly',
        __wxConfig: 'readonly',
        module: 'writable',
        require: 'readonly',
        exports: 'writable',
        process: 'readonly',
        console: 'readonly',
        Buffer: 'readonly',
        __dirname: 'readonly',
        setTimeout: 'readonly',
        clearTimeout: 'readonly',
        setInterval: 'readonly',
        clearInterval: 'readonly',
      },
    },
    rules: {
      'no-unused-vars': ['warn', { args: 'none', ignoreRestSiblings: true }],
      'no-empty': ['warn', { allowEmptyCatch: true }],
      'no-constant-condition': ['error', { checkLoops: false }],
      'no-prototype-builtins': 'off',
    },
  },
];
