/**
 * TranscriptionPanel.videoFollow.test.js — the Cursor follows the playing video through
 * the timed part of the document (ADR-0082 rule 3, services/videoFollow.js), moves by a
 * seek that writes nothing, never seeks the video back, holds still after a placement by
 * hand so that `v` times the Action the user chose, and the duration column says it.
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
    PickTranscriptionVideo: vi.fn().mockResolvedValue('')
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetPruneK: vi.fn().mockResolvedValue(0)
}));
vi.mock('../components/VideoPane.svelte', async () => await import('./fixtures/FakeVideoPane.svelte'));

import { ApplyTranscriptionGesture } from '../../wailsjs/go/database/Database.js';
import { LegalMoves, EvaluatePositionImmediate } from '../../wailsjs/go/gui/App.js';

import TranscriptionPanel from '../components/TranscriptionPanel.svelte';
import { transcriptionStore, transcriptionNoticeStore, clearTranscription } from '../stores/transcriptionStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';
import { fakeVideo } from './fixtures/fakeVideo.js';

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
const seeksOfCursor = () => gestures().filter((g) => g.Kind === 'seek_cursor');

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

/** The video plays on to `ms`; the panel's clock ticks once. */
async function playTo(/** @type {number} */ ms) {
    fakeVideo.now = ms;
    vi.advanceTimersByTime(250);
    await settle(20);
}

beforeEach(() => {
    vi.useFakeTimers({ toFake: ['setInterval', 'clearInterval'] });
    vi.clearAllMocks();
    fakeVideo.reset();
    source = 'C:/videos/match.mov';
    ACTIONS = [
        { side: 0, kind: 'checker', dice: [3, 1], roll: 10000, tick: 15000 },
        { side: 1, kind: 'checker', dice: [5, 2], roll: 20000, tick: 26000 },
        { side: 0, kind: 'checker', dice: [6, 4], roll: 30000 }
    ];
    databasePathStore.set('/tmp/test.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
    /** @type {any} */ (LegalMoves).mockResolvedValue([{ notation: '8/5 6/5' }]);
    /** @type {any} */ (EvaluatePositionImmediate).mockResolvedValue({ moves: [{ index: 0, move: '8/5 6/5', equity: 0.1 }] });
    /** @type {any} */ (ApplyTranscriptionGesture).mockImplementation((/** @type {any} */ _id, /** @type {any} */ gesture) => {
        if (gesture.Kind === 'cursor_back') at = Math.max(0, at - 1);
        if (gesture.Kind === 'cursor_forward') at = Math.min(ACTIONS.length, at + 1);
        if (gesture.Kind === 'seek_cursor') at = Math.max(0, Math.min(ACTIONS.length, gesture.At));
        if (gesture.Kind === 'set_timecode' && at < ACTIONS.length) {
            if (gesture.HasTick) ACTIONS[at] = { ...ACTIONS[at], tick: gesture.TickMS };
            if (gesture.HasRollTick) ACTIONS[at] = { ...ACTIONS[at], roll: gesture.RollTickMS };
        }
        return Promise.resolve({ id: 1, annotated: annotated(at) });
    });
});

afterEach(() => {
    cleanup();
    clearTranscription();
    selectedMoveStore.set(null);
    statusBarModeStore.set('NORMAL');
    activeTabStore.set('position');
    vi.useRealTimers();
});

describe('the Cursor follows the video', () => {
    test('playing into an Action’s stretch puts the Cursor on it, without seeking the video back', async () => {
        await openedPanel();
        await playTo(5000);
        await playTo(12000);
        expect(seeksOfCursor()).toEqual([{ Kind: 'seek_cursor', At: 0 }]);
        await playTo(17000);
        expect(seeksOfCursor()).toHaveLength(1);
        await playTo(21000);
        expect(seeksOfCursor()).toEqual([
            { Kind: 'seek_cursor', At: 0 },
            { Kind: 'seek_cursor', At: 1 }
        ]);
        expect(document.querySelector('.cell[aria-current="true"]')?.getAttribute('data-index')).toBe('1');
        // The Cursor moved by the video never moves the video.
        expect(fakeVideo.seeks).toEqual([]);
        expect(gestures().some((g) => g.Kind === 'cursor_back' || g.Kind === 'cursor_forward')).toBe(false);
    });

    test('past the last Repère the Cursor goes back to the end of the document, once', async () => {
        await openedPanel();
        await playTo(12000);
        await playTo(12500);
        expect(seeksOfCursor()).toEqual([{ Kind: 'seek_cursor', At: 0 }]);
        await playTo(35000);
        await playTo(36000);
        await playTo(37000);
        expect(seeksOfCursor()).toEqual([
            { Kind: 'seek_cursor', At: 0 },
            { Kind: 'seek_cursor', At: 3 }
        ]);
    });

    test('a paused video takes the Cursor nowhere', async () => {
        await openedPanel();
        await playTo(12000);
        await playTo(12000);
        await playTo(12000);
        expect(seeksOfCursor()).toEqual([]);
    });

    test('after a click on a play with no end Repère, the Cursor holds and v times that play', async () => {
        await openedPanel();
        await playTo(40000);
        await playTo(41000);
        const cell = /** @type {HTMLElement} */ (document.querySelector('.cell[data-index="2"]'));
        await fireEvent.click(cell);
        await settle(20);
        expect(fakeVideo.seeks).toEqual([29000]);
        // The video, a second ahead of the roll, plays on through the play and a little past.
        await playTo(29000);
        await playTo(31000);
        await playTo(34000);
        expect(seeksOfCursor()).toEqual([]);
        await fireEvent.keyDown(document.activeElement ?? document, { code: 'KeyV', key: 'v', bubbles: true, cancelable: true });
        await settle(20);
        expect(gestures()).toContainEqual({ Kind: 'set_timecode', TickMS: 34000, HasTick: true });
        const duration = /** @type {HTMLElement} */ (document.querySelector('td.time[data-duration="2"]'));
        expect(duration.textContent).toBe('4 s · 4 s');
    });

    test('after a click on a timed play, the Cursor holds until the video leaves it', async () => {
        await openedPanel();
        await playTo(40000);
        await playTo(41000);
        await fireEvent.click(/** @type {HTMLElement} */ (document.querySelector('.cell[data-index="0"]')));
        await settle(20);
        expect(fakeVideo.seeks).toEqual([9000]);
        await playTo(9000);
        await playTo(14000);
        expect(seeksOfCursor()).toEqual([]);
        await playTo(25000);
        expect(seeksOfCursor()).toEqual([{ Kind: 'seek_cursor', At: 1 }]);
    });

    test('nothing moves while a field has the focus', async () => {
        await openedPanel();
        const field = document.createElement('input');
        document.body.appendChild(field);
        field.focus();
        await playTo(12000);
        await playTo(13000);
        await playTo(22000);
        expect(seeksOfCursor()).toEqual([]);
        field.remove();
    });

    test('without a video nothing follows', async () => {
        source = '';
        await openedPanel();
        await playTo(12000);
        await playTo(13000);
        expect(seeksOfCursor()).toEqual([]);
    });
});

describe('the duration column', () => {
    test('shows each play’s decision, the cube decision second', async () => {
        await openedPanel();
        const head = [...document.querySelectorAll('th.time')].map((th) => th.textContent);
        expect(head).toEqual(['time', 'time']);
        expect(document.querySelector('td.time[data-duration="0"]')?.textContent).toBe('5 s');
        expect(document.querySelector('td.time[data-duration="1"]')?.textContent).toBe('6 s · 5 s');
        expect(document.querySelector('td.time[data-duration="2"]')?.textContent).toBe('4 s');
    });

    test('is absent without a video or a Repère', async () => {
        source = '';
        ACTIONS = ACTIONS.map(({ roll: _r, tick: _t, ...a }) => a);
        await openedPanel();
        expect(document.querySelector('th.time')).toBeNull();
    });

    test('an estimated duration reads apart, with what it takes in', async () => {
        ACTIONS[1] = { ...ACTIONS[1], tick: undefined, estimate: 10000 };
        await openedPanel();
        const cell = /** @type {HTMLElement} */ (document.querySelector('td.time[data-duration="1"]'));
        expect(cell.querySelector('.estimated')?.textContent).toBe('≈ 10 s');
        expect(cell.getAttribute('title')).toContain('up to the next roll');
        // A measured one does not.
        expect(document.querySelector('td.time[data-duration="0"] .estimated')).toBeNull();
    });
});

describe('the cube decision has no estimate', () => {
    test('a double after a play with no action Repère says how to measure it', async () => {
        await openedPanel();
        await fireEvent.keyDown(document.activeElement ?? document, { code: 'KeyD', key: 'd', bubbles: true, cancelable: true });
        await settle(20);
        expect(get(transcriptionNoticeStore)?.key).toBe('transcription.notice.cubeUntimed');
    });

    test('after a timed play it says nothing', async () => {
        ACTIONS[2] = { ...ACTIONS[2], tick: 33000 };
        await openedPanel();
        await fireEvent.keyDown(document.activeElement ?? document, { code: 'KeyD', key: 'd', bubbles: true, cancelable: true });
        await settle(20);
        expect(get(transcriptionNoticeStore)?.key).not.toBe('transcription.notice.cubeUntimed');
    });
});
