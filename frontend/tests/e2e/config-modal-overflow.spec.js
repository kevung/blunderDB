/**
 * La fenêtre des Paramètres tient ses onglets : aucun n'est coupé par le cadre de la
 * fenêtre ni par l'écran, à 1024×700 comme à 1280×800.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, dismissHomeScreen } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';

for (const [width, height] of [
    [1024, 700],
    [1280, 800]
]) {
    test(`les onglets des Paramètres restent dans la fenêtre à ${width}×${height}`, async ({ page }) => {
        await page.setViewportSize({ width, height });
        await installWailsMock(page, openLibraryMock());
        await page.goto('/');
        await dismissHomeScreen(page);
        await page.locator('button[aria-label="Settings"]').click();
        const box = page.locator('.modal-box').first();
        await expect(box).toBeVisible();

        const frame = await box.boundingBox();
        const tabs = page.locator('.modal-box [role="tab"]');
        const count = await tabs.count();
        expect(count).toBeGreaterThanOrEqual(8);
        for (let i = 0; i < count; i++) {
            const r = await tabs.nth(i).boundingBox();
            expect(r.x, `onglet ${i} à gauche`).toBeGreaterThanOrEqual(frame.x - 1);
            expect(r.x + r.width, `onglet ${i} à droite`).toBeLessThanOrEqual(frame.x + frame.width + 1);
            expect(r.y + r.height, `onglet ${i} en bas de l'écran`).toBeLessThanOrEqual(height + 1);
        }
        expect(frame.x + frame.width).toBeLessThanOrEqual(width);
    });
}
