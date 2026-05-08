import { defineConfig } from '@playwright/test';

export default defineConfig({
    testDir: 'tests/e2e',
    timeout: 10000,
    expect: { timeout: 2000 },
    use: {
        browserName: 'chromium',
        viewport: { width: 1280, height: 800 },
    },
    webServer: {
        command: 'npm run dev',
        url: 'http://localhost:5173',
        reuseExistingServer: !process.env.CI,
        timeout: 30000,
    },
    reporter: [['list'], ['html', { open: 'never' }]],
});
