/**
 * anki-answer-reveal.spec.js — la réponse masquée d'une carte de révision
 * (ADR-0025), dans l'application réelle et non dans un composant monté seul.
 *
 * Le parcours vérifié est celui de l'utilisateur : ouvrir l'onglet Anki,
 * lancer une session, constater que la réponse est masquée, la révéler, noter,
 * et retrouver la carte suivante masquée. Ce qu'un test de composant ne peut
 * pas voir et qui se joue ici : la touche Espace traverse le répartiteur
 * clavier global (elle ouvre la ligne de commande partout ailleurs), et
 * l'analyse affichée est bien celle que le backend a rendue POUR CETTE CARTE.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, overrideDbMethodByArg } from './helpers/wailsMock.js';

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
    // Le panneau peut être une bande basse large ou une colonne étroite. Une
    // colonne haute avait deux défauts que seule une mesure révèle : la réponse
    // centrée flottait à une demi-hauteur des boutons qui la notent, et le
    // tableau de coups, plus large que la colonne, était coupé au lieu de
    // défiler — on masquait donc des colonnes au moment même de les révéler.
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
