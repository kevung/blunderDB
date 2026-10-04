/**
 * direction-cycle.spec.js — le cycle de vie de la Direction : un double geste sur « Diriger »
 * l'ouvre sans la refermer, ; un rechargement la rouvre sur le même onglet sans un clic (la session Go est simulée par sessionStorage).
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installDirectionEngine, S2_HALL } from './helpers/directionEngine.js';

async function selectTournament(page) {
    await installWailsMock(page, openLibraryMock());
    await installDirectionEngine(page, S2_HALL);
    await page.addInitScript(() => {
        const db = window.go.database.Database;
        db.SaveSessionState = async (state) => sessionStorage.setItem('e2e-session', JSON.stringify(state));
        db.LoadSessionState = async () => JSON.parse(sessionStorage.getItem('e2e-session') || 'null');
    });
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await page.locator('#tournamentPanel tbody tr').first().dblclick();
}

test('un double clic sur « Diriger » ouvre la Direction et la laisse ouverte', async ({ page }) => {
    await selectTournament(page);
    await page.locator('#tournamentPanel .direction-btn').dblclick();
    await expect(page.locator('.direction-view')).toBeVisible();
    await page.waitForTimeout(300);
    await expect(page.locator('.direction-view')).toBeVisible();
});

test('un rechargement rouvre la Direction sur son onglet, sans un clic', async ({ page }) => {
    await selectTournament(page);
    await page.locator('#tournamentPanel .direction-btn').click();
    await page.locator('[data-testid="direction-tab-standings"]').click();
    await expect(page.locator('[data-testid="direction-tab-standings"]')).toHaveClass(/active/);
    await page.waitForFunction(() => (sessionStorage.getItem('e2e-session') || '').includes('standings'));
    await page.reload();
    await expect(page.locator('[data-testid="direction-tab-standings"]')).toHaveClass(/active/);
});
