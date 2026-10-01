/**
 * direction-drag.spec.js
 *
 * Le glisser-déposer d'une case de la grille des tables, au pointeur : déplacer sur une case
 * libre, échanger sur une case occupée (avec confirmation), annuler avec Échap, et le refus du
 * service pour une table hors service.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installDirectionEngine } from './helpers/directionEngine.js';

const cell = (n) => `.grid [data-testid="direction-table-${n}"]`;

async function openHall(page, opts = { running: 2 }) {
    await installWailsMock(page, openLibraryMock());
    await installDirectionEngine(page, opts);
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await expect(page.locator('#tournamentPanel')).toBeVisible();
    await page.locator('#tournamentPanel tbody tr').first().click();
    await page.locator('#tournamentPanel .direction-btn').click();
    await expect(page.locator('.direction-view')).toBeVisible();
    await page.locator('[data-testid="direction-tab-direction"]').click();
    await expect(page.locator(cell(1))).toBeVisible();
}

async function drag(page, from, to, { release = true } = {}) {
    const a = await page.locator(cell(from)).boundingBox();
    const b = await page.locator(cell(to)).boundingBox();
    await page.mouse.move(a.x + a.width / 2, a.y + a.height / 2);
    await page.mouse.down();
    await page.mouse.move(a.x + a.width / 2 + 20, a.y + a.height / 2 + 20, { steps: 3 });
    await page.mouse.move(b.x + b.width / 2, b.y + b.height / 2, { steps: 5 });
    if (release) await page.mouse.up();
}

test('glisser une case occupée sur une case libre déplace le match', async ({ page }) => {
    await openHall(page);
    await expect(page.locator(cell(4))).toHaveClass(/idle/);
    await drag(page, 1, 4);
    await expect(page.locator(cell(4))).toHaveClass(/busy/);
    await expect(page.locator(cell(1))).toHaveClass(/idle/);
    expect(await page.evaluate(() => window.__moves.length)).toBe(1);
    // Le clic qui suit le geste n'ouvre pas la fiche.
    await expect(page.locator('.result-card')).toHaveCount(0);
});

test('le glisser montre le fantôme et la case visée', async ({ page }) => {
    await openHall(page);
    await drag(page, 1, 3, { release: false });
    await expect(page.locator('[data-testid="direction-drag-ghost"]')).toBeVisible();
    await expect(page.locator(`${cell(3)}.drop-over`)).toHaveCount(1);
    await page.mouse.up();
    await expect(page.locator('[data-testid="direction-drag-ghost"]')).toHaveCount(0);
});

test('Échap annule le glisser : rien ne bouge', async ({ page }) => {
    await openHall(page);
    await drag(page, 1, 4, { release: false });
    await page.keyboard.press('Escape');
    await expect(page.locator('[data-testid="direction-drag-ghost"]')).toHaveCount(0);
    await page.mouse.up();
    await expect(page.locator(cell(1))).toHaveClass(/busy/);
    await expect(page.locator(cell(4))).toHaveClass(/idle/);
    expect(await page.evaluate(() => window.__moves.length)).toBe(0);
});

test('glisser sur une case occupée demande confirmation, puis échange', async ({ page }) => {
    await openHall(page);
    const one = await page.locator(cell(1)).innerText();
    const two = await page.locator(cell(2)).innerText();
    await drag(page, 1, 2);
    const dialog = page.locator('[aria-modal="true"]');
    await expect(dialog).toContainText('1');
    await expect(dialog).toContainText('2');
    await dialog.getByRole('button', { name: 'Swap' }).click();
    await expect(page.locator(cell(1))).toContainText(two.split('\n').find((l) => l.includes('–')) || '');
    await expect(page.locator(cell(2))).toContainText(one.split('\n').find((l) => l.includes('–')) || '');
    expect(await page.evaluate(() => window.__moves.length)).toBe(1);
});

test("refuser l'échange ne change rien", async ({ page }) => {
    await openHall(page);
    await drag(page, 1, 2);
    await page
        .locator('[aria-modal="true"]')
        .getByRole('button', { name: /cancel/i })
        .click();
    expect(await page.evaluate(() => window.__moves.length)).toBe(0);
});

test('une table hors service est refusée par le service, et le dit', async ({ page }) => {
    await openHall(page, { running: 2, unavailable: [4] });
    await drag(page, 1, 4);
    await expect(page.locator(cell(1))).toHaveClass(/busy/);
    await expect(page.locator('.status-bar, [data-testid="status-bar"]').first()).toContainText(/out of service/);
});
