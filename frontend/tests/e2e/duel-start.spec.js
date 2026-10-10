/**
 * Le lanceur du Duel : partant du plateau, le plateau devient le brouillon d'Eval ; le score,
 * la partie unique et la cadence en deux nombres partent avec la création.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, getWailsCalls } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';

const OFFER = {
    cadences: [
        { name: 'standard', reservePerPoint: 120, delay: 12 },
        { name: 'speed', reservePerPoint: 24, delay: 10 }
    ],
    botLevels: ['instant', 'normal', 'thorough'],
    levels: []
};

test.beforeEach(async ({ page }) => {
    await installWailsMock(page, openLibraryMock({ database: { DuelOffer: OFFER, ListDuels: [] }, config: { GetPanelWidth: 520 } }));
    await page.goto('/');
    await expect(page.getByTestId('status-bar')).toContainText('3 / 3');
});

test('du plateau, à un score, une seule partie, en cadence Speed', async ({ page }) => {
    await page.getByTestId('tab-duel').click();
    const panel = page.getByTestId('duel-panel');
    await panel.getByTestId('duel-start').selectOption('board');
    await expect(page.getByTestId('board-situation-banner')).toContainText('Evaluation board');

    await panel.getByTestId('duel-length').selectOption('5');
    await panel.getByTestId('duel-away-1').selectOption('1');
    await panel.getByTestId('duel-away-2').selectOption('3');
    await panel.getByTestId('duel-single-game').check();
    await panel.getByTestId('duel-cadence').selectOption('speed');
    await expect(panel.getByTestId('duel-minutes')).toHaveValue('0.4');
    await expect(panel.getByTestId('duel-delay')).toHaveValue('10');

    const shot = process.env.DUEL_SCREENSHOT;
    if (shot) await page.screenshot({ path: shot });

    await panel.getByTestId('duel-play').click();
    await expect.poll(async () => (await getWailsCalls(page, 'CreateDuel')).length).toBe(1);
    const [settings] = /** @type {any[]} */ ((await getWailsCalls(page, 'CreateDuel'))[0].args);
    expect(settings.matchLength).toBe(5);
    expect(settings.singleGame).toBe(true);
    expect(settings.start.score).toEqual([1, 3]);
    expect(settings.cadence).toMatchObject({ name: 'speed', reservePerPoint: 24, delay: 10 });
});
