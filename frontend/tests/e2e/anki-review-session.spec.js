/**
 * anki-review-session.spec.js — une révision Anki de bout en bout
 *
 * Ouvrir le paquet, révéler, noter au clavier chaque carte jusqu'à la dernière : chaque note
 * part au planificateur avec sa carte, la session se clôt sur son bilan, et le retour à la
 * liste des paquets (Échap) rend la main. La révélation elle-même est tenue par
 * anki-answer-reveal.spec.js.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, overrideDbMethodByArg, getWailsCalls } from './helpers/wailsMock.js';

const POSITION_10 = {
    id: 10,
    board: { points: Array.from({ length: 26 }, () => ({ color: -1, checkers: 0 })), bearoff: [0, 0] },
    dice: [6, 1],
    cube: { value: 0, owner: -1 },
    score: [-1, -1],
    player_on_roll: 0
};
const POSITION_11 = { ...POSITION_10, id: 11 };
const CARD_10 = { card: { id: 100, deckId: 1, state: 0 }, position: POSITION_10 };
const CARD_11 = { card: { id: 101, deckId: 1, state: 0 }, position: POSITION_11 };
/** @param {number} positionId @param {string} move */
const analysis = (positionId, move) => ({
    positionId,
    analysisType: 'CheckerMove',
    analysisEngineVersion: 'XG2',
    checkerAnalysis: { moves: [{ move, equity: 0.5, error: 0, analysisDepth: '3-ply' }] },
    playedMoves: [],
    playedCubeActions: []
});
const DECK = { id: 1, name: 'Blunders', description: '', sourceType: 'collection', sourceId: 7, cardCount: 2, newCount: 2, dueCount: 2 };

test('deux cartes notées au clavier, puis le bilan et le retour aux paquets', async ({ page }) => {
    await installWailsMock(page, {
        config: { GetLastDatabasePath: '/tmp/e2e-anki.db' },
        app: { PathExists: true, IsProtectedCopyPath: false },
        database: {
            GetAllAnkiDecks: [DECK],
            GetAnkiDeckStats: { newCount: 2, learningCount: 0, reviewCount: 0, totalCount: 2, dueCount: 2 },
            GetAnkiDeckPositions: [POSITION_10, POSITION_11],
            GetNextAnkiCard: CARD_10,
            SyncAnkiDeck: null,
            ListPositionIDs: [10, 11],
            CheckDatabaseVersion: '2.15.0',
            GetDatabaseVersion: '2.15.0'
        }
    });
    await page.goto('/');
    await expect(page.getByTestId('status-bar')).toBeVisible({ timeout: 8000 });
    await page.keyboard.press('Escape'); // le catalogue de la visite, s'il s'est ouvert

    await overrideDbMethodByArg(page, 'LoadAnalysis', { 10: analysis(10, '13/7 8/7'), 11: analysis(11, '24/18 6/5') }, null);
    // Le planificateur rend la carte 11 après la 10, puis plus rien.
    await page.evaluate(
        ([second]) => {
            window.go.database.Database.ReviewAnkiCard = async (cardId) => (cardId === 100 ? second : null);
        },
        [CARD_11]
    );

    await page.getByTestId('tab-anki').click();
    await page.click('tbody tr');
    await page.click('.btn-study');
    await expect(page.locator('.review-body')).toBeVisible();

    await page.keyboard.press('Space');
    await expect(page.locator('.review-body')).toContainText('13/7');
    await page.keyboard.press('Digit3');
    await expect(page.locator('.answer-masked')).toBeVisible();

    await page.keyboard.press('Space');
    await expect(page.locator('.review-body')).toContainText('24/18');
    await page.keyboard.press('Digit1');

    await expect(page.getByText('Review complete! 2 cards reviewed.')).toBeVisible();
    const graded = await getWailsCalls(page, 'ReviewAnkiCard');
    expect(graded.map((c) => c.args)).toEqual([
        [100, 3],
        [101, 1]
    ]);

    await page.keyboard.press('Escape');
    await expect(page.locator('.review-body')).toHaveCount(0);
    await expect(page.locator('tbody tr', { hasText: 'Blunders' })).toBeVisible();
});
