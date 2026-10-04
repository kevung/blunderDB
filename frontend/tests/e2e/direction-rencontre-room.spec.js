/**
 * direction-rencontre-room.spec.js — une salle partagée par deux épreuves (ADR-0056).
 *
 * L'état S3 du samedi, 20 h : le speed s'ouvre dans la salle de 14 tables où le principal joue
 * encore sur les tables 1 à 7. La file du speed ne propose que les tables 8 à 14 — aucune
 * collision —, la grille dit qui occupe les autres, et une table hors service se déclare une
 * seule fois, depuis la Rencontre.
 */
import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installDirectionEngine, S3_SATURDAY_20H } from './helpers/directionEngine.js';

test('S3 20 h : le speed ne propose aucune table du principal', async ({ page }) => {
    await installWailsMock(page, openLibraryMock());
    await installDirectionEngine(page, S3_SATURDAY_20H);
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await page.locator('#tournamentPanel tbody tr').first().dblclick();
    await page.locator('#tournamentPanel .direction-btn').click();
    await page.locator('[data-testid="direction-tab-direction"]').click();

    // Les sept tables du principal sont dites occupées, par qui.
    for (let n = 1; n <= 7; n++) {
        await expect(page.locator(`[data-testid="direction-table-${n}"]`)).toContainText(/Principal/);
    }
    const queue = page.locator('.proposals .queue li:not(.empty)');
    await expect(queue).toHaveCount(16);
    await page.locator('[data-testid="direction-proposals-all"]').click();
    await expect(page.locator('[data-testid="direction-proposals-confirm"] ul li')).toHaveCount(7);
    await page.locator('[data-testid="direction-proposals-confirm-all"]').click();

    // Zéro collision : les tables 1 à 7 restent au principal, les 7 matchs lancés sont ailleurs.
    for (let n = 1; n <= 7; n++) {
        await expect(page.locator(`[data-testid="direction-table-${n}"]`)).toContainText(/Principal/);
    }
    for (let n = 8; n <= 14; n++) {
        await expect(page.locator(`[data-testid="direction-table-${n}"]`)).toHaveClass(/busy/);
    }
    await expect(queue).toHaveCount(9);

    // Une table cassée : un seul geste, pour toute la salle.
    await page.locator('[data-testid="direction-tab-settings"]').click();
    // Le panneau d'une Rencontre déjà rattachée est replié : on le déplie avant de cocher.
    await page.locator('[data-testid="rencontre-summary"]').click();
    await expect(page.locator('[data-testid="rencontre-current"]')).toContainText('Festival de Lyon');
    await page.locator('[data-testid="rencontre-out-12"]').check();
    await expect(page.locator('[data-testid="rencontre-out-12"]')).toBeChecked();
    expect(await page.evaluate(() => window.__roomGestures)).toEqual([{ table: 12, out: true }]);
});
