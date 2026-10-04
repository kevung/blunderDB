/**
 * direction-phase-settings.spec.js — les réglages de phase.
 *
 * Ajouter une consolante au tableau doit avoir un chemin à l'écran. Budget : ≤ 4 gestes avant
 * le tirage (Réglages, la case, Enregistrer, Confirmer). Après le tirage le moteur refuse la
 * consolante : l'aperçu affiche son refus, avec sa raison, et n'offre pas de confirmer.
 */
import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { countGestures } from './helpers/gestureCount.js';
import { installDirectionEngine } from './helpers/directionEngine.js';

const PHASES = [
    { kind: 'swiss_lives', length: 7, lives: 2, mode: 'continuous', target: 16 },
    { kind: 'bracket', length: 9 }
];

async function openDirection(page, opts) {
    await installWailsMock(page, openLibraryMock());
    await installDirectionEngine(page, { phases: PHASES, tables: 8, running: 2, ...opts });
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await page.locator('#tournamentPanel tbody tr').first().dblclick();
    await page.locator('#tournamentPanel .direction-btn').click();
    await expect(page.locator('.direction-view')).toBeVisible();
    await page.locator('[data-testid="direction-tab-direction"]').click();
}

test('ajouter une consolante avant le tirage tient en quatre gestes', async ({ page }) => {
    await openDirection(page);
    const counted = await countGestures(page, async (g) => {
        await g.click(page.locator('[data-testid="direction-tab-settings"]'));
        await g.click(page.locator('[data-testid="direction-settings-consolation-2"]'));
        await g.click(page.locator('[data-testid="direction-settings-apply"]'));
        await expect(page.locator('[data-testid="direction-settings-changes"]')).toBeVisible();
        await g.click(page.locator('[data-testid="direction-settings-confirm"]'));
        await expect(page.locator('[data-testid="direction-settings-changes"]')).toHaveCount(0);
    });
    expect(counted.total, `ajouter une consolante : ${counted.total} gestes`).toBeLessThanOrEqual(4);
    // Le barème de la consolante est à remplir pour qu'elle ait son classement : la vue le dit.
    await expect(page.locator('[data-testid="direction-settings-conso-scale-hint"]')).toBeVisible();
});

test("après le tirage, l'aperçu affiche le refus du moteur et sa raison", async ({ page }) => {
    await openDirection(page, { locks: [{ phase: 2, kind: 'bracket', locked: true, reason: 'drawn' }] });
    await page.locator('[data-testid="direction-tab-settings"]').click();
    await page.locator('[data-testid="direction-settings-consolation-2"]').click();
    await page.locator('[data-testid="direction-settings-apply"]').click();
    const refused = page.locator('[data-testid="direction-settings-changes"] .refused');
    await expect(refused).toHaveCount(1);
    await expect(refused).toContainText(/draw|tirage/i);
    await expect(page.locator('[data-testid="direction-settings-confirm"]')).toHaveCount(0);
});
