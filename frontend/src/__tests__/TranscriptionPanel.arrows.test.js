/**
 * TranscriptionPanel.arrows.test.js — le plateau dessine la décision transcrite.
 *
 * Une correction en amont a vidé le point 8 : la décision « 8/5 6/5 » ne se joue plus. Aucun
 * candidat ne porte cette notation, et c'est là que le meilleur coup du moteur prenait la place
 * du coup écrit. L'oracle : les flèches partent de la case écrite, l'étape fautive marquée ; une
 * ligne de l'analyse ne se dessine qu'une fois désignée, dans son propre ton.
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

import { ApplyTranscriptionGesture } from '../../wailsjs/go/database/Database.js';
import { LegalMoves, EvaluatePositionImmediate } from '../../wailsjs/go/gui/App.js';

import TranscriptionPanel from '../components/TranscriptionPanel.svelte';
import { transcriptionStore, transcriptionKeyStore, transcriptionArrowsStore, clearTranscription } from '../stores/transcriptionStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';

/** @param {number} onRoll @param {number[]} dice @param {Record<number, [number, number]>} stacks */
function position(onRoll, dice, stacks) {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    for (const [point, [color, checkers]] of Object.entries(stacks)) points[Number(point)] = { color, checkers };
    return { board: { points, bearoff: [0, 0] }, cube: { owner: -1, value: 0 }, dice, score: [7, 7], player_on_roll: onRoll, decision_type: 0 };
}

// Action 0 : Alice (camp 1) joue un coup légal, mais pas le meilleur. Action 1 : le 8 de Kévin a
// été vidé en amont, « 8/5 6/5 » part d'une case vide.
const ACTIONS = [
    { side: 1, kind: 'checker', dice: [5, 2], notation: '13/8 13/11', before: position(1, [5, 2], { 12: [1, 5], 6: [0, 5], 13: [0, 5] }), flaws: [] },
    {
        side: 0,
        kind: 'checker',
        dice: [3, 1],
        notation: '8/5 6/5',
        before: position(0, [3, 1], { 6: [0, 5], 13: [0, 5], 12: [1, 3] }),
        flaws: [{ kind: 'illegal_move', rule: 'unreachable_board', args: {}, detail: '' }]
    }
];

function annotated(/** @type {number} */ cursor) {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice' }, actions: ACTIONS.map((a) => ({ side: a.side, kind: a.kind, dice: a.dice })), cursor },
        actions: ACTIONS.map((a, index) => ({
            index,
            side: a.side,
            kind: a.kind,
            before: a.before,
            has_position: true,
            notation: a.notation,
            game_index: 0,
            game_number: 1,
            score: [0, 0],
            move_number: index,
            inconsistencies: a.flaws
        })),
        games: [{ number: 1, initial_score: [0, 0], winner: -1, points_won: 0, crawford: false, finished: false, first: 0, last: 1 }],
        next: { expects: 'checker', side: 1, position: position(1, [0, 0], {}), crawford: false },
        score: [0, 0],
        cursor
    };
}

const PLAYS = {
    52: [{ notation: '13/11 13/8' }, { notation: '13/8 13/11' }],
    31: [{ notation: '13/10 6/5' }, { notation: '6/3 6/5' }]
};
const RANKED = {
    52: [
        { index: 0, move: '13/11 13/8', equity: 0.1 },
        { index: 1, move: '13/8 13/11', equity: 0.05, equityError: 0.05 }
    ],
    31: [
        { index: 0, move: '13/10 6/5', equity: 0.2 },
        { index: 1, move: '6/3 6/5', equity: 0.1, equityError: 0.1 }
    ]
};
const roll = (/** @type {any} */ pos) => /** @type {31 | 52} */ (pos.dice[0] * 10 + pos.dice[1]);

async function settle(times = 10) {
    for (let i = 0; i < times; i++) {
        await tick();
        await Promise.resolve();
    }
}

/** Le panneau ouvert au bout du document, puis `n` fois `h`. */
async function cursorBack(/** @type {number} */ n) {
    transcriptionStore.set({ id: 1, annotated: annotated(2) });
    const rendered = render(TranscriptionPanel);
    await tick();
    document.getElementById('transcriptionPanel')?.focus();
    for (let i = 0; i < n; i++) {
        await fireEvent.keyDown(document, { code: 'KeyH', key: 'h' });
        await settle();
    }
    return rendered;
}

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/tmp/test.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
    /** @type {any} */ (LegalMoves).mockImplementation((/** @type {any} */ pos) => Promise.resolve(PLAYS[roll(pos)] ?? []));
    /** @type {any} */ (EvaluatePositionImmediate).mockImplementation((/** @type {any} */ pos) => Promise.resolve({ moves: RANKED[roll(pos)] ?? [] }));
    let at = 2;
    /** @type {any} */ (ApplyTranscriptionGesture).mockImplementation((/** @type {any} */ _id, /** @type {any} */ gesture) => {
        if (gesture.Kind === 'cursor_back') at = Math.max(0, at - 1);
        if (gesture.Kind === 'cursor_forward') at = Math.min(ACTIONS.length, at + 1);
        return Promise.resolve({ id: 1, annotated: annotated(at) });
    });
});

afterEach(() => {
    cleanup();
    clearTranscription();
    selectedMoveStore.set(null);
    transcriptionArrowsStore.set(null);
    statusBarModeStore.set('NORMAL');
    activeTabStore.set('position');
});

describe('the board draws the transcribed decision', () => {
    test('a decision a correction upstream made illegal: arrows from the point written, the faulty step flagged, no line standing in', async () => {
        await cursorBack(1);
        expect(/** @type {any} */ (get(transcriptionStore)).annotated.cursor).toBe(1);

        const arrows = get(transcriptionArrowsStore);
        expect(arrows?.tone).toBe('record');
        expect(arrows?.notation).toBe('8/5 6/5');
        expect(arrows?.moves).toEqual([
            { from: 8, to: 5, index: 0, illegal: true },
            { from: 6, to: 5, index: 0 }
        ]);
        // The engine's best play is listed, never selected nor drawn in its place.
        expect(get(selectedMoveStore)).toBeNull();
        expect(get(transcriptionKeyStore).selected).toBe(-1);
    });

    test('`j` points at the first line: its arrows, in the engine tone', async () => {
        await cursorBack(1);
        await fireEvent.keyDown(document, { code: 'KeyJ', key: 'j' });
        await settle();

        expect(get(selectedMoveStore)).toBe('13/10 6/5');
        expect(get(transcriptionArrowsStore)).toMatchObject({ tone: 'engine', notation: '13/10 6/5' });
    });

    test('hovering a line draws it while the pointer stays, then the written move comes back', async () => {
        const { container } = await cursorBack(1);
        const row = /** @type {HTMLElement} */ (container.querySelector('tr[data-move="6/3 6/5"]'));
        expect(row).not.toBeNull();

        await fireEvent.mouseEnter(row);
        await settle();
        expect(get(transcriptionArrowsStore)).toMatchObject({ tone: 'engine', notation: '6/3 6/5' });

        await fireEvent.mouseLeave(row);
        await settle();
        expect(get(transcriptionArrowsStore)).toMatchObject({ tone: 'record', notation: '8/5 6/5' });
    });

    test('a legal decision that is not the best: its own arrows, its line highlighted', async () => {
        await cursorBack(2);
        expect(/** @type {any} */ (get(transcriptionStore)).annotated.cursor).toBe(0);

        expect(get(selectedMoveStore)).toBe('13/8 13/11');
        const arrows = get(transcriptionArrowsStore);
        expect(arrows?.tone).toBe('record');
        expect(arrows?.notation).toBe('13/8 13/11');
        expect(arrows?.moves.some((m) => m.illegal)).toBe(false);
    });
});

describe('choosing a line of the analysis replaces the decision, the first line included', () => {
    const sent = () => /** @type {any} */ (ApplyTranscriptionGesture).mock.calls.map((/** @type {any} */ call) => (call[1].Kind === 'select_candidate' ? `select ${call[1].Candidate}` : call[1].Kind));

    test('on a decision no line carries, `j` then Enter writes the first line', async () => {
        await cursorBack(1);
        /** @type {any} */ (ApplyTranscriptionGesture).mockClear();
        await fireEvent.keyDown(document, { code: 'KeyJ', key: 'j' });
        await settle();
        await fireEvent.keyDown(document, { code: 'Enter', key: 'Enter' });
        await settle();
        // '13/10 6/5' is the generator's play 0.
        expect(sent()).toEqual(['select 0', 'validate']);
    });

    test('on a decision no line carries, `k` at the top still picks the first line', async () => {
        await cursorBack(1);
        /** @type {any} */ (ApplyTranscriptionGesture).mockClear();
        await fireEvent.keyDown(document, { code: 'KeyK', key: 'k' });
        await settle();
        expect(sent()).toEqual(['select 0']);
    });

    test('on a legal decision that is not the best, `k` up to the first line writes it', async () => {
        await cursorBack(2);
        /** @type {any} */ (ApplyTranscriptionGesture).mockClear();
        await fireEvent.keyDown(document, { code: 'KeyK', key: 'k' });
        await settle();
        await fireEvent.keyDown(document, { code: 'Enter', key: 'Enter' });
        await settle();
        // '13/11 13/8', ranked first, is the generator's play 0.
        expect(sent()).toEqual(['select 0', 'validate']);
    });

    test('a click on the first line picks it, a double-click writes it', async () => {
        const { container } = await cursorBack(1);
        /** @type {any} */ (ApplyTranscriptionGesture).mockClear();
        await fireEvent.dblClick(/** @type {HTMLElement} */ (container.querySelector('tr[data-move="13/10 6/5"]')));
        await settle();
        expect(sent()).toEqual(['select 0', 'validate']);
    });
});
