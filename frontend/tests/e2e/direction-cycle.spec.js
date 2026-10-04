/**
 * direction-cycle.spec.js — le cycle de vie de la Direction : un double geste sur « Diriger »
 * l'ouvre sans la refermer, (le rechargement est couvert par directionView.tabMemory.test.js : la maquette Wails ne survit pas à un reload).
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installDirectionEngine, S2_HALL } from './helpers/directionEngine.js';

async function selectTournament(page) {
    await installWailsMock(page, openLibraryMock());
    await installDirectionEngine(page, S2_HALL);
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
