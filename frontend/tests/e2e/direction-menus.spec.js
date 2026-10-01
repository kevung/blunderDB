/**
 * direction-menus.spec.js
 *
 * Les menus contextuels de la page Direction, dans l'application réelle : clic droit et
 * Maj+F10 sur une proposition de la file, Échap qui rend le focus à la ligne.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installDirectionEngine } from './helpers/directionEngine.js';

const proposal = '.proposals .queue li';

async function openDirection(page) {
    await installWailsMock(page, openLibraryMock());
    await installDirectionEngine(page);
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await expect(page.locator('#tournamentPanel')).toBeVisible();
    await page.locator('#tournamentPanel tbody tr').first().click();
    await page.locator('#tournamentPanel .direction-btn').click();
    await expect(page.locator('.direction-view')).toBeVisible();
    await page.locator('[data-testid="direction-tab-direction"]').click();
    await expect(page.locator(proposal).first()).toBeVisible();
}

test('clic droit sur une proposition ouvre son menu, Échap le ferme', async ({ page }) => {
    await openDirection(page);
    await page.locator(proposal).first().click({ button: 'right' });
    const menu = page.locator('.context-menu');
    await expect(menu).toBeVisible();
    await expect(menu.locator('.context-menu-item').first()).toBeFocused();
    await page.keyboard.press('Escape');
    await expect(menu).toHaveCount(0);
});

test('Maj+F10 sur une proposition focalisée ouvre le menu, ENTRÉE lance', async ({ page }) => {
    await openDirection(page);
    const before = await page.locator(proposal).count();
    await page.locator(proposal).first().focus();
    await page.keyboard.press('Shift+F10');
    await expect(page.locator('.context-menu')).toBeVisible();
    await page.keyboard.press('Enter');
    await expect(page.locator(proposal)).toHaveCount(before - 1);
});

test('un seul menu contextuel à la fois : en ouvrir un second ferme le premier', async ({ page }) => {
    await openDirection(page);
    await page.locator(proposal).first().click({ button: 'right' });
    await expect(page.locator('.context-menu')).toHaveCount(1);
    await page.locator('.grid .cell').first().click({ button: 'right' });
    await expect(page.locator('.context-menu')).toHaveCount(1);
    await expect(page.locator('.context-menu .context-menu-item', { hasText: /Start the selected pairing here|Take the table out of service/ }).first()).toBeVisible();
    await page.locator(proposal).first().click({ button: 'right' });
    await expect(page.locator('.context-menu')).toHaveCount(1);
});

test('la grille est un seul arrêt de Tab ; les flèches vont de case en case', async ({ page }) => {
    await openDirection(page);
    const cells = page.locator('.grid [role="gridcell"].cell');
    expect(await cells.count()).toBeGreaterThan(1);
    expect(await page.locator('.grid .cell[tabindex="0"]').count()).toBe(1);
    await cells.first().focus();
    await page.keyboard.press('ArrowRight');
    await expect(cells.nth(1)).toBeFocused();
    expect(await page.locator('.grid .cell[tabindex="0"]').count()).toBe(1);
});
