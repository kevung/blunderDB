/**
 * TranscriptionPanel.inplace.test.js — corriger la cellule où l'on est.
 *
 * Trois promesses, toutes nées du même écart : ce que l'utilisateur voit à
 * l'écran et ce que le document dit doivent être la même chose, tout de suite.
 *
 *  1. Le plateau suit le Cursor sur TOUTE Action — le premier coup d'une partie
 *     et l'abandon compris. `has_position` dit qu'une Action ne produit ni Move ni
 *     Position dans le Match enregistré, jamais qu'il n'y a rien à montrer : le
 *     lire comme une absence renvoyait le plateau à la fin du document dès
 *     qu'on cliquait sur la première cellule d'une partie.
 *  2. `t` sur une passe écrit une prise À LA PLACE de la passe. Le moteur écrit
 *     au rang de l'Entry ; la machine à touches, elle, avalait la touche parce
 *     qu'aucune offre n'était en attente EN BOUT DE DOCUMENT.
 *  3. Le premier coup d'une partie se ressaisit sur place : deux dés, sans
 *     validation, et c'est l'ordre des dés qui décide qui commence.
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
import { transcriptionStore, clearTranscription } from '../stores/transcriptionStore.js';
import { positionStore } from '../stores/positionStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';

/** `score` sert de marqueur : il dit de quelle Action vient la position lue. */
const POSITION = (/** @type {any} */ mark) => ({
    board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [0, 0] },
    cube: { owner: -1, value: 0 },
    dice: [0, 0],
    score: [mark, mark],
    player_on_roll: 0,
    decision_type: 0
});

const ACTIONS = [
    { side: 0, kind: 'checker', dice: [6, 3] },
    { side: 1, kind: 'checker', dice: [5, 2] },
    { side: 0, kind: 'double' },
    { side: 1, kind: 'pass' }
];
const NOTATIONS = ['24/18 13/10', '13/8 13/11', '', ''];
const END = 99;

/**
 * L'annoté du moteur : le Cursor où on le lui demande, et l'Entry que
 * `loadEntry` pose dès que le Cursor tient une Action.
 *
 * @param {number} cursor
 */
function annotated(cursor) {
    const entry =
        cursor < ACTIONS.length
            ? {
                  at: cursor,
                  replacing: true,
                  side: ACTIONS[cursor].side,
                  dice: ACTIONS[cursor].dice ?? [0, 0],
                  selected: false,
                  review: false,
                  kind: 'checker',
                  game_start: cursor === 0,
                  notation: NOTATIONS[cursor]
              }
            : null;
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice' }, actions: ACTIONS, cursor },
        actions: ACTIONS.map((a, index) => ({
            index,
            side: a.side,
            kind: a.kind,
            // Le marqueur EST le rang : la position d'une Action est la sienne.
            before: POSITION(index),
            has_position: true,
            opens_game: index === 0,
            notation: NOTATIONS[index],
            game_index: 0,
            game_number: 1,
            score: [0, 0],
            move_number: index,
            inconsistencies: []
        })),
        games: [{ number: 1, initial_score: [0, 0], winner: 1, points_won: 1, crawford: false, finished: true, first: 0, last: 3 }],
        next: { expects: 'checker', game_start: true, side: 0, position: POSITION(END), crawford: false },
        entry,
        score: [0, 1],
        cursor
    };
}

const gestures = () => /** @type {any} */ (ApplyTranscriptionGesture).mock.calls.map((/** @type {any} */ call) => call[1]);

async function openedPanel(cursor = ACTIONS.length) {
    transcriptionStore.set({ id: 1, annotated: annotated(cursor) });
    const rendered = render(TranscriptionPanel);
    await tick();
    document.getElementById('transcriptionPanel')?.focus();
    return rendered;
}

async function settle(times = 8) {
    for (let i = 0; i < times; i++) {
        await tick();
        await Promise.resolve();
    }
}

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/tmp/test.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
    /** @type {any} */ (LegalMoves).mockResolvedValue([{ notation: '24/18 13/10', steps: [] }]);
    /** @type {any} */ (EvaluatePositionImmediate).mockResolvedValue({ moves: [{ index: 0, move: '24/18 13/10', equity: 0 }] });
    let at = ACTIONS.length;
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
    statusBarModeStore.set('NORMAL');
    activeTabStore.set('position');
});

describe('le plateau suit le Cursor', () => {
    test('un clic sur le premier coup d’une partie montre sa position', async () => {
        const { container } = await openedPanel();
        await settle();

        await fireEvent.click(/** @type {Element} */ (container.querySelector('.cell[data-index="0"]')));
        await settle();

        expect(/** @type {any} */ (get(positionStore)).score).toEqual([0, 0]);
    });

    test('un clic sur une décision de videau montre la position de cette décision', async () => {
        const { container } = await openedPanel();
        await settle();

        await fireEvent.click(/** @type {Element} */ (container.querySelector('.cell[data-index="2"]')));
        await settle();

        expect(/** @type {any} */ (get(positionStore)).score).toEqual([2, 2]);
    });
});

describe('le premier coup ressaisi', () => {
    test('les deux dés se retapent sur place, sans validation : l’ordre dit qui commence', async () => {
        const { container } = await openedPanel();
        await settle();

        await fireEvent.click(/** @type {Element} */ (container.querySelector('.cell[data-index="0"]')));
        await settle();
        /** @type {any} */ (ApplyTranscriptionGesture).mockClear();

        // Le petit dé d'abord : le joueur 2 commence. Le moteur en tire le camp ;
        // ce qui est tenu ici est que le jet se corrige en place, le coup restant
        // à choisir.
        await fireEvent.keyDown(document, { code: 'Digit3', key: '3' });
        await fireEvent.keyDown(document, { code: 'Digit5', key: '5' });
        await settle();

        expect(gestures().slice(0, 2)).toEqual([
            { Kind: 'enter_die', Die: 3 },
            { Kind: 'enter_die', Die: 5 }
        ]);
        expect(gestures().map((/** @type {any} */ g) => g.Kind)).not.toContain('validate');
    });
});

describe('les gestes de videau corrigent la cellule tenue par le Cursor', () => {
    test('`t` sur une passe demande une prise, sans passer par une suppression', async () => {
        const { container } = await openedPanel();
        await settle();

        await fireEvent.click(/** @type {Element} */ (container.querySelector('.cell[data-index="3"]')));
        await settle();
        /** @type {any} */ (ApplyTranscriptionGesture).mockClear();

        await fireEvent.keyDown(document, { code: 'KeyT', key: 't' });
        await settle();

        expect(gestures()).toEqual([{ Kind: 'take' }]);
    });

    test('le bouton [T] est allumé sur une cellule relue, hors de toute offre', async () => {
        const { container } = await openedPanel();
        await settle();
        // La rangée est [D] [T] [P] [R] : le second bouton est la prise.
        const takeButton = () => /** @type {HTMLButtonElement | null} */ (container.querySelectorAll('.cube-row button')[1]);
        // En bout de document, plus rien à prendre : le bouton est éteint.
        expect(takeButton()?.disabled).toBe(true);

        await fireEvent.click(/** @type {Element} */ (container.querySelector('.cell[data-index="3"]')));
        await settle();

        expect(takeButton()?.disabled).toBe(false);
    });

    test('`p` reste au répartiteur global pendant la saisie d’un jet', async () => {
        await openedPanel();
        await settle();
        /** @type {any} */ (ApplyTranscriptionGesture).mockClear();

        // Un dé tapé : l'utilisateur saisit un jet, `p` n'est pas une réponse.
        await fireEvent.keyDown(document, { code: 'Digit4', key: '4' });
        await settle();
        const before = gestures().length;
        await fireEvent.keyDown(document, { code: 'KeyP', key: 'p' });
        await settle();

        expect(gestures().slice(before)).toEqual([]);
    });
});
