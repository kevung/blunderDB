/**
 * TranscriptionPanel.boardPlay.test.js — T2.3 et T2.4 dans le panneau.
 *
 * Le réducteur est éprouvé à part (transcriptionPlay.test.js) et le clic qui le
 * fait avancer aussi (boardInteractions.test.js). Ce qui ne se voit qu'ici est
 * la couture : aucun coup armé sans jet, le coup armé du jet saisi avec son
 * rappel de validation, l'enregistrement seulement à la validation. Le coup
 * hors des règles est dans TranscriptionPanel.directPlay.test.js.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListTranscriptions: vi.fn().mockResolvedValue([]),
    CreateTranscription: vi.fn(),
    OpenTranscription: vi.fn(),
    ApplyTranscriptionGesture: vi.fn(),
    TranscriptionMAT: vi.fn().mockResolvedValue('')
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    LegalMoves: vi.fn(),
    EvaluatePositionImmediate: vi.fn()
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetPruneK: vi.fn().mockResolvedValue(0)
}));

import { ApplyTranscriptionGesture, ListTranscriptions } from '../../wailsjs/go/database/Database.js';
import { LegalMoves, EvaluatePositionImmediate } from '../../wailsjs/go/gui/App.js';

import TranscriptionPanel from '../components/TranscriptionPanel.svelte';
import { transcriptionListStore, transcriptionStore, clearTranscription } from '../stores/transcriptionStore.js';
import { quizPlayStore, quizPlayValidateStore } from '../stores/quizPlayStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';
import { selectSource, playHop } from '../services/quizPlay.js';

const BLACK = 0;
const WHITE = 1;

function positionWith(/** @type {any} */ stacks, mover = BLACK) {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    for (const [point, [checkers, color]] of Object.entries(stacks)) points[Number(point)] = { checkers, color };
    return {
        id: 0,
        board: { points, bearoff: [0, 0] },
        cube: { owner: -1, value: 0 },
        dice: [0, 0],
        score: [7, 7],
        player_on_roll: mover,
        decision_type: 0
    };
}

const POSITION = positionWith({ 13: [5, BLACK], 8: [3, BLACK], 6: [5, BLACK], 12: [2, WHITE] });

const step = (/** @type {number} */ from, /** @type {number} */ to) => ({ from, to, hit: false });
const play = (/** @type {any[]} */ ...steps) => ({ steps, notation: steps.map((s) => `${s.from}/${s.to}`).join(' '), result: {} });

// Ce que `LegalMoves` rendrait, jet par jet. Tout ce qui n'est pas nommé ici est
// un jet sans coup, et il s'écarte de lui-même de l'union.
/** @type {Record<string, any[]>} */
const PLAYS_BY_ROLL = {
    61: [play(step(13, 7), step(8, 7))],
    62: [play(step(13, 7), step(13, 11))],
    66: [play(step(13, 7), step(13, 7), step(8, 2), step(8, 2))],
    21: [play(step(13, 11), step(13, 12))]
};

function annotated(extra = {}) {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice' }, actions: [], cursor: 0 },
        actions: [],
        games: [],
        next: { expects: 'checker', side: BLACK, position: POSITION, crawford: false },
        score: [0, 0],
        cursor: 0,
        ...extra
    };
}

const state = (/** @type {any} */ ann) => ({ id: 1, annotated: ann });
const gestures = () => /** @type {any} */ (ApplyTranscriptionGesture).mock.calls.map((/** @type {any} */ call) => call[1]);

async function settle(times = 10) {
    for (let i = 0; i < times; i++) {
        await tick();
        await Promise.resolve();
    }
}

function press(/** @type {string} */ code) {
    const digit = /^Digit([1-9])$/.exec(code);
    return fireEvent.keyDown(document, { code, key: digit ? digit[1] : code });
}

/** Le panneau ouvert sur un brouillon, le jet tapé, et le plateau armé du coup à jouer. */
async function armed(first = 'Digit6', second = 'Digit1') {
    transcriptionStore.set(state(annotated()));
    render(TranscriptionPanel);
    await tick();
    document.getElementById('transcriptionPanel')?.focus();
    await press(first);
    await press(second);
    await vi.waitFor(() => expect(get(quizPlayStore)?.rolled).toBeTruthy());
    return /** @type {any} */ (get(quizPlayStore));
}

/** Un pas joué au plateau, tel que le clic le joue. */
function hop(/** @type {number} */ from, /** @type {number} */ to) {
    quizPlayStore.update((/** @type {any} */ s) => playHop(selectSource(s, from), from, to));
}

const diceCells = () => [...document.querySelectorAll('.dice-triangle button')];
const cell = (/** @type {string} */ label) => /** @type {HTMLButtonElement} */ (diceCells().find((b) => b.textContent.trim() === label));

beforeEach(() => {
    vi.clearAllMocks();
    /** @type {any} */ (ListTranscriptions).mockResolvedValue([]);
    /** @type {any} */ (ApplyTranscriptionGesture).mockImplementation(() => Promise.resolve(state(annotated())));
    /** @type {any} */ (LegalMoves).mockImplementation((/** @type {any} */ pos) => {
        const [a, b] = pos.dice;
        return Promise.resolve(PLAYS_BY_ROLL[a >= b ? `${a}${b}` : `${b}${a}`] ?? []);
    });
    /** @type {any} */ (EvaluatePositionImmediate).mockResolvedValue({ moves: [] });
    transcriptionListStore.set([]);
    clearTranscription();
    quizPlayStore.set(null);
    selectedMoveStore.set(null);
    databasePathStore.set('/tmp/library.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
});

afterEach(() => {
    cleanup();
    quizPlayStore.set(null);
});

describe('le plateau suit la grammaire commune, les dés d’abord', () => {
    test('sans jet saisi, aucun coup n’est armé', async () => {
        transcriptionStore.set(state(annotated()));
        render(TranscriptionPanel);
        await tick();
        await settle();
        expect(get(quizPlayStore)).toBeNull();
        expect(get(quizPlayValidateStore)).toBeNull();
    });

    test('le jet tapé arme les coups de ce jet, avec son rappel de validation', async () => {
        const play = await armed();
        expect(play.free).toBe(false);
        expect(play.rolled).toEqual([6, 1]);
        expect(play.plays).toHaveLength(1);
        expect(get(quizPlayValidateStore)).toBeTypeOf('function');
    });

    test('les quatre pas d’un double sont enregistrés par la validation, pas avant', async () => {
        await armed('Digit6', 'Digit6');
        hop(13, 7);
        hop(13, 7);
        hop(8, 2);
        hop(8, 2);
        await settle();
        expect(gestures().some((/** @type {any} */ g) => g.Kind === 'enter_play')).toBe(false);

        get(quizPlayValidateStore)?.();
        await vi.waitFor(() => expect(gestures().some((/** @type {any} */ g) => g.Kind === 'validate')).toBe(true));
        const sent = gestures();
        const at = sent.findIndex((/** @type {any} */ g) => g.Kind === 'enter_play');
        expect(sent[at].Steps).toEqual([
            { from: 13, to: 7, hit: false },
            { from: 13, to: 7, hit: false },
            { from: 8, to: 2, hit: false },
            { from: 8, to: 2, hit: false }
        ]);
        // Un coup LÉGAL ne porte pas de plateau : il n'a rien d'illégal à dire.
        expect(sent[at].BoardAfter).toBeNull();
    });

    test('un coup à moitié joué n’enregistre rien, même validé', async () => {
        await armed('Digit6', 'Digit6');
        hop(13, 7);
        get(quizPlayValidateStore)?.();
        await settle();
        expect(gestures().some((/** @type {any} */ g) => g.Kind === 'enter_play')).toBe(false);
    });

    test('une case du triangle enregistre le coup achevé avant de commencer le jet suivant', async () => {
        await armed();
        hop(13, 7);
        hop(8, 7);
        await fireEvent.click(cell('62'));
        await vi.waitFor(() => expect(gestures().filter((/** @type {any} */ g) => g.Kind === 'enter_die')).toHaveLength(4));
        const kinds = gestures().map((/** @type {any} */ g) => g.Kind);
        expect(kinds.indexOf('validate')).toBeLessThan(kinds.lastIndexOf('enter_die'));
    });
});
