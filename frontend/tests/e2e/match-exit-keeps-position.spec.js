/**
 * match-exit-keeps-position.spec.js — #201 (D.1)
 *
 * Quitter la revue d'un match (commande `m`) recharge la bibliothèque, qui
 * repartait de sa dernière position : le coup que l'on étudiait — une
 * position de la bibliothèque comme les autres — était perdu. La sortie doit
 * se poser sur lui.
 *
 * La bibliothèque factice contient ici les trois positions habituelles plus
 * les six positions du match, dans cet ordre.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, getWailsCalls } from './helpers/wailsMock.js';
import { openLibraryMock, libraryMockAfter, libraryPositions, matchSample, matchGames, matchMovePositions } from './helpers/fixtures.js';

const statusBar = (page) => page.getByTestId('status-bar');
const infoBar = (page) => page.getByTestId('match-info-bar');
const board = (page) => page.locator('#backgammon-board');

test.beforeEach(async ({ page }) => {
    const library = [...libraryPositions, ...matchMovePositions.map((mp) => mp.position)];
    await installWailsMock(
        page,
        openLibraryMock({
            database: {
                ...libraryMockAfter(library),
                GetAllMatches: [matchSample],
                GetMatchMovePositions: matchMovePositions,
                GetGamesByMatch: matchGames
            }
        })
    );
    await page.goto('/');
    await expect(statusBar(page)).toContainText('9 / 9');
});

test('quitter le match laisse le coup étudié sur le damier', async ({ page }) => {
    const panel = page.getByRole('region', { name: 'Match navigator' });
    await panel.getByRole('row', { name: /Alice/ }).click();
    await panel.getByRole('button', { name: /Review/ }).click();
    await expect(infoBar(page)).toBeVisible();
    await expect(statusBar(page)).toContainText('move 1/6');

    // Deuxième coup : la position 2012, cinquième de la bibliothèque.
    await page.keyboard.press('j');
    await expect(statusBar(page)).toContainText('move 2/6');
    const studied = await board(page).getAttribute('aria-label');

    await page.keyboard.press('Space');
    const commandLine = page.getByPlaceholder('Type command...');
    await commandLine.fill('m');
    await commandLine.press('Enter');

    await expect(infoBar(page)).toBeHidden();
    await expect(statusBar(page)).toContainText('5 / 9');
    await expect(board(page)).toHaveAttribute('aria-label', studied);
    const saved = await getWailsCalls(page, 'SaveLastVisitedPosition');
    expect(saved.at(-1).args).toEqual([7, 1]);
});
