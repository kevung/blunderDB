/**
 * manage-players.spec.js
 *
 * La fenêtre « Gérer les joueurs » du panneau Match : chaque joueur avec son
 * nombre de matchs et ses alias, l'étoile du joueur principal. Cliquer l'étoile
 * montre d'abord combien de matchs seront inversés ; rien ne s'inverse avant
 * la confirmation.
 *
 * MANAGE_PLAYERS_SHOTS=<dossier> y dépose les captures de la fenêtre.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock, matchSample, matchGames, matchMovePositions } from './helpers/fixtures.js';

const shots = process.env.MANAGE_PLAYERS_SHOTS;

test.beforeEach(async ({ page }) => {
    await installWailsMock(
        page,
        openLibraryMock({
            database: {
                GetAllMatches: [matchSample, { ...matchSample, id: 8, player1_name: 'Gildas', player2_name: 'Kevin' }],
                GetMatchMovePositions: matchMovePositions,
                GetGamesByMatch: matchGames,
                GetAllPlayerNames: [
                    { Name: 'Kevin', Count: 42 },
                    { Name: 'Gildas', Count: 17 },
                    { Name: 'Alice', Count: 3 },
                    { Name: 'Bob', Count: 2 }
                ],
                ListAliases: [
                    { alias: 'K. Unger', canonical: 'Kevin' },
                    { alias: 'kevung', canonical: 'Kevin' },
                    { alias: 'Gildas L.', canonical: 'Gildas' }
                ],
                MainPlayer: '',
                PlanMainPlayer: { name: 'Kevin', swap: [8, 11, 12, 15, 21, 23, 30, 31, 40, 41, 52, 60, 61, 70, 71, 77, 80, 88, 90], refused: [95] },
                SetMainPlayer: { name: 'Kevin', swap: [8], refused: [] }
            },
            config: { GetPanelWidth: 420 }
        })
    );
    await page.goto('/');
});

test('the star previews the swaps before anything is swapped', async ({ page }) => {
    const panel = page.getByRole('region', { name: 'Match navigator' });
    await panel.getByRole('button', { name: /Manage players/ }).click();
    const dialog = page.getByRole('dialog');
    await expect(dialog.getByText('Kevin', { exact: true })).toBeVisible();
    await expect(dialog.getByTestId('alias-chip')).toHaveCount(3);
    if (shots) await dialog.screenshot({ path: `${shots}/manage-players-list.png` });

    await dialog.getByTestId('main-player-star').first().click();
    await expect(dialog.getByTestId('main-player-swap-count')).toHaveText('19');
    await expect(dialog.getByTestId('main-player-refused-count')).toHaveText('1');
    if (shots) await dialog.screenshot({ path: `${shots}/manage-players-confirm.png` });
});
