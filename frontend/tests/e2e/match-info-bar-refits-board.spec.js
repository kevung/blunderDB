/**
 * match-info-bar-refits-board.spec.js — #201 (D.1)
 *
 * La barre d'information du match s'insère au-dessus du plateau à l'entrée
 * en revue et disparaît à la sortie. Le plateau (two.js) ne se mesure que sur
 * un événement 'resize' : sans lui, il restait 22 px trop grand pendant toute
 * la revue, puis trop petit après. La spec mesure le SVG contre son conteneur
 * dans les deux sens.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock, matchSample, matchGames, matchMovePositions } from './helpers/fixtures.js';

const statusBar = (page) => page.getByTestId('status-bar');
const infoBar = (page) => page.getByTestId('match-info-bar');

/** Hauteurs (px) du conteneur du plateau et du SVG qu'il contient. */
async function heights(page) {
    const container = await page.locator('.scrollable-content').evaluate((el) => el.clientHeight);
    const svg = await page.locator('#backgammon-board svg').evaluate((el) => el.getBoundingClientRect().height);
    return { container, svg };
}

/** Le SVG occupe toute la hauteur du conteneur, au pixel d'arrondi près. */
const fitted = ({ container, svg }) => Math.abs(svg - container) <= 1;

test.beforeEach(async ({ page }) => {
    await installWailsMock(
        page,
        openLibraryMock({
            database: { GetAllMatches: [matchSample], GetMatchMovePositions: matchMovePositions, GetGamesByMatch: matchGames },
            // Une hauteur de panneau fixe, comme en production : sans réponse le
            // panneau prend la hauteur de son contenu, et la zone du plateau
            // bougerait à chaque changement d'onglet — ici seule la barre compte.
            config: { GetPanelHeight: 250, GetPanelWidth: 420 }
        })
    );
    await page.goto('/');
    await expect(statusBar(page)).toContainText('3 / 3');
});

test('le plateau se réajuste quand la barre du match apparaît puis disparaît', async ({ page }) => {
    await expect(infoBar(page)).toBeHidden();
    // À 1280×800 le plateau est contraint par la hauteur : ajusté, son SVG
    // fait exactement la hauteur du conteneur.
    await expect.poll(async () => fitted(await heights(page))).toBe(true);
    const before = await heights(page);

    // Entrée en revue : la barre prend sa place, le conteneur rétrécit, le
    // plateau doit tenir dedans.
    const panel = page.getByRole('region', { name: 'Match navigator' });
    await panel.getByRole('row', { name: /Alice/ }).click();
    await panel.getByRole('button', { name: /Review/ }).click();
    await expect(infoBar(page)).toBeVisible();

    await expect.poll(async () => (await heights(page)).container).toBeLessThan(before.container);
    await expect.poll(async () => fitted(await heights(page))).toBe(true);
    const during = await heights(page);
    expect(during.svg).toBeLessThan(before.svg);

    // Sortie : la barre disparaît, le plateau reprend sa taille.
    await page.keyboard.press('Space');
    const commandLine = page.getByPlaceholder('Type command...');
    await commandLine.fill('m');
    await commandLine.press('Enter');
    await expect(infoBar(page)).toBeHidden();

    await expect.poll(async () => (await heights(page)).container).toBe(before.container);
    await expect.poll(async () => fitted(await heights(page))).toBe(true);
});
