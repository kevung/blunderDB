/**
 * Le panneau Entraînement : ce qu'il montre, et surtout ce qu'il ne doit pas
 * escamoter.
 *
 * Le défaut que ce fichier tient : quand la question suivante ne peut pas être
 * bâtie (la position tirée a été supprimée entre-temps), le panneau
 * rebasculait sur le lanceur. « Terminer » disparaissait, les nombres déjà
 * répondus devenaient inatteignables, et « Démarrer » les écrasait — tout un
 * journal de session partait sans un mot. L'oracle est donc la présence des
 * boutons de la session, pas la valeur d'un store.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetTrainingSeedSources: vi.fn(() => Promise.resolve({})),
    SaveTrainingSeedSource: vi.fn(() => Promise.resolve())
}));
vi.mock('../services/trainingTabService.js', () => ({
    startTrainingSession: vi.fn(),
    revealQuestion: vi.fn(),
    markFault: vi.fn(),
    setTrainingAnswer: vi.fn(),
    nextTrainingQuestion: vi.fn(),
    retryTrainingQuestion: vi.fn(),
    finishTrainingSession: vi.fn(),
    quitTrainingSession: vi.fn(),
    refreshTrainingJournal: vi.fn(() => Promise.resolve()),
    answerDecisionBoard: vi.fn(),
    answerDecisionCube: vi.fn(),
    undoDecisionStep: vi.fn(),
    resetDecisionPlay: vi.fn(),
    refusalMessageKey: (/** @type {string} */ code) => (code && code !== 'noQuestion' ? `training.refusal.${code}` : 'training.noQuestion')
}));

import TrainingPanel from '../components/TrainingPanel.svelte';
import { trainingSessionStore, trainingJournalStore } from '../stores/trainingTabStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { newSession, askQuestion, reveal, recordQuestion, failNextQuestion, answerChosen, attachCorrection, setAnswer } from '../services/trainingTab.js';
import * as serviceModule from '../services/trainingTabService.js';
import { quizPlayStore } from '../stores/quizPlayStore.js';
import { newPlay, playHop } from '../services/quizPlay.js';
import en from '../i18n/locales/en.json';

const service = vi.mocked(serviceModule);

function pipsQuestion() {
    return {
        kind: 'pips',
        key: '7',
        positionId: 7,
        numbers: [
            { type: 'pips.bottom', value: 12 },
            { type: 'pips.top', value: 10 }
        ]
    };
}

/** Une session dont une question a été répondue, et dont la suivante a échoué. */
function sessionWithAFailedNextQuestion() {
    let s = askQuestion(newSession({ exercise: 'pips', seedSource: 'library' }), pipsQuestion(), 0);
    s = recordQuestion(reveal(s, 2000));
    return failNextQuestion(s, 'noQuestion');
}

beforeEach(() => {
    trainingSessionStore.set(null);
    trainingJournalStore.set({});
});

afterEach(() => {
    cleanup();
    trainingSessionStore.set(null);
    quizPlayStore.set(null);
});

describe('au repos', () => {
    test('montre le lanceur et le bilan', () => {
        const { container } = render(TrainingPanel);
        expect(container.querySelector('[data-testid="training-start"]')).not.toBeNull();
        expect(container.querySelector('[data-testid="training-summary-scores"]')).not.toBeNull();
    });
});

describe('pendant une question', () => {
    test('montre « Révéler », « Terminer » et « Quitter », et pas le lanceur', () => {
        trainingSessionStore.set(askQuestion(newSession({ exercise: 'pips', seedSource: 'board' }), pipsQuestion(), 0));
        const { container } = render(TrainingPanel);
        expect(container.querySelector('[data-testid="training-reveal"]')).not.toBeNull();
        expect(container.querySelector('[data-testid="training-finish"]')).not.toBeNull();
        expect(container.querySelector('[data-testid="training-quit"]')).not.toBeNull();
        expect(container.querySelector('[data-testid="training-start"]')).toBeNull();
    });

    test('la consigne de cochage est à l’écran une fois révélée, pas seulement dans une infobulle', () => {
        const asked = askQuestion(newSession({ exercise: 'pips', seedSource: 'board' }), pipsQuestion(), 0);
        trainingSessionStore.set(reveal(asked, 1000));
        const { container } = render(TrainingPanel);
        expect(container.querySelector('.hint')).not.toBeNull();
    });
});

describe('quand la question suivante a échoué', () => {
    test('la session reste à l’écran : l’échec se dit, « Terminer » et « Réessayer » sont là', () => {
        trainingSessionStore.set(sessionWithAFailedNextQuestion());
        const { container } = render(TrainingPanel);
        expect(container.querySelector('[data-testid="training-question-failed"]'), "l'échec doit se dire").not.toBeNull();
        expect(container.querySelector('[data-testid="training-finish"]'), 'les nombres répondus doivent rester enregistrables').not.toBeNull();
        expect(container.querySelector('[data-testid="training-retry"]')).not.toBeNull();
        expect(container.querySelector('[data-testid="training-start"]'), 'le lanceur écraserait la session').toBeNull();
    });

    test('le chronomètre ne compte plus rien et disparaît', () => {
        trainingSessionStore.set(sessionWithAFailedNextQuestion());
        const { container } = render(TrainingPanel);
        expect(container.querySelector('[data-testid="training-clock"]')).toBeNull();
    });
});

describe('la source « base » (ADR-0041 règle 2)', () => {
    // Un bouton qui accepte le clic pour refuser ensuite fait faire le geste
    // avant de dire qu'il ne mène nulle part.
    test('est grisée sans bibliothèque ouverte, et cliquable avec', async () => {
        trainingSessionStore.set(null);
        databasePathStore.set('');
        const closed = render(TrainingPanel);
        closed.getByTestId('training-exercise-bearoff').click();
        await Promise.resolve();
        expect(/** @type {HTMLButtonElement} */ (closed.getByTestId('training-source-library')).disabled).toBe(true);
        expect(/** @type {HTMLButtonElement} */ (closed.getByTestId('training-source-pool')).disabled).toBe(false);
        cleanup();

        databasePathStore.set('/tmp/some.db');
        const open = render(TrainingPanel);
        open.getByTestId('training-exercise-bearoff').click();
        await Promise.resolve();
        expect(/** @type {HTMLButtonElement} */ (open.getByTestId('training-source-library')).disabled).toBe(false);
    });
});

// ── Décision (#323) ──────────────────────────────────────────────────────────

/** @param {'checker'|'cube'} prompt */
function decisionSession(prompt = 'checker') {
    const question = { kind: 'decision', key: '7', positionId: 7, prompt, numbers: [{ type: prompt === 'cube' ? 'decision.cube' : 'decision.checker', value: 0 }] };
    return askQuestion(newSession({ exercise: 'decision', seedSource: 'library' }), question, 0);
}

/** @param {any} extra */
function verdict(extra = {}) {
    return { legal: true, matched: true, notation: '6/4 6/3', best: '6/4 6/3', errorMp: 0, ...extra };
}

const POSITION = {
    player_on_roll: 0,
    board: { points: Array.from({ length: 26 }, (_, i) => (i === 6 ? { checkers: 2, color: 0 } : { checkers: 0, color: -1 })), bearoff: [0, 0] }
};
const PLAY = {
    notation: '6/4 6/3',
    steps: [
        { from: 6, to: 4, hit: false },
        { from: 6, to: 3, hit: false }
    ],
    result: { board: {} }
};

describe('une décision de pions', () => {
    beforeEach(() => vi.clearAllMocks());

    test('se joue sur le plateau, et « Valider » attend un coup complet', async () => {
        trainingSessionStore.set(decisionSession('checker'));
        quizPlayStore.set(newPlay(POSITION, [PLAY]));
        const panel = render(TrainingPanel);
        expect(panel.container.textContent).toContain(en.training.playOnBoard);
        const validate = /** @type {HTMLButtonElement} */ (panel.getByTestId('training-validate-move'));
        expect(validate.disabled).toBe(true);

        quizPlayStore.update((s) => (s ? playHop(playHop(s, 6, 4), 6, 3) : s));
        await tick();
        expect(validate.disabled).toBe(false);
        validate.click();
        expect(service.answerDecisionBoard).toHaveBeenCalled();
    });

    test('« Annuler le pas » et « Recommencer » sont des boutons du panneau', async () => {
        trainingSessionStore.set(decisionSession('checker'));
        quizPlayStore.set(newPlay(POSITION, [PLAY]));
        const panel = render(TrainingPanel);
        const undo = /** @type {HTMLButtonElement} */ (panel.getByTestId('training-undo-step'));
        const reset = /** @type {HTMLButtonElement} */ (panel.getByTestId('training-reset-play'));
        expect(undo.disabled, 'rien à annuler avant le premier pas').toBe(true);
        expect(reset.disabled).toBe(true);

        quizPlayStore.update((s) => (s ? playHop(s, 6, 4) : s));
        await tick();
        undo.click();
        reset.click();
        expect(service.undoDecisionStep).toHaveBeenCalled();
        expect(service.resetDecisionPlay).toHaveBeenCalled();
    });

    test('ni « Révéler » ni case à cocher : c’est le juge qui répond', () => {
        trainingSessionStore.set(answerChosen(decisionSession('checker'), verdict({ errorMp: 42 }), 1000));
        const panel = render(TrainingPanel);
        expect(panel.queryByTestId('training-reveal')).toBeNull();
        expect(panel.container.querySelector('.hint')).toBeNull();
        expect(panel.queryByTestId('training-validate-move'), 'la question est jugée').toBeNull();
    });
});

describe('une action de videau', () => {
    beforeEach(() => vi.clearAllMocks());

    test('trois boutons dans le panneau, chacun est la réponse', () => {
        trainingSessionStore.set(decisionSession('cube'));
        const panel = render(TrainingPanel);
        expect(panel.queryByTestId('training-reveal')).toBeNull();
        panel.getByTestId('training-cube-dt').click();
        expect(service.answerDecisionCube).toHaveBeenCalledWith('dt');
        expect(panel.getByTestId('training-cube-nd').textContent).toContain(en.training.noDouble);
        expect(panel.getByTestId('training-cube-dp').textContent).toContain(en.training.doublePass);
    });

    test('une fois jugée, les trois boutons ne répondent plus', () => {
        trainingSessionStore.set(answerChosen(decisionSession('cube'), verdict({ notation: 'nd' }), 1000));
        const panel = render(TrainingPanel);
        expect(/** @type {HTMLButtonElement} */ (panel.getByTestId('training-cube-nd')).disabled).toBe(true);
    });
});

describe('les trois issues restent distinguées', () => {
    /** @param {any} v */
    function verdictText(v) {
        trainingSessionStore.set(answerChosen(decisionSession('checker'), v, 1000));
        const panel = render(TrainingPanel);
        const text = panel.getByTestId('training-verdict').textContent ?? '';
        cleanup();
        return text;
    }

    test('illégal, non évalué, coût en mMWC — et juste', () => {
        expect(verdictText(verdict({ legal: false, matched: false }))).toContain(en.training.illegal);
        expect(verdictText(verdict({ matched: false }))).toContain(en.training.unranked);
        const cost = verdictText(verdict({ errorMp: 42, best: '24/18 13/11' }));
        expect(cost).toContain(en.training.cost.replace('{mp}', '42'));
        expect(cost).toContain('24/18 13/11');
        expect(verdictText(verdict())).toContain(en.training.right);
    });

    test('hors délai, la correction se montre sans issue inventée', () => {
        const late = reveal(decisionSession('checker'), 15000, { outOfTime: true });
        trainingSessionStore.set(attachCorrection(late, '7', verdict({ legal: false, matched: false, notation: '', best: '24/18 13/11' })));
        const panel = render(TrainingPanel);
        const text = panel.getByTestId('training-verdict').textContent ?? '';
        expect(text).toContain('24/18 13/11');
        expect(text).not.toContain(en.training.illegal);
    });
});

describe('le lanceur et le bilan de Décision', () => {
    test('une seule source possible : aucun choix de source à faire', async () => {
        databasePathStore.set('/tmp/some.db');
        const panel = render(TrainingPanel);
        panel.getByTestId('training-exercise-decision').click();
        await tick();
        expect(panel.queryByTestId('training-source-library')).toBeNull();
    });

    test('le bilan dit le PR de la dernière session', () => {
        trainingJournalStore.set({ decision: { sessions: [{ exercise: 'decision', numbersAsked: 5, faults: 2, deviations: 0, meanDeviation: 0, medianMs: 3000, pr: 6.5 }], numbers: [] } });
        const panel = render(TrainingPanel);
        expect(panel.getByTestId('training-summary-decision').textContent).toContain(en.training.lastPr.replace('{value}', '6.50'));
    });
});

describe('le focus après la réponse (défaut hérité de #321)', () => {
    // Le bouton cliqué disparaît avec la question ouverte : le focus retombait
    // sur la page, et J / K parcouraient la liste sous la question.
    test('après « Valider », le focus est sur « Suivante »', async () => {
        const question = {
            kind: 'bearoff',
            key: 'pool:1',
            numbers: [
                { type: 'epc.bottom', value: 20, tolerance: 0.5, precision: 1 },
                { type: 'epc.top', value: 30, tolerance: 0.5, precision: 1 }
            ]
        };
        const asked = setAnswer(askQuestion(newSession({ exercise: 'bearoff', seedSource: 'pool' }), question, 0), 0, '20');
        trainingSessionStore.set(asked);
        const panel = render(TrainingPanel);
        panel.getByTestId('training-reveal').focus();
        trainingSessionStore.set(reveal(asked, 1000));
        await tick();
        expect(document.activeElement).toBe(panel.getByTestId('training-next'));
    });

    test('après le verdict d’une décision, aussi', async () => {
        trainingSessionStore.set(decisionSession('cube'));
        const panel = render(TrainingPanel);
        panel.getByTestId('training-cube-nd').focus();
        trainingSessionStore.set(answerChosen(decisionSession('cube'), verdict({ notation: 'nd' }), 1000));
        await tick();
        expect(document.activeElement).toBe(panel.getByTestId('training-next'));
    });
});

describe('une question d’Évaluation (#322)', () => {
    /** @param {any} [extra] */
    function evaluationQuestion(extra = {}) {
        return {
            kind: 'evaluation',
            key: 'pool:4',
            cubeVerdict: 'too_good',
            regime: 'evaluated',
            depth: '2-ply',
            epc: null,
            numbers: [
                { type: 'eval.win', value: 93.4, tolerance: 5, precision: 1 },
                { type: 'eval.cube', value: 0, mode: 'chosen', answer: 'nd' }
            ],
            ...extra
        };
    }

    /** @param {any} [extra] */
    function evaluationSession(extra) {
        return askQuestion(newSession({ exercise: 'evaluation', seedSource: 'pool' }), /** @type {any} */ (evaluationQuestion(extra)), 0);
    }

    beforeEach(() => vi.clearAllMocks());

    test('un champ pour les chances de gain et trois boutons pour le videau, dans la même question', () => {
        trainingSessionStore.set(evaluationSession());
        const panel = render(TrainingPanel);
        expect(panel.getByLabelText(en.training.numbers.evalWin)).toBeTruthy();
        for (const id of ['nd', 'dt', 'dp']) expect(panel.getByTestId(`training-choice-1-${id}`)).toBeTruthy();
        expect(panel.queryByTestId('training-answer-1'), 'le videau ne se tape pas').toBeNull();
        expect(panel.getByText(en.training.toleranceEvaluation)).toBeTruthy();
        expect(panel.getByTestId('training-reveal').textContent).toContain(en.training.validate);
    });

    test('choisir retient l’option sans juger, et porte le focus sur « Valider »', async () => {
        trainingSessionStore.set(evaluationSession());
        const panel = render(TrainingPanel);
        await fireEvent.click(panel.getByTestId('training-choice-1-dt'));
        expect(service.setTrainingAnswer).toHaveBeenCalledWith(1, 'dt');
        expect(service.revealQuestion).not.toHaveBeenCalled();
        expect(document.activeElement).toBe(panel.getByTestId('training-reveal'));

        trainingSessionStore.set(setAnswer(evaluationSession(), 1, 'dt'));
        await tick();
        expect(panel.getByTestId('training-choice-1-dt').getAttribute('aria-pressed')).toBe('true');
        expect(panel.getByTestId('training-choice-1-nd').getAttribute('aria-pressed')).toBe('false');
    });

    test('ENTRÉE dans le champ va au videau tant qu’il n’est pas choisi, puis valide', async () => {
        trainingSessionStore.set(setAnswer(evaluationSession(), 0, '90'));
        const panel = render(TrainingPanel);
        await fireEvent.keyDown(panel.getByTestId('training-answer-0'), { key: 'Enter' });
        expect(service.revealQuestion, 'une action de videau pas encore donnée ne se juge pas').not.toHaveBeenCalled();
        expect(document.activeElement).toBe(panel.getByTestId('training-choice-1-nd'));

        trainingSessionStore.set(setAnswer(setAnswer(evaluationSession(), 0, '90'), 1, 'nd'));
        await tick();
        await fireEvent.keyDown(panel.getByTestId('training-answer-0'), { key: 'Enter' });
        expect(service.revealQuestion).toHaveBeenCalledTimes(1);
    });

    test('après « Valider » : le verdict du moteur à quatre issues, la faute marquée, d’où vient la vérité', () => {
        let s = setAnswer(setAnswer(evaluationSession(), 0, '80'), 1, 'dp');
        trainingSessionStore.set(reveal(s, 3000));
        const panel = render(TrainingPanel);
        const truth = panel.getByTestId('training-truth-1');
        expect(truth.textContent).toContain(en.cube.verdicts.too_good);
        expect(truth.textContent, 'la faute se dit par un glyphe').toContain('×');
        expect(panel.getByTestId('training-truth-0').textContent).toContain('93.4');
        expect(panel.getByTestId('training-choice-1-dp').hasAttribute('disabled')).toBe(true);
        expect(panel.getByTestId('training-truth-source').textContent).toContain(en.training.truthEvaluated.replace('{depth}', '2-ply'));
        expect(panel.queryByTestId('training-epc'), 'pas d’EPC exact, rien à montrer').toBeNull();
    });

    test('l’EPC n’apparaît qu’après la réponse, et seulement quand la position en a un exact', async () => {
        const epc = { bottom: { epc: { epc: 12.34 } }, top: { epc: { epc: 9.87 } } };
        const open = evaluationSession({ epc, regime: 'exact', depth: '' });
        trainingSessionStore.set(open);
        const panel = render(TrainingPanel);
        expect(panel.queryByTestId('training-epc'), 'jamais demandé, jamais montré avant la réponse').toBeNull();

        trainingSessionStore.set(reveal(setAnswer(setAnswer(open, 0, '93'), 1, 'nd'), 1000));
        await tick();
        const shown = panel.getByTestId('training-epc').textContent ?? '';
        expect(shown).toContain('12.3');
        expect(shown).toContain('9.9');
        expect(panel.getByTestId('training-truth-source').textContent).toContain(en.training.truthExact);
        expect(panel.getByTestId('training-truth-1').textContent).not.toContain('×');
    });

    test('le lanceur propose Évaluation et ses trois sources', async () => {
        const panel = render(TrainingPanel);
        await fireEvent.click(panel.getByTestId('training-exercise-evaluation'));
        for (const source of ['pool', 'board', 'library']) expect(panel.getByTestId(`training-source-${source}`)).toBeTruthy();
    });
});
