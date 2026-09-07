/**
 * TranscriptionPanel.cursor.test.js — T1.6 : le Cursor se promène.
 *
 * Ce que le panneau doit à la vue et que la vue ne peut pas tenir seule : une
 * touche `h`/`l` ou un clic sur une cellule demande au moteur de déplacer le
 * Cursor (`cursor_back`/`cursor_forward`, qui rechargent l'Action visée), puis
 * le panneau REMONTE le reste de l'écran — le plateau suit `annotated.cursor`
 * tout seul, mais les candidats sont une Évaluation, jamais stockée (ADR-0045
 * règle 8), et doivent être redemandés avec le coup joué présélectionné.
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
import { transcriptionStore, transcriptionKeyStore, clearTranscription } from '../stores/transcriptionStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';

const POSITION = (dice) => ({
    board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [0, 0] },
    cube: { owner: -1, value: 0 },
    dice,
    score: [7, 7],
    player_on_roll: 0,
    decision_type: 0
});

const ACTIONS = [
    { side: 0, kind: 'opening', dice: [3, 1] },
    { side: 0, kind: 'checker', dice: [3, 1] },
    { side: 1, kind: 'checker', dice: [5, 2] }
];
const NOTATIONS = ['', '8/5 6/5', '13/8 13/11'];

/** L'annoté que le moteur rend, le Cursor où on le lui demande. */
function annotated(cursor) {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice' }, actions: ACTIONS, cursor },
        actions: ACTIONS.map((a, index) => ({
            index,
            side: a.side,
            kind: a.kind,
            before: POSITION(a.dice),
            has_position: a.kind !== 'opening',
            notation: NOTATIONS[index],
            game_index: 0,
            game_number: 1,
            score: [0, 0],
            move_number: index,
            inconsistencies: []
        })),
        games: [{ number: 1, initial_score: [0, 0], winner: -1, points_won: 0, crawford: false, finished: false, first: 0, last: 2 }],
        next: { expects: 'checker', side: 0, position: POSITION([0, 0]), crawford: false },
        score: [0, 0],
        cursor
    };
}

// Le générateur numérote, l'évaluation classe : le coup JOUÉ n'est pas le
// premier de la liste, ce qui est tout l'objet du test.
const PLAYS = [{ notation: '24/23 13/11' }, { notation: '8/5 6/5' }, { notation: '13/11 6/5' }];
const RANKED = [
    { index: 0, move: '13/11 6/5', equity: 0.12 },
    { index: 1, move: '8/5 6/5', equity: 0.04, equityError: 0.08 },
    { index: 2, move: '24/23 13/11', equity: -0.1, equityError: 0.22 }
];

const gestures = () => ApplyTranscriptionGesture.mock.calls.map((call) => call[1]);

async function openedPanel(cursor = 3) {
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
    LegalMoves.mockResolvedValue(PLAYS);
    EvaluatePositionImmediate.mockResolvedValue({ moves: RANKED });
    // Le moteur déplace le Cursor et rend le document entier, comme la liaison.
    let at = 3;
    ApplyTranscriptionGesture.mockImplementation((_id, gesture) => {
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

describe('le Cursor au clavier', () => {
    test('`h` recule d’une Action et présélectionne le coup qui a été joué', async () => {
        await openedPanel();

        await fireEvent.keyDown(document, { code: 'KeyH', key: 'h' });
        await settle();

        expect(gestures()).toEqual([{ Kind: 'cursor_back' }]);
        expect(get(transcriptionStore).annotated.cursor).toBe(2);
        // L'Action visée est le 52: 13/8 13/11 d'Alice... dont la notation
        // n'est pas dans la liste classée ici ; c'est le cas suivant qui tient
        // la présélection. Ce qui compte à cette ligne : les dés de l'Action
        // visée sont ceux que la machine porte de nouveau.
        expect(get(transcriptionKeyStore).dice).toEqual([5, 2]);
    });

    test('deux `h` amènent sur le coup 8/5 6/5, qui est celui que la liste montre choisi', async () => {
        await openedPanel();

        await fireEvent.keyDown(document, { code: 'KeyH', key: 'h' });
        await settle();
        await fireEvent.keyDown(document, { code: 'KeyH', key: 'h' });
        await settle();

        expect(get(transcriptionStore).annotated.cursor).toBe(1);
        // Le coup joué est le SECOND de la liste classée : c'est lui qui est
        // sélectionné, pas le meilleur.
        expect(get(transcriptionKeyStore).selected).toBe(1);
        expect(get(selectedMoveStore)).toBe('8/5 6/5');
    });

    test('`l` avance', async () => {
        await openedPanel(1);
        // Le pilote du moteur part de la fin ; on le remet là où le panneau est.
        ApplyTranscriptionGesture.mockImplementation(() => Promise.resolve({ id: 1, annotated: annotated(2) }));

        await fireEvent.keyDown(document, { code: 'KeyL', key: 'l' });
        await settle();

        expect(gestures()).toEqual([{ Kind: 'cursor_forward' }]);
        expect(get(transcriptionStore).annotated.cursor).toBe(2);
    });
});

describe('le Cursor à la souris', () => {
    test('un clic sur une cellule du Transcript marche jusqu’à elle', async () => {
        const { container } = await openedPanel();
        await settle();

        const cell = container.querySelector('.cell[data-index="1"]');
        expect(cell).not.toBeNull();
        await fireEvent.click(cell);
        await settle();

        // Du bout du document (3) jusqu'à l'Action 1 : deux pas en arrière.
        expect(gestures()).toEqual([{ Kind: 'cursor_back' }, { Kind: 'cursor_back' }]);
        expect(get(transcriptionStore).annotated.cursor).toBe(1);
    });

    test('la cellule du Cursor est encadrée dans le Transcript du panneau', async () => {
        const { container } = await openedPanel(1);
        await settle();
        expect(container.querySelector('.cell.cursor')?.dataset.index).toBe('1');
    });
});
