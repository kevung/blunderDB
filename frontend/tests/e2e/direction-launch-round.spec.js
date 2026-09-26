/**
 * direction-launch-round.spec.js — lancer une ronde plus grande que la salle.
 *
 * L'état S4 du vendredi : 25 inscrits, 6 tables, la ronde 3 proposée (12 matchs). « Tout
 * lancer » lance les 6 matchs qui ont une table ; les 6 autres restent dans la file, marqués
 * « aucune table libre », au lieu de disparaître.
 */
import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installDirectionEngine, S4_FRIDAY } from './helpers/directionEngine.js';

test('« Tout lancer » laisse en file les appariements sans table', async ({ page }) => {
    await installWailsMock(page, openLibraryMock());
    await installDirectionEngine(page, S4_FRIDAY);
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await page.locator('#tournamentPanel tbody tr').first().click();
    await page.locator('#tournamentPanel .direction-btn').click();
    await page.locator('[data-testid="direction-tab-direction"]').click();

    const queue = page.locator('.proposals .queue li:not(.empty)');
    await expect(queue).toHaveCount(12);
    await page.locator('[data-testid="direction-proposals-all"]').click();
    // La confirmation ne compte que ce qui sera lancé.
    await expect(page.locator('[data-testid="direction-proposals-confirm"] ul li')).toHaveCount(6);
    await page.locator('[data-testid="direction-proposals-confirm-all"]').click();

    await expect(queue).toHaveCount(6);
    await expect(queue.filter({ hasText: /aucune table libre|no free table/i })).toHaveCount(6);
});
