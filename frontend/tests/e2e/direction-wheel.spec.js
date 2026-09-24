/**
 * direction-wheel.spec.js — la molette et Tab au-dessus de la page Direction (#434, #435).
 *
 * La page Direction remplace le plateau dans la zone principale (ADR-0047). La molette y
 * changeait la position de la bibliothèque cachée derrière — 757 → 753 sur la base de la
 * simulation 2026-09 — sans faire défiler la page ; et Tab, du score A au score B, ouvrait
 * l'onglet Recherche. Sur la page Direction, ce sont une page et ses champs.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installDirectionEngine } from './helpers/directionEngine.js';

test.use({ viewport: { width: 1024, height: 768 } });

async function openDirection(page) {
    // Le dock à sa hauteur par défaut : sans elle, le mock rend null et le dock prend toute
    // la hauteur de son contenu.
    await installWailsMock(page, openLibraryMock({ config: { GetPanelHeight: 250, GetPanelPosition: 'bottom' } }));
    await installDirectionEngine(page);
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await page.locator('#tournamentPanel tbody tr').first().click();
    await page.locator('#tournamentPanel .direction-btn').click();
    await expect(page.locator('.direction-view')).toBeVisible();
}

test('la molette fait défiler la page Direction et ne change pas de position', async ({ page }) => {
    await openDirection(page);
    await page.locator('[data-testid="direction-tab-settings"]').click();
    const body = page.locator('.direction-view .body');
    const overflow = await body.evaluate((e) => e.scrollHeight - e.clientHeight);
    expect(overflow, 'les Réglages débordent à 1024×768, dock ouvert').toBeGreaterThan(100);
    const status = page.locator('.status-bar');
    const before = await status.textContent();

    const box = await body.boundingBox();
    await page.mouse.move(box.x + 40, box.y + box.height / 2);
    for (let i = 0; i < 4; i++) await page.mouse.wheel(0, 100);

    await expect.poll(() => body.evaluate((e) => e.scrollTop)).toBeGreaterThan(0);
    expect(await status.textContent()).toBe(before);
});

test('Tab passe au champ suivant dans la page Direction', async ({ page }) => {
    await openDirection(page);
    await page.locator('[data-testid="direction-tab-players"]').click();
    const name = page.locator('.players .entry input').first();
    await name.click();
    await page.keyboard.press('Tab');
    await expect(page.locator('.direction-view')).toBeVisible();
    await expect(page.locator('.players .entry input').nth(1)).toBeFocused();
});
