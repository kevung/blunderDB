/**
 * direction-fullscreen.spec.js — le plein écran dédié de la page Direction.
 *
 * F11 masque le chrome (barre d'outils, onglets, panneau, barre d'état) et rend la fenêtre à
 * la direction ; la grille reste utilisable ; Échap ferme un menu ouvert avant de sortir ;
 * quitter la page met fin au mode.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installDirectionEngine, S2_HALL } from './helpers/directionEngine.js';

const chrome = ['.toolbar', '[data-testid="status-bar"]', '.panel-wrapper'];

async function openDirection(page) {
    await installWailsMock(page, openLibraryMock());
    await installDirectionEngine(page, S2_HALL);
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await page.locator('#tournamentPanel tbody tr').first().click();
    await page.locator('#tournamentPanel .direction-btn').click();
    await page.locator('[data-testid="direction-tab-direction"]').click();
    await expect(page.locator('[data-testid="direction-table-1"]')).toBeVisible();
}

async function expectChrome(page, visible) {
    for (const sel of chrome) {
        if (visible) await expect(page.locator(sel).first()).toBeVisible();
        else await expect(page.locator(sel).first()).toBeHidden();
    }
}

test('F11 masque le chrome, la grille reste utilisable, F11 le rend', async ({ page }) => {
    await openDirection(page);
    await expectChrome(page, true);
    await page.keyboard.press('F11');
    await expectChrome(page, false);
    await expect(page.locator('.direction-view')).toBeVisible();

    await page.locator('[data-testid="direction-table-1"]').click();
    await expect(page.locator('[data-testid="direction-result-card"]')).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(page.locator('[data-testid="direction-result-card"]')).toHaveCount(0);
    await expectChrome(page, false);

    await page.keyboard.press('F11');
    await expectChrome(page, true);
});

test('Échap ferme le menu ouvert avant de quitter le plein écran', async ({ page }) => {
    await openDirection(page);
    await page.locator('[data-testid="direction-fullscreen-toggle"]').click();
    await expectChrome(page, false);

    await page.locator('.grid .cell').first().click({ button: 'right' });
    await expect(page.locator('.context-menu')).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(page.locator('.context-menu')).toHaveCount(0);
    await expectChrome(page, false);

    await page.keyboard.press('Escape');
    await expectChrome(page, true);
});

test('le mode survit aux onglets de direction ; F11 hors de la Direction ne fait rien', async ({ page }) => {
    await openDirection(page);
    await page.keyboard.press('F11');
    await page.locator('[data-testid="direction-tab-players"]').click();
    await expectChrome(page, false);
    await page.locator('[data-testid="direction-tab-direction"]').click();
    await expectChrome(page, false);
    await page.keyboard.press('Escape');
    await expectChrome(page, true);

    await page.locator('[data-testid="tab-search"]').click();
    await page.keyboard.press('F11');
    await expectChrome(page, true);
});
