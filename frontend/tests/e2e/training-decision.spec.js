/**
 * training-decision.spec.js — l'exercice Décision, dans l'application réelle.
 *
 * Le parcours est celui de l'utilisateur : taper `train decision`, trouver la
 * question dans l'onglet Entraînement, JOUER le coup sur le plateau — deux
 * clics par dé, sur le damier dessiné —, le valider depuis le panneau, et lire
 * le verdict. Ce qu'un test de composant ne voit pas et qui se joue ici : la
 * commande traverse la ligne de commande et l'application, la question passe
 * par la liste parcourue sans quitter l'onglet, et le clic sur le damier arrive
 * au coup armé par la session (boardInteractions → quizPlayStore → panneau).
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, overrideDbMethodByArg, getWailsCalls } from './helpers/wailsMock.js';
import { openLibraryMock, libraryPositions, positionA } from './helpers/fixtures.js';

// positionA : le joueur 1 au trait, dés 3-1, des pions sur 4 et 6. Le moteur
// factice n'offre qu'un coup, 6/3 4/3 — c'est lui que le plateau doit accepter.
const PLAY = {
    notation: '6/3 4/3',
    steps: [
        { from: 6, to: 3, hit: false },
        { from: 4, to: 3, hit: false }
    ],
    result: { board: positionA.board }
};

// Une seule position analysée : la question est donc positionA, quel que soit
// le tirage.
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

const VERDICT = { legal: true, matched: true, notation: '6/3 4/3', best: '8/5 6/5', errorMp: 42 };

/**
 * Clique le point `point` du modèle sur le damier dessiné. Les coordonnées
 * sortent des fonctions mêmes qui dessinent le plateau et qui lisent le clic
 * (boardMetrics, stackSlotCenter), converties en pixels comme le fait
 * `boardMouseToDrawing` dans l'autre sens : un clic « au jugé » donnerait une
 * spec qui dépend de la taille de la fenêtre.
 *
 * @param {import('@playwright/test').Page} page
 * @param {number} point
 */
async function clickPoint(page, point) {
    const at = await page.evaluate(async (p) => {
        const { boardMetrics } = await import('/src/utils/boardGeometry.js');
        const { stackSlotCenter } = await import('/src/utils/boardScene.js');
        const { defaultBoardConfig } = await import('/src/utils/boardConfig.js');
        const host = /** @type {HTMLElement} */ (document.getElementById('backgammon-board'));
        const drawing = /** @type {Element} */ (host.firstElementChild);
        const width = Number(drawing.getAttribute('width'));
        const height = Number(drawing.getAttribute('height'));
        const rect = host.getBoundingClientRect();
        const cfg = defaultBoardConfig();
        const { x, y } = stackSlotCenter(boardMetrics(width, height, cfg.widthFactor), cfg, p, 0);
        return { x: rect.left + (x * rect.width) / width, y: rect.top + (y * rect.height) / height };
    }, point);
    await page.mouse.click(at.x, at.y);
}

test.beforeEach(async ({ page }) => {
    await installWailsMock(
        page,
        openLibraryMock({
            database: { GradeQuizChecker: VERDICT, LoadTrainingSessions: [], LoadTrainingNumberStats: [], SaveTrainingSession: 1 },
            app: { LegalMoves: [PLAY] }
        })
    );
    await page.goto('/');
    await expect(page.getByTestId('status-bar')).toContainText('3 / 3');
    await page.keyboard.press('Escape'); // le catalogue de visite du premier lancement, s'il s'est ouvert

    await overrideDbMethodByArg(page, 'LoadAnalysis', { [positionA.id]: ANALYSIS_A }, null);
    await overrideDbMethodByArg(page, 'LoadPosition', Object.fromEntries(libraryPositions.map((p) => [p.id, p])), null);
});

test('train decision : le coup se joue sur le plateau, se valide dans le panneau, et le verdict se lit', async ({ page }) => {
    await page.keyboard.press('Space');
    await page.locator('.command-input').fill('train decision');
    await page.keyboard.press('Enter');

    const panel = page.getByTestId('training-panel');
    const validate = panel.getByTestId('training-validate-move');
    await expect(validate).toBeVisible();
    await expect(validate, 'rien n’est joué : rien à valider').toBeDisabled();
    await expect(page.locator('[data-testid="tab-training"]')).toHaveClass(/active/);

    // Deux dés, deux pas, chacun source puis destination.
    await clickPoint(page, 6);
    await clickPoint(page, 3);
    await expect(panel.getByTestId('training-undo-step')).toBeEnabled();
    await clickPoint(page, 4);
    await clickPoint(page, 3);

    await expect(validate).toBeEnabled();
    await validate.click();

    const verdict = panel.getByTestId('training-verdict');
    await expect(verdict).toContainText('42 mMWC');
    await expect(verdict).toContainText('8/5 6/5');
    await expect(panel.getByTestId('training-next'), 'le focus va au geste suivant').toBeFocused();

    const graded = await getWailsCalls(page, 'GradeQuizChecker');
    expect(graded).toHaveLength(1);
    expect(graded[0].args[0]).toBe(positionA.id);
});
