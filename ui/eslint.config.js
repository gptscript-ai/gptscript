import antfu from '@antfu/eslint-config'
import { FlatCompat } from '@eslint/eslintrc'

const compat = new FlatCompat()

export default antfu({
  ...compat.config({
    env: {
      node: true,
      commonjs: true,
      browser: true,
      es6: true,
    },
    rules: {
      'n/prefer-global/process': 'off',
      '@stylistic/ts/brace-style': 'off',
      '@stylistic/js/key-spacing': 'off',
      '@stylistic/js/space-in-parens': 'off',
      '@stylistic/js/template-curly-spacing': 'off',
      '@typescript-eslint/brace-style': ['warn', '1tbs'],
      '@typescript-eslint/consistent-type-definitions': 'off',
      'array-bracket-spacing': 'warn',
      'arrow-parens': 'warn',
      'arrow-spacing': ['warn', { before: true, after: true }],
      'block-spacing': ['warn', 'always'],
      'brace-style': 'off',
      'comma-dangle': ['warn', 'only-multiline'],
      'comma-spacing': 'warn',
      'curly': ['warn', 'all'],
      'func-call-spacing': ['warn', 'never'],
      'implicit-arrow-linebreak': 'warn',
      'keyword-spacing': 'warn',
      'lines-between-class-members': ['warn', 'always', { exceptAfterSingleLine: true }],
      'multiline-ternary': ['warn', 'never'],
      'newline-per-chained-call': ['warn', { ignoreChainWithDepth: 4 }],
      'no-caller': 'warn',
      'no-cond-assign': ['warn', 'except-parens'],
      'no-console': ['warn', { allow: ['debug', 'info', 'warn', 'error'] }],
      'no-debugger': 'warn',
      'no-eq-null': 'warn',
      'no-eval': 'warn',
      'no-trailing-spaces': 'warn',
      'no-unused-vars': 'warn',
      'no-whitespace-before-property': 'warn',
      'object-curly-spacing': ['warn', 'always'],
      'object-property-newline': 'off',
      'object-shorthand': 'warn',
      'padded-blocks': ['warn', 'never'],
      'prefer-arrow-callback': 'warn',
      'prefer-template': 'warn',
      'quote-props': 'warn',
      'react/display-name': 'off',
      'react/no-unknown-property': 'off',
      'rest-spread-spacing': 'warn',
      'space-before-function-paren': 'off',
      'space-in-parens': 'off',
      'space-infix-ops': 'warn',
      'spaced-comment': 'warn',
      'switch-colon-spacing': 'warn',
      'template-curly-spacing': ['warn', 'always'],
      'yield-star-spacing': ['warn', 'both'],

      'key-spacing': ['warn', {
        align: {
          beforeColon: false,
          afterColon: true,
          on: 'value',
          mode: 'minimum',
        },
        multiLine: {
          beforeColon: false,
          afterColon: true,
        },
      }],

      'object-curly-newline': ['warn', {
        ObjectExpression: {
          multiline: true,
          minProperties: 4,
        },
        ObjectPattern: {
          multiline: true,
          minProperties: 4,
        },
        ImportDeclaration: {
          multiline: true,
          minProperties: 5,
        },
        ExportDeclaration: {
          multiline: true,
          minProperties: 3,
        },
      }],

      'padding-line-between-statements': [
        'warn',
        {
          blankLine: 'always',
          prev: '*',
          next: 'return',
        },
        {
          blankLine: 'always',
          prev: 'function',
          next: 'function',
        },
        {
          blankLine: 'always',
          prev: ['const', 'let', 'var'],
          next: '*',
        },
        {
          blankLine: 'any',
          prev: ['const', 'let', 'var'],
          next: ['const', 'let', 'var'],
        },
      ],

      '@typescript-eslint/quotes': [
        'warn',
        'single',
        {
          avoidEscape: true,
          allowTemplateLiterals: true,
        },
      ],

      'quotes': [
        'warn',
        'single',
        {
          avoidEscape: true,
          allowTemplateLiterals: true,
        },
      ],

      'vue/component-options-name-casing': ['warn', 'kebab-case'],
    },
  }),
})
