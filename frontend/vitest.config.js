import { defineConfig } from 'vitest/config';
import { svelte } from '@sveltejs/vite-plugin-svelte';

export default defineConfig({
    plugins: [svelte({ hot: !process.env.VITEST })],
    resolve: {
        conditions: ['browser']
    },
    test: {
        environment: 'jsdom',
        include: ['src/**/*.{test,spec}.{js,ts}'],
        globals: true,
        setupFiles: ['./src/test-setup.js'],
        coverage: {
            provider: 'v8',
            reporter: ['text', 'lcov'],
            include: ['src/**/*.{js,svelte}'],
            // Generated Wails bindings, the help corpus, and the tests themselves
            // are not code we measure. No threshold yet: the number is informative
            // until a baseline is agreed on.
            exclude: ['src/**/*.{test,spec}.{js,ts}', 'src/__tests__/**', 'src/test-setup.js', 'src/i18n/help/**', '**/wailsjs/**']
        }
    }
});
