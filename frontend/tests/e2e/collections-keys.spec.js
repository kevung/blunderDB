/**
 * collections-keys.spec.js — le panneau Collections reçoit ses touches par le répartiteur
 * commun, qui lui laisse les touches qu'il réserve (Suppr) sans les passer au plateau.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';

test('Suppr dans le panneau Collections ne propose pas de supprimer la position du plateau', async ({ page }) => {
    await installWailsMock(page, openLibraryMock());
    await page.goto('/');
    const status = page.getByTestId('status-bar');
    await expect(status).toContainText('3 / 3');

    // Témoin : la suppression de la position demande bien cette confirmation.
    const del = page.getByRole('button', { name: 'Delete Position' });
    await expect(del).toBeEnabled();
    await del.click();
    await expect(page.getByText('Delete this position and its associated analysis?')).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(page.getByText('Delete this position and its associated analysis?')).toHaveCount(0);

    await page.getByTestId('tab-collections').click();
    await page.locator('#collectionPanel').focus();
    await page.keyboard.press('Delete');
    await expect(page.getByText('Delete this position and its associated analysis?')).toHaveCount(0);
    await expect(status).toContainText('3 / 3');
});
