/**
 * Le panneau Transcription ne mange pas le plateau : à 1024×700, dock bas, le plateau entier
 * (rangée 12-1 comprise) reste dans la fenêtre et dans sa zone, et la zone ne s'écrase pas sous
 * une taille lisible.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installTranscriptionEngine } from './helpers/transcriptionDraft.js';

test.use({ viewport: { width: 1024, height: 700 } });

test('le plateau entier est visible avec le panneau Transcription ouvert', async ({ page }) => {
    await installWailsMock(page, openLibraryMock({ config: { GetPanelPosition: 'bottom', GetPanelHeight: 250 } }));
    await installTranscriptionEngine(page);
    await page.goto('/');
    await page.locator('[data-testid="tab-transcription"]').click();
    await page.locator('#transcriptionPanel tbody tr').first().click();
    await expect(page.locator('#transcriptionPanel .draft-bar')).toBeVisible();

    const area = page.locator('.scrollable-content');
    const svg = page.locator('#backgammon-board svg');
    await expect.poll(async () => (await svg.boundingBox()).height).toBeGreaterThan(200);

    const box = await svg.boundingBox();
    const zone = await area.boundingBox();
    expect(box.y).toBeGreaterThanOrEqual(zone.y - 1);
    expect(box.y + box.height).toBeLessThanOrEqual(zone.y + zone.height + 1);
    expect(box.y + box.height).toBeLessThanOrEqual(700);
    expect(box.x + box.width).toBeLessThanOrEqual(1024);

    // Le panneau, lui, reste dans la fenêtre : rien de la colonne « Match » n'est coupé à droite.
    const panel = await page.locator('#transcriptionPanel').boundingBox();
    expect(panel.x + panel.width).toBeLessThanOrEqual(1024 + 1);
});
