/**
 * transcription-bar-menus.spec.js — les deux menus de la barre du brouillon, dans un vrai
 * navigateur : le clic qui ouvre un menu le traverse encore quand le menu se monte, ce que
 * jsdom ne reproduit pas. Le menu doit rester ouvert, se fermer à Échap ou au clic dehors,
 * et rendre le focus au panneau.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installTranscriptionEngine } from './helpers/transcriptionDraft.js';

test.beforeEach(async ({ page }) => {
    await installWailsMock(page, openLibraryMock({ config: { GetPanelPosition: 'side', GetPanelWidth: 420 } }));
    await installTranscriptionEngine(page);
    await page.goto('/');
    await page.locator('[data-testid="tab-transcription"]').click();
    await page.locator('#transcriptionPanel tbody tr').first().click();
    await expect(page.locator('#transcriptionPanel .draft-bar')).toBeVisible();
});

test('le menu Vidéo reste ouvert, montre le champ YouTube et se ferme à Échap', async ({ page }) => {
    await page.locator('[data-testid="transcription-video-button"]').click();
    const menu = page.locator('[data-testid="transcription-video-menu"]');
    await expect(menu).toBeVisible();
    await menu.getByRole('menuitem', { name: /YouTube/ }).click();
    await expect(menu.locator('input')).toBeFocused();
    await page.keyboard.press('Escape');
    await expect(menu).toHaveCount(0);
    await expect(page.locator('#transcriptionPanel')).toBeFocused();
});

test('le menu ⋯ tient dans la fenêtre et se ferme au clic dehors', async ({ page }) => {
    await page.locator('[data-testid="transcription-more-button"]').click();
    const menu = page.locator('[data-testid="transcription-more-menu"]');
    await expect(menu).toBeVisible();
    const box = await menu.boundingBox();
    const width = page.viewportSize()?.width ?? 0;
    expect((box?.x ?? 0) + (box?.width ?? 0)).toBeLessThanOrEqual(width);
    await page.locator('#transcriptionPanel .save-state').click();
    await expect(menu).toHaveCount(0);
});
