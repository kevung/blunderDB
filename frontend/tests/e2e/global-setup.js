/**
 * Charge l'application une fois avant les tests : Vite transforme et optimise ses modules à la
 * première requête, et ce coût à froid tombait sur les deux premiers tests des workers, dont le
 * budget (10 s) est calibré pour un serveur chaud.
 */
import { chromium } from '@playwright/test';

export default async function globalSetup(config) {
    const browser = await chromium.launch();
    try {
        const page = await browser.newPage();
        await page.goto(config.projects[0].use.baseURL, { waitUntil: 'networkidle', timeout: 25000 });
    } finally {
        await browser.close();
    }
}
