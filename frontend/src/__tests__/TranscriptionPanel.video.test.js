/**
 * TranscriptionPanel.video.test.js — the video of a Transcription (ADR-0079 rule 3).
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
import { fakeVideo } from './fixtures/fakeVideo.js';

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

/** @param {string} key */
async function press(key, extra = {}) {
    const code = /^[1-6]$/.test(key) ? `Digit${key}` : undefined;
    await fireEvent.keyDown(document, { key, code, ...extra });
    await settle();
}

let at = ACTIONS.length;

beforeEach(() => {
    vi.clearAllMocks();
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
    cleanup();
    clearTranscription();
    selectedMoveStore.set(null);
    statusBarModeStore.set('NORMAL');
    activeTabStore.set('position');
});

describe('the automatic Repères', () => {
    test('a new roll and its explicit validation carry the instant of the video', async () => {
        await openedPanel();
        await press('3');
        fakeVideo.now = 71000;
        await press('1');
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
        await press('3');
        await press('1');
        await vi.waitFor(() => expect(gestures().some((g) => g.Kind === 'select_candidate')).toBe(true));
        /** @type {any} */ (ApplyTranscriptionGesture).mockClear();
        fakeVideo.now = 80000;
        await press('5');

        expect(gestures()).toEqual([{ Kind: 'validate' }, { Kind: 'enter_die', Die: 5, TickMS: 80000, HasTick: true }]);
        expect(get(transcriptionNoticeStore)?.key).toBe('transcription.notice.untimedAction');
    });

    test('a cube gesture carries the instant of its own action', async () => {
        await openedPanel();
        await press('d');
        expect(gestures()).toContainEqual({ Kind: 'double', TickMS: 65000, HasTick: true });
    });

    test('a correction in place sends no instant', async () => {
        entry = { at: 0, kind: 'checker', replacing: true, dice: [0, 0], game_start: false };
        await openedPanel(0);
        await press('4');
        const die = gestures().find((g) => g.Kind === 'enter_die');
        expect(die).toEqual({ Kind: 'enter_die', Die: 4 });
    });

    test('without a source nothing is timed and the video keys do nothing', async () => {
        source = '';
        await openedPanel();
        expect(screen.queryByTestId('video-pane')).toBeNull();
        await press('3');
        await press('1');
        await press(' ');
        await press('v');
        await press('.');
        const sent = gestures();
        expect(sent).toContainEqual({ Kind: 'enter_die', Die: 3 });
        expect(sent.some((g) => g.Kind === 'set_timecode' || 'HasTick' in g)).toBe(false);
        expect(fakeVideo.toggles).toBe(0);
        expect(fakeVideo.seeks).toEqual([]);
    });
});

describe('the video keys', () => {
    test('Space plays or pauses, comma and period step 5 s, with Shift 1 s', async () => {
        await openedPanel();
        await press(' ');
        expect(fakeVideo.toggles).toBe(1);
        await press('.');
        await press(',');
        await press('.', { shiftKey: true });
        await press('<', { shiftKey: true });
        expect(fakeVideo.seeks).toEqual([70000, 60000, 66000, 64000]);
        expect(gestures()).toEqual([]);
    });

    test('v times the action under the Cursor, MAJ-V its roll', async () => {
        await openedPanel(1);
        await press('v');
        fakeVideo.now = 88000;
        await press('V', { shiftKey: true });
        expect(gestures()).toEqual([
            { Kind: 'set_timecode', TickMS: 65000, HasTick: true },
            { Kind: 'set_timecode', RollTickMS: 88000, HasRollTick: true }
        ]);
    });

    test('v at the end of the document, with no Action under the Cursor, says so', async () => {
        await openedPanel();
        await press('v');
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
        await press('h');
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
