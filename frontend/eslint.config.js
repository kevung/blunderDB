import js from '@eslint/js';
import svelte from 'eslint-plugin-svelte';
import prettier from 'eslint-config-prettier';
import globals from 'globals';

export default [
    js.configs.recommended,
    ...svelte.configs['flat/recommended'],
    prettier,
    ...svelte.configs['flat/prettier'],
    {
        languageOptions: {
            globals: {
                ...globals.browser,
                ...globals.node,
            },
        },
        rules: {
            'no-unused-vars': [
                'error',
                {
                    varsIgnorePattern: '^_',
                    argsIgnorePattern: '^_',
                    caughtErrorsIgnorePattern: '^_',
                },
            ],
            'no-console': 'error',
        },
    },
    {
        files: ['**/*.svelte'],
        rules: {
            // Many "unused" vars in Svelte are used in the template
            'no-unused-vars': [
                'error',
                {
                    varsIgnorePattern: '^_|^\\$',
                    argsIgnorePattern: '^_',
                    caughtErrorsIgnorePattern: '^_',
                },
            ],
            // Adding keys to all each blocks is a large refactor — warn only
            'svelte/require-each-key': 'warn',
            // SvelteSet migration is non-trivial — warn only
            'svelte/prefer-svelte-reactivity': 'warn',
            // Reactive loops are possible but mostly false positives — warn only
            'svelte/infinite-reactive-loop': 'warn',
            // Svelte reactive assignments can look "useless" to the JS analyser
            'no-useless-assignment': 'warn',
        },
    },
    {
        ignores: ['wailsjs/**', 'dist/**', 'node_modules/**'],
    },
];
