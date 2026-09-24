/**
 * direction-upcoming-sheet.spec.js — imprimer la feuille d'une ronde proposée, datée, avant de
 * la lancer (#451, D7.2).
 *
 * S4, le vendredi (rapport/S4.md O6) : les rondes 1 et 2 sont jouées, la ronde 3 est dans la
 * file. Le sélecteur de la feuille ne proposait que les rondes lancées, et les lancer pour les
 * imprimer les aurait horodatées au vendredi. Budget : ≤ 3 gestes hors saisie de la date, et
 * rien de lancé. Le faux moteur (helpers/directionEngine.js) garde ce qui a été demandé.
 */
import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { countGestures } from './helpers/gestureCount.js';
import { installDirectionEngine, S4_FRIDAY } from './helpers/directionEngine.js';

test('la feuille de la ronde 3 s’imprime le vendredi, datée, sans rien lancer', async ({ page }) => {
    await installWailsMock(page, openLibraryMock());
    await installDirectionEngine(page, S4_FRIDAY);
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await page.locator('#tournamentPanel tbody tr').first().click();
    await page.locator('#tournamentPanel .direction-btn').click();
    await page.locator('[data-testid="direction-tab-direction"]').click();
    const queued = await page.locator('.proposals .queue li').count();
    expect(queued).toBeGreaterThan(0);

    const counted = await countGestures(page, async (g) => {
        await g.click(page.locator('[data-testid="direction-sheet-upcoming"]'));
        // La saisie de la date n'est pas un geste de plus : c'est ce que le directeur annonce.
        await page.locator('[data-testid="direction-sheet-announced"]').fill('lundi 12/10, 20 h');
        await g.click(page.locator('[data-testid="direction-sheet-upcoming-print"]'));
        await expect.poll(() => page.evaluate(() => window.__announcedSheets.length)).toBe(1);
    });
    expect(counted.total, `feuille d’une ronde à venir : ${counted.total} gestes hors saisie`).toBeLessThanOrEqual(3);

    const [asked] = await page.evaluate(() => window.__announcedSheets);
    expect(asked.announced).toBe('lundi 12/10, 20 h');
    // Rien de lancé : la file est intacte, aucune table n'est occupée.
    await expect(page.locator('.proposals .queue li')).toHaveCount(queued);
    expect(await page.evaluate(() => window.__announcedSheets[0].proposals)).toBe(queued);
});
