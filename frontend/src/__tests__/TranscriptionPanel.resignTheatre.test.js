/**
 * TranscriptionPanel.resignTheatre.test.js — `r` in the theatre: the panel's row of levels is
 * covered, so a small dialog asks for the level over it. Same keys, same gesture as beside the
 * board; Escape cancels the resignation, not the theatre.
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
    TranscriptionMAT: vi.fn().mockResolvedValue('')
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    LegalMoves: vi.fn(),
    PickTranscriptionVideo: vi.fn().mockResolvedValue(''),
    YouTubeWatchURL: vi.fn().mockResolvedValue('')
}));
vi.mock('../components/VideoPane.svelte', async () => await import('./fixtures/FakeVideoPane.svelte'));

import { ApplyTranscriptionGesture, TranscriptionCandidates } from '../../wailsjs/go/database/Database.js';
import { LegalMoves } from '../../wailsjs/go/gui/App.js';

import TranscriptionPanel from '../components/TranscriptionPanel.svelte';
import TranscriptionTheatre from '../components/TranscriptionTheatre.svelte';
import { theatreStore, exitTheatre, enterTheatre } from '../services/transcriptionTheatre.js';
import { handleEscapeCapture } from '../services/escapeService.js';
import { fakeVideo } from './fixtures/fakeVideo.js';
import { transcriptionStore, clearTranscription } from '../stores/transcriptionStore.js';
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

function annotated() {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice', video_source: 'C:/videos/match.mov' }, actions: [], cursor: 0 },
        actions: [],
        games: [],
        next: { expects: 'checker', side: 1, position: POSITION, crawford: false, game_number: 1 },
        score: [0, 0],
        cursor: 0
    };
}

/** @returns {any[]} */
const gestures = () => /** @type {any} */ (ApplyTranscriptionGesture).mock.calls.map((/** @type {any} */ call) => call[1]);

async function settle(times = 10) {
    for (let i = 0; i < times; i++) {
        await tick();
        await Promise.resolve();
    }
}

const panel = () => /** @type {HTMLElement} */ (document.getElementById('transcriptionPanel'));
const modal = () => document.querySelector('[data-testid="resign-modal"]');

async function press(/** @type {string} */ code, /** @type {string} */ key) {
    await fireEvent.keyDown(panel(), { code, key, bubbles: true, cancelable: true });
    await settle();
}

async function inTheatre() {
    transcriptionStore.set({ id: 1, annotated: annotated() });
    render(TranscriptionTheatre);
    render(TranscriptionPanel);
    await settle();
    await enterTheatre();
    await settle();
    panel().focus();
}

beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    databasePathStore.set('/tmp/test.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
    /** @type {any} */ (LegalMoves).mockResolvedValue([]);
    /** @type {any} */ (TranscriptionCandidates).mockResolvedValue({ side: 0, dice: [0, 0], unranked: false, candidates: [] });
    /** @type {any} */ (ApplyTranscriptionGesture).mockImplementation(() => Promise.resolve({ id: 1, annotated: annotated() }));
});

afterEach(async () => {
    await exitTheatre();
    fakeVideo.reset();
    cleanup();
    clearTranscription();
});

describe('r in the theatre', () => {
    test('opens a dialog over the theatre naming the side on roll, and the focus stays on the panel', async () => {
        await inTheatre();
        expect(modal()).toBeNull();
        await press('KeyR', 'r');
        const dialog = /** @type {HTMLElement} */ (modal()?.querySelector('[role="dialog"]'));
        expect(dialog.getAttribute('aria-modal')).toBe('true');
        expect(dialog.getAttribute('aria-label')).toContain('Alice');
        // Outside the panel, so no ancestor's stacking context sinks it under the theatre.
        expect(panel().contains(modal())).toBe(false);
        expect(document.activeElement).toBe(panel());
        expect(gestures()).toEqual([]);
    });

    test.each([
        ['Digit1', '1', 1],
        ['Digit2', '2', 2],
        ['Digit3', '3', 3],
        ['Enter', 'Enter', 1]
    ])('%s records a resign of level %i and closes the dialog', async (code, key, level) => {
        await inTheatre();
        await press('KeyR', 'r');
        await press(code, key);
        await vi.waitFor(() => expect(gestures()).toEqual([{ Kind: 'resign', Level: level }]));
        expect(modal()).toBeNull();
        expect(get(theatreStore)).toBe(true);
    });

    test.each([1, 2, 3])('the button of level %i records it, and the focus goes back to the panel', async (level) => {
        await inTheatre();
        await press('KeyR', 'r');
        await fireEvent.click(/** @type {HTMLElement} */ (modal()?.querySelector(`[data-level="${level}"]`)));
        await settle();
        await vi.waitFor(() => expect(gestures()).toEqual([{ Kind: 'resign', Level: level }]));
        expect(modal()).toBeNull();
        expect(document.activeElement).toBe(panel());
    });

    test('Escape cancels the resignation and leaves the theatre open', async () => {
        await inTheatre();
        await press('KeyR', 'r');
        handleEscapeCapture(new KeyboardEvent('keydown', { key: 'Escape', cancelable: true }));
        await settle();
        expect(modal()).toBeNull();
        expect(get(theatreStore)).toBe(true);
        expect(gestures()).toEqual([]);
        expect(document.activeElement).toBe(panel());
    });
});
