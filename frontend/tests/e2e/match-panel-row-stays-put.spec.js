/**
 * match-panel-row-stays-put.spec.js — #201 (D.1)
 *
 * Panneau Match : le premier clic sur une ligne ouvre le volet de détail. Ce
 * volet rétrécissait la liste de 100 % à 45 % : la ligne cliquée bougeait
 * sous le curseur avant le second clic d'un double-clic. La largeur du volet
 * est désormais réservée (une invite l'occupe tant que rien n'est
 * sélectionné) : la ligne ne bouge pas d'un pixel.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock, matchSample, matchGames, matchMovePositions } from './helpers/fixtures.js';

const statusBar = (page) => page.getByTestId('status-bar');

test.beforeEach(async ({ page }) => {
    await installWailsMock(
        page,
        openLibraryMock({
            database: { GetAllMatches: [matchSample], GetMatchMovePositions: matchMovePositions, GetGamesByMatch: matchGames },
            // Une hauteur de panneau fixe, comme en production : sans réponse le
            // panneau prend la hauteur de son contenu et monterait quand la
            // transcription remplit le volet — ici seule la liste compte.
            config: { GetPanelHeight: 250, GetPanelWidth: 420 }
        })
    );
    await page.goto('/');
    await expect(statusBar(page)).toContainText('3 / 3');
});

test('la ligne cliquée ne bouge pas quand le volet de détail s’ouvre', async ({ page }) => {
    const panel = page.getByRole('region', { name: 'Match navigator' });
    const row = panel.getByRole('row', { name: /Alice/ });
    const list = panel.locator('.match-list-pane');

    // Avant tout clic : l'invite occupe le volet, la liste a déjà sa largeur.
    await expect(panel.getByText('Select a match to show its transcript')).toBeVisible();
    const rowBefore = await row.boundingBox();
    const listBefore = await list.boundingBox();

    await row.click();
    await expect(panel.getByRole('button', { name: /Review/ })).toBeVisible();

    expect(await row.boundingBox()).toEqual(rowBefore);
    expect(await list.boundingBox()).toEqual(listBefore);

    // Le second clic retombe sur la même ligne : il la désélectionne.
    await row.click();
    await expect(panel.getByText('Select a match to show its transcript')).toBeVisible();
    expect(await row.boundingBox()).toEqual(rowBefore);
});
