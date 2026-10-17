/**
 * L'exercice Ouverture (ADR-0094) : le jet choisi au lanceur — 1st roll, le
 * coup d'ouverture ; 2nd roll, la réponse après un coup à égalité avec le meilleur —,
 * joué au plateau et jugé contre le livre.
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
    OpeningBookStatus: vi.fn(),
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
    app.OpeningBookStatus.mockResolvedValue(/** @type {any} */ ({ lines: 9, scores: ['4-2', '2-1', '1-2'] }));
    quitTrainingSession();
    quizPlayStore.set(null);
    databasePathStore.set('/tmp/some.db');
    positionStore.set(board());
});

describe('une session d’Ouverture', () => {
    test('ne tire que les clés que le livre tient, selon les catégories choisies', async () => {
        app.OpeningBookStatus.mockResolvedValue(/** @type {any} */ ({ lines: 9, scores: ['4-2', '2-1', '1-2'] }));
        await startTrainingSession({ exercise: 'opening', seedSource: 'first', categories: ['money', 'dmp', 'gammonSave'] });
        expect(app.OpeningBookQuestion.mock.calls[0][0]).toEqual({ roll: 'first', scores: ['2-1'] });
        quitTrainingSession();
        await startTrainingSession({ exercise: 'opening', seedSource: 'first', categories: ['grid'] });
        expect(app.OpeningBookQuestion.mock.calls.at(-1)?.[0]).toEqual({ roll: 'first', scores: ['4-2', '2-1', '1-2'] });
    });

    test('Money et DMP tirés dès que le livre les tient', async () => {
        app.OpeningBookStatus.mockResolvedValue(/** @type {any} */ ({ lines: 9, scores: ['money', '1-1', '2-1', '1-2'] }));
        await startTrainingSession({ exercise: 'opening', seedSource: 'first' });
        expect(app.OpeningBookQuestion.mock.calls[0][0].scores).toHaveLength(1);
        expect(['money', '1-1', '2-1', '1-2']).toContain(app.OpeningBookQuestion.mock.calls[0][0].scores?.[0]);
    });

    test('chaque tirage joue une seule catégorie, et toutes les catégories cochées sortent à égalité', async () => {
        app.OpeningBookStatus.mockResolvedValue(/** @type {any} */ ({ lines: 9, scores: ['money', '1-1', '2-1', '1-2', '4-2'] }));
        const random = vi.spyOn(Math, 'random');
        const sent = [];
        for (const r of [0, 0.3, 0.55, 0.8]) {
            random.mockReturnValue(r);
            app.OpeningBookQuestion.mockClear();
            await startTrainingSession({ exercise: 'opening', seedSource: 'first', categories: ['money', 'dmp', 'gammonSave', 'gammonGo'] });
            sent.push(app.OpeningBookQuestion.mock.calls[0][0].scores);
            quitTrainingSession();
        }
        random.mockRestore();
        expect(sent).toEqual([['money'], ['1-1'], ['2-1'], ['1-2']]);
    });

    test('aucune catégorie tenue par le livre : refus nommé, aucun tirage', async () => {
        const started = await startTrainingSession({ exercise: 'opening', seedSource: 'first', categories: ['money'] });
        expect(started).toBe(false);
        expect(app.OpeningBookQuestion).not.toHaveBeenCalled();
    });

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

    test('1st roll : chaque question est un coup d’ouverture, tiré à nouveau', async () => {
        expect(await startTrainingSession({ exercise: 'opening', seedSource: 'first' })).toBe(true);
        expect(app.OpeningBookQuestion.mock.calls[0][0].roll).toBe('first');
        expect(['2-1', '1-2']).toContain(app.OpeningBookQuestion.mock.calls[0][0].scores?.[0]);
        expect(current().seedSource).toBe('first');
        expect(current().question).toMatchObject({ kind: 'opening', step: 'opening', opening: '31', reply: '', openingPlay: '', score: '4-2' });

        playOnBoard();
        app.OpeningBookGrade.mockResolvedValueOnce(/** @type {any} */ ({ legal: true, notation: '13/10 6/5', played: 2, right: false, ranked: RANKED, games: 216, signature: 'test' }));
        await answerDecisionBoard();
        expect(app.OpeningBookGrade.mock.calls[0][0]).toMatchObject({ opening: '31', openingPlay: '', score: '4-2', board: RESULT });
        expect(current().revealed).toBe(true);
        expect(current().faults).toEqual([true]);

        const draws = app.OpeningBookQuestion.mock.calls.length;
        await nextTrainingQuestion();
        expect(current().question.step).toBe('opening');
        expect(current().revealed).toBe(false);
        expect(app.OpeningBookQuestion.mock.calls.length, 'la suivante est un nouveau tirage').toBeGreaterThanOrEqual(draws);
        const row = await finishTrainingSession();
        expect(row?.items.map((/** @type {any} */ item) => [item.numberType, item.wrong])).toEqual([['opening.first', true]]);
        expect(db.SaveTrainingSession).toHaveBeenCalledTimes(1);
    });

    test('2nd roll : la réponse se pose après le coup d’ouverture du livre, et se juge comme telle', async () => {
        app.OpeningBookQuestion.mockResolvedValue(/** @type {any} */ ({ empty: false, roll: 'second', opening: '31', openingPlay: '8/5 6/5', reply: '21', score: '4-2', position: board() }));
        expect(await startTrainingSession({ exercise: 'opening', seedSource: 'second' })).toBe(true);
        expect(app.OpeningBookQuestion.mock.calls[0][0].roll).toBe('second');
        expect(['2-1', '1-2']).toContain(app.OpeningBookQuestion.mock.calls[0][0].scores?.[0]);
        expect(current().question).toMatchObject({ kind: 'opening', step: 'reply', opening: '31', openingPlay: '8/5 6/5', reply: '21' });
        expect(current().question.numbers[0].type).toBe('opening.reply');

        playOnBoard();
        app.OpeningBookGrade.mockResolvedValueOnce(/** @type {any} */ ({ legal: true, notation: '13/11 6/5', played: 0, right: true, ranked: RANKED }));
        await answerDecisionBoard();
        expect(app.OpeningBookGrade.mock.calls[0][0]).toMatchObject({ opening: '31', openingPlay: '8/5 6/5', reply: '21', score: '4-2' });
        expect(current().faults).toEqual([false]);
        await nextTrainingQuestion();
        expect(current().question.step).toBe('reply');
        const row = await finishTrainingSession();
        expect(row?.items.map((/** @type {any} */ item) => item.numberType)).toEqual(['opening.reply']);
    });

    test('un jet inconnu (mémoire d’une version passée) retombe sur 1st roll', async () => {
        expect(await startTrainingSession({ exercise: 'opening', seedSource: 'pool' })).toBe(true);
        expect(current().seedSource).toBe('first');
        expect(app.OpeningBookQuestion.mock.calls[0][0].roll).toBe('first');
        expect(['2-1', '1-2']).toContain(app.OpeningBookQuestion.mock.calls[0][0].scores?.[0]);
    });

    test('hors délai, la correction vient du livre', async () => {
        expect(await startTrainingSession({ exercise: 'opening', limitSeconds: 15 })).toBe(true);
        app.OpeningBookGrade.mockResolvedValueOnce(/** @type {any} */ ({ legal: false, notation: '', played: -1, right: false, ranked: RANKED }));
        revealQuestion({ outOfTime: true });
        await new Promise((resolve) => setTimeout(resolve, 0));
        expect(app.OpeningBookGrade.mock.calls[0][0].board).toBeUndefined();
        expect(current().verdict?.ranked).toHaveLength(3);
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
