/**
 * scale-navigation.spec.js — une bibliothèque de cent mille positions
 *
 * Home / End vont aux extrémités, PageUp / PageDown sautent une page de cent
 * positions, `:N%` va à une proportion de la liste ; l'ouverture reste dans un
 * budget de temps.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { scaleLibraryMock } from './helpers/fixtures.js';

const N = 100000;
const statusBar = (page) => page.getByTestId('status-bar');

test.beforeEach(async ({ page }) => {
    await installWailsMock(page, scaleLibraryMock({ ids: N }));
    const t0 = Date.now();
    await page.goto('/');
    await expect(statusBar(page)).toContainText(`${N} / ${N}`);
    // Budget d'ouverture, large : il garde l'ordre de grandeur, pas la milliseconde.
    expect(Date.now() - t0).toBeLessThan(6000);
});

test('End, PageUp, Home, PageDown parcourent la liste', async ({ page }) => {
    // La base s'ouvre sur l'onglet Match, qui garde toutes les touches tant qu'il est affiché :
    // Échap le ferme et rend les touches au plateau.
    await page.keyboard.press('Escape');
    await page.keyboard.press('Home');
    await expect(statusBar(page)).toContainText(`1 / ${N}`);
    await page.keyboard.press('PageDown');
    await expect(statusBar(page)).toContainText(`101 / ${N}`);
    await page.keyboard.press('End');
    await expect(statusBar(page)).toContainText(`${N} / ${N}`);
    await page.keyboard.press('PageUp');
    await expect(statusBar(page)).toContainText(`${N - 100} / ${N}`);
});

test(':N% va à la proportion demandée de la liste', async ({ page }) => {
    await page.keyboard.press('Space');
    await page.keyboard.type('50%');
    await page.keyboard.press('Enter');
    await expect(statusBar(page)).toContainText(`${N / 2 + 0} / ${N}`.replace(/^\d+/, String(Math.round(0.5 * (N - 1)) + 1)));
});

test.describe('pas de page à 10 % de la liste', () => {
    test('PageDown saute un dixième de la liste', async ({ page }) => {
        await installWailsMock(page, scaleLibraryMock({ ids: N, extra: { config: { GetPageStep: '10%' } } }));
        await page.goto('/');
        await expect(statusBar(page)).toContainText(`${N} / ${N}`);
        await page.keyboard.press('Escape');
        await page.keyboard.press('Home');
        await expect(statusBar(page)).toContainText(`1 / ${N}`);
        await page.keyboard.press('PageDown');
        await expect(statusBar(page)).toContainText(`${N / 10 + 1} / ${N}`);
    });
});
