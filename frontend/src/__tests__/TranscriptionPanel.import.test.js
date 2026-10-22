/**
 * TranscriptionPanel.import.test.js — choisir un .mat ou un document JSON crée le brouillon
 * et l'ouvre ; un refus s'affiche sans rien ouvrir.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    TranscriptionCandidates: vi.fn(),
    ListTranscriptions: vi.fn().mockResolvedValue([]),
    CreateTranscription: vi.fn(),
    OpenTranscription: vi.fn(),
    ApplyTranscriptionGesture: vi.fn(),
    ImportTranscription: vi.fn()
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    LegalMoves: vi.fn(),
    PickTranscriptionFile: vi.fn(),
    ReadFileContent: vi.fn()
}));
vi.mock('../services/transcriptionSave.js', async (importOriginal) => ({
    .../** @type {any} */ (await importOriginal()),
    finishDraft: vi.fn().mockResolvedValue(null),
    exportDraftMat: vi.fn().mockResolvedValue(false)
}));

import { ImportTranscription, ListTranscriptions, TranscriptionCandidates } from '../../wailsjs/go/database/Database.js';
import { LegalMoves, PickTranscriptionFile, ReadFileContent } from '../../wailsjs/go/gui/App.js';

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

beforeEach(() => {
    vi.clearAllMocks();
    /** @type {any} */ (ListTranscriptions).mockResolvedValue([]);
    /** @type {any} */ (LegalMoves).mockResolvedValue([]);
    /** @type {any} */ (TranscriptionCandidates).mockResolvedValue({ side: 0, dice: [0, 0], unranked: false, candidates: [] });
    transcriptionListStore.set([]);
    clearTranscription();
    selectedMoveStore.set(null);
    databasePathStore.set('/tmp/library.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
});

afterEach(cleanup);

async function clickImport() {
    render(TranscriptionPanel);
    await tick();
    await fireEvent.click(/** @type {HTMLElement} */ (document.getElementById('transcriptionImport')));
}

describe('importer un brouillon', () => {
    test('le texte du fichier choisi crée le brouillon et l’ouvre', async () => {
        /** @type {any} */ (PickTranscriptionFile).mockResolvedValue('/tmp/m.mat');
        /** @type {any} */ (ReadFileContent).mockResolvedValue({ content: '7 point match', error: '' });
        /** @type {any} */ (ImportTranscription).mockResolvedValue({ id: 4, annotated: annotated(3) });
        await clickImport();
        await vi.waitFor(() => expect(get(transcriptionStore)?.id).toBe(4));
        expect(ImportTranscription).toHaveBeenCalledWith('7 point match');
        expect(document.querySelector('.error')).toBeNull();
    });

    test('un refus affiche le message et n’ouvre rien', async () => {
        /** @type {any} */ (PickTranscriptionFile).mockResolvedValue('/tmp/m.json');
        /** @type {any} */ (ReadFileContent).mockResolvedValue({ content: 'oops', error: '' });
        /** @type {any} */ (ImportTranscription).mockRejectedValue('not a transcription');
        await clickImport();
        await vi.waitFor(() => expect(document.querySelector('.error')?.textContent).toMatch(/not a transcription/));
        expect(get(transcriptionStore)).toBeNull();
    });

    test('annuler le choix du fichier ne fait rien', async () => {
        /** @type {any} */ (PickTranscriptionFile).mockResolvedValue('');
        await clickImport();
        await tick();
        expect(ReadFileContent).not.toHaveBeenCalled();
        expect(ImportTranscription).not.toHaveBeenCalled();
    });

    test('la touche i importe, comme le bouton', async () => {
        /** @type {any} */ (PickTranscriptionFile).mockResolvedValue('/tmp/m.mat');
        /** @type {any} */ (ReadFileContent).mockResolvedValue({ content: '7 point match', error: '' });
        /** @type {any} */ (ImportTranscription).mockResolvedValue({ id: 5, annotated: annotated(1) });
        render(TranscriptionPanel);
        await tick();
        document.getElementById('transcriptionPanel')?.focus();
        await fireEvent.keyDown(document, { key: 'i', code: 'KeyI' });
        await vi.waitFor(() => expect(get(transcriptionStore)?.id).toBe(5));
    });

    test('annuler le sélecteur efface le refus précédent', async () => {
        /** @type {any} */ (PickTranscriptionFile).mockResolvedValueOnce('/tmp/m.json').mockResolvedValueOnce('');
        /** @type {any} */ (ReadFileContent).mockResolvedValue({ content: 'oops', error: '' });
        /** @type {any} */ (ImportTranscription).mockRejectedValue('not a transcription');
        await clickImport();
        await vi.waitFor(() => expect(document.querySelector('.error')).not.toBeNull());
        await fireEvent.click(/** @type {HTMLElement} */ (document.getElementById('transcriptionImport')));
        await vi.waitFor(() => expect(document.querySelector('.error')).toBeNull());
    });
});
