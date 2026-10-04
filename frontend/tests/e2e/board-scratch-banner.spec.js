/**
 * Les onglets Recherche et Eval remplacent le plateau par un brouillon : le plateau le dit par
 * un bandeau, et la barre d'info du match, qui décrit la position étudiée, s'efface. Au retour
 * sur l'analyse, la position étudiée est de retour et le bandeau a disparu.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock, matchSample, matchGames, matchMovePositions } from './helpers/fixtures.js';

const statusBar = (page) => page.getByTestId('status-bar');
const infoBar = (page) => page.getByTestId('match-info-bar');
const banner = (page) => page.getByTestId('board-situation-banner');
const board = (page) => page.locator('#backgammon-board');
const tab = (page, id) => page.getByTestId(`tab-${id}`);

test.beforeEach(async ({ page }) => {
    await installWailsMock(
        page,
        openLibraryMock({
            database: { GetAllMatches: [matchSample], GetMatchMovePositions: matchMovePositions, GetGamesByMatch: matchGames },
            config: { GetPanelWidth: 420 }
        })
    );
    await page.goto('/');
    await expect(statusBar(page)).toContainText('3 / 3');
});

test('changer d’onglet annonce le plateau brouillon et masque la barre du match', async ({ page }) => {
    const panel = page.getByRole('region', { name: 'Match navigator' });
    await panel.getByRole('row', { name: /Alice/ }).click();
    await panel.getByRole('button', { name: /Review/ }).click();
    await expect(infoBar(page)).toBeVisible();
    await expect(banner(page)).toBeHidden();
    const studied = await board(page).getAttribute('aria-label');

    await tab(page, 'search').click();
    await expect(banner(page)).toContainText('Search board');
    await expect(infoBar(page)).toBeHidden();

    await tab(page, 'eval').click();
    await expect(banner(page)).toContainText('Evaluation board');
    await expect(infoBar(page)).toBeHidden();

    await tab(page, 'analysis').click();
    await expect(banner(page)).toBeHidden();
    await expect(infoBar(page)).toBeVisible();
    await expect(board(page)).toHaveAttribute('aria-label', studied);
});

for (const [width, height] of [
    [1280, 800],
    [1024, 700]
]) {
    test(`le bandeau ne recouvre rien du plateau à ${width}×${height}`, async ({ page }) => {
        await page.setViewportSize({ width, height });
        await tab(page, 'search').click();
        await expect(banner(page)).toBeVisible();
        const bar = await banner(page).boundingBox();
        const svg = await page.locator('#backgammon-board svg').boundingBox();
        // Le canevas colle en haut de sa zone : tout ce qui le recouvre cache les numéros de points.
        const overlaps = bar.x < svg.x + svg.width && bar.x + bar.width > svg.x && bar.y < svg.y + svg.height && bar.y + bar.height > svg.y;
        expect(overlaps).toBe(false);
    });
}
