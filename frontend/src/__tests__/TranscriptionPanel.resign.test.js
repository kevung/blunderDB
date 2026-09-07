/**
 * TranscriptionPanel.resign.test.js — T1.5 : la résignation dans le panneau.
 *
 * Les transitions sont tenues à part (transcriptionKeys.resign.test.js). Ce qui
 * est vérifié ici est la jointure : `r` puis un niveau produit UN geste `resign`
 * portant ce niveau et AUCUN camp — le camp qui abandonne est celui au trait, et
 * c'est le moteur qui le sait (fonctionnel.md §1.2) — et l'attente du niveau se
 * voit à l'écran, faute de quoi `r` serait une touche qui ne fait rien.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, screen, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListTranscriptions: vi.fn().mockResolvedValue([]),
    CreateTranscription: vi.fn(),
    OpenTranscription: vi.fn(),
    ApplyTranscriptionGesture: vi.fn()
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
import { selectedMoveStore } from '../stores/analysisStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';

const POSITION = {
    board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [0, 0] },
    cube: { owner: -1, value: 0 },
    dice: [0, 0],
    score: [7, 7],
    player_on_roll: 0,
    decision_type: 0
};

function annotated({ expects = 'checker', side = 0, score = [0, 0] } = {}) {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice' }, actions: [], cursor: 0 },
        actions: [],
        games: [],
        next: { expects, side, position: POSITION, crawford: false, game_number: 1 },
        score,
        finished: false,
        winner: -1,
        cursor: 0
    };
}

const state = (ann) => ({ id: 1, annotated: ann });

function press(code) {
    const digit = /^Digit([1-9])$/.exec(code);
    const letter = /^Key([A-Z])$/.exec(code);
    return fireEvent.keyDown(document, { code, key: digit ? digit[1] : letter ? letter[1].toLowerCase() : code });
}

async function openedPanel(ann = annotated()) {
    transcriptionStore.set(state(ann));
    render(TranscriptionPanel);
    await tick();
    document.getElementById('transcriptionPanel')?.focus();
}

const gestures = () => ApplyTranscriptionGesture.mock.calls.map((call) => call[1]);

beforeEach(() => {
    vi.clearAllMocks();
    ListTranscriptions.mockResolvedValue([]);
    ApplyTranscriptionGesture.mockImplementation(() => Promise.resolve(state(annotated())));
    LegalMoves.mockResolvedValue([]);
    EvaluatePositionImmediate.mockResolvedValue({ moves: [] });
    transcriptionListStore.set([]);
    clearTranscription();
    selectedMoveStore.set(null);
    databasePathStore.set('/tmp/library.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
});

afterEach(cleanup);

describe('la résignation', () => {
    test('r puis 2 envoie un resign de niveau 2, sans camp', async () => {
        await openedPanel();
        await press('KeyR');
        await press('Digit2');
        await vi.waitFor(() => expect(gestures()).toEqual([{ Kind: 'resign', Level: 2 }]));
    });

    test('l’attente du niveau se voit : le camp au trait, les trois niveaux, l’échappatoire', async () => {
        await openedPanel(annotated({ side: 1 }));
        await press('KeyR');
        expect(await screen.findByText('Alice resigns')).toBeTruthy();
        expect(screen.getByText('1 = single, 2 = gammon, 3 = backgammon; Escape cancels.')).toBeTruthy();
        // Les dés n'ont plus de sens tant que le niveau n'est pas donné.
        expect(document.querySelectorAll('.die')).toHaveLength(0);
    });

    test('Échap annule sans envoyer le moindre geste', async () => {
        await openedPanel();
        await press('KeyR');
        await press('Escape');
        expect(gestures()).toEqual([]);
        expect(await screen.findByText("Kévin's dice")).toBeTruthy();
    });

    test('entre r et son niveau, un 5 n’enregistre pas de dé', async () => {
        await openedPanel();
        await press('KeyR');
        await press('Digit5');
        expect(gestures()).toEqual([]);
        expect(screen.getByText('Kévin resigns')).toBeTruthy();
    });
});
