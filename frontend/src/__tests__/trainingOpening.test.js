/**
 * L'exercice Ouverture (ADR-0094) : une question, deux temps joués au plateau —
 * l'ouverture, puis la réponse dans la position que le livre suit.
 */
import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    LoadPosition: vi.fn(() => Promise.resolve(null)),
    SaveTrainingSession: vi.fn(() => Promise.resolve(1)),
    LoadTrainingSessions: vi.fn(() => Promise.resolve([])),
    LoadTrainingNumberStats: vi.fn(() => Promise.resolve([])),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    GradeQuizChecker: vi.fn(),
    GradeQuizCheckerMove: vi.fn(),
    GradeQuizCube: vi.fn()
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    GenerateBearoffQuestion: vi.fn(),
    GenerateEvaluationQuestion: vi.fn(),
    LegalMoves: vi.fn(() => Promise.resolve([])),
    OpeningBookQuestion: vi.fn(),
    OpeningBookGrade: vi.fn()
}));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));
vi.mock('../utils/logger.js', () => ({ logger: { error: vi.fn(), log: vi.fn() } }));

import * as dbModule from '../../wailsjs/go/database/Database.js';
import * as appModule from '../../wailsjs/go/gui/App.js';
import * as databaseServiceModule from '../services/databaseService.js';
import { quizPlayStore } from '../stores/quizPlayStore.js';
import { playHop } from '../services/quizPlay.js';
import { positionStore } from '../stores/positionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { trainingSessionStore } from '../stores/trainingTabStore.js';
import { tMsg } from '../i18n';
import { startTrainingSession, nextTrainingQuestion, finishTrainingSession, quitTrainingSession, answerDecisionBoard, revealQuestion } from '../services/trainingTabService.js';
import { exerciseForCommand } from '../services/trainingTab.js';

const db = vi.mocked(dbModule);
const app = vi.mocked(appModule);
const statusBar = vi.mocked(databaseServiceModule);

/** @returns {any} */
function current() {
    return get(trainingSessionStore);
}

/** @returns {any} */
function board() {
    return {
        player_on_roll: 0,
        board: {
            points: Array.from({ length: 26 }, (_, i) => {
                if (i === 6) return { checkers: 2, color: 0 };
                if (i === 20) return { checkers: 2, color: 1 };
                return { checkers: 0, color: -1 };
            }),
            bearoff: [0, 0]
        }
    };
}

const RESULT = { points: [], bearoff: [0, 0] };
const PLAY = {
    notation: '6/4 6/3',
    steps: [
        { from: 6, to: 4, hit: false },
        { from: 6, to: 3, hit: false }
    ],
    result: { board: RESULT }
};

const RANKED = [
    { move: '8/5 6/5', gap: 0, ci95: 0, tied: true },
    { move: '24/23 13/10', gap: 0.004, ci95: 0.008, tied: true },
    { move: '13/10 6/5', gap: 0.06, ci95: 0.008, tied: false }
];

function playOnBoard() {
    quizPlayStore.update((s) => (s ? playHop(playHop(s, 6, 4), 6, 3) : s));
}

beforeEach(() => {
    vi.clearAllMocks();
    app.LegalMoves.mockResolvedValue(/** @type {any} */ ([PLAY]));
    app.OpeningBookQuestion.mockResolvedValue(/** @type {any} */ ({ empty: false, opening: '31', reply: '21', score: '4-2', position: board() }));
    quitTrainingSession();
    quizPlayStore.set(null);
    databasePathStore.set('/tmp/some.db');
    positionStore.set(board());
});

describe('une session d’Ouverture', () => {
    test('le mot de la commande `train` nomme l’exercice', () => {
        expect(exerciseForCommand('ouverture')).toBe('opening');
        expect(exerciseForCommand('opening')).toBe('opening');
    });

    test('un livre vide refuse de démarrer en le disant, sans erreur', async () => {
        app.OpeningBookQuestion.mockResolvedValue(/** @type {any} */ ({ empty: true }));
        expect(await startTrainingSession({ exercise: 'opening' })).toBe(false);
        expect(statusBar.setStatusBarMessage).toHaveBeenCalledWith(tMsg('training.refusal.bookEmpty'));
        expect(current()).toBeNull();
    });

    test('une ouverture fausse montre l’analyse, puis la réponse se pose après le meilleur coup du livre', async () => {
        expect(await startTrainingSession({ exercise: 'opening' })).toBe(true);
        expect(current().question).toMatchObject({ kind: 'opening', step: 'opening', opening: '31', reply: '21', score: '4-2' });

        playOnBoard();
        app.OpeningBookGrade.mockResolvedValueOnce(
            /** @type {any} */ ({
                legal: true,
                notation: '13/10 6/5',
                played: 2,
                right: false,
                ranked: RANKED,
                games: 216,
                signature: 'test',
                next: { openingPlay: '8/5 6/5', followsBest: true, reply: '21', position: board() }
            })
        );
        await answerDecisionBoard();
        const draws = app.OpeningBookQuestion.mock.calls.length;
        expect(app.OpeningBookGrade.mock.calls[0][0]).toMatchObject({ opening: '31', openingPlay: '', reply: '21', score: '4-2', board: RESULT });
        expect(current().revealed).toBe(true);
        expect(current().faults).toEqual([true]);

        await nextTrainingQuestion();
        expect(current().question).toMatchObject({ kind: 'opening', step: 'reply', openingPlay: '8/5 6/5', followsBest: true });
        // La réponse naît du verdict, pas d'un nouveau tirage.
        expect(app.OpeningBookQuestion).toHaveBeenCalledTimes(draws);

        playOnBoard();
        app.OpeningBookGrade.mockResolvedValueOnce(/** @type {any} */ ({ legal: true, notation: '13/11 6/5', played: 0, right: true, ranked: RANKED }));
        await answerDecisionBoard();
        expect(app.OpeningBookGrade.mock.calls[1][0]).toMatchObject({ openingPlay: '8/5 6/5', reply: '21' });
        expect(current().faults).toEqual([false]);

        await nextTrainingQuestion();
        expect(current().question.step).toBe('opening');
        const row = await finishTrainingSession();
        expect(row?.items.map((/** @type {any} */ item) => [item.numberType, item.wrong])).toEqual([
            ['opening.first', true],
            ['opening.reply', false]
        ]);
        expect(row?.pr).toBe(0);
        expect(db.SaveTrainingSession).toHaveBeenCalledTimes(1);
    });

    test('hors délai, la correction vient du livre et la réponse suit quand même', async () => {
        expect(await startTrainingSession({ exercise: 'opening', limitSeconds: 15 })).toBe(true);
        app.OpeningBookGrade.mockResolvedValueOnce(
            /** @type {any} */ ({ legal: false, notation: '', played: -1, right: false, ranked: RANKED, next: { openingPlay: '8/5 6/5', followsBest: true, reply: '21', position: board() } })
        );
        revealQuestion({ outOfTime: true });
        await new Promise((resolve) => setTimeout(resolve, 0));
        expect(app.OpeningBookGrade.mock.calls[0][0].board).toBeUndefined();
        expect(current().verdict?.ranked).toHaveLength(3);
        await nextTrainingQuestion();
        expect(current().question.step).toBe('reply');
    });

    test('« Terminer » rend le plateau à celui d’avant la séance, dés d’exercice ôtés', async () => {
        const before = { ...board(), id: 7, dice: [0, 0] };
        positionStore.set(before);
        app.OpeningBookQuestion.mockResolvedValue(/** @type {any} */ ({ empty: false, opening: '31', reply: '21', score: '4-2', position: { ...board(), id: 0, dice: [3, 1] } }));
        expect(await startTrainingSession({ exercise: 'opening' })).toBe(true);
        expect(get(positionStore).dice).toEqual([3, 1]);
        await finishTrainingSession();
        expect(get(positionStore)).toMatchObject({ id: 7, dice: [0, 0] });
        expect(get(quizPlayStore)).toBeNull();
    });
});
