/**
 * escape-closes-overlays.spec.js — #414
 *
 * Échap ferme ce qui est ouvert, dans l'application réelle, répartiteur global actif. Le menu
 * contextuel des onglets et la reprise de la direction écoutaient Échap par
 * `<svelte:window onkeydown>` ; montés après App.svelte, ils ne recevaient aucune touche — le
 * `stopPropagation()` du répartiteur suffisait à Svelte 5 pour ne pas les appeler.
 */

import { test, expect } from '@playwright/test';
import { dismissHomeScreen, installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installDirectionEngine } from './helpers/directionEngine.js';

/** Une Direction ouverte, la file des propositions devant soi (quatre inscrits : deux propositions). */
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
    await expect(page.locator('.proposals .queue li').first()).toBeVisible();
}

/** Une Direction ouverte, un résultat saisi : la dernière décision est là, corrigeable. */
async function openDirectionWithResult(page) {
    await openDirection(page);
    await page.locator('.proposals .all').click();
    await page.locator('.proposals .confirm .primary').click();
    await page.locator('.grid .cell.busy').first().click();
    await page.locator('.card .winner').first().click();
    await expect(page.locator('.last')).toBeVisible();
}

test('le menu contextuel des onglets se ferme au clavier', async ({ page }) => {
    await installWailsMock(page);
    await page.goto('/');
    await expect(page.locator('[data-testid="status-bar"]')).toBeVisible({ timeout: 8000 });
    await dismissHomeScreen(page);

    await page.locator('[data-testid="tab-stats"]').click({ button: 'right' });
    const menu = page.locator('.context-menu');
    await expect(menu).toBeVisible();
    await expect(menu.locator('.context-menu-item').first()).toBeFocused();

    await page.keyboard.press('Escape');
    await expect(menu).toHaveCount(0);
    // Fermer le menu n'a rien masqué : l'onglet est toujours là.
    await expect(page.locator('[data-testid="tab-stats"]')).toBeVisible();
});

test('la reprise de la direction s’ouvre par CTRL-Z et se ferme par Échap', async ({ page }) => {
    await openDirectionWithResult(page);
    const correction = page.locator('.correct');
    await expect(correction).toHaveCount(0);

    await page.keyboard.press('Control+z');
    await expect(correction).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(correction).toHaveCount(0);

    // Ouverte à la souris, même chose.
    await page.locator('.last button').first().click();
    await expect(correction).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(correction).toHaveCount(0);
});

test('la fiche de résultat se ferme par Échap', async ({ page }) => {
    await openDirectionWithResult(page);

    await page.locator('.grid .cell.busy').first().click();
    await expect(page.locator('.card')).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(page.locator('.card')).toHaveCount(0);
});
