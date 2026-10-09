/**
 * transcription-theatre.spec.js — le mode théâtre de la Transcription, dans un vrai
 * navigateur : la vidéo prend toute la fenêtre, le mini-plateau flotte au coin et se
 * déplace, le clavier de saisie reste actif, F11 et Échap en sortent.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installTranscriptionEngine, sentKinds, resetGestures } from './helpers/transcriptionDraft.js';

test.beforeEach(async ({ page }) => {
    await installWailsMock(page, openLibraryMock({ config: { GetPanelPosition: 'side', GetPanelWidth: 420 } }));
    await installTranscriptionEngine(page, { video: '/videos/match.mp4' });
    await page.goto('/');
    await page.locator('[data-testid="tab-transcription"]').click();
    await page.locator('#transcriptionPanel tbody tr').first().click();
    await expect(page.locator('#transcriptionPanel .draft-bar')).toBeVisible();
});

const theatre = (/** @type {import('@playwright/test').Page} */ page) => page.locator('[data-testid="transcription-theatre"]');

test('F11 ouvre le théâtre : la vidéo couvre la fenêtre, le mini-plateau est au coin', async ({ page }) => {
    await page.locator('#transcriptionPanel').focus();
    await page.keyboard.press('F11');
    await expect(theatre(page)).toBeVisible();
    const view = page.viewportSize() ?? { width: 0, height: 0 };
    const video = await theatre(page).locator('[data-testid="video-dock"]').boundingBox();
    expect(video?.width).toBe(view.width);
    expect(video?.height).toBe(view.height);
    const board = await page.locator('[data-testid="theatre-board"]').boundingBox();
    expect((board?.x ?? 0) + (board?.width ?? 0)).toBeGreaterThan(view.width - 40);
    expect((board?.y ?? 0) + (board?.height ?? 0)).toBeGreaterThan(view.height - 100);
    await page.keyboard.press('F11');
    await expect(theatre(page)).toHaveCount(0);
    // La vidéo revient où elle était : à côté du plateau, la place par défaut.
    await expect(page.locator('[data-testid="video-dock"]')).toHaveAttribute('data-placement', 'board');
});

test('la saisie continue dans le théâtre, et Échap en sort', async ({ page }) => {
    await page.locator('[data-testid="video-theatre"]').click();
    await expect(theatre(page)).toBeVisible();
    await resetGestures(page);
    await page.keyboard.press('3');
    await expect.poll(() => sentKinds(page)).toContain('enter_die');
    await page.keyboard.press('Escape');
    await expect(theatre(page)).toHaveCount(0);
});

test('le mini-plateau se déplace à la souris et garde sa place', async ({ page }) => {
    await page.locator('#transcriptionPanel').focus();
    await page.keyboard.press('F11');
    const board = page.locator('[data-testid="theatre-board"]');
    const before = await board.boundingBox();
    if (!before) throw new Error('mini-plateau absent');
    await page.mouse.move(before.x + 40, before.y + 10);
    await page.mouse.down();
    await page.mouse.move(80, 60, { steps: 5 });
    await page.mouse.up();
    const after = await board.boundingBox();
    expect(after?.x).toBeLessThan(before.x - 100);
    expect(after?.y).toBeLessThan(before.y - 100);
    // Le glisser ne prend pas le focus : le panneau garde ses touches.
    await expect(page.locator('#transcriptionPanel')).toBeFocused();
    await page.keyboard.press('Escape');
    await page.keyboard.press('F11');
    const again = await page.locator('[data-testid="theatre-board"]').boundingBox();
    expect(Math.abs((again?.x ?? 0) - (after?.x ?? 0))).toBeLessThan(2);
});
