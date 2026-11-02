/**
 * vision.available.panel.test.js — the vision switch seen from the Transcription panel. Cut, the
 * panel shows nothing of the vision, leaves `g` to the rest of the application and a retained
 * calibration starts no session; on, all of it is there.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent, screen } from '@testing-library/svelte';
import { tick } from 'svelte';

const visionSwitch = vi.hoisted(() => ({ on: false }));
const session = vi.hoisted(() => ({ calibrate: vi.fn(), dispose: vi.fn() }));
vi.mock('../vision/available.js', () => ({
    get VISION_AVAILABLE() {
        return visionSwitch.on;
    }
}));
vi.mock('../services/visionSession.js', () => ({ createVisionSession: vi.fn(() => session) }));
vi.mock('../components/VideoPane.svelte', async () => await import('./fixtures/FakeVideoPane.svelte'));
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    TranscriptionCandidates: vi.fn().mockResolvedValue({}),
    ListTranscriptions: vi.fn().mockResolvedValue([]),
    CreateTranscription: vi.fn(),
    OpenTranscription: vi.fn(),
    ApplyTranscriptionGesture: vi.fn(),
    TranscriptionMAT: vi.fn().mockResolvedValue('')
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    LegalMoves: vi.fn().mockResolvedValue([]),
    PickTranscriptionVideo: vi.fn().mockResolvedValue('')
}));

import { ApplyTranscriptionGesture } from '../../wailsjs/go/database/Database.js';
import { createVisionSession } from '../services/visionSession.js';
import TranscriptionPanel from '../components/TranscriptionPanel.svelte';
import { transcriptionStore, clearTranscription } from '../stores/transcriptionStore.js';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { visionEventsStore, resetVisionEvents } from '../stores/visionEventsStore.js';
import { makeEvent, EVENT_TYPES as T, STATES as S } from '../vision/contract.js';
import { registerKeys } from '../services/keyDispatch.js';
import { fakeVideo } from './fixtures/fakeVideo.js';

const POSITION = (/** @type {number[]} */ dice) => ({
    board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [0, 0] },
    cube: { owner: -1, value: 0 },
    dice,
    score: [7, 7],
    player_on_roll: 0,
    decision_type: 0
});

const annotated = () => ({
    document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice', video_source: 'C:/videos/match.mov' }, actions: [], cursor: 0 },
    actions: [],
    games: [{ number: 1, initial_score: [0, 0], winner: -1, points_won: 0, crawford: false, finished: false, first: 0, last: -1 }],
    entry: null,
    next: { expects: 'checker', side: 0, position: POSITION([0, 0]), crawford: false },
    score: [0, 0],
    cursor: 0
});

const globalKeys = vi.fn();
/** @type {(() => void) | null} */
let unregisterGlobal = null;

async function settle(times = 10) {
    for (let i = 0; i < times; i++) {
        await tick();
        await Promise.resolve();
    }
}

async function openedPanel() {
    transcriptionStore.set({ id: 1, annotated: annotated() });
    render(TranscriptionPanel);
    await settle();
    document.getElementById('transcriptionPanel')?.focus();
}

/** @param {string} key */
async function press(key, shiftKey = false) {
    await fireEvent.keyDown(document.activeElement ?? document, { code: 'KeyG', key, shiftKey, bubbles: true, cancelable: true });
    await settle();
}

// A store fed by hand: what the panel does with the events, whichever way they got there.
const feedAll = () => visionEventsStore.set([makeEvent(T.DICE_REST, 62, S.SURE), makeEvent(T.DICE_PICKUP, 70, S.CHECK), makeEvent(T.OVERLAY_CHANGED, 64, S.CHECK)]);

beforeEach(() => {
    vi.clearAllMocks();
    unregisterGlobal = registerKeys('global', globalKeys);
    fakeVideo.reset();
    resetVisionEvents();
    fakeVideo.now = 65000;
    fakeVideo.duration = 600000;
    databasePathStore.set('/tmp/test.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
    vi.mocked(ApplyTranscriptionGesture).mockResolvedValue(/** @type {any} */ ({ id: 1, annotated: annotated() }));
});

afterEach(() => {
    unregisterGlobal?.();
    cleanup();
    clearTranscription();
    statusBarModeStore.set('NORMAL');
    activeTabStore.set('position');
});

describe('the switch cut', () => {
    beforeEach(() => {
        visionSwitch.on = false;
    });

    test('no line of markers, no drift pictogram, no bench export', async () => {
        feedAll();
        await openedPanel();
        await new Promise((r) => setTimeout(r, 600));
        expect(screen.queryByTestId('video-timeline')).toBeNull();
        expect(screen.queryByTestId('video-overlay-drift')).toBeNull();
        expect(document.querySelector('.bench-export')).toBeNull();
    });

    test.each([
        ['g', false],
        ['G', true]
    ])('%s is not taken: it goes on to the rest of the application and moves nothing', async (key, shift) => {
        feedAll();
        await openedPanel();
        await press(key, shift);
        expect(fakeVideo.seeks).toEqual([]);
        expect(globalKeys).toHaveBeenCalled();
    });

    test('a retained calibration creates no session', async () => {
        await openedPanel();
        fakeVideo.oncalibration?.({ name: 'Calibration', clicks: [], imageToModel: [], modelToImage: [] });
        await settle();
        expect(createVisionSession).not.toHaveBeenCalled();
    });

    test('with no event, the first die keeps the current instant as its Repère', async () => {
        await openedPanel();
        await fireEvent.keyDown(document.activeElement ?? document, { code: 'Digit3', key: '3', bubbles: true, cancelable: true });
        await settle();
        const sent = vi.mocked(ApplyTranscriptionGesture).mock.calls.map((/** @type {any} */ call) => call[1]);
        expect(sent).toContainEqual({ Kind: 'enter_die', Die: 3, TickMS: 65000, HasTick: true });
    });
});

describe('the switch on', () => {
    beforeEach(() => {
        visionSwitch.on = true;
    });

    test('the line, the drift pictogram and the bench export show', async () => {
        feedAll();
        await openedPanel();
        await vi.waitFor(() => expect(screen.queryByTestId('video-timeline')).not.toBeNull(), { timeout: 2000 });
        await vi.waitFor(() => expect(screen.queryByTestId('video-overlay-drift')).not.toBeNull(), { timeout: 2000 });
        expect(document.querySelector('.bench-export')).not.toBeNull();
    });

    test('g jumps to the next marker and is kept from the rest of the application', async () => {
        feedAll();
        await openedPanel();
        await press('g');
        expect(fakeVideo.seeks).toEqual([70000]);
        expect(globalKeys).not.toHaveBeenCalled();
    });

    test('a retained calibration creates the session and feeds it', async () => {
        await openedPanel();
        const calibration = { name: 'Calibration', clicks: [], imageToModel: [], modelToImage: [] };
        fakeVideo.oncalibration?.(calibration);
        await settle();
        expect(createVisionSession).toHaveBeenCalledTimes(1);
        expect(session.calibrate).toHaveBeenCalledWith(calibration);
    });
});
