// @ts-check

import eslint from '@eslint/js';
import tseslint from 'typescript-eslint';
import eslintPluginPrettierRecommended from 'eslint-plugin-prettier/recommended';
// import * as reactQuery from '@tanstack/eslint-plugin-query';

export default tseslint.config(
    eslint.configs.recommended,
    ...tseslint.configs.recommended,
    // {
    //     plugins: {
    //         '@tanstack/query': reactQuery
    //     },
    //     rules: {
    //         ...reactQuery.configs.recommended.rules
    //     }
    // },
    eslintPluginPrettierRecommended,
    {
        files: ['src/**/*.ts', 'eslint.config.js']
    }
);
