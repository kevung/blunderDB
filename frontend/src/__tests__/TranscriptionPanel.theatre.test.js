/**
 * TranscriptionPanel.theatre.test.js — the theatre of the transcription: the video over the
 * whole window and a floating board, opened from the video's button or F11,
 * left by F11, Escape or its button; the transcription keys stay live inside it.
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
    EvaluatePositionImmediate: vi.fn(),
    PickTranscriptionVideo: vi.fn().mockResolvedValue(''),
    YouTubeWatchURL: vi.fn((/** @type {string} */ s) => Promise.resolve(/youtu/.test(s) ? (s.startsWith('https://') ? s : 'https://www.youtube.com/watch?v=dQw4w9WgXcQ') : ''))
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetPruneK: vi.fn().mockResolvedValue(0)
}));
vi.mock('../components/VideoPane.svelte', async () => await import('./fixtures/FakeVideoPane.svelte'));

import { ApplyTranscriptionGesture } from '../../wailsjs/go/database/Database.js';
import { LegalMoves, EvaluatePositionImmediate } from '../../wailsjs/go/gui/App.js';

import TranscriptionPanel from '../components/TranscriptionPanel.svelte';
import TranscriptionTheatre from '../components/TranscriptionTheatre.svelte';
import { theatreStore, exitTheatre, enterTheatre, theatreKey, theatreAvailableStore } from '../services/transcriptionTheatre.js';
import { handleEscapeCapture } from '../services/escapeService.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { fakeVideo } from './fixtures/fakeVideo.js';
import { transcriptionStore, clearTranscription } from '../stores/transcriptionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';

const POSITION = (/** @type {any} */ dice) => ({
    board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [0, 0] },
    cube: { owner: -1, value: 0 },
    dice,
    score: [7, 7],
    player_on_roll: 0,
    decision_type: 0
});

// Three plays: two timed in full, the last one timed only by its roll.
/** @type {{side: number, kind: string, dice: number[], roll?: number, tick?: number, estimate?: number}[]} */
let ACTIONS = [];

let source = 'C:/videos/match.mov';

function annotated(/** @type {number} */ cursor) {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice', video_source: source || undefined }, actions: ACTIONS, cursor },
        actions: ACTIONS.map((a, index) => ({
            index,
            side: a.side,
            kind: a.kind,
            before: POSITION(a.dice),
            has_position: true,
            notation: '8/5 6/5',
            game_index: 0,
            game_number: 1,
            score: [0, 0],
            move_number: index,
            inconsistencies: [],
            roll_tick_ms: a.roll,
            tick_ms: a.tick,
            decision_ms: a.roll != null && a.tick != null ? a.tick - a.roll : a.estimate,
            decision_estimated: a.estimate != null ? true : undefined,
            cube_decision_ms: index > 0 && a.roll != null && ACTIONS[index - 1].tick != null ? a.roll - /** @type {number} */ (ACTIONS[index - 1].tick) : undefined
        })),
        games: [{ number: 1, initial_score: [0, 0], winner: -1, points_won: 0, crawford: false, finished: false, first: 0, last: ACTIONS.length - 1 }],
        entry: cursor < ACTIONS.length ? { at: cursor, replacing: true, side: ACTIONS[cursor].side, dice: ACTIONS[cursor].dice, selected: true, kind: 'checker', notation: '8/5 6/5' } : null,
        next: { expects: 'checker', side: 0, position: POSITION([0, 0]), crawford: false },
        score: [0, 0],
        cursor
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

let at = 0;

async function openedPanel(cursor = ACTIONS.length) {
    at = cursor;
    transcriptionStore.set({ id: 1, annotated: annotated(cursor) });
    render(TranscriptionTheatre);
    render(TranscriptionPanel);
    await settle();
    document.getElementById('transcriptionPanel')?.focus();
}

const panel = () => /** @type {HTMLElement} */ (document.getElementById('transcriptionPanel'));

beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    source = 'C:/videos/match.mov';
    ACTIONS = [{ side: 0, kind: 'checker', dice: [3, 1] }];
    databasePathStore.set('/tmp/test.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
    /** @type {any} */ (LegalMoves).mockResolvedValue([{ notation: '8/5 6/5' }]);
    /** @type {any} */ (EvaluatePositionImmediate).mockResolvedValue({ moves: [{ index: 0, move: '8/5 6/5', equity: 0.1 }] });
    /** @type {any} */ (ApplyTranscriptionGesture).mockImplementation(() => Promise.resolve({ id: 1, annotated: annotated(at) }));
});

afterEach(async () => {
    await exitTheatre();
    fakeVideo.reset();
    cleanup();
    clearTranscription();
});

const theatre = () => document.querySelector('[data-testid="transcription-theatre"]');
const dock = () => /** @type {HTMLElement} */ (document.querySelector('[data-testid="video-dock"]'));
const key = (/** @type {string} */ k, init = {}) => new KeyboardEvent('keydown', { key: k, cancelable: true, ...init });

async function openFromMenu() {
    await fireEvent.click(/** @type {HTMLElement} */ (document.querySelector('[data-testid="video-theatre"]')));
    await settle();
}

describe('the theatre of the transcription', () => {
    test('the button on the video opens it: the video moves over the whole window, the mini-board floats over it', async () => {
        await openedPanel();
        await openFromMenu();
        expect(get(theatreStore)).toBe(true);
        expect(dock().dataset.placement).toBe('theatre');
        expect(dock().classList.contains('in-theatre')).toBe(true);
        expect(document.querySelector('[data-testid="theatre-board"] svg')).not.toBeNull();
        // The dock's own buttons stay out of the theatre: it has its own way out.
        expect(document.querySelector('[data-testid="video-placement"]')).toBeNull();
    });

    test('the player is not moved by entering or leaving: same dock, same pane, so the instant and the play state stay', async () => {
        await openedPanel();
        const before = dock();
        const pane = before.querySelector('[data-testid="video-pane"]');
        const parent = before.parentNode;
        await enterTheatre();
        await settle();
        expect(dock()).toBe(before);
        expect(before.parentNode).toBe(parent);
        expect(before.querySelector('[data-testid="video-pane"]')).toBe(pane);
        handleEscapeCapture(key('Escape'));
        await settle();
        expect(dock()).toBe(before);
        expect(before.parentNode).toBe(parent);
        expect(before.querySelector('[data-testid="video-pane"]')).toBe(pane);
    });

    test('the dock’s buttons sit in a bar above the image, not over the video', async () => {
        await openedPanel();
        const bar = dock().querySelector('.video-dock-bar');
        expect(bar).not.toBeNull();
        expect(bar?.contains(document.querySelector('[data-testid="video-theatre"]'))).toBe(true);
        expect(bar?.contains(document.querySelector('[data-testid="video-placement"]'))).toBe(true);
        expect(dock().querySelector('[data-testid="video-pane"] [data-testid="video-placement"]')).toBeNull();
    });

    test('the theatre bar turns the video too, sharing the dock’s angle', async () => {
        await openedPanel();
        await openFromMenu();
        const button = /** @type {HTMLElement} */ (document.querySelector('[data-testid="transcription-theatre"] [data-testid="video-rotate"]'));
        expect(button).not.toBeNull();
        await fireEvent.click(button);
        await settle();
        expect(localStorage.getItem('blunderdb.transcription.videoRotation.1')).toBe('90');
        await fireEvent.click(button);
        await settle();
        expect(localStorage.getItem('blunderdb.transcription.videoRotation.1')).toBe('180');
    });

    test('the button on the video opens it too, without taking the focus', async () => {
        await openedPanel();
        const button = /** @type {HTMLElement} */ (document.querySelector('[data-testid="video-theatre"]'));
        expect(button).not.toBeNull();
        await fireEvent.click(button);
        await settle();
        expect(get(theatreStore)).toBe(true);
    });

    test('the transcription keys stay live: the focus comes back to the panel, Space plays the video', async () => {
        await openedPanel();
        /** @type {HTMLElement} */ (document.activeElement)?.blur?.();
        await enterTheatre();
        await settle();
        expect(document.activeElement).toBe(panel());
        await fireEvent.keyDown(panel(), { code: 'Space', key: ' ', bubbles: true, cancelable: true });
        await settle();
        expect(fakeVideo.toggles).toBe(1);
        await fireEvent.keyDown(panel(), { code: 'Digit3', key: '3', bubbles: true, cancelable: true });
        await settle();
        expect(gestures().some((g) => g.Kind === 'enter_die')).toBe(true);
    });

    test('Escape and the exit button leave it, and the video goes back to the panel', async () => {
        await openedPanel();
        await enterTheatre();
        await settle();
        handleEscapeCapture(key('Escape'));
        await settle();
        expect(get(theatreStore)).toBe(false);
        expect(theatre()).toBeNull();
        expect(panel().contains(dock())).toBe(true);
        await enterTheatre();
        await settle();
        await fireEvent.click(/** @type {HTMLElement} */ (document.querySelector('[data-testid="theatre-exit"]')));
        await settle();
        expect(get(theatreStore)).toBe(false);
    });

    test('F11 is the theatre’s only in a transcription with a video; leaving the tab ends it', async () => {
        await openedPanel();
        expect(theatreKey(key('F11'))).toBe(true);
        expect(theatreKey(key('F11', { ctrlKey: true }))).toBe(false);
        await enterTheatre();
        await settle();
        activeTabStore.set('search');
        await settle();
        expect(get(theatreStore)).toBe(false);
        expect(get(theatreAvailableStore)).toBe(false);
        expect(theatreKey(key('F11'))).toBe(false);
    });

    test('without a video there is no theatre', async () => {
        source = '';
        await openedPanel();
        expect(get(theatreAvailableStore)).toBe(false);
        await enterTheatre();
        expect(get(theatreStore)).toBe(false);
    });

    test('the mini-board names the selected candidate and folds into a tab for this theatre only', async () => {
        await openedPanel();
        await enterTheatre();
        await settle();
        selectedMoveStore.set('8/5 6/5');
        await settle();
        expect(document.querySelector('[data-testid="theatre-board-move"]')?.textContent).toBe('8/5 6/5');
        await fireEvent.click(/** @type {HTMLElement} */ (document.querySelector('[data-testid="theatre-board-hide"]')));
        await settle();
        expect(document.querySelector('[data-testid="theatre-board"]')).toBeNull();
        await fireEvent.click(/** @type {HTMLElement} */ (document.querySelector('[data-testid="theatre-board-show"]')));
        await settle();
        expect(document.querySelector('[data-testid="theatre-board"]')).not.toBeNull();
        await fireEvent.click(/** @type {HTMLElement} */ (document.querySelector('[data-testid="theatre-board-hide"]')));
        await exitTheatre();
        await settle();
        await enterTheatre();
        await settle();
        expect(document.querySelector('[data-testid="theatre-board"]')).not.toBeNull();
    });

    test('folded, the tab stands where the board stood, whole inside the window', async () => {
        localStorage.setItem('blunderdb.theatre.board', JSON.stringify({ x: 0.25, y: 0.5, size: 'm' }));
        await openedPanel();
        await enterTheatre();
        await settle();
        const board = /** @type {HTMLElement} */ (document.querySelector('[data-testid="theatre-board"]'));
        expect(board.style.left).toBe(`${Math.round(0.25 * window.innerWidth)}px`);
        await fireEvent.click(/** @type {HTMLElement} */ (document.querySelector('[data-testid="theatre-board-hide"]')));
        await settle();
        const tab = /** @type {HTMLElement} */ (document.querySelector('[data-testid="theatre-board-show"]'));
        expect(tab.style.left).toBe(`${Math.round(0.25 * window.innerWidth)}px`);
        expect(tab.style.top).toBe(`${Math.round(0.5 * window.innerHeight)}px`);
        localStorage.setItem('blunderdb.theatre.board', JSON.stringify({ x: 1, y: 1, size: 'm' }));
        await exitTheatre();
        await settle();
        await enterTheatre();
        await settle();
        await fireEvent.click(/** @type {HTMLElement} */ (document.querySelector('[data-testid="theatre-board-hide"]')));
        await settle();
        const corner = /** @type {HTMLElement} */ (document.querySelector('[data-testid="theatre-board-show"]'));
        expect(parseInt(corner.style.left, 10)).toBeLessThanOrEqual(window.innerWidth - 110);
        expect(parseInt(corner.style.top, 10)).toBeLessThanOrEqual(window.innerHeight - 34);
    });

    test('a fold remembered by an earlier version does not hide the board: the theatre opens with it', async () => {
        // The state read in a user's webview: placed, then folded in an earlier theatre.
        localStorage.setItem('blunderdb.theatre.board', JSON.stringify({ x: 0.0453125, y: 0.17833333333333334, size: 'm', hidden: true }));
        await openedPanel();
        await enterTheatre();
        await settle();
        expect(document.querySelector('[data-testid="theatre-board-show"]')).toBeNull();
        const board = /** @type {HTMLElement} */ (document.querySelector('[data-testid="theatre-board"]'));
        expect(board).not.toBeNull();
        expect(board.dataset.size).toBe('m');
    });
});
