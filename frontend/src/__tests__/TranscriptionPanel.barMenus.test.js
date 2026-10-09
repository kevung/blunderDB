/**
 * TranscriptionPanel.barMenus.test.js — the draft bar's two menus: Video, anchored to its
 * button, names the attached source and holds the YouTube field; ⋯ holds the .mat text and
 * the export. Escape or a click outside closes them, and the focus goes back to the panel
 * so the next keystroke is a transcription key.
 */

import { handleEscapeCapture } from '../services/escapeService.js';
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

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
import { LegalMoves, EvaluatePositionImmediate, PickTranscriptionVideo } from '../../wailsjs/go/gui/App.js';

import TranscriptionPanel from '../components/TranscriptionPanel.svelte';
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
    render(TranscriptionPanel);
    await tick();
    document.getElementById('transcriptionPanel')?.focus();
}

const panel = () => /** @type {HTMLElement} */ (document.getElementById('transcriptionPanel'));
const videoButton = () => /** @type {HTMLElement} */ (document.querySelector('[data-testid="transcription-video-button"]'));
const moreButton = () => /** @type {HTMLElement} */ (document.querySelector('[data-testid="transcription-more-button"]'));
const videoMenu = () => document.querySelector('[data-testid="transcription-video-menu"]');
const moreMenu = () => document.querySelector('[data-testid="transcription-more-menu"]');
const itemsOf = (/** @type {Element | null} */ menu) => [...(menu?.querySelectorAll('[role="menuitem"]') ?? [])].map((b) => b.textContent?.trim());
const item = (/** @type {Element | null} */ menu, /** @type {RegExp} */ name) =>
    /** @type {HTMLElement} */ ([...(menu?.querySelectorAll('[role="menuitem"]') ?? [])].find((b) => name.test(b.textContent ?? '')));

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

afterEach(() => {
    cleanup();
    clearTranscription();
});

describe('the Video button', () => {
    test('names the attached file, or YouTube, or says Video when none is attached', async () => {
        await openedPanel();
        expect(videoButton().textContent).toContain('match.mov');
        cleanup();
        source = 'youtu.be/dQw4w9WgXcQ';
        await openedPanel();
        expect(videoButton().textContent).toContain('YouTube');
        cleanup();
        source = '';
        await openedPanel();
        expect(videoButton().textContent).toContain('Video');
    });

    test('opens a menu under it: a local file, a YouTube link, and Remove the video while one is attached', async () => {
        await openedPanel();
        await fireEvent.click(videoButton());
        await settle();
        expect(videoMenu()?.getAttribute('role')).toBe('menu');
        expect(videoButton().getAttribute('aria-expanded')).toBe('true');
        expect(itemsOf(videoMenu())).toEqual(['Local file…', 'YouTube link…', 'Remove video']);
        expect(videoMenu()?.contains(document.activeElement)).toBe(true);
        cleanup();
        source = '';
        await openedPanel();
        await fireEvent.click(videoButton());
        await settle();
        expect(itemsOf(videoMenu())).toEqual(['Local file…', 'YouTube link…']);
    });

    test('a local file chosen is attached, the menu closes and the panel has the focus', async () => {
        /** @type {any} */ (PickTranscriptionVideo).mockResolvedValue('/home/k/new.mp4');
        await openedPanel();
        await fireEvent.click(videoButton());
        await settle();
        await fireEvent.click(item(videoMenu(), /Local file/));
        await settle();
        expect(videoMenu()).toBeNull();
        expect(gestures()).toContainEqual({ Kind: 'set_video', VideoSource: '/home/k/new.mp4' });
        expect(document.activeElement).toBe(panel());
    });

    test('the YouTube field opens inside the menu, prefilled and selected over a YouTube video', async () => {
        source = 'youtu.be/dQw4w9WgXcQ';
        await openedPanel();
        await fireEvent.click(videoButton());
        await settle();
        await fireEvent.click(item(videoMenu(), /YouTube/));
        await settle(20);
        const input = /** @type {HTMLInputElement} */ (videoMenu()?.querySelector('input'));
        expect(input.value).toBe('https://www.youtube.com/watch?v=dQw4w9WgXcQ');
        expect(document.activeElement).toBe(input);
        expect([input.selectionStart, input.selectionEnd]).toEqual([0, input.value.length]);
        // Typing a URL is not transcription: the digits stay in the field.
        await fireEvent.keyDown(input, { key: '3', code: 'Digit3' });
        expect(gestures()).toEqual([]);
        input.value = 'https://youtu.be/abcdefghijk';
        await fireEvent.input(input);
        await fireEvent.submit(/** @type {HTMLFormElement} */ (input.form));
        await settle();
        expect(gestures()).toContainEqual({ Kind: 'set_video', VideoSource: 'https://youtu.be/abcdefghijk' });
        expect(videoMenu()).toBeNull();
        expect(document.activeElement).toBe(panel());
    });

    test('Remove video detaches the video', async () => {
        await openedPanel();
        await fireEvent.click(videoButton());
        await settle();
        await fireEvent.click(item(videoMenu(), /Remove video/));
        await settle();
        expect(gestures()).toContainEqual({ Kind: 'set_video', VideoSource: '' });
        expect(document.activeElement).toBe(panel());
    });

    test('Escape closes the menu, even from the YouTube field, and gives the focus back to the panel', async () => {
        await openedPanel();
        await fireEvent.click(videoButton());
        await settle();
        await fireEvent.keyDown(/** @type {HTMLElement} */ (document.activeElement), { key: 'Escape', code: 'Escape' });
        await settle();
        expect(videoMenu()).toBeNull();
        expect(document.activeElement).toBe(panel());

        await fireEvent.click(videoButton());
        await settle();
        await fireEvent.click(item(videoMenu(), /YouTube/));
        await settle(20);
        await fireEvent.keyDown(/** @type {HTMLElement} */ (document.activeElement), { key: 'Escape', code: 'Escape' });
        await settle();
        expect(videoMenu()).toBeNull();
        expect(document.activeElement).toBe(panel());
    });

    test('a click outside closes it', async () => {
        await openedPanel();
        await fireEvent.click(videoButton());
        await settle();
        await fireEvent.click(document.body);
        await settle();
        expect(videoMenu()).toBeNull();
    });

    test('a digit typed while the menu has the focus does not reach the dice', async () => {
        await openedPanel();
        await fireEvent.click(videoButton());
        await settle();
        await fireEvent.keyDown(/** @type {HTMLElement} */ (document.activeElement), { key: '3', code: 'Digit3' });
        await settle();
        expect(gestures()).toEqual([]);
    });
});

describe('the ⋯ menu', () => {
    test('holds the .mat text and the export, and Escape gives the focus back to the panel', async () => {
        await openedPanel();
        expect(moreMenu()).toBeNull();
        await fireEvent.click(moreButton());
        await settle();
        expect(moreButton().getAttribute('aria-expanded')).toBe('true');
        expect(itemsOf(moreMenu())).toEqual(['.mat text', 'Export .mat']);
        await fireEvent.keyDown(/** @type {HTMLElement} */ (document.activeElement), { key: 'Escape', code: 'Escape' });
        await settle();
        expect(moreMenu()).toBeNull();
        expect(document.activeElement).toBe(panel());
    });

    test('.mat text opens the dialog', async () => {
        await openedPanel();
        await fireEvent.click(moreButton());
        await settle();
        await fireEvent.click(item(moreMenu(), /\.mat text/));
        await settle();
        expect(moreMenu()).toBeNull();
        expect(document.querySelector('.mat-text')).not.toBeNull();
    });
});

describe('the Metadata button', () => {
    const metadataButton = () => /** @type {HTMLElement} */ ([...document.querySelectorAll('.draft-bar button')].find((b) => b.textContent?.trim() === 'Metadata'));

    test('the open form replaces the entry; the same button and Escape bring it back', async () => {
        await openedPanel();
        expect(document.querySelector('[data-testid="transcription-dice"]')).not.toBeNull();
        await fireEvent.click(metadataButton());
        await settle();
        expect(document.querySelector('[data-testid="transcription-metadata"]')).not.toBeNull();
        expect(document.querySelector('[data-testid="transcription-dice"]')).toBeNull();
        expect(document.querySelector('[data-testid="transcription-candidates"]')).toBeNull();
        await fireEvent.click(metadataButton());
        await settle();
        expect(document.querySelector('[data-testid="transcription-metadata"]')).toBeNull();
        expect(document.querySelector('[data-testid="transcription-dice"]')).not.toBeNull();
        await fireEvent.click(metadataButton());
        await settle();
        handleEscapeCapture(new KeyboardEvent('keydown', { key: 'Escape', cancelable: true }));
        await settle();
        expect(document.querySelector('[data-testid="transcription-metadata"]')).toBeNull();
        expect(document.querySelector('[data-testid="transcription-dice"]')).not.toBeNull();
    });
});
