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

// À reprendre : sous ce mock, aucune touche de navigation (même ←) ne déplace le compteur ; la
// couverture des touches est celle de keyboardService.pageKeys.test.js et positionNavigation.page.test.js.
test.fixme('End, PageUp, Home, PageDown parcourent la liste', async ({ page }) => {
    // Le focus part dans le panneau Match, qui garde ses touches : on le rend à la page.
    await page.getByTestId('status-bar').click();
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
