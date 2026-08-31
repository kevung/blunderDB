import { defineConfig } from '@playwright/test';

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
        // Specs navigate with page.goto('/') so a local run can point the whole
        // suite at another port — 5173 is a busy default and a squatted one
        // silently tests somebody else's app.
        baseURL: 'http://localhost:5173'
    },
    webServer: {
        command: 'npm run dev',
        url: 'http://localhost:5173',
        reuseExistingServer: !process.env.CI,
        timeout: 30000
    },
    reporter: [['list'], ['html', { open: 'never' }]]
});
