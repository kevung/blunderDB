/**
 * TranscriptionPanel.conflict.test.js — un autre écrivain a changé le brouillon.
 *
 * Le geste n'est pas enregistré : le panneau redessine le brouillon tel qu'il
 * est (document, curseur, annulation) et le dit dans la langue de l'utilisateur,
 * au lieu d'afficher l'erreur brute.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

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
vi.mock('../services/transcriptionSave.js', async (importOriginal) => ({
    .../** @type {any} */ (await importOriginal()),
    finishDraft: vi.fn().mockResolvedValue(null),
    exportDraftMat: vi.fn().mockResolvedValue(false)
}));

import { ApplyTranscriptionGesture, ListTranscriptions } from '../../wailsjs/go/database/Database.js';
import { LegalMoves, EvaluatePositionImmediate } from '../../wailsjs/go/gui/App.js';

import TranscriptionPanel from '../components/TranscriptionPanel.svelte';
import { transcriptionListStore, transcriptionStore, transcriptionHistoryStore, clearTranscription, setTranscription } from '../stores/transcriptionStore.js';
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

function annotated(actions = 0) {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice' }, actions: Array.from({ length: actions }, () => ({ kind: 'resign' })), cursor: actions },
        actions: [],
        games: [],
        next: { expects: 'checker', side: 0, position: POSITION, crawford: false, game_number: 1 },
        score: [0, 0],
        finished: false,
        winner: -1,
        cursor: actions
    };
}

function press(/** @type {string} */ code) {
    const digit = /^Digit([1-9])$/.exec(code);
    const letter = /^Key([A-Z])$/.exec(code);
    return fireEvent.keyDown(document, { code, key: digit ? digit[1] : letter ? letter[1].toLowerCase() : code });
}

async function openedPanel() {
    transcriptionStore.set({ id: 1, annotated: annotated() });
    render(TranscriptionPanel);
    await tick();
    document.getElementById('transcriptionPanel')?.focus();
}

beforeEach(() => {
    vi.clearAllMocks();
    /** @type {any} */ (ListTranscriptions).mockResolvedValue([]);
    /** @type {any} */ (LegalMoves).mockResolvedValue([]);
    /** @type {any} */ (EvaluatePositionImmediate).mockResolvedValue({ moves: [] });
    transcriptionListStore.set([]);
    clearTranscription();
    selectedMoveStore.set(null);
    databasePathStore.set('/tmp/library.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
});

afterEach(cleanup);

describe('un brouillon modifié ailleurs', () => {
    test('le conflit redessine le brouillon, remet l’annulation à zéro et le dit', async () => {
        await openedPanel();
        setTranscription({ id: 1, annotated: annotated(), can_undo: true, can_redo: false });
        /** @type {any} */ (ApplyTranscriptionGesture).mockResolvedValueOnce({ id: 1, annotated: annotated(2), can_undo: false, can_redo: false, conflict: true });
        await press('KeyR');
        await press('Digit2');
        await vi.waitFor(() => expect(document.querySelector('.error')?.textContent).toMatch(/changed elsewhere/));
        expect(document.querySelector('.error')?.textContent).toMatch(/undo history was reset/);
        expect(get(transcriptionStore)?.annotated.document.actions).toHaveLength(2);
        expect(get(transcriptionHistoryStore)).toEqual({ canUndo: false, canRedo: false });
    });

    test('le geste suivant, accepté, efface le message', async () => {
        await openedPanel();
        /** @type {any} */ (ApplyTranscriptionGesture).mockResolvedValueOnce({ id: 1, annotated: annotated(1), conflict: true }).mockResolvedValue({ id: 1, annotated: annotated(1) });
        await press('KeyR');
        await press('Digit2');
        await vi.waitFor(() => expect(document.querySelector('.error')).not.toBeNull());
        await press('Digit3');
        await vi.waitFor(() => expect(document.querySelector('.error')).toBeNull());
    });
});
