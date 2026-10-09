/**
 * TranscriptionPanel.video.test.js — the video of a Transcription (ADR-0082 rule 3).
 *
 * The player is a clock the panel reads: a gesture that writes a new Action carries the
 * video's instant, an explicit validation carries the action's, and the validation the
 * panel sends ahead of the next roll's digit carries none — the play stays untimed rather
 * than timed wrong. Without a source, nothing changes: no field, no key.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent, screen } from '@testing-library/svelte';
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
import { setVideoPlacement } from '../stores/videoStageStore.js';
import VideoStageHarness from './fixtures/VideoStageHarness.svelte';
import { fakeVideo } from './fixtures/fakeVideo.js';
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
    fakeVideo.now = 65000;
    source = 'C:/videos/match.mov';
    entry = null;
    databasePathStore.set('/tmp/test.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
    /** @type {any} */ (LegalMoves).mockResolvedValue(PLAYS);
    /** @type {any} */ (EvaluatePositionImmediate).mockResolvedValue({ moves: RANKED });
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

describe('the automatic Repères', () => {
    test('a new roll and its explicit validation carry the instant of the video', async () => {
        await openedPanel();
        await press('Digit3');
        fakeVideo.now = 71000;
        await press('Digit1');
        await vi.waitFor(() => expect(gestures().some((g) => g.Kind === 'select_candidate')).toBe(true));
        fakeVideo.now = 76000;
        await press('Enter');

        const sent = gestures();
        expect(sent).toContainEqual({ Kind: 'enter_die', Die: 3, TickMS: 65000, HasTick: true });
        expect(sent).toContainEqual({ Kind: 'enter_die', Die: 1, TickMS: 71000, HasTick: true });
        expect(sent).toContainEqual({ Kind: 'validate', TickMS: 76000, HasTick: true });
        // Choosing a candidate is not an instant of the game.
        expect(sent.find((g) => g.Kind === 'select_candidate')).not.toHaveProperty('HasTick');
    });

    test('the validation sent ahead of the next roll’s digit carries no instant, and the status bar says so', async () => {
        await openedPanel();
        await press('Digit3');
        await press('Digit1');
        await vi.waitFor(() => expect(gestures().some((g) => g.Kind === 'select_candidate')).toBe(true));
        /** @type {any} */ (ApplyTranscriptionGesture).mockClear();
        fakeVideo.now = 80000;
        await press('Digit5');

        expect(gestures()).toEqual([{ Kind: 'validate' }, { Kind: 'enter_die', Die: 5, TickMS: 80000, HasTick: true }]);
        expect(get(transcriptionNoticeStore)?.key).toBe('transcription.notice.untimedAction');
    });

    test('a cube gesture carries the instant of its own action', async () => {
        await openedPanel();
        await press('KeyD');
        expect(gestures()).toContainEqual({ Kind: 'double', TickMS: 65000, HasTick: true });
    });

    test('a cube gesture on a preselected play validates it without an instant, and says so', async () => {
        entry = { at: ACTIONS.length, kind: 'checker', replacing: false, dice: [3, 1], selected: true, game_start: false };
        await openedPanel();
        await press('KeyD');
        expect(gestures()).toEqual([{ Kind: 'double', TickMS: 65000, HasTick: true }]);
        expect(get(transcriptionNoticeStore)?.key).toBe('transcription.notice.untimedAction');
    });

    test('a play finished on the board times the action, not the roll its dice were not typed for', async () => {
        entry = { at: ACTIONS.length, kind: 'checker', replacing: false, dice: [3, 1], selected: false, game_start: false };
        await openedPanel();
        const cell = /** @type {HTMLElement} */ (document.querySelector('.cell[data-pending="true"]'));
        await fireEvent.dblClick(cell);
        await settle();
        const field = /** @type {HTMLInputElement} */ (document.querySelector('.transcript-view input'));
        await fireEvent.input(field, { target: { value: '8/5 6/5' } });
        await fireEvent.keyDown(field, { key: 'Enter', code: 'Enter' });
        await settle(20);
        const sent = gestures();
        expect(sent.filter((g) => g.Kind === 'enter_die')).toEqual([
            { Kind: 'enter_die', Die: 3 },
            { Kind: 'enter_die', Die: 1 }
        ]);
        expect(sent.find((g) => g.Kind === 'enter_play')).toMatchObject({ TickMS: 65000, HasTick: true });
        expect(sent.find((g) => g.Kind === 'validate')).toEqual({ Kind: 'validate', TickMS: 65000, HasTick: true });
    });

    test('a correction of the last Action validated by the next digit: the new roll is timed, nothing to remind', async () => {
        entry = { at: ACTIONS.length - 1, kind: 'checker', replacing: true, dice: [5, 2], selected: true, game_start: false };
        await openedPanel(ACTIONS.length - 1);
        // Retyped on the last Action, the roll is validated by the digit that follows.
        await press('Digit6');
        await press('Digit4');
        await vi.waitFor(() => expect(gestures().some((g) => g.Kind === 'select_candidate')).toBe(true));
        /** @type {any} */ (ApplyTranscriptionGesture).mockClear();
        fakeVideo.now = 99000;
        await press('Digit2');
        expect(gestures()).toEqual([{ Kind: 'validate' }, { Kind: 'enter_die', Die: 2, TickMS: 99000, HasTick: true }]);
        expect(get(transcriptionNoticeStore)?.key).not.toBe('transcription.notice.untimedAction');
    });

    test('a correction in place sends no instant', async () => {
        entry = { at: 0, kind: 'checker', replacing: true, dice: [0, 0], game_start: false };
        await openedPanel(0);
        await press('Digit4');
        const die = gestures().find((g) => g.Kind === 'enter_die');
        expect(die).toEqual({ Kind: 'enter_die', Die: 4 });
    });

    test('without a source nothing is timed and the video keys do nothing', async () => {
        source = '';
        await openedPanel();
        expect(screen.queryByTestId('video-pane')).toBeNull();
        await press('Digit3');
        await press('Digit1');
        await press('KeyV');
        await press('ArrowRight', { shiftKey: true });
        const sent = gestures();
        expect(sent).toContainEqual({ Kind: 'enter_die', Die: 3 });
        expect(sent.some((g) => g.Kind === 'set_timecode' || 'HasTick' in g)).toBe(false);
        expect(fakeVideo.toggles).toBe(0);
        expect(fakeVideo.seeks).toEqual([]);
    });
});

describe('the video keys', () => {
    test('Space plays or pauses, and never reaches the global tier', async () => {
        await openedPanel();
        await press('Space');
        expect(fakeVideo.toggles).toBe(1);
        expect(globalKeys).not.toHaveBeenCalled();
    });

    test('without a video, Space goes on to the global tier (the command line)', async () => {
        source = '';
        await openedPanel();
        await press('Space');
        expect(fakeVideo.toggles).toBe(0);
        expect(globalKeys).toHaveBeenCalledTimes(1);
        expect(globalKeys.mock.calls[0][0].code).toBe('Space');
    });

    test('Shift+Left/Right step 5 s, Ctrl+Shift+Left/Right 1 s, and Ctrl+Left/Right still turn the board', async () => {
        await openedPanel();
        await press('ArrowRight', { shiftKey: true });
        await press('ArrowLeft', { shiftKey: true });
        await press('ArrowRight', { ctrlKey: true, shiftKey: true });
        await press('ArrowLeft', { ctrlKey: true, shiftKey: true });
        expect(fakeVideo.seeks).toEqual([70000, 60000, 66000, 64000]);
        await press('ArrowLeft', { ctrlKey: true });
        expect(fakeVideo.seeks).toHaveLength(4);
        expect(globalKeys).toHaveBeenCalledTimes(1);
        expect(gestures()).toEqual([]);
    });

    test('v times the action under the Cursor, MAJ-V its roll', async () => {
        await openedPanel(1);
        await press('KeyV');
        fakeVideo.now = 88000;
        await press('KeyV', { shiftKey: true });
        expect(gestures()).toEqual([
            { Kind: 'set_timecode', TickMS: 65000, HasTick: true },
            { Kind: 'set_timecode', RollTickMS: 88000, HasRollTick: true }
        ]);
    });

    test('v is read by its label, not by its place', async () => {
        await openedPanel(1);
        // Dvorak: the key labelled v sits where QWERTY has the period.
        await press('Period', { key: 'v' });
        await press('KeyV', { key: 'k' });
        expect(gestures().filter((g) => g.Kind === 'set_timecode')).toEqual([{ Kind: 'set_timecode', TickMS: 65000, HasTick: true }]);
    });

    test('v at the end of the document, with no Action under the Cursor, says so', async () => {
        await openedPanel();
        await press('KeyV');
        expect(gestures()).toEqual([]);
        expect(get(transcriptionNoticeStore)?.key).toBe('transcription.notice.noAction');
    });
});

describe('the jump on a cell', () => {
    test('placing the Cursor on a cell brings the video a second before its roll', async () => {
        await openedPanel();
        const cell = /** @type {HTMLElement} */ (document.querySelector('.cell[data-index="0"]'));
        await fireEvent.click(cell);
        await settle(20);
        expect(fakeVideo.seeks).toEqual([69000]);
    });

    test('h onto an Action timed only by its action seeks there', async () => {
        await openedPanel();
        await press('KeyH');
        await settle(20);
        expect(fakeVideo.seeks).toEqual([89000]);
    });

    test('a timed cell says its Repères and the deduced duration', async () => {
        await openedPanel();
        const cell = /** @type {HTMLElement} */ (document.querySelector('.cell[data-index="0"]'));
        expect(cell.dataset.timed).toBe('true');
        expect(cell.getAttribute('title')).toBe('roll 1:10, play 1:16, 6.5 s');
    });
});

describe('the video beside the board', () => {
    /** @type {(() => void) | null} */
    let unregisterBoard = null;
    const boardKeys = vi.fn();

    beforeEach(() => {
        boardKeys.mockClear();
        unregisterBoard = registerKeys('boardOrientation', boardKeys);
    });
    afterEach(() => {
        unregisterBoard?.();
        setVideoPlacement('board');
    });

    /** The main area beside the panel, the focus on the video's own button. */
    async function besideTheBoard() {
        setVideoPlacement('board');
        render(VideoStageHarness, { props: { withVideo: false } });
        await openedPanel();
        await settle();
        const stage = screen.getByTestId('video-stage');
        expect(stage.contains(screen.getByTestId('video-pane'))).toBe(true);
        /** @type {HTMLElement} */ (screen.getByTestId('video-placement')).focus();
    }

    test('focus in the video beside the board keeps the panel keys, and Ctrl+Left/Right still turn the board', async () => {
        await besideTheBoard();
        await press('Space');
        expect(fakeVideo.toggles).toBe(1);
        await press('ArrowRight', { shiftKey: true });
        expect(fakeVideo.seeks).toEqual([70000]);
        await press('ArrowLeft', { ctrlKey: true });
        await press('ArrowRight', { ctrlKey: true });
        expect(boardKeys).toHaveBeenCalledTimes(2);
        expect(fakeVideo.seeks).toEqual([70000]);
    });

    test('in the panel, the slot takes its height and Ctrl+Left/Right still turn the board', async () => {
        setVideoPlacement('panel');
        render(VideoStageHarness, { props: { withVideo: false } });
        await openedPanel();
        await settle();
        expect(screen.queryByTestId('video-stage')).toBeNull();
        expect(document.getElementById('transcriptionPanel')?.contains(screen.getByTestId('video-pane'))).toBe(true);
        expect(screen.getByRole('separator', { name: /./ })).toBeTruthy();
        await press('ArrowLeft', { ctrlKey: true });
        expect(boardKeys).toHaveBeenCalledTimes(1);
    });

    test('[ and ] step the speed, AltGr included, never from a field', async () => {
        await besideTheBoard();
        await press('BracketRight', { key: ']' });
        await press('BracketLeft', { key: '[' });
        // AZERTY: AltGr+5 types [, AltGr+° types ].
        await press('Digit5', { key: '[', altKey: true });
        expect(fakeVideo.rateSteps).toEqual([1, -1, -1]);
        expect(globalKeys).not.toHaveBeenCalled();
        const field = document.createElement('input');
        document.getElementById('transcriptionPanel')?.appendChild(field);
        field.focus();
        await press('BracketRight', { key: ']' });
        expect(fakeVideo.rateSteps).toEqual([1, -1, -1]);
        field.remove();
    });
});
