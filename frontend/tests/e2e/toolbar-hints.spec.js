/**
 * toolbar-hints.spec.js — la barre d'outils nomme ses touches, la barre de vues son « + »
 *
 * Les infobulles passent par la table de noms de touches (utils/keyNames.js) ; la barre de
 * vues repliée dit « New view » à côté du « + » au lieu d'un signe orphelin.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';

test("infobulles de la barre d'outils et bouton « + New view »", async ({ page }) => {
    await installWailsMock(page, openLibraryMock());
    await page.goto('/');
    await expect(page.getByTestId('status-bar')).toContainText('3 / 3');

    await expect(page.locator('button[title$="(Left, k)"]')).toHaveCount(1);
    await expect(page.locator('button[title$="(PageDown, l)"]')).toHaveCount(1);
    await expect(page.locator('button[title$="(Del)"]')).toHaveCount(1);

    const add = page.locator('.view-tabs .add-btn');
    await expect(add).toContainText('New view');
});
