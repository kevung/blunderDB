/**
 * direction-queue-keys.spec.js — #415
 *
 * La file des propositions au clavier, dans l'application réelle : J choisit la proposition
 * suivante, ENTRÉE la confirme. Le panneau Tournois est visible en même temps que la page
 * Direction ; son écouteur `document` arrêtait toute touche nue, et la file ne recevait rien.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installDirectionEngine } from './helpers/directionEngine.js';

const proposal = '.proposals .queue li';

/** Une Direction ouverte, la file des propositions devant soi. */
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

test('J choisit la proposition suivante, ENTRÉE la confirme', async ({ page }) => {
    await openDirection(page);
    const before = await page.locator(proposal).count();
    expect(before).toBeGreaterThan(1);
    const second = await page.locator(`${proposal} .what`).nth(1).textContent();
    await expect(page.locator(proposal).first()).toHaveClass(/selected/);

    await page.keyboard.press('j');
    await expect(page.locator(proposal).nth(1)).toHaveClass(/selected/);

    await page.keyboard.press('Enter');
    await expect(page.locator(proposal)).toHaveCount(before - 1);
    // C'est bien la proposition choisie qui est partie, pas la première.
    await expect(page.locator(`${proposal} .what`, { hasText: second ?? '' })).toHaveCount(0);
});

test('ouvrir la direction depuis le panneau lui donne le clavier', async ({ page }) => {
    await installWailsMock(page, openLibraryMock());
    await installDirectionEngine(page);
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await page.locator('#tournamentPanel tbody tr').first().click();
    // Le bouton est dans le panneau : si la page ne prenait pas le focus en s'ouvrant, il y
    // resterait, et J / K iraient au panneau. Une direction en cours s'ouvre directement sur
    // la file (DirectionView), sans autre clic pour déplacer le focus.
    await page.locator('#tournamentPanel .direction-btn').click();
    await expect(page.locator('.direction-view')).toBeVisible();

    await expect.poll(() => page.evaluate(() => !!document.activeElement?.closest('.direction-view'))).toBe(true);
    // Et la prise de focus différée du panneau ne la reprend pas.
    await page.waitForTimeout(300);
    expect(await page.evaluate(() => !!document.activeElement?.closest('#tournamentPanel'))).toBe(false);
});
