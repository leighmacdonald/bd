// @ts-check

import eslint from '@eslint/js';
import tseslint from 'typescript-eslint';
import eslintPluginPrettierRecommended from 'eslint-plugin-prettier/recommended';

export default tseslint.config(
    eslint.configs.recommended,
    ...tseslint.configs.strict,
    ...tseslint.configs.stylistic,
    eslintPluginPrettierRecommended,
    {
        files: ['src/**/*.ts', 'src/**/*.tss', 'eslint.config.js'],
        // Doesn't actually work? It's ignored via cli flag in the Makefile for now.
        ignores: ['dist/']
    }
);
