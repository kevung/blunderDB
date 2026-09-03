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
                ...globals.node
            }
        },
        rules: {
            // varsIgnorePattern deliberately does NOT cover plain let/const
            // declarations — an unused `_foo` local should be an error (dead
            // code), not silently tolerated. destructuredArrayIgnorePattern
            // keeps the escape hatch for array-destructuring positions you
            // must name but don't need (e.g. `const [, _second] = pair`).
            'no-unused-vars': [
                'error',
                {
                    destructuredArrayIgnorePattern: '^_',
                    argsIgnorePattern: '^_',
                    caughtErrorsIgnorePattern: '^_'
                }
            ],
            'no-console': 'error'
        }
    },
    {
        files: ['**/*.svelte'],
        rules: {
            // Many "unused" vars in Svelte are used in the template; store
            // auto-subscriptions ($store) are the one pattern still allowed
            // through varsIgnorePattern here. Plain `_foo` locals are not.
            'no-unused-vars': [
                'error',
                {
                    varsIgnorePattern: '^\\$',
                    destructuredArrayIgnorePattern: '^_',
                    argsIgnorePattern: '^_',
                    caughtErrorsIgnorePattern: '^_'
                }
            ],
            // #205: every `{#each}` in the codebase is now keyed (verified
            // 2026-09-03, zero warnings from either rule below) — promoted
            // from 'warn' so a future unkeyed each-block or reactive loop
            // fails CI instead of accumulating silently the way the 35
            // Svelte-compiler warnings this fiche put a ceiling on did.
            'svelte/require-each-key': 'error',
            'svelte/prefer-svelte-reactivity': 'error',
            'svelte/infinite-reactive-loop': 'error',
            // Svelte reactive assignments can look "useless" to the JS analyser
            'no-useless-assignment': 'warn'
        }
    },
    {
        // src/i18n/help/*.js are generated data files: each exports a single
        // multi-KB HTML template-literal string per help tab. They parse fine at
        // runtime (node/Vite) but exceed eslint's espree parser limits, so we skip
        // linting them — they contain no logic to lint.
        ignores: ['wailsjs/**', 'dist/**', 'node_modules/**', 'src/i18n/help/*.js']
    }
];
