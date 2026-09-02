/**
 * search-escape-leaves-field.spec.js — #201 (D.1)
 *
 * Dans le panneau Recherche, un champ qui a le focus garde les touches nues
 * pour lui (j/k, flèches, Tab) — mais Escape doit remonter au dispatcher
 * global, qui rend le focus au plateau : c'est la seule sortie clavier du
 * formulaire. Le panneau stoppait toutes les touches, Escape compris, et
 * l'utilisateur restait prisonnier du champ.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';

const statusBar = (page) => page.getByTestId('status-bar');
const tab = (page, id) => page.getByTestId(`tab-${id}`);

test.beforeEach(async ({ page }) => {
    await installWailsMock(page, openLibraryMock());
    await page.goto('/');
    await expect(statusBar(page)).toContainText('3 / 3');
});

test('Escape quitte le champ de recherche et rend les raccourcis nus', async ({ page }) => {
    await page.keyboard.press('Control+f');
    await expect(tab(page, 'search')).toHaveClass(/active/);

    const pipDiff = page.locator('.filter-item', { hasText: 'Pipcount Difference' });
    await pipDiff.getByRole('checkbox').check();
    const field = pipDiff.getByRole('spinbutton').first();
    await field.fill('10');
    await expect(field).toBeFocused();

    // Tant que le champ a le focus, Espace n'ouvre pas la ligne de commande.
    await page.keyboard.press('Space');
    await expect(page.getByPlaceholder('Type command...')).toBeHidden();
    await expect(field).toBeFocused();

    // Escape remonte au dispatcher : le champ est quitté…
    await page.keyboard.press('Escape');
    await expect(field).not.toBeFocused();
    await expect(field).toHaveValue('10');

    // …et les touches nues sont de nouveau à l'application.
    await page.keyboard.press('Space');
    await expect(page.getByPlaceholder('Type command...')).toBeVisible();
});
