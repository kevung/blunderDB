/**
 * TranscriptionPanel.videoFocus.test.js — a click in the video leaves the keyboard to the
 * Transcription. The real player this time: a focused <video> reads the arrows and Space
 * itself, and a focused YouTube frame swallows every key into a document the app cannot read.
 * Whether the video sits beside the board or in the panel, the next die typed is the draft's.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent, screen } from '@testing-library/svelte';
import { tick } from 'svelte';

const host = vi.hoisted(() => ({ kind: 'file' }));

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListTranscriptions: vi.fn().mockResolvedValue([]),
    CreateTranscription: vi.fn(),
    OpenTranscription: vi.fn(),
    ApplyTranscriptionGesture: vi.fn(),
    TranscriptionMAT: vi.fn().mockResolvedValue('')
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    LegalMoves: vi.fn().mockResolvedValue([]),
    EvaluatePositionImmediate: vi.fn().mockResolvedValue({ moves: [] }),
    PickTranscriptionVideo: vi.fn().mockResolvedValue(''),
    VideoSourceKind: vi.fn(() => Promise.resolve(host.kind)),
    MediaURL: vi.fn(() => Promise.resolve('http://127.0.0.1:1/m/tok')),
    YouTubeEmbedURL: vi.fn(() => Promise.resolve('http://127.0.0.1:1/yt/abc')),
    ReleaseMedia: vi.fn(() => Promise.resolve())
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetPruneK: vi.fn().mockResolvedValue(0)
}));

import { ApplyTranscriptionGesture } from '../../wailsjs/go/database/Database.js';
import TranscriptionPanel from '../components/TranscriptionPanel.svelte';
import { transcriptionStore, clearTranscription } from '../stores/transcriptionStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';
import { setVideoPlacement } from '../stores/videoStageStore.js';
import VideoStageHarness from './fixtures/VideoStageHarness.svelte';

const POSITION = {
    board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [0, 0] },
    cube: { owner: -1, value: 0 },
    dice: [0, 0],
    score: [7, 7],
    player_on_roll: 0,
    decision_type: 0
};

function annotated(source) {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice', video_source: source }, actions: [], cursor: 0 },
        actions: [],
        games: [{ number: 1, initial_score: [0, 0], winner: -1, points_won: 0, crawford: false, finished: false, first: 0, last: -1 }],
        entry: null,
        next: { expects: 'checker', side: 0, position: POSITION, crawford: false },
        score: [0, 0],
        cursor: 0
    };
}

async function settle(times = 10) {
    for (let i = 0; i < times; i++) {
        await tick();
        await Promise.resolve();
    }
}

const dice = () => /** @type {any} */ (ApplyTranscriptionGesture).mock.calls.map((/** @type {any} */ c) => c[1]).filter((g) => g.Kind === 'enter_die');

async function draftWithVideo(/** @type {string} */ source, /** @type {'board' | 'panel'} */ placement) {
    setVideoPlacement(placement);
    render(VideoStageHarness, { props: { withVideo: false } });
    transcriptionStore.set({ id: 1, annotated: annotated(source) });
    render(TranscriptionPanel);
    await settle();
    // The keyboard is elsewhere before the click: on the board, say.
    /** @type {HTMLElement} */ (document.activeElement)?.blur?.();
    expect(document.activeElement).toBe(document.body);
}

async function typeDie() {
    await fireEvent.keyDown(document.activeElement ?? document.body, { code: 'Digit3', key: '3', bubbles: true, cancelable: true });
    await settle();
}

beforeEach(() => {
    vi.clearAllMocks();
    host.kind = 'file';
    databasePathStore.set('/tmp/test.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
    /** @type {any} */ (ApplyTranscriptionGesture).mockImplementation(() => Promise.resolve({ id: 1, annotated: annotated('/v/match.mp4') }));
});

afterEach(() => {
    cleanup();
    clearTranscription();
    statusBarModeStore.set('NORMAL');
    activeTabStore.set('position');
    setVideoPlacement('board');
});

describe('a click in the video leaves the keyboard to the Transcription', () => {
    for (const placement of /** @type {const} */ (['board', 'panel'])) {
        test(`a file, ${placement === 'board' ? 'beside the board' : 'in the panel'}: clicking the <video> then a die`, async () => {
            await draftWithVideo('/v/match.mp4', placement);
            const video = /** @type {HTMLVideoElement} */ (await vi.waitFor(() => document.querySelector('video') ?? expect.fail('no video')));
            expect(!!screen.queryByTestId('video-stage')?.contains(video)).toBe(placement === 'board');
            await fireEvent.pointerDown(video);
            video.focus();
            await fireEvent.focus(video);
            await new Promise((r) => setTimeout(r, 0));
            expect(document.activeElement).not.toBe(video);
            await typeDie();
            expect(dice()).toContainEqual(expect.objectContaining({ Kind: 'enter_die', Die: 3 }));
        });
    }

    test('a YouTube frame that took the focus gives it back, and the die is the draft’s', async () => {
        host.kind = 'youtube';
        await draftWithVideo('https://youtu.be/dQw4w9WgXcQ', 'board');
        const frame = /** @type {HTMLIFrameElement} */ (await vi.waitFor(() => document.querySelector('iframe') ?? expect.fail('no frame')));
        frame.focus();
        window.dispatchEvent(new Event('blur'));
        await new Promise((r) => setTimeout(r, 0));
        expect(document.activeElement).not.toBe(frame);
        await typeDie();
        expect(dice()).toContainEqual(expect.objectContaining({ Kind: 'enter_die', Die: 3 }));
    });

    test('a drag on the separator leaves the keyboard where it was', async () => {
        await draftWithVideo('/v/match.mp4', 'board');
        document.getElementById('transcriptionPanel')?.focus();
        const handle = screen.getByTestId('video-stage-resize');
        const down = await fireEvent.mouseDown(handle);
        expect(down).toBe(false);
        await typeDie();
        expect(dice()).toContainEqual(expect.objectContaining({ Kind: 'enter_die', Die: 3 }));
    });
});
