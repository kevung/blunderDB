/**
 * TranscriptionPanel.rollSnap.test.js — what the vision events do to the panel (ADR-0109):
 * the first die of a new Action takes the instant of a sure rest of the dice that covers the
 * current instant, and the current instant otherwise; `v` / `Shift+V` and the other gestures
 * keep the instant asked; `g` / `Shift+G` jump between the markers; the line, the lost view
 * and the overlay drift show.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent, screen } from '@testing-library/svelte';
import { tick } from 'svelte';

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
    PickTranscriptionVideo: vi.fn().mockResolvedValue('')
}));
vi.mock('../components/VideoPane.svelte', async () => await import('./fixtures/FakeVideoPane.svelte'));

import { ApplyTranscriptionGesture, TranscriptionCandidates } from '../../wailsjs/go/database/Database.js';
import { LegalMoves } from '../../wailsjs/go/gui/App.js';

import { candidatesAnswer } from './helpers/transcriptionCandidates.js';
import TranscriptionPanel from '../components/TranscriptionPanel.svelte';
import { transcriptionStore, clearTranscription } from '../stores/transcriptionStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';
import { fakeVideo } from './fixtures/fakeVideo.js';
import { visionEventsStore, resetVisionEvents } from '../stores/visionEventsStore.js';
import { makeEvent, EVENT_TYPES as T, STATES as S } from '../vision/contract.js';
import { registerKeys } from '../services/keyDispatch.js';

// Stands for keyboardService.handleKeyDown, the global tier App.svelte registers: what
// reaches it is what the panel let through.
const globalKeys = vi.fn();
/** @type {(() => void) | null} */
let unregisterGlobal = null;

const POSITION = (/** @type {any} */ dice) => ({
    board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [0, 0] },
    cube: { owner: -1, value: 0 },
    dice,
    score: [7, 7],
    player_on_roll: 0,
    decision_type: 0
});

const ACTIONS = [
    { side: 0, kind: 'checker', dice: [3, 1] },
    { side: 1, kind: 'checker', dice: [5, 2] }
];

let source = 'C:/videos/match.mov';
/** @type {any} */
let entry = null;

/** The annotated document the engine returns, the Cursor where it was asked. */
function annotated(/** @type {number} */ cursor) {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice', video_source: source || undefined }, actions: ACTIONS, cursor },
        actions: ACTIONS.map((a, index) => ({
            index,
            side: a.side,
            kind: a.kind,
            before: POSITION(a.dice),
            has_position: true,
            notation: index === 0 ? '8/5 6/5' : '13/8 13/11',
            game_index: 0,
            game_number: 1,
            score: [0, 0],
            move_number: index,
            inconsistencies: [],
            roll_tick_ms: index === 0 ? 70000 : undefined,
            tick_ms: index === 0 ? 76000 : 90000,
            decision_ms: index === 0 ? 6500 : undefined
        })),
        games: [{ number: 1, initial_score: [0, 0], winner: -1, points_won: 0, crawford: false, finished: false, first: 0, last: 1 }],
        entry,
        next: { expects: 'checker', side: 0, position: POSITION([0, 0]), crawford: false },
        score: [0, 0],
        cursor
    };
}

const PLAYS = [{ notation: '24/23 13/11' }, { notation: '8/5 6/5' }];
const RANKED = [
    { index: 1, move: '8/5 6/5', equity: 0.12 },
    { index: 0, move: '24/23 13/11', equity: -0.1, equityError: 0.22 }
];

/** @returns {any[]} */
const gestures = () => /** @type {any} */ (ApplyTranscriptionGesture).mock.calls.map((/** @type {any} */ call) => call[1]);

async function settle(times = 10) {
    for (let i = 0; i < times; i++) {
        await tick();
        await Promise.resolve();
    }
}

async function openedPanel(cursor = ACTIONS.length) {
    at = cursor;
    transcriptionStore.set({ id: 1, annotated: annotated(cursor) });
    render(TranscriptionPanel);
    await tick();
    document.getElementById('transcriptionPanel')?.focus();
}

/**
 * A real keydown, as the webview sends it: the physical key in `code`, the character it
 * types in `key`, through the app's dispatcher (keyDispatch).
 *
 * @param {string} code
 */
async function press(code, extra = /** @type {Record<string, any>} */ ({})) {
    const digit = /^Digit([1-9])$/.exec(code);
    const letter = /^Key([A-Z])$/.exec(code);
    const key = digit ? digit[1] : letter ? (extra.shiftKey ? letter[1] : letter[1].toLowerCase()) : code === 'Space' ? ' ' : code;
    await fireEvent.keyDown(document.activeElement ?? document, { code, key, bubbles: true, cancelable: true, ...extra });
    await settle();
}

let at = ACTIONS.length;

beforeEach(() => {
    vi.clearAllMocks();
    unregisterGlobal = registerKeys('global', globalKeys);
    fakeVideo.reset();
    resetVisionEvents();
    fakeVideo.now = 65000;
    source = 'C:/videos/match.mov';
    entry = null;
    databasePathStore.set('/tmp/test.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
    /** @type {any} */ (LegalMoves).mockResolvedValue(PLAYS);
    /** @type {any} */ (TranscriptionCandidates).mockResolvedValue(candidatesAnswer(PLAYS, RANKED));
    /** @type {any} */ (ApplyTranscriptionGesture).mockImplementation((/** @type {any} */ _id, /** @type {any} */ gesture) => {
        if (gesture.Kind === 'cursor_back') at = Math.max(0, at - 1);
        if (gesture.Kind === 'cursor_forward') at = Math.min(ACTIONS.length, at + 1);
        return Promise.resolve({ id: 1, annotated: annotated(at) });
    });
});

afterEach(() => {
    unregisterGlobal?.();
    cleanup();
    clearTranscription();
    selectedMoveStore.set(null);
    statusBarModeStore.set('NORMAL');
    activeTabStore.set('position');
});

const ev = (/** @type {string} */ type, /** @type {number} */ t, state = S.SURE) => makeEvent(type, t, state);
const feed = (/** @type {any[]} */ ...events) => visionEventsStore.set(events);
const die3 = () => gestures().find((g) => g.Kind === 'enter_die' && g.Die === 3);

describe('the Repère of the roll', () => {
    test('a sure rest covering the instant gives the roll its own instant', async () => {
        feed(ev(T.DICE_REST, 62));
        await openedPanel();
        await press('Digit3');
        expect(die3()).toEqual({ Kind: 'enter_die', Die: 3, TickMS: 62000, HasTick: true });
    });

    test.each([
        ['a rest to check', [ev(T.DICE_REST, 62, S.CHECK)]],
        ['an uncertain rest', [ev(T.DICE_REST, 62, S.UNCERTAIN)]],
        ['a rest picked up before the instant', [ev(T.DICE_REST, 62), ev(T.DICE_PICKUP, 64)]],
        ['a rest older than a minute', [ev(T.DICE_REST, 4)]],
        ['a lost view', [ev(T.DICE_REST, 62), ev(T.VIEW_LOST, 63)]],
        ['no event', []]
    ])('%s leaves the current instant', async (_name, events) => {
        feed(...events);
        await openedPanel();
        await press('Digit3');
        expect(die3()).toEqual({ Kind: 'enter_die', Die: 3, TickMS: 65000, HasTick: true });
    });

    test('the action of the validation keeps the current instant', async () => {
        feed(ev(T.DICE_REST, 62));
        await openedPanel();
        await press('Digit3');
        await press('Digit1');
        await vi.waitFor(() => expect(gestures().some((g) => g.Kind === 'select_candidate')).toBe(true));
        fakeVideo.now = 76000;
        feed(ev(T.DICE_REST, 62));
        await press('Enter');
        expect(gestures()).toContainEqual({ Kind: 'validate', TickMS: 76000, HasTick: true });
    });

    test('the explicit gesture keeps the current instant', async () => {
        feed(ev(T.DICE_REST, 62));
        await openedPanel(0);
        await press('KeyV', { shiftKey: true });
        expect(gestures()).toContainEqual({ Kind: 'set_timecode', RollTickMS: 65000, HasRollTick: true });
    });

    test('a correction in place sends no instant at all', async () => {
        feed(ev(T.DICE_REST, 62));
        entry = { at: 0, kind: 'checker', replacing: true, dice: [0, 0], game_start: false };
        await openedPanel(0);
        await press('KeyE');
        await press('Escape');
        await press('Digit3');
        expect(die3()).toEqual({ Kind: 'enter_die', Die: 3 });
    });
});

describe('the jump between markers', () => {
    const MARKS = [ev(T.DICE_REST, 20), ev(T.DICE_PICKUP, 40, S.CHECK), ev(T.DICE_THROWN, 90, S.UNCERTAIN)];

    test('g goes to the next marker, Shift+G to the previous one, a marker to check or uncertain being a stop', async () => {
        feed(...MARKS);
        await openedPanel();
        await press('KeyG');
        expect(fakeVideo.seeks).toEqual([90000]);
        await press('KeyG', { shiftKey: true });
        expect(fakeVideo.seeks).toEqual([90000, 40000]);
    });

    test('a jump writes nothing: no gesture, no toggle of the playback', async () => {
        feed(...MARKS);
        await openedPanel();
        /** @type {any} */ (ApplyTranscriptionGesture).mockClear();
        await press('KeyG');
        await press('KeyG', { shiftKey: true });
        expect(gestures()).toEqual([]);
        expect(fakeVideo.toggles).toBe(0);
    });

    test('past the last marker or without any, g does nothing', async () => {
        await openedPanel();
        await press('KeyG');
        feed(ev(T.DICE_REST, 20));
        await press('KeyG');
        expect(fakeVideo.seeks).toEqual([]);
    });

    test('hands and clock events are no stop', async () => {
        feed(ev(T.HANDS_IN, 70), ev(T.CLOCK, 71), ev(T.DICE_REST, 80));
        await openedPanel();
        await press('KeyG');
        expect(fakeVideo.seeks).toEqual([80000]);
    });
});

describe('the line and the indicators', () => {
    test('a marker per event, its shape saying its state, a click on it seeks', async () => {
        fakeVideo.duration = 600000;
        feed(ev(T.DICE_REST, 20), ev(T.DICE_PICKUP, 40, S.CHECK), ev(T.DICE_THROWN, 90, S.UNCERTAIN));
        await openedPanel();
        const marks = await screen.findAllByTestId('video-marker');
        expect(marks.map((m) => m.getAttribute('data-shape'))).toEqual(['circle', 'diamond', 'triangle']);
        await fireEvent.click(marks[1]);
        expect(fakeVideo.seeks).toEqual([40000]);
    });

    test('the lost view reaches the player until the board is found again', async () => {
        feed(ev(T.VIEW_LOST, 10));
        await openedPanel();
        expect(screen.getByTestId('video-pane').getAttribute('data-view-lost')).toBe('true');
        feed(ev(T.VIEW_LOST, 10), ev(T.VIEW_FOUND, 20));
        await settle();
        expect(screen.getByTestId('video-pane').getAttribute('data-view-lost')).toBe('false');
    });

    test('no marker, no line', async () => {
        fakeVideo.duration = 600000;
        await openedPanel();
        expect(screen.queryByTestId('video-timeline')).toBeNull();
    });

    test('a recent overlay change is reported, an old one is not', async () => {
        fakeVideo.duration = 600000;
        feed(ev(T.OVERLAY_CHANGED, 40, S.CHECK));
        fakeVideo.now = 50000;
        await openedPanel();
        await vi.waitFor(() => expect(screen.queryByTestId('video-overlay-drift')).not.toBeNull(), { timeout: 2000 });
        cleanup();
        fakeVideo.now = 80000;
        await openedPanel();
        await new Promise((r) => setTimeout(r, 600));
        expect(screen.queryByTestId('video-overlay-drift')).toBeNull();
    });
});
