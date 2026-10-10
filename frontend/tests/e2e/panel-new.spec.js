/**
 * panel-new.spec.js — un seul geste de création : « + Nouveau… » dans l'en-tête du panneau
 *
 * Les champs de création de Collections et de Tournois ne sont plus posés en bas de liste :
 * ils s'ouvrent depuis le bouton de l'en-tête.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';

test("Collections : le champ de création s'ouvre depuis « + New collection »", async ({ page }) => {
    await installWailsMock(page, openLibraryMock());
    await page.goto('/');
    await expect(page.getByTestId('status-bar')).toContainText('3 / 3');
    await page.getByTestId('tab-collections').click();

    const panel = page.locator('#collectionPanel');
    await expect(panel.locator('.add-input')).toHaveCount(0);
    await panel.getByTestId('panel-new').click();
    await expect(panel.locator('.add-input').first()).toBeFocused();
});
