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
        // HelpModal renders the in-app help, which is app-owned HTML loaded from
        // src/i18n/help/*.js (no user input), so {@html} is safe here by design.
        files: ['**/components/HelpModal.svelte'],
        rules: {
            'svelte/no-at-html-tags': 'off',
        },
    },
    {
        // src/i18n/help/*.js are generated data files: each exports a single
        // multi-KB HTML template-literal string per help tab. They parse fine at
        // runtime (node/Vite) but exceed eslint's espree parser limits, so we skip
        // linting them — they contain no logic to lint.
        ignores: ['wailsjs/**', 'dist/**', 'node_modules/**', 'src/i18n/help/*.js'],
    },
];
