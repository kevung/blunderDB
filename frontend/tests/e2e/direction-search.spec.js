/**
 * direction-search.spec.js — la recherche rapide de la Direction.
 *
 * `/` sur la page ouvre la palette sur la salle seule : on tape un nom ou un numéro de table,
 * Entrée amène à l'objet — la fiche de la table du joueur, la table elle-même, la liste d'un
 * joueur libre. `/` reste une barre oblique dans un champ de saisie.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installDirectionEngine, S2_HALL } from './helpers/directionEngine.js';

const palette = '[data-testid="command-palette"]';

async function openDirection(page) {
    await installWailsMock(page, openLibraryMock());
    await installDirectionEngine(page, S2_HALL);
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await page.locator('#tournamentPanel tbody tr').first().dblclick();
    await page.locator('#tournamentPanel .direction-btn').click();
    await page.locator('[data-testid="direction-tab-direction"]').click();
    await expect(page.locator('[data-testid="direction-table-1"]')).toBeVisible();
}

test('/ ouvre la recherche, un nom mène à la fiche de sa table', async ({ page }) => {
    await openDirection(page);
    await page.keyboard.press('/');
    await expect(page.locator(palette)).toBeVisible();
    // La portée est la salle : aucune commande dans la liste.
    await expect(page.locator(`${palette} .kind-command`)).toHaveCount(0);

    await page.locator(`${palette} input`).fill('Joueur 03');
    await expect(page.locator(`${palette} [role="option"]`).first()).toContainText('Joueur 03');
    await page.keyboard.press('Enter');

    await expect(page.locator(palette)).toHaveCount(0);
    await expect(page.locator('[data-testid="direction-result-card"]')).toBeVisible();
    await expect(page.locator('[data-testid="direction-result-card"]')).toContainText('Joueur 03');
});

test('un numéro de table mène à cette table', async ({ page }) => {
    await openDirection(page);
    await page.keyboard.press('/');
    await page.locator(`${palette} input`).fill('t5');
    await expect(page.locator(`${palette} [role="option"]`).first()).toContainText(/5/);
    await page.keyboard.press('Enter');
    await expect(page.locator('[data-testid="direction-result-card"]')).toBeVisible();
    await expect(page.locator('[data-testid="direction-table-5"]')).toBeFocused();
});

test('un joueur libre mène à la liste des joueurs, filtrée sur son nom', async ({ page }) => {
    await openDirection(page);
    await page.keyboard.press('/');
    await page.locator(`${palette} input`).fill('Joueur 70');
    await page.keyboard.press('Enter');
    await expect(page.locator('[data-testid="direction-player-filter"]')).toHaveValue('Joueur 70');
    await expect(page.locator('[data-testid="direction-player-filter"]')).toBeVisible();
});

test('Échap ferme la recherche sans rien ouvrir', async ({ page }) => {
    await openDirection(page);
    await page.keyboard.press('/');
    await expect(page.locator(palette)).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(page.locator(palette)).toHaveCount(0);
    await expect(page.locator('[data-testid="direction-result-card"]')).toHaveCount(0);
});

test('dans un champ de saisie, / reste une barre oblique', async ({ page }) => {
    await openDirection(page);
    await page.locator('[data-testid="direction-tab-players"]').click();
    const filter = page.locator('[data-testid="direction-player-filter"]');
    await filter.focus();
    await page.keyboard.type('a/b');
    await expect(filter).toHaveValue('a/b');
    await expect(page.locator(palette)).toHaveCount(0);
});
