import js from '@eslint/js'
import boundaries from 'eslint-plugin-boundaries'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import { defineConfig, globalIgnores } from 'eslint/config'
import globals from 'globals'
import tseslint from 'typescript-eslint'

export default defineConfig([
  globalIgnores(['dist', 'src/shared/api/schema.d.ts']),
  {
    files: ['**/*.{ts,tsx}'],
    extends: [
      js.configs.recommended,
      tseslint.configs.recommended,
      reactHooks.configs.flat.recommended,
      reactRefresh.configs.vite,
    ],
    plugins: {
      boundaries,
    },
    settings: {
      'boundaries/elements': [
        { type: 'app', pattern: 'src/app/**' },
        { type: 'pages', pattern: 'src/pages/**' },
        { type: 'widgets', pattern: 'src/widgets/**' },
        { type: 'features', pattern: 'src/features/**' },
        { type: 'entities', pattern: 'src/entities/**' },
        { type: 'shared', pattern: 'src/shared/**' },
      ],
    },
    rules: {
      '@typescript-eslint/no-explicit-any': 'error',
      '@typescript-eslint/no-unused-vars': ['error', { argsIgnorePattern: '^_' }],
      'react-hooks/exhaustive-deps': 'warn',
      'boundaries/dependencies': [
        'error',
        {
          default: 'disallow',
          rules: [
            { from: { type: 'app' }, allow: { to: { type: ['pages', 'widgets', 'features', 'entities', 'shared'] } } },
            { from: { type: 'pages' }, allow: { to: { type: ['widgets', 'features', 'entities', 'shared'] } } },
            { from: { type: 'widgets' }, allow: { to: { type: ['features', 'entities', 'shared'] } } },
            { from: { type: 'features' }, allow: { to: { type: ['entities', 'shared'] } } },
            { from: { type: 'entities' }, allow: { to: { type: 'shared' } } },
            { from: { type: 'shared' }, allow: { to: { type: 'shared' } } },
          ],
        },
      ],
    },
    languageOptions: {
      ecmaVersion: 2023,
      globals: globals.browser,
    },
  },
])
