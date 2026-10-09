/**
 * TranscriptionPanel.copyMat.test.js — la copie du texte .mat dans le
 * presse-papiers : un refus du webview est annoncé dans la barre d'état, il ne
 * remonte pas en promesse rejetée non gérée.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListTranscriptions: vi.fn().mockResolvedValue([]),
    CreateTranscription: vi.fn(),
    OpenTranscription: vi.fn(),
    ApplyTranscriptionGesture: vi.fn(),
    TranscriptionMAT: vi.fn().mockResolvedValue('; [Site "blunderDB"]')
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    LegalMoves: vi.fn().mockResolvedValue([]),
    EvaluatePositionImmediate: vi.fn().mockResolvedValue({ moves: [] })
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetPruneK: vi.fn().mockResolvedValue(0)
}));
vi.mock('../services/clipboardService.js', () => ({
    writeTextToClipboard: vi.fn()
}));

import TranscriptionPanel from '../components/TranscriptionPanel.svelte';
import { transcriptionListStore, transcriptionStore, clearTranscription } from '../stores/transcriptionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarModeStore, statusBarTextStore } from '../stores/uiStore.js';

const POSITION = {
    board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [0, 0] },
    cube: { owner: -1, value: 0 },
    dice: [0, 0],
    score: [7, 7],
    player_on_roll: 0,
    decision_type: 0
};

const annotated = {
    document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice' }, actions: [], cursor: 0 },
    actions: [],
    games: [],
    next: { expects: 'checker', side: 0, position: POSITION, crawford: false, game_number: 1 },
    score: [0, 0],
    finished: false,
    winner: -1,
    cursor: 0
};

const buttonNamed = (/** @type {string} */ label) => [...document.querySelectorAll('button')].find((b) => b.textContent.trim() === label);

beforeEach(() => {
    vi.clearAllMocks();
    transcriptionListStore.set([]);
    clearTranscription();
    databasePathStore.set('/tmp/library.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
    statusBarTextStore.set('');
});

afterEach(cleanup);

describe('Dialogue du texte .mat', () => {
    test('un brouillon rechargé dialogue ouvert laisse le focus dans le dialogue', async () => {
        transcriptionStore.set({ id: 1, annotated });
        render(TranscriptionPanel);
        await tick();
        // .mat text and the export live in the bar's ⋯ menu.
        await fireEvent.click(/** @type {HTMLElement} */ (document.querySelector('[data-testid="transcription-more-button"]')));
        await fireEvent.click(/** @type {HTMLElement} */ (buttonNamed('.mat text')));
        await tick();
        const overlay = /** @type {HTMLElement} */ (document.querySelector('.modal-overlay'));
        expect(overlay.contains(document.activeElement)).toBe(true);

        transcriptionStore.set({ id: 1, annotated: { ...annotated } });
        await tick();

        expect(overlay.contains(document.activeElement)).toBe(true);
    });
});
