/**
 * training-evaluation.spec.js — l'exercice Évaluation, dans l'application réelle.
 *
 * Le parcours de l'utilisateur : taper `train evaluation`, trouver la question
 * dans l'onglet Entraînement, SAISIR les chances de gain, CHOISIR l'action de
 * videau, valider, et lire la vérité. Ce qu'un test de composant ne voit pas et
 * qui se joue ici : la commande traverse la ligne de commande et l'application,
 * la position engendrée arrive sur le plateau sans quitter l'onglet, et les
 * deux nombres d'une même question partent au juge en un seul geste.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, getWailsCalls } from './helpers/wailsMock.js';
import { openLibraryMock, positionA } from './helpers/fixtures.js';

// La question telle que le moteur la rend : une course, 71,3 % pour le joueur
// au trait, « double, prend », et un EPC exact à montrer après la réponse.
const QUESTION = {
    generated: true,
    refusal: '',
    source: 'pool',
    plies: 3,
    position: { ...positionA, id: 0, dice: [0, 0] },
    winChance: 71.3,
    cubeVerdict: 'double_take',
    cubeAnswer: 'dt',
    regime: 'evaluated',
    depth: '2-ply',
    epc: { bottom: { epc: { epc: 21.7 } }, top: { epc: { epc: 30.2 } } }
};

test.beforeEach(async ({ page }) => {
    await installWailsMock(
        page,
        openLibraryMock({
            database: { LoadTrainingSessions: [], LoadTrainingNumberStats: [], SaveTrainingSession: 1 },
            app: { GenerateEvaluationQuestion: QUESTION }
        })
    );
    await page.goto('/');
    await expect(page.getByTestId('status-bar')).toContainText('3 / 3');
    await page.keyboard.press('Escape'); // le catalogue de visite du premier lancement, s'il s'est ouvert
});

test('train evaluation : on saisit les chances, on choisit le videau, et la vérité se lit', async ({ page }) => {
    await page.keyboard.press('Space');
    await page.locator('.command-input').fill('train evaluation');
    await page.keyboard.press('Enter');

    const panel = page.getByTestId('training-panel');
    const win = panel.getByTestId('training-answer-0');
    await expect(win).toBeVisible();
    await expect(page.locator('[data-testid="tab-training"]')).toHaveClass(/active/);
    await expect(panel.getByTestId('training-epc'), 'l’EPC n’est pas demandé, et pas montré avant la réponse').toHaveCount(0);

    await win.fill('68');
    // ENTRÉE avant d'avoir choisi le videau : le focus va au videau, rien n'est jugé.
    await win.press('Enter');
    await expect(panel.getByTestId('training-choice-1-nd')).toBeFocused();
    await expect(panel.getByTestId('training-truth-0')).toBeEmpty();

    await panel.getByTestId('training-choice-1-dt').click();
    const validate = panel.getByTestId('training-reveal');
    await expect(validate, 'le geste suivant est de valider').toBeFocused();
    await validate.click();

    // Les deux nombres jugés : 68 est dans les cinq points de 71,3, et « prend » est le bouton juste.
    await expect(panel.getByTestId('training-truth-0')).toContainText('71.3');
    await expect(panel.getByTestId('training-truth-0')).not.toContainText('×');
    await expect(panel.getByTestId('training-truth-1')).not.toContainText('×');
    await expect(panel.getByTestId('training-choice-1-dt')).toBeDisabled();
    await expect(panel.getByTestId('training-epc')).toContainText('21.7');
    await expect(panel.getByTestId('training-truth-source')).toContainText('2-ply');
    await expect(panel.getByTestId('training-next'), 'le focus va au geste suivant').toBeFocused();

    const calls = await getWailsCalls(page, 'GenerateEvaluationQuestion');
    expect(calls.length).toBeGreaterThanOrEqual(1);
    expect(calls[0].args[0]).toEqual({ source: 'pool' });
});
