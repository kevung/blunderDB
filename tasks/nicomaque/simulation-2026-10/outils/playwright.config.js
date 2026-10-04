// Configuration Playwright de la simulation #380 : le vrai front (Vite, frontend/) sur le vrai
// backend (shim, voir e2e/shimProcess.js). Un seul worker : chaque spec tient son shim et sa base.
import { defineConfig } from '@playwright/test';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const here = path.dirname(fileURLToPath(import.meta.url));
const FRONT = path.resolve(here, '../../../../frontend');
const PORT = process.env.BLUNDERDB_E2E_PORT || '5183';

export default defineConfig({
    testDir: 'e2e',
    timeout: Number(process.env.SIM_TIMEOUT || 30 * 60 * 1000),
    expect: { timeout: 8000 },
    workers: 1,
    retries: 0,
    use: {
        browserName: 'chromium',
        launchOptions: { executablePath: process.env.CHROMIUM || '/usr/bin/chromium' },
        viewport: { width: 1366, height: 768 },
        baseURL: `http://localhost:${PORT}`,
        actionTimeout: 8000
    },
    webServer: {
        // --strictPort : un port déjà pris échoue au lieu de tester une autre application.
        command: `npm run dev -- --port ${PORT} --strictPort`,
        cwd: FRONT,
        url: `http://localhost:${PORT}`,
        reuseExistingServer: false,
        timeout: 60000
    },
    reporter: [['line']],
    outputDir: process.env.SIM_DIR ? path.join(process.env.SIM_DIR, 'pw-out') : '/tmp/blunderdb-sim/pw-out'
});
