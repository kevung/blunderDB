/**
 * help-search.spec.js — la recherche dans l'aide
 *
 * « ? » ouvre l'aide, « / » met le focus dans le champ de recherche sans la fermer, la saisie
 * compte les occurrences et Entrée passe à la suivante.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';

test('« / » cherche dans l’aide, Entrée passe à l’occurrence suivante', async ({ page }) => {
    await installWailsMock(page, openLibraryMock());
    await page.goto('/');
    await expect(page.getByTestId('status-bar')).toContainText('3 / 3');

    await page.keyboard.press('?');
    const search = page.getByTestId('help-search');
    await expect(search).toBeVisible();
    await expect(page.getByTestId('help-loading')).toHaveCount(0);

    await page.keyboard.press('/');
    await expect(search).toBeFocused();
    await search.fill('position');
    await expect(page.getByTestId('help-search-count')).toHaveText(/^1 \/ \d+$/);
    await search.press('Enter');
    await expect(page.getByTestId('help-search-count')).toHaveText(/^2 \/ \d+$/);
    await expect(search).toBeVisible();
});
