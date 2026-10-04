/**
 * Mise en page de la vue Direction : pleine largeur de la zone, un défilement par onglet dont la
 * position survit au changement d'onglet, cibles d'au moins 40 px.
 */
import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installDirectionEngine } from './helpers/directionEngine.js';

async function openDirection(page) {
    await page.setViewportSize({ width: 1024, height: 768 });
    await installWailsMock(page, openLibraryMock({ config: { GetPanelPosition: 'bottom' } }));
    await installDirectionEngine(page);
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await page.locator('#tournamentPanel tbody tr').first().click();
    await page.locator('#tournamentPanel .direction-btn').click();
    await expect(page.locator('.direction-view')).toBeVisible();
}

test('la vue prend toute la largeur de la zone, quel que soit l’onglet', async ({ page }) => {
    await openDirection(page);
    const widths = [];
    for (const id of ['settings', 'direction', 'players']) {
        await page.locator(`[data-testid="direction-tab-${id}"]`).click();
        widths.push(await page.locator('.direction-view').evaluate((e) => Math.round(e.getBoundingClientRect().width)));
    }
    expect(new Set(widths).size).toBe(1);
    const area = await page.locator('.scrollable-content').evaluate((e) => Math.round(e.getBoundingClientRect().width));
    expect(widths[0]).toBe(area);
});

test('la position de défilement d’un onglet est gardée quand on le quitte', async ({ page }) => {
    await openDirection(page);
    await page.locator('[data-testid="direction-tab-settings"]').click();
    const pane = page.locator('[data-testid="direction-pane-settings"]');
    await pane.evaluate((e) => (e.scrollTop = 120));
    await page.locator('[data-testid="direction-tab-players"]').click();
    await page.locator('[data-testid="direction-tab-settings"]').click();
    await expect.poll(() => pane.evaluate((e) => e.scrollTop)).toBe(120);
});

test('les onglets sont des cibles d’au moins 40 px', async ({ page }) => {
    await openDirection(page);
    const h = await page.locator('[data-testid="direction-tab-settings"]').evaluate((e) => e.getBoundingClientRect().height);
    expect(h).toBeGreaterThanOrEqual(40);
});

test('PageDown fait défiler l’onglet actif, et la position revient après un détour', async ({ page }) => {
    await openDirection(page);
    await page.locator('[data-testid="direction-tab-players"]').click();
    await page.locator('[data-testid="direction-tab-settings"]').click();
    const pane = page.locator('[data-testid="direction-pane-settings"]');
    await expect(pane).toBeFocused();
    await page.keyboard.press('PageDown');
    await expect.poll(() => pane.evaluate((e) => e.scrollTop)).toBeGreaterThan(0);
    // Le défilement du clavier est animé : on attend qu'il se soit arrêté.
    let y = -1;
    await expect
        .poll(async () => {
            const now = await pane.evaluate((e) => e.scrollTop);
            const settled = now === y;
            y = now;
            return settled;
        })
        .toBe(true);
    await page.locator('[data-testid="direction-tab-players"]').click();
    await page.locator('[data-testid="direction-tab-settings"]').click();
    await expect.poll(() => pane.evaluate((e) => e.scrollTop)).toBe(y);
});

test('un onglet quitté garde son état (section repliée)', async ({ page }) => {
    await openDirection(page);
    await page.locator('[data-testid="direction-tab-settings"]').click();
    const sec = page.locator('.direction-view details.sec').first();
    await sec.locator('summary').click();
    await expect(sec).not.toHaveAttribute('open', '');
    await page.locator('[data-testid="direction-tab-players"]').click();
    await page.locator('[data-testid="direction-tab-settings"]').click();
    await expect(sec).not.toHaveAttribute('open', '');
});

test('la page murale est accessible depuis l’en-tête', async ({ page }) => {
    await openDirection(page);
    await expect(page.locator('[data-testid="direction-open-page"]')).toBeVisible();
});
