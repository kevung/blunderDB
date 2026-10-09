/**
 * MatchPanel.video.test.js
 *
 * A match that carries a video source offers « View in the video » on each decision that
 * carries a Repère (the icon of its row, or the `v` key), by the path its source calls for:
 * the pane for a file, the browser at the timestamped link for YouTube. Without a source,
 * nothing changes on screen.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

const state = vi.hoisted(() => ({ source: '', kind: '' }));

const MOVES = [
    { move_id: 101, game_number: 1, move_number: 1, move_type: 'checker', player_on_roll: 0, position: { dice: [3, 1] }, checker_move: '8/5 6/5', roll_tick_ms: 65000, tick_ms: 70000 },
    { move_id: 102, game_number: 1, move_number: 2, move_type: 'checker', player_on_roll: 1, position: { dice: [6, 5] }, checker_move: '24/13' },
    { move_id: 103, game_number: 1, move_number: 3, move_type: 'cube', player_on_roll: 0, position: { dice: [0, 0] }, cube_action: 'Double', tick_ms: 500 }
];

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    VideoSourceKind: vi.fn(() => Promise.resolve(state.kind)),
    OpenVideoExternally: vi.fn(() => Promise.resolve()),
    PickTranscriptionVideo: vi.fn(() => Promise.resolve('')),
    MediaURL: vi.fn(() => Promise.resolve('http://127.0.0.1:1/m/x')),
    YouTubeEmbedURL: vi.fn(() => Promise.resolve('http://127.0.0.1:1/yt/x')),
    ReleaseMedia: vi.fn(() => Promise.resolve())
}));

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListMatches: vi.fn(() =>
        Promise.resolve([{ id: 7, player1_name: 'Alice', player2_name: 'Bob', match_length: 7, match_date: '2026-01-15', game_count: 1, video_source: state.source || undefined }])
    ),
    CountMatches: vi.fn(() => Promise.resolve(1)),
    GetMatchByID: vi.fn(() => Promise.resolve(null)),
    GetAllTournaments: vi.fn(() => Promise.resolve([])),
    ListTranscriptions: vi.fn(() => Promise.resolve([])),
    TrashMatch: vi.fn(() => Promise.resolve()),
    UpdateMatch: vi.fn(() => Promise.resolve()),
    UpdateMatchComment: vi.fn(() => Promise.resolve()),
    GetMatchMovePositions: vi.fn(() => Promise.resolve(MOVES)),
    GetGamesByMatch: vi.fn(() => Promise.resolve([{ game_number: 1, initial_score: [0, 0], winner: 1, points_won: 1 }])),
    GetMatchDetailStats: vi.fn(() => Promise.resolve(null)),
    GetMatchMoveGrades: vi.fn(() => Promise.resolve([])),
    GetMatchTimeSummary: vi.fn(() => Promise.resolve(null)),
    GetMatchOrigin: vi.fn(() => Promise.resolve(null)),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    SetMatchVideoSource: vi.fn(() => Promise.resolve()),
    SetMatchTournamentByName: vi.fn(() => Promise.resolve()),
    SwapMatchPlayers: vi.fn(() => Promise.resolve()),
    SaveLastVisitedPosition: vi.fn(() => Promise.resolve()),
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(() => Promise.resolve())
}));

import { openPanels, PANEL } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { lastVisitedMatchStore, matchContextStore } from '../stores/positionStore.js';
import { ListMatches, SetMatchVideoSource } from '../../wailsjs/go/database/Database.js';
import MatchPanel from '../components/MatchPanel.svelte';
import VideoStageHarness from './fixtures/VideoStageHarness.svelte';
import { setVideoPlacement } from '../stores/videoStageStore.js';

import { OpenVideoExternally, PickTranscriptionVideo } from '../../wailsjs/go/gui/App.js';

async function openTranscript() {
    const view = render(MatchPanel);
    const { container } = view;
    await vi.waitFor(() => expect(ListMatches).toHaveBeenCalledTimes(2));
    await new Promise((r) => setTimeout(r, 0));
    for (let i = 0; i < 6; i++) await tick();
    if (!container.querySelector('tbody tr.selected')) {
        const cell = [...container.querySelectorAll('tbody tr td')].find((td) => td.textContent.includes('Alice'));
        await fireEvent.click(cell);
    }
    await vi.waitFor(() => expect(container.querySelector('details.game-section')).not.toBeNull());
    for (let i = 0; i < 4; i++) await tick();
    return container;
}

describe('MatchPanel — View in the video', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        state.source = '';
        state.kind = '';
        databasePathStore.set('/tmp/test.db');
        openPanels.set(new Set([PANEL.MATCH]));
        lastVisitedMatchStore.set({ matchID: 7, currentIndex: 0, gameNumber: 1 });
    });
    afterEach(() => {
        cleanup();
        lastVisitedMatchStore.set({ matchID: null, currentIndex: 0, gameNumber: 1 });
        matchContextStore.set({ isMatchMode: false, matchID: null, currentIndex: 0 });
        openPanels.set(new Set());
    });

    test('a match without a source shows no video gesture', async () => {
        const container = await openTranscript();
        expect(container.querySelector('[data-testid="view-in-video"]')).toBeNull();
        expect(container.querySelector('[data-testid="match-video"]')).toBeNull();
    });

    test('only the decisions that carry a Repère get the gesture', async () => {
        state.source = 'https://youtu.be/dQw4w9WgXcQ';
        state.kind = 'youtube';
        const container = await openTranscript();
        await vi.waitFor(() => expect(container.querySelectorAll('[data-testid="view-in-video"]').length).toBe(2));
        expect(container.querySelector('[data-testid="match-video"]')).not.toBeNull();
    });

    test('a YouTube source opens the browser one second before the dice fell', async () => {
        state.source = 'https://youtu.be/dQw4w9WgXcQ';
        state.kind = 'youtube';
        const container = await openTranscript();
        await vi.waitFor(() => expect(container.querySelectorAll('[data-testid="view-in-video"]').length).toBe(2));
        await fireEvent.click(container.querySelectorAll('[data-testid="view-in-video"]')[0]);
        expect(OpenVideoExternally).toHaveBeenCalledWith('https://youtu.be/dQw4w9WgXcQ', 64000);
        // A cube action has only its action instant; the start never goes below zero.
        await fireEvent.click(container.querySelectorAll('[data-testid="view-in-video"]')[1]);
        expect(OpenVideoExternally).toHaveBeenLastCalledWith('https://youtu.be/dQw4w9WgXcQ', 0);
    });

    test('a file opens the pane in the panel, not the browser', async () => {
        state.source = '/videos/final.mp4';
        state.kind = 'file';
        const container = await openTranscript();
        await vi.waitFor(() => expect(container.querySelectorAll('[data-testid="view-in-video"]').length).toBe(2));
        await fireEvent.click(container.querySelectorAll('[data-testid="view-in-video"]')[0]);
        await vi.waitFor(() => expect(container.querySelector('[data-testid="video-pane"]')).not.toBeNull());
        expect(OpenVideoExternally).not.toHaveBeenCalled();
    });

    test('the source is detached from the match information', async () => {
        state.source = '/videos/final.mp4';
        state.kind = 'file';
        const container = await openTranscript();
        await fireEvent.click([...container.querySelectorAll('button')].find((b) => b.classList.contains('detail-tab') && b.textContent.trim() === 'Info'));
        await fireEvent.click(container.querySelector('[data-testid="video-detach"]'));
        await vi.waitFor(() => expect(SetMatchVideoSource).toHaveBeenCalledWith(7, ''));
    });

    test('relocating the file keeps the pane open and the source is written once', async () => {
        state.source = '/videos/final.mp4';
        state.kind = 'file';
        vi.mocked(PickTranscriptionVideo).mockResolvedValueOnce('/moved/final.mp4');
        const container = await openTranscript();
        await vi.waitFor(() => expect(container.querySelectorAll('[data-testid="view-in-video"]').length).toBe(2));
        await fireEvent.click(container.querySelectorAll('[data-testid="view-in-video"]')[0]);
        await vi.waitFor(() => expect(container.querySelector('[data-testid="video-pane"]')).not.toBeNull());
        await fireEvent.click([...container.querySelectorAll('button')].find((b) => b.classList.contains('detail-tab') && b.textContent.trim() === 'Info'));
        await fireEvent.click([...container.querySelectorAll('.meta-value button')].find((b) => b.textContent.trim() === 'File…'));
        await vi.waitFor(() => expect(SetMatchVideoSource).toHaveBeenCalledWith(7, '/moved/final.mp4'));
        for (let i = 0; i < 6; i++) await tick();
        expect(SetMatchVideoSource).toHaveBeenCalledTimes(1);
        expect(container.querySelector('[data-testid="video-pane"]')).not.toBeNull();
    });

    test('the file plays beside the board, and [ ] set its speed there, never from a field', async () => {
        setVideoPlacement('board');
        render(VideoStageHarness, { props: { withVideo: false } });
        state.source = '/videos/final.mp4';
        state.kind = 'file';
        const container = await openTranscript();
        await vi.waitFor(() => expect(container.querySelectorAll('[data-testid="view-in-video"]').length).toBe(2));
        await fireEvent.click(container.querySelectorAll('[data-testid="view-in-video"]')[0]);
        const stage = await vi.waitFor(() => document.querySelector('[data-testid="video-stage"]') ?? expect.fail('no stage'));
        const video = /** @type {HTMLVideoElement} */ (await vi.waitFor(() => stage.querySelector('video') ?? expect.fail('no video in the stage')));
        expect(container.querySelector('video')).toBeNull();
        await fireEvent(video, new Event('loadedmetadata'));
        await fireEvent.keyDown(document.body, { key: ']', code: 'BracketRight', bubbles: true, cancelable: true });
        await fireEvent.keyDown(document.body, { key: ']', code: 'Minus', altKey: true, bubbles: true, cancelable: true });
        expect(video.playbackRate).toBe(1.5);
        await fireEvent.keyDown(document.body, { key: '[', code: 'BracketLeft', bubbles: true, cancelable: true });
        expect(video.playbackRate).toBe(1.25);
        const field = /** @type {HTMLInputElement} */ (container.querySelector('input'));
        field.focus();
        await fireEvent.keyDown(field, { key: '[', code: 'BracketLeft', bubbles: true, cancelable: true });
        expect(video.playbackRate).toBe(1.25);
    });

    test('in the panel, the file plays under the match card', async () => {
        setVideoPlacement('panel');
        state.source = '/videos/final.mp4';
        state.kind = 'file';
        const container = await openTranscript();
        await fireEvent.click(container.querySelector('[data-testid="match-video"]'));
        await vi.waitFor(() => expect(container.querySelector('.match-video-slot [data-testid="video-pane"]')).not.toBeNull());
        setVideoPlacement('board');
    });
});
