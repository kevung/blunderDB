/**
 * direction-absence.spec.js — D7.1 : absenter un joueur pour une ronde ou jusqu'à une heure,
 * sans le retirer.
 *
 * Avant ce geste, le seul chemin trouvé était « Retirer » puis « Corriger → Enregistrer » (une
 * réinscription par effet de bord) : l'absent y comptait « forfait » au classement pendant son
 * absence. Ce spec tient le budget de gestes de l'acceptation : absenter en au plus trois clics,
 * revenir en un seul, le classement inchangé pendant l'absence.
 */
import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installDirectionEngine } from './helpers/directionEngine.js';

/** Une Direction ouverte sur l'onglet Joueurs. */
async function openPlayers(page) {
    await installWailsMock(page, openLibraryMock());
    await installDirectionEngine(page);
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await expect(page.locator('#tournamentPanel')).toBeVisible();
    await page.locator('#tournamentPanel tbody tr').first().click();
    await page.locator('#tournamentPanel .direction-btn').click();
    await expect(page.locator('.direction-view')).toBeVisible();
    await page.locator('[data-testid="direction-tab-players"]').click();
    await expect(page.locator('[data-testid="direction-player-ha"]')).toBeVisible();
}

test('absenter en trois clics au plus, revenir en un', async ({ page }) => {
    await openPlayers(page);
    const row = page.locator('[data-testid="direction-player-ha"]');
    await expect(row).toContainText('free');

    // 1er clic : ouvre le petit formulaire d'absence, sous la ligne.
    await row.locator('[data-testid="direction-player-absent"]').click();
    const form = page.locator('[data-testid="direction-player-absent-form"]');
    await expect(form).toBeVisible();

    // 2e clic : valide l'heure par défaut (déjà pré-remplie, rien à taper).
    await form.locator('[data-testid="direction-player-absent-confirm-time"]').click();

    await expect(row).toContainText('absent');
    await expect(row.locator('[data-testid="direction-player-withdraw-now"]')).toBeVisible(); // pas retiré

    // Revenir : un clic.
    await row.locator('[data-testid="direction-player-return"]').click();
    await expect(row).toContainText('free');
});

test('le classement ne bouge pas pendant une absence', async ({ page }) => {
    await openPlayers(page);
    await page.locator('[data-testid="direction-tab-standings"]').click();
    const before = await page.locator('.standings').textContent();

    await page.locator('[data-testid="direction-tab-players"]').click();
    const row = page.locator('[data-testid="direction-player-ha"]');
    await row.locator('[data-testid="direction-player-absent"]').click();
    await page.locator('[data-testid="direction-player-absent-form"] [data-testid="direction-player-absent-confirm-time"]').click();

    await page.locator('[data-testid="direction-tab-standings"]').click();
    await expect(page.locator('.standings')).toHaveText(before);
});

test("absenter jusqu'à une ronde, dans un suisse par rondes", async ({ page }) => {
    await installWailsMock(page, openLibraryMock());
    await installDirectionEngine(page, { phases: [{ kind: 'swiss_lives', length: 7, lives: 2, mode: 'rounds', target: 0 }] });
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await page.locator('#tournamentPanel tbody tr').first().click();
    await page.locator('#tournamentPanel .direction-btn').click();
    await page.locator('[data-testid="direction-tab-players"]').click();

    const row = page.locator('[data-testid="direction-player-ha"]');
    await row.locator('[data-testid="direction-player-absent"]').click();
    const form = page.locator('[data-testid="direction-player-absent-form"]');
    await form.locator('[data-testid="direction-player-absent-round"]').fill('3');
    await form.locator('[data-testid="direction-player-absent-confirm-round"]').click();

    await expect(row).toContainText('absent');
    await expect(row).toContainText('3');
});
