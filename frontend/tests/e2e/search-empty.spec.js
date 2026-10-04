/**
 * search-empty.spec.js — une recherche sans résultat se voit dans le panneau
 *
 * Le backend renvoie aucun id : le panneau Recherche affiche l'état « aucune
 * position » avec son bouton d'effacement, au lieu de ne changer que la barre
 * d'état. Le sous-onglet des critères ne porte plus le nom du bouton Search.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';

const statusBar = (page) => page.getByTestId('status-bar');

test('recherche sans résultat : état vide dans le panneau, bouton primaire, sous-onglet Criteria', async ({ page }) => {
    await installWailsMock(page, openLibraryMock({ database: { LoadPositionIDsByFilters: [] } }));
    await page.goto('/');
    await expect(statusBar(page)).toContainText('3 / 3');
    await page.keyboard.press('Control+f');

    await expect(page.getByRole('button', { name: 'Criteria', exact: true })).toBeVisible();
    await expect(page.locator('.top-action-bar').getByRole('button', { name: 'Search', exact: true })).toBeVisible();
    await expect(page.locator('.no-results')).toHaveCount(0);

    const pipDiff = page.locator('.filter-item', { hasText: 'Pipcount Difference' });
    await pipDiff.getByRole('checkbox').check();
    await pipDiff.getByRole('spinbutton').first().fill('10');
    await page.locator('.top-action-bar').getByRole('button', { name: 'Search', exact: true }).click();

    const empty = page.locator('.no-results');
    await expect(empty).toContainText('No position matches');
    await empty.getByRole('button', { name: 'Clear filters' }).click();
    await expect(page.locator('.active-count')).toHaveText('0 active');
    await expect(page.locator('.no-results')).toHaveCount(0);

    // The banner reports a search; listing the whole library ends that search too.
    await pipDiff.getByRole('checkbox').check();
    await pipDiff.getByRole('spinbutton').first().fill('10');
    await page.locator('.top-action-bar').getByRole('button', { name: 'Search', exact: true }).click();
    await expect(page.locator('.no-results')).toBeVisible();
    await page.keyboard.press('Control+r');
    await expect(statusBar(page)).toContainText('3 / 3');
    await expect(page.locator('.no-results')).toHaveCount(0);
});
