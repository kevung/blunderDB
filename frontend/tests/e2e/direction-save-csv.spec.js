/**
 * direction-save-csv.spec.js — enregistrer le classement et l'annuaire dans un fichier.
 *
 * Le classement s'enregistre par le dialogue natif, sous un nom proposé, et le fichier est le
 * CSV copié : les deux gestes (Copier, Enregistrer…) reçoivent le même texte. Le nom proposé
 * suit la langue de l'interface (anglaise ici, comme dans les autres specs). Le dialogue est
 * remplacé par le faux moteur (helpers/directionEngine.js), qui rend le chemin choisi et garde
 * ce qui a été écrit.
 */
import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { countGestures } from './helpers/gestureCount.js';
import { installDirectionEngine } from './helpers/directionEngine.js';

async function openDirection(page) {
    await installWailsMock(page, openLibraryMock());
    await installDirectionEngine(page);
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await page.locator('#tournamentPanel tbody tr').first().click();
    await page.locator('#tournamentPanel .direction-btn').click();
    await expect(page.locator('.direction-view')).toBeVisible();
}

const today = () => {
    const d = new Date();
    const p = (n) => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`;
};

test('le classement s’enregistre en un clic, identique au CSV copié', async ({ page }) => {
    await openDirection(page);
    await page.locator('[data-testid="direction-tab-standings"]').click();
    await page.locator('[data-testid="direction-standings-csv"]').click();
    await expect.poll(() => page.evaluate(() => window.__copied.length)).toBe(1);

    const counted = await countGestures(page, async (g) => {
        await g.click(page.locator('[data-testid="direction-standings-save"]'));
        await expect(page.getByTestId('status-bar-message')).toContainText('/home/nadia/');
    });
    expect(counted.total, `enregistrer le classement : ${counted.total} gestes`).toBeLessThanOrEqual(1);

    const [saved] = await page.evaluate(() => window.__savedCSV);
    const copied = await page.evaluate(() => window.__copied[0]);
    expect(saved.name).toBe(`Open-de-Lyon-standings-${today()}.csv`);
    expect(saved.body).toBe(copied);
});

test('l’annuaire s’enregistre, identique au CSV copié', async ({ page }) => {
    await openDirection(page);
    await page.locator('[data-testid="direction-tab-players"]').click();
    await page.locator('[data-testid="direction-directory-toggle"]').click();
    await page.locator('[data-testid="direction-directory-export"]').click();
    await expect.poll(() => page.evaluate(() => window.__copied.length)).toBe(1);
    await page.locator('[data-testid="direction-directory-save"]').click();
    await expect.poll(() => page.evaluate(() => window.__savedCSV.length)).toBe(1);

    const [saved] = await page.evaluate(() => window.__savedCSV);
    const copied = await page.evaluate(() => window.__copied[0]);
    expect(saved.name).toBe(`Open-de-Lyon-directory-${today()}.csv`);
    expect(saved.body).toBe(copied);
    expect(saved.body).toContain('Hugo Andrieu');
});
