/**
 * match-navigation.spec.js — flux Match
 *
 * Ouvre un match depuis le panneau Match (sélection d'une ligne puis
 * « Review », ou Entrée), parcourt ses coups avec j/k et les flèches sur deux parties,
 * quitte le mode match par la commande `m` et vérifie le retour à la
 * bibliothèque.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, getWailsCalls } from './helpers/wailsMock.js';
import { openLibraryMock, matchSample, matchGames, matchMovePositions } from './helpers/fixtures.js';

const statusBar = (page) => page.getByTestId('status-bar');
const infoBar = (page) => page.getByTestId('match-info-bar');

// La barre d'état lit le contexte de match, que l'appelant met à jour AVANT
// d'afficher la position : son compteur avance donc même quand l'affichage
// échoue. C'est ce qui est arrivé le 2026-09-03 (structuredClone jetait sur un
// proxy Svelte dans showPosition, et la seule assertion qui l'a vu était le
// bilan des LoadAnalysis en fin de test). `expectsId` fait donc constater, coup
// par coup, que le coup demandé a bien été chargé — l'échec nomme alors le
// coup fautif au lieu d'un tableau final.
async function expectMove(page, move, game, expectsId = null) {
    await expect(statusBar(page)).toContainText(`move ${move}/6`);
    await expect(statusBar(page)).toContainText(`game ${game}/2`);
    if (expectsId !== null) {
        await expect.poll(async () => (await getWailsCalls(page, 'LoadAnalysis')).map((c) => c.args[0]).at(-1)).toBe(expectsId);
    }
}

test.beforeEach(async ({ page }) => {
    await installWailsMock(
        page,
        openLibraryMock({
            database: { GetAllMatches: [matchSample], GetMatchMovePositions: matchMovePositions, GetGamesByMatch: matchGames }
        })
    );
    await page.goto('/');
    await expect(statusBar(page)).toContainText('3 / 3');
});

async function openMatch(page) {
    // Le panneau Match est l'onglet ouvert au chargement : sélectionner la
    // ligne du match puis lancer la revue.
    const panel = page.getByRole('region', { name: 'Match navigator' });
    await panel.getByRole('row', { name: /Alice/ }).click();
    await panel.getByRole('button', { name: /Review/ }).click();
    await expect(infoBar(page)).toBeVisible();
    await expectMove(page, 1, 1);
}

test('ouvrir un match depuis le panneau et parcourir ses coups (j/k, flèches)', async ({ page }) => {
    await openMatch(page);
    await expect(infoBar(page)).toContainText('Alice');
    await expect(infoBar(page)).toContainText('Bob');

    await page.keyboard.press('j');
    await expectMove(page, 2, 1, 2012);
    await page.keyboard.press('ArrowRight');
    await expectMove(page, 3, 1, 2013);
    await page.keyboard.press('k');
    await expectMove(page, 2, 1, 2012);
    await page.keyboard.press('ArrowLeft');
    await expectMove(page, 1, 1, 2011);

    // Franchir la frontière de partie
    for (let i = 0; i < 3; i++) await page.keyboard.press('j');
    await expectMove(page, 4, 2, 2021);

    // Chaque coup affiché a chargé son analyse
    const analysed = (await getWailsCalls(page, 'LoadAnalysis')).map((c) => c.args[0]);
    expect(analysed).toEqual(expect.arrayContaining([2011, 2012, 2013, 2021]));
});

test('sortir du mode match ramène à la bibliothèque', async ({ page }) => {
    await openMatch(page);
    await page.keyboard.press('j');
    await expectMove(page, 2, 1);

    // Commande `m` : retour à la bibliothèque. Les positions de ce match ne
    // figurent pas dans la bibliothèque factice, la sortie retombe donc sur
    // la dernière position (voir match-exit-keeps-position.spec.js pour le
    // cas où elles y sont) ; position du match mémorisée côté backend.
    await page.keyboard.press('Space');
    const commandLine = page.getByPlaceholder('Type command...');
    await commandLine.fill('m');
    await commandLine.press('Enter');

    await expect(infoBar(page)).toBeHidden();
    await expect(statusBar(page)).toContainText('3 / 3');
    await expect(statusBar(page)).not.toContainText('move');
    const saved = await getWailsCalls(page, 'SaveLastVisitedPosition');
    expect(saved.at(-1).args).toEqual([7, 1]);

    // La bibliothèque se parcourt à nouveau normalement — depuis l'onglet
    // Analyse : l'onglet Matchs garde les flèches pour sa propre liste.
    await page.getByTestId('tab-analysis').click();
    await page.keyboard.press('ArrowLeft');
    await expect(statusBar(page)).toContainText('2 / 3');
});

test('Entrée sur la ligne sélectionnée entre directement en revue', async ({ page }) => {
    const panel = page.getByRole('region', { name: 'Match navigator' });
    await panel.getByRole('row', { name: /Alice/ }).click();
    await page.keyboard.press('Enter');

    await expect(infoBar(page)).toContainText('Alice');
    await expectMove(page, 1, 1);
    await expect(page.getByTestId('tab-analysis')).toHaveClass(/active/);
});
