/**
 * view-tabs-layout.spec.js
 *
 * Passer d'un groupe de bibliothèque à l'autre ne déplace pas le plateau, même quand l'un des
 * deux montre la barre d'information de la position et l'autre non : son emplacement est
 * réservé dès que deux bibliothèques sont ouvertes.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock, matchSample, matchGames, matchMovePositions } from './helpers/fixtures.js';

const pathB = '/libs/club.db';

test.beforeEach(async ({ page }) => {
    await installWailsMock(
        page,
        openLibraryMock({
            database: { GetAllMatches: [matchSample], GetMatchMovePositions: matchMovePositions, GetGamesByMatch: matchGames },
            config: { GetPanelWidth: 420 },
            app: {
                OpenDatabaseDialog: pathB,
                LibraryGroupHolding: 0,
                OpenLibraryGroup: { id: 2, path: pathB, active: true, readOnly: false },
                ActivateLibraryGroup: null,
                CloseLibraryGroup: null
            }
        })
    );
    await page.goto('/');
    await expect(page.getByTestId('status-bar')).toContainText('3 / 3');
});

/** Vertical position and height (px) of the board, rounded to the pixel. */
const boardBox = async (page) =>
    page.locator('#backgammon-board svg').evaluate((el) => {
        const r = el.getBoundingClientRect();
        return { top: Math.round(r.top), height: Math.round(r.height) };
    });

test('the board does not move between a group with the position bar and one without', async ({ page }) => {
    // Group 1 reviews a match: its info bar is shown.
    const panel = page.getByRole('region', { name: 'Match navigator' });
    await panel.getByRole('row', { name: /Alice/ }).click();
    await panel.getByRole('button', { name: /Review/ }).click();
    await expect(page.getByTestId('match-info-bar')).toBeVisible();

    // A second library opens: its group has no position to describe.
    await page.getByTestId('open-library-group').click();
    await expect(page.getByTestId('library-group')).toHaveCount(2);
    await expect(page.getByTestId('match-info-bar')).toBeHidden();
    await expect(page.getByTestId('match-info-slot')).toBeVisible();
    await expect.poll(async () => (await boardBox(page)).height).toBeGreaterThan(100);
    const without = await boardBox(page);

    // Back to group 1 (bar shown), then to group 2 again.
    await page.locator('[role="tab"][data-group-id="1"]').click();
    await expect(page.getByTestId('library-group').nth(0)).toHaveClass(/active/);
    await expect(page.getByTestId('match-info-bar')).toBeVisible();
    await expect.poll(async () => (await boardBox(page)).top).toBe(without.top);
    expect(await boardBox(page)).toEqual(without);

    await page.locator('[role="tab"][data-group-id="2"]').click();
    await expect(page.getByTestId('library-group').nth(1)).toHaveClass(/active/);
    await expect(page.getByTestId('match-info-bar')).toBeHidden();
    await expect.poll(async () => (await boardBox(page)).top).toBe(without.top);
    expect(await boardBox(page)).toEqual(without);
});
