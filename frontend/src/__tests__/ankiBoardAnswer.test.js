/**
 * ankiBoardAnswer.test.js — « répondre au damier » : la note proposée, l'option par paquet et le
 * damier que l'on ne rend qu'à condition de l'avoir posé.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

const meta = vi.hoisted(() => ({ value: /** @type {Record<string,string>} */ ({}) }));

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    LoadMetadata: vi.fn(() => Promise.resolve({ ...meta.value })),
    SaveMetadata: vi.fn((m) => {
        meta.value = { ...m };
        return Promise.resolve();
    }),
    GradeQuizChecker: vi.fn(),
    GetLibrarySettings: vi.fn(() => Promise.resolve({ errorThresholdMP: 20, blunderThresholdMP: 40 }))
}));

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    LegalMoves: vi.fn(() => Promise.resolve([{ steps: [{ from: 8, to: 3, hit: false }], result: { board: {} } }]))
}));

import * as bg from '../services/ankiBoardAnswer.js';
import { quizPlayStore } from '../stores/quizPlayStore.js';
import { hideAnkiAnswer } from '../stores/ankiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { GradeQuizChecker } from '../../wailsjs/go/database/Database.js';

const board = { points: Array.from({ length: 26 }, () => ({ color: -1, checkers: 0 })), bearoff: [0, 0] };
const card = (id, extra = {}) => ({ card: { id }, position: { id: 10, board, dice: [3, 1], decision_type: 0, player_on_roll: 0, ...extra } });

describe('la note proposée', () => {
    test('une bonne réponse rapide est Facile, lente Bien', () => {
        const v = { legal: true, matched: true, errorMp: 0 };
        expect(bg.suggestRating(v, 2000)).toBe(4);
        expect(bg.suggestRating(v, bg.FAST_ANSWER_MS + 1)).toBe(3);
    });
    test('une erreur est Difficile, un blunder ou un coup illégal À revoir', () => {
        expect(bg.suggestRating({ legal: true, matched: true, errorMp: 99 }, 0)).toBe(2);
        expect(bg.suggestRating({ legal: true, matched: true, errorMp: 100 }, 0)).toBe(1);
        expect(bg.suggestRating({ legal: false, matched: false, errorMp: 0 }, 0)).toBe(1);
    });
    test("un coup légal non classé n'a pas de coût connu : pas de suggestion", () => {
        expect(bg.suggestRating({ legal: true, matched: false, errorMp: 0 }, 0)).toBeNull();
    });
});

describe('le seuil du blunder', () => {
    test('suit le seuil donné, pas un seuil fixe', () => {
        const v = { legal: true, matched: true, errorMp: 60 };
        expect(bg.suggestRating(v, 0, 50)).toBe(1);
        expect(bg.suggestRating(v, 0, 100)).toBe(2);
    });

    test('la validation lit le seuil des Réglages de la bibliothèque', async () => {
        meta.value = {};
        quizPlayStore.set(null);
        bg.disarmBoardAnswer();
        await bg.setBoardAnswer(9, true);
        const withBoard = card(9);
        await bg.armBoardAnswer({ id: 9 }, withBoard);
        quizPlayStore.update((st) => ({ ...st, steps: [{ from: 8, to: 3, hit: false }] }));
        // Seuil mocké à 40 : 60 mp est un blunder, alors que le défaut (100) dirait Difficile.
        /** @type {any} */ (GradeQuizChecker).mockResolvedValue({ legal: true, matched: true, errorMp: 60, notation: '8/3' });
        await bg.validateBoardAnswer(withBoard);
        expect(get(bg.ankiBoardAnswerStore)).toMatchObject({ phase: 'graded', suggested: 1 });
    });
});

describe("l'option décochée", () => {
    test('écrit « 0 » : le stockage ne fait que remplacer, une clé retirée reviendrait', async () => {
        await bg.setBoardAnswer(6, true);
        await bg.setBoardAnswer(6, false);
        expect(meta.value.anki_board_answer_6).toBe('0');
        // Relecture à froid, comme au relancement.
        databasePathStore.set('/autre.db');
        expect(await bg.boardAnswerEnabled(6)).toBe(false);
    });

    test("ne passe pas d'une base à l'autre", async () => {
        await bg.setBoardAnswer(7, true);
        expect(await bg.boardAnswerEnabled(7)).toBe(true);
        meta.value = {};
        databasePathStore.set('/b.db');
        expect(await bg.boardAnswerEnabled(7)).toBe(false);
    });
});

describe('les cartes qui se jouent au damier', () => {
    test('une décision de pions oui, un videau ou un jet nul non', () => {
        expect(bg.isBoardPlayable(card(1))).toBe(true);
        expect(bg.isBoardPlayable(card(1, { decision_type: 1 }))).toBe(false);
        expect(bg.isBoardPlayable(card(1, { dice: [0, 0] }))).toBe(false);
        expect(bg.isBoardPlayable({ card: { id: 1 }, position: null })).toBe(false);
    });
});

describe("l'armement du damier", () => {
    beforeEach(() => {
        meta.value = {};
        hideAnkiAnswer();
        quizPlayStore.set(null);
        bg.disarmBoardAnswer();
    });

    test('sans option, le damier reste libre : auto-notation', async () => {
        await bg.armBoardAnswer({ id: 1 }, card(1));
        expect(get(quizPlayStore)).toBeNull();
        expect(get(bg.ankiBoardAnswerStore)).toBeNull();
    });

    test("l'option par paquet arme le damier, et suit la base", async () => {
        await bg.setBoardAnswer(2, true);
        expect(meta.value.anki_board_answer_2).toBe('1');
        await bg.armBoardAnswer({ id: 2 }, card(1));
        expect(get(quizPlayStore)).not.toBeNull();
        expect(get(bg.ankiBoardAnswerStore)).toEqual({ phase: 'play' });
    });

    test("un paquet de scores n'est jamais armé", async () => {
        await bg.setBoardAnswer(3, true);
        await bg.armBoardAnswer({ id: 3, sourceType: 'scores' }, card(1));
        expect(get(quizPlayStore)).toBeNull();
    });

    test("le damier de l'Entraînement n'est pas rendu par une carte qui ne l'a pas posé", async () => {
        const training = { steps: [], plays: [] };
        quizPlayStore.set(/** @type {any} */ (training));
        await bg.armBoardAnswer({ id: 4 }, card(1));
        bg.disarmBoardAnswer();
        expect(get(quizPlayStore)).toBe(training);
    });

    test('relire la même carte ne rejoue pas le coup', async () => {
        await bg.setBoardAnswer(5, true);
        await bg.armBoardAnswer({ id: 5 }, card(7));
        const first = get(quizPlayStore);
        await bg.armBoardAnswer({ id: 5 }, card(7));
        expect(get(quizPlayStore)).toBe(first);
    });
});
