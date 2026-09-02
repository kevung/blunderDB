import { defineConfig } from '@playwright/test';

// 5173 is a busy default — gammonGo and other Vite projects squat it, and a
// squatted port silently runs the whole suite against somebody else's app.
// BLUNDERDB_E2E_PORT moves both the dev server and the baseURL together.
const PORT = process.env.BLUNDERDB_E2E_PORT || '5173';

export default defineConfig({
    testDir: 'tests/e2e',
    timeout: 10000,
    expect: { timeout: 2000 },
    // CI runners are shared/noisy; retry a flaky run twice there instead of
    // failing the whole job outright. Local runs stay retry-free so a real
    // failure is never masked on a dev machine.
    retries: process.env.CI ? 2 : 0,
    use: {
        browserName: 'chromium',
        viewport: { width: 1280, height: 800 },
        // Specs navigate with page.goto('/'), so pointing the whole suite at
        // another port is a matter of setting BLUNDERDB_E2E_PORT.
        baseURL: `http://localhost:${PORT}`
    },
    webServer: {
        command: `npm run dev -- --port ${PORT} --strictPort`,
        url: `http://localhost:${PORT}`,
        reuseExistingServer: !process.env.CI,
        timeout: 30000
    },
    reporter: [['list'], ['html', { open: 'never' }]]
});
