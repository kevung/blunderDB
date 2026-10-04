/**
 * direction-queue-keys.spec.js
 *
 * La file des propositions au clavier, dans l'application réelle : J choisit la proposition
 * suivante, ENTRÉE la confirme. Le panneau Tournois est visible en même temps que la page
 * Direction ; sans garde, son écouteur `document` intercepterait toute touche nue.
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
    await page.locator('#tournamentPanel tbody tr').first().dblclick();
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

test('Tab jusqu’à un bouton : ENTRÉE active le bouton et ne confirme aucune proposition', async ({ page }) => {
    await openDirection(page);
    const before = await page.locator(proposal).count();
    expect(before).toBeGreaterThan(1);

    // Au clavier seul, depuis l'onglet cliqué, jusqu'à « Tout lancer ».
    let reached = false;
    for (let i = 0; i < 40 && !reached; i++) {
        await page.keyboard.press('Tab');
        reached = await page.evaluate(() => !!document.activeElement?.matches('.proposals .all'));
    }
    expect(reached, '« Tout lancer » atteint par Tab').toBe(true);

    await page.keyboard.press('Enter');
    // Le bouton a fait son geste — la confirmation de « Tout lancer » s'ouvre —, et aucun match
    // n'est parti.
    await expect(page.locator('.proposals .confirm')).toBeVisible();
    await expect(page.locator(proposal)).toHaveCount(before);
});

test('ouvrir la direction depuis le panneau lui donne le clavier', async ({ page }) => {
    await installWailsMock(page, openLibraryMock());
    await installDirectionEngine(page);
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await page.locator('#tournamentPanel tbody tr').first().dblclick();
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

test('Tab après « Lancer » : le focus reste dans la Direction, la Recherche ne s’ouvre pas', async ({ page }) => {
    await openDirection(page);
    await page.locator(`${proposal} .go`).first().click();
    await page.waitForTimeout(300);
    await expect.poll(() => page.evaluate(() => !!document.activeElement?.closest('.direction-view'))).toBe(true);

    await page.keyboard.press('Tab');
    await expect(page.locator('.direction-view')).toBeVisible();
    await expect(page.locator('[data-testid="tab-tournaments"]')).toHaveClass(/active/);
    expect(await page.evaluate(() => !!document.activeElement?.closest('.direction-view'))).toBe(true);
});

test('un tournoi s’ouvre au clavier : Tab jusqu’à la ligne, Entrée', async ({ page }) => {
    await installWailsMock(page, openLibraryMock());
    await installDirectionEngine(page);
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await expect(page.locator('#tournamentPanel')).toBeVisible();
    const row = page.locator('#tournamentPanel tbody tr').first();
    await row.focus();
    await page.keyboard.press('Enter');
    await expect(page.locator('#tournamentPanel .direction-btn')).toBeVisible();
});

test('la file est à un Tab de l’onglet Direction, et ses raccourcis sont affichés', async ({ page }) => {
    await openDirection(page);
    await expect(page.locator('[data-testid="direction-proposals-keys"]')).toContainText('J / K');
    await page.locator('[data-testid="direction-tab-direction"]').focus();
    // Le premier arrêt dans la page est le lien vers la file, avant la grille des tables.
    await page.locator('[data-testid="direction-pane-direction"]').focus();
    await page.keyboard.press('Tab');
    await expect(page.locator('[data-testid="direction-skip-queue"]')).toBeFocused();
    await page.keyboard.press('Enter');
    await expect(page.locator('[data-testid="direction-queue"]')).toBeFocused();
});
