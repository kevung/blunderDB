/**
 * zoom-layout.spec.js
 *
 * L'échelle de l'interface est un `zoom` CSS sur le conteneur principal. Les moteurs multiplient
 * les unités de fenêtre (vw, vh) par ce facteur, et les coordonnées des événements restent en
 * pixels de fenêtre : à toute échelle, l'interface occupe exactement la fenêtre, le plateau est
 * centré dans sa zone et un menu contextuel s'ouvre sous le pointeur.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { showcaseGalleryMock } from './helpers/showcase.js';

/** Règle l'échelle par les raccourcis, comme l'utilisateur. @param {import('@playwright/test').Page} page @param {number} scale */
async function setScale(page, scale) {
    await page.keyboard.press('Control+0');
    const steps = Math.round((scale - 1) * 10);
    for (let i = 0; i < Math.abs(steps); i++) await page.keyboard.press(steps > 0 ? 'Control+Equal' : 'Control+Minus');
    await expect(page.locator('.main-container')).toHaveCSS('zoom', String(scale));
}

/** @param {import('@playwright/test').Page} page @param {string} selector */
async function box(page, selector) {
    const b = await page.locator(selector).first().boundingBox();
    if (!b) throw new Error(`${selector} absent`);
    return b;
}

for (const scale of [0.5, 1.5, 2]) {
    test(`à ${scale * 100} %, l'interface occupe toute la fenêtre et le plateau est centré`, async ({ page }) => {
        await installWailsMock(page, showcaseGalleryMock());
        await page.goto('/');
        await expect(page.locator('.panel-wrapper')).toBeVisible();
        await setScale(page, scale);
        const view = page.viewportSize() ?? { width: 0, height: 0 };

        await expect(async () => {
            const main = await box(page, '.main-container');
            expect(main.x).toBeCloseTo(0, 0);
            expect(main.y).toBeCloseTo(0, 0);
            expect(main.width).toBeCloseTo(view.width, 0);
            expect(main.height).toBeCloseTo(view.height, 0);

            const panel = await box(page, '.panel-wrapper');
            expect(panel.width).toBeCloseTo(view.width, 0);
            const status = await box(page, '[data-testid="status-bar"]');
            expect(status.y + status.height).toBeCloseTo(view.height, 0);

            const area = await box(page, '.scrollable-content');
            const board = await box(page, '#backgammon-board svg');
            expect(Math.abs(board.x + board.width / 2 - (area.x + area.width / 2))).toBeLessThan(2);
            expect(board.y + board.height).toBeLessThanOrEqual(area.y + area.height + 1);
        }).toPass();
    });
}

test('à 150 %, le menu contextuel s’ouvre sous le pointeur', async ({ page }) => {
    await installWailsMock(page, showcaseGalleryMock());
    await page.goto('/');
    await expect(page.locator('.panel-wrapper')).toBeVisible();
    await setScale(page, 1.5);
    const tab = await box(page, '[data-testid^="tab-"]');
    const at = { x: tab.x + tab.width / 2, y: tab.y + tab.height / 2 };
    await page.mouse.click(at.x, at.y, { button: 'right' });
    const menu = await box(page, '.context-menu');
    expect(Math.abs(menu.x - at.x)).toBeLessThan(2);
    expect(Math.abs(menu.y - at.y)).toBeLessThan(2);
});

test('à 150 %, un clic droit sur le plateau y ouvre son menu, sous le pointeur', async ({ page }) => {
    await installWailsMock(page, showcaseGalleryMock());
    await page.goto('/');
    await expect(page.locator('.panel-wrapper')).toBeVisible();
    await setScale(page, 1.5);
    const board = await box(page, '#backgammon-board svg');
    const at = { x: board.x + board.width * 0.3, y: board.y + board.height * 0.3 };
    await page.mouse.click(at.x, at.y, { button: 'right' });
    await expect(page.getByRole('menuitem', { name: 'Evaluate this position' })).toBeVisible();
    const menu = await box(page, '.context-menu');
    expect(Math.abs(menu.x - at.x)).toBeLessThan(2);
    expect(Math.abs(menu.y - at.y)).toBeLessThan(2);
});
