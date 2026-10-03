/**
 * anki-answer-reveal.spec.js — la réponse masquée d'une carte de révision
 * (ADR-0025), dans l'application réelle et non dans un composant monté seul.
 *
 * Ce qu'un test de composant ne voit pas : Espace traverse le répartiteur
 * clavier global (il ouvre la ligne de commande ailleurs), et l'analyse
 * affichée est celle que le backend a rendue pour cette carte.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, overrideDbMethodByArg, getWailsCalls } from './helpers/wailsMock.js';
import { positionA } from './helpers/fixtures.js';

const POSITION_10 = {
    id: 10,
    board: { points: Array.from({ length: 26 }, () => ({ color: -1, checkers: 0 })), bearoff: [0, 0] },
    dice: [6, 1],
    cube: { value: 0, owner: -1 },
    score: [-1, -1],
    player_on_roll: 0
};
const POSITION_11 = { ...POSITION_10, id: 11 };

const CARD_10 = { card: { id: 100, state: 0 }, position: POSITION_10 };
const CARD_11 = { card: { id: 101, state: 0 }, position: POSITION_11 };

const DECK = { id: 1, name: 'Blunders', description: '', sourceType: 'collection', sourceId: 7, cardCount: 2, newCount: 2, dueCount: 2 };

// Two DIFFERENT analyses: the moves below are what tells us which position's
// answer is on screen.
const ANALYSIS_10 = {
    positionId: 10,
    analysisType: 'CheckerMove',
    analysisEngineVersion: 'XG2',
    checkerAnalysis: { moves: [{ move: '13/7 8/7', equity: 0.512, error: 0, analysisDepth: '3-ply' }] },
    playedMoves: [],
    playedCubeActions: []
};
const ANALYSIS_11 = {
    positionId: 11,
    analysisType: 'CheckerMove',
    analysisEngineVersion: 'XG2',
    checkerAnalysis: { moves: [{ move: '24/18 6/5', equity: 0.203, error: 0, analysisDepth: '3-ply' }] },
    playedMoves: [],
    playedCubeActions: []
};

async function startReview(page, { reviewReturns = CARD_11, side = false } = {}) {
    await installWailsMock(page, {
        config: { GetLastDatabasePath: '/tmp/e2e-anki.db', ...(side ? { GetPanelPosition: 'side', GetPanelWidth: 460 } : {}) },
        // PathExists gates the reopen of the last database at startup; without
        // it the app forgets the path and no deck ever loads.
        app: { PathExists: true, IsProtectedCopyPath: false },
        database: {
            GetAllAnkiDecks: [DECK],
            GetAnkiDeckStats: { newCount: 2, learningCount: 0, reviewCount: 0, totalCount: 2, dueCount: 2 },
            GetAnkiDeckPositions: [POSITION_10, POSITION_11],
            GetNextAnkiCard: CARD_10,
            ReviewAnkiCard: reviewReturns,
            SyncAnkiDeck: null,
            ListPositionIDs: [10, 11],
            CheckDatabaseVersion: '2.15.0',
            GetDatabaseVersion: '2.15.0'
        }
    });
    await page.goto('/');
    await expect(page.locator('[data-testid="status-bar"]')).toBeVisible({ timeout: 8000 });
    await page.keyboard.press('Escape'); // the first-run tour catalog, if it opened

    // Each position answers with its own analysis — a constant would make them
    // all look alike and hide the very bug this feature depends on.
    await overrideDbMethodByArg(page, 'LoadAnalysis', { 10: ANALYSIS_10, 11: ANALYSIS_11 }, null);

    await page.click('[data-testid="tab-anki"]');
    await expect(page.locator('[data-testid="tab-anki"]')).toHaveClass(/active/);
    await page.click('tbody tr');
    await page.click('.btn-study');
    await expect(page.locator('.review-body')).toBeVisible();
}

test('la réponse est masquée à l’ouverture de la carte', async ({ page }) => {
    await startReview(page);

    await expect(page.locator('.answer-masked')).toBeVisible();
    await expect(page.locator('.checker-table')).toHaveCount(0);
    await expect(page.locator('.review-body')).not.toContainText('13/7');
    // Noter reste possible sans révéler.
    await expect(page.locator('.btn-rating')).toHaveCount(4);
    for (const b of await page.locator('.btn-rating').all()) await expect(b).toBeEnabled();
});

test('Espace révèle l’analyse de CETTE carte, sans ouvrir la ligne de commande', async ({ page }) => {
    await startReview(page);

    await page.keyboard.press('Space');

    await expect(page.locator('.answer-masked')).toHaveCount(0);
    await expect(page.locator('.checker-table')).toBeVisible();
    await expect(page.locator('.review-body')).toContainText('13/7');
    await expect(page.locator('.review-body')).not.toContainText('24/18'); // l'autre position
    await expect(page.locator('.command-input')).toHaveCount(0);
    await page.screenshot({ path: 'test-results/anki-answer-revealed.png' });
});

test('un clic sur la zone masquée révèle la même chose', async ({ page }) => {
    await startReview(page);

    await page.click('.answer-masked');

    await expect(page.locator('.checker-table')).toBeVisible();
    await expect(page.locator('.review-body')).toContainText('13/7');
});

test('la bande de notes reste au-dessus de la réponse et ne défile pas', async ({ page }) => {
    await startReview(page);
    await page.keyboard.press('Space');
    await expect(page.locator('.checker-table')).toBeVisible();

    const strip = await page.locator('.review-strip').boundingBox();
    const answer = await page.locator('.review-answer').boundingBox();
    expect(strip.y + strip.height).toBeLessThanOrEqual(answer.y + 1);

    const scrolls = await page.locator('.review-answer').evaluate((el) => getComputedStyle(el).overflowY);
    expect(scrolls).toBe('auto');
});

test('noter passe à la carte suivante, réponse masquée à nouveau', async ({ page }) => {
    await startReview(page);
    await page.keyboard.press('Space');
    await expect(page.locator('.review-body')).toContainText('13/7');

    await page.keyboard.press('Digit3'); // « Correct »

    await expect(page.locator('.answer-masked')).toBeVisible();
    await expect(page.locator('.review-body')).not.toContainText('13/7');

    // …et c'est bien la réponse de la nouvelle carte qui se cache dessous.
    await page.keyboard.press('Space');
    await expect(page.locator('.review-body')).toContainText('24/18');
    await expect(page.locator('.review-body')).not.toContainText('13/7');
});

test('changer d’onglet et revenir ne remasque pas la réponse', async ({ page }) => {
    await startReview(page);
    await page.keyboard.press('Space');
    await expect(page.locator('.review-body')).toContainText('13/7');

    await page.click('[data-testid="tab-analysis"]');
    await expect(page.locator('[data-testid="tab-analysis"]')).toHaveClass(/active/);
    await page.click('[data-testid="tab-anki"]');

    await expect(page.locator('.answer-masked')).toHaveCount(0);
    await expect(page.locator('.review-body')).toContainText('13/7');
});

test('l’onglet Analyse montre la même position que la carte, pas une autre', async ({ page }) => {
    await startReview(page);

    await page.click('[data-testid="tab-analysis"]');

    // Le correctif de showCard : sans lui, l'onglet affichait l'analyse de la
    // dernière position parcourue — vide ici, ou celle d'une autre carte.
    await expect(page.locator('[data-testid="tab-content"]')).toContainText('13/7');
});

test('une position sans analyse enregistrée le dit, sans zone masquée', async ({ page }) => {
    await startReview(page);
    await overrideDbMethodByArg(page, 'LoadAnalysis', {}, null);

    await page.keyboard.press('Digit3'); // carte suivante, sans analyse

    await expect(page.locator('.answer-absent')).toBeVisible();
    await expect(page.locator('.answer-masked')).toHaveCount(0);
});

test('en colonne latérale, la réponse se pose sous la bande et défile au lieu d’être coupée', async ({ page }) => {
    // En colonne étroite, la réponse doit rester près des boutons qui la
    // notent, et le tableau de coups défiler plutôt qu'être coupé.
    await startReview(page, { side: true });
    await page.keyboard.press('Space');
    await expect(page.locator('.checker-table')).toBeVisible();

    const strip = await page.locator('.review-strip').boundingBox();
    const table = await page.locator('.checker-table').boundingBox();
    expect(table.y - (strip.y + strip.height)).toBeLessThan(40);

    const { scrollable, clipped } = await page.locator('.review-answer').evaluate((el) => ({
        scrollable: el.scrollWidth > el.clientWidth,
        clipped: getComputedStyle(el).overflowX === 'hidden'
    }));
    expect(clipped).toBe(false);
    if (scrollable) {
        await page.locator('.review-answer').evaluate((el) => (el.scrollLeft = el.scrollWidth));
        const moved = await page.locator('.review-answer').evaluate((el) => el.scrollLeft > 0);
        expect(moved).toBe(true);
    }
});

// « Répondre au damier » (ADR-0040) : l'option du paquet fait jouer le coup au damier, le quiz le
// juge et propose une note que le joueur garde la main de corriger.
const PLAY = {
    notation: '6/3 4/3',
    steps: [
        { from: 6, to: 3, hit: false },
        { from: 4, to: 3, hit: false }
    ],
    result: { board: positionA.board }
};
const VERDICT = { legal: true, matched: true, notation: '6/3 4/3', best: '8/5 6/5', errorMp: 42 };
const ANALYSIS_A = {
    positionId: positionA.id,
    analysisType: 'CheckerMove',
    checkerAnalysis: {
        moves: [
            { move: '8/5 6/5', equity: 0.1, equityError: 0 },
            { move: '6/3 4/3', equity: 0.058, equityError: -0.042 }
        ]
    },
    playedMoves: [],
    playedCubeActions: []
};

async function clickPoint(page, point) {
    const at = await page.evaluate(async (p) => {
        const { boardMetrics } = await import('/src/utils/boardGeometry.js');
        const { stackSlotCenter } = await import('/src/utils/boardScene.js');
        const { defaultBoardConfig } = await import('/src/utils/boardConfig.js');
        const host = document.getElementById('backgammon-board');
        const drawing = host.firstElementChild;
        const width = Number(drawing.getAttribute('width'));
        const height = Number(drawing.getAttribute('height'));
        const rect = host.getBoundingClientRect();
        const cfg = defaultBoardConfig();
        const { x, y } = stackSlotCenter(boardMetrics(width, height, cfg.widthFactor), cfg, p, 0);
        return { x: rect.left + (x * rect.width) / width, y: rect.top + (y * rect.height) / height };
    }, point);
    await page.mouse.click(at.x, at.y);
}

async function startBoardReview(page, { option = true } = {}) {
    const card = { card: { id: 200, state: 0, deckId: 1 }, position: positionA };
    await installWailsMock(page, {
        config: { GetLastDatabasePath: '/tmp/e2e-anki.db' },
        app: { PathExists: true, IsProtectedCopyPath: false, LegalMoves: [PLAY] },
        database: {
            GetAllAnkiDecks: [{ ...DECK, cardCount: 1, newCount: 1, dueCount: 1 }],
            GetAnkiDeckStats: { newCount: 1, learningCount: 0, reviewCount: 0, totalCount: 1, dueCount: 1 },
            GetAnkiDeckPositions: [positionA],
            GetNextAnkiCard: card,
            ReviewAnkiCard: card,
            SyncAnkiDeck: null,
            ListPositionIDs: [positionA.id],
            LoadMetadata: option ? { anki_board_answer_1: '1' } : {},
            GradeQuizChecker: VERDICT,
            CheckDatabaseVersion: '2.15.0',
            GetDatabaseVersion: '2.15.0'
        }
    });
    await page.goto('/');
    await expect(page.locator('[data-testid="status-bar"]')).toBeVisible({ timeout: 8000 });
    await page.keyboard.press('Escape');
    await overrideDbMethodByArg(page, 'LoadAnalysis', { [positionA.id]: ANALYSIS_A }, null);
    await page.click('[data-testid="tab-anki"]');
    await page.click('tbody tr');
    await page.click('.btn-study');
    await expect(page.locator('.review-body')).toBeVisible();
}

test('répondre au damier : le coup se joue, le quiz le juge, la note est proposée et reste corrigeable', async ({ page }) => {
    await startBoardReview(page);

    const validate = page.getByTestId('anki-board-validate');
    await expect(validate).toBeDisabled();
    await expect(page.locator('.answer-masked'), 'la réponse reste masquée tant que rien n’est joué').toBeVisible();

    await clickPoint(page, 6);
    await clickPoint(page, 3);
    await clickPoint(page, 4);
    await clickPoint(page, 3);
    await expect(validate).toBeEnabled();
    await validate.click();

    await expect(page.getByTestId('anki-board-verdict')).toContainText('42');
    await expect(page.locator('.checker-table')).toBeVisible();
    // 42 mp : une erreur sous le blunder, donc « Difficile » (2) proposé…
    await expect(page.locator('.btn-rating.suggested')).toHaveCount(1);
    await expect(page.locator('.btn-rating.suggested .rating-key')).toHaveText('2');
    const graded = await getWailsCalls(page, 'GradeQuizChecker');
    expect(graded).toHaveLength(1);
    expect(graded[0].args[0]).toBe(positionA.id);

    // …et le joueur note ce qu'il veut : rien ne l'impose.
    await page.keyboard.press('Digit3');
    const reviews = await getWailsCalls(page, 'ReviewAnkiCard');
    expect(reviews[0].args[1]).toBe(3);
});

test('sans l’option, la carte reste en auto-notation : aucun damier armé', async ({ page }) => {
    await startBoardReview(page, { option: false });

    await expect(page.getByTestId('anki-board-play')).toHaveCount(0);
    await page.keyboard.press('Space');
    await expect(page.locator('.checker-table')).toBeVisible();
    await expect(page.locator('.btn-rating.suggested')).toHaveCount(0);
    expect(await getWailsCalls(page, 'GradeQuizChecker')).toHaveLength(0);
});

test('révéler sans jouer abandonne le coup : pas de suggestion, la main reste au joueur', async ({ page }) => {
    await startBoardReview(page);
    await expect(page.getByTestId('anki-board-play')).toBeVisible();

    await page.keyboard.press('Space');

    await expect(page.locator('.checker-table')).toBeVisible();
    await expect(page.getByTestId('anki-board-play')).toHaveCount(0);
    await expect(page.locator('.btn-rating.suggested')).toHaveCount(0);
});
