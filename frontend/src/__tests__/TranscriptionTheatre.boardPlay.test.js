/**
 * TranscriptionTheatre.boardPlay.test.js — a play taken on the theatre's board, through the whole
 * hierarchy (panel, theatre, mini board) with ranked candidates: the selected candidate's arrows
 * leave the drawing at the first step and stay off while the play is in progress, and a right
 * click takes the play back.
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
    YouTubeWatchURL: vi.fn().mockResolvedValue('')
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetPruneK: vi.fn().mockResolvedValue(0)
}));
vi.mock('../components/VideoPane.svelte', async () => await import('./fixtures/FakeVideoPane.svelte'));

import { ApplyTranscriptionGesture } from '../../wailsjs/go/database/Database.js';
import { LegalMoves, EvaluatePositionImmediate } from '../../wailsjs/go/gui/App.js';

import TranscriptionPanel from '../components/TranscriptionPanel.svelte';
import TranscriptionTheatre from '../components/TranscriptionTheatre.svelte';
import { exitTheatre, enterTheatre } from '../services/transcriptionTheatre.js';
import { fakeVideo } from './fixtures/fakeVideo.js';
import { transcriptionStore, clearTranscription } from '../stores/transcriptionStore.js';
import { quizPlayStore } from '../stores/quizPlayStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';
import { DIAGRAM_WIDTH, DIAGRAM_HEIGHT } from '../services/diagramService.js';
import { boardMetrics } from '../utils/boardGeometry.js';
import { defaultBoardConfig } from '../utils/boardConfig.js';
import { ARROW_COLORS, stackSlotCenter } from '../utils/boardScene.js';

const BLACK = 0;
const WHITE = 1;

function positionWith(/** @type {any} */ stacks) {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    for (const [point, [checkers, color]] of Object.entries(stacks)) points[Number(point)] = { checkers, color };
    return { board: { points, bearoff: [0, 0] }, cube: { owner: -1, value: 0 }, dice: [0, 0], score: [7, 7], player_on_roll: BLACK, decision_type: 0 };
}
const POSITION = positionWith({ 13: [5, BLACK], 8: [3, BLACK], 6: [5, BLACK], 12: [2, WHITE] });

const step = (/** @type {number} */ from, /** @type {number} */ to) => ({ from, to, hit: false });
const play = (/** @type {any[]} */ ...steps) => ({ steps, notation: steps.map((s) => `${s.from}/${s.to}`).join(' '), result: {} });
const PLAYS_61 = [play(step(8, 2), step(6, 5)), play(step(13, 7), step(13, 12)), play(step(13, 7), step(8, 7))];
const RANKED_61 = PLAYS_61.map((p, index) => ({ index, move: p.notation, equity: -index / 10 }));

function annotated() {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice', video_source: 'C:/videos/match.mov' }, actions: [], cursor: 0 },
        actions: [],
        games: [],
        next: { expects: 'checker', side: BLACK, position: POSITION, crawford: false, game_number: 1 },
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

const panel = () => /** @type {HTMLElement} */ (document.getElementById('transcriptionPanel'));
const drawing = () => /** @type {HTMLElement} */ (document.querySelector('[data-testid="theatre-board-drawing"]'));
// The arrows' colours, as boardScene.drawMoveArrows paints them: a written move or a line of the analysis.
const arrows = () => Object.values(ARROW_COLORS).reduce((n, colour) => n + drawing().innerHTML.split(colour).length - 1, 0);
const steps = () => get(quizPlayStore)?.steps.map((/** @type {any} */ s) => [s.from, s.to]);

const W = DIAGRAM_WIDTH;
const H = DIAGRAM_HEIGHT + 24;
const cfg = { ...defaultBoardConfig(), orientation: /** @type {'right'} */ ('right') };
const geom = boardMetrics(W, H, cfg.widthFactor);
const slot = (/** @type {number} */ point) => /** @type {{ x: number, y: number }} */ (stackSlotCenter(geom, cfg, point, 0));

/**
 * @param {string} type
 * @param {{ x: number, y: number }} at
 * @param {number} button
 */
function fire(type, at, button) {
    const Ctor = type.startsWith('pointer') && typeof PointerEvent === 'function' ? PointerEvent : MouseEvent;
    drawing().dispatchEvent(new Ctor(type, { clientX: at.x, clientY: at.y, button, buttons: button === 2 ? 2 : 1, bubbles: true, cancelable: true }));
}
async function clickAt(/** @type {{ x: number, y: number }} */ at) {
    fire('pointerdown', at, 0);
    fire('pointerup', at, 0);
    await settle();
}
/** A right click as a mouse gives it: press, release, then the menu's event. */
async function rightClickAt(/** @type {{ x: number, y: number }} */ at) {
    fire('pointerdown', at, 2);
    fire('pointerup', at, 2);
    fire('contextmenu', at, 2);
    await settle();
}

/** The theatre open over a draft, the roll 61 typed, its candidates ranked. */
async function armedInTheatre() {
    transcriptionStore.set({ id: 1, annotated: annotated() });
    render(TranscriptionTheatre);
    render(TranscriptionPanel);
    await settle();
    await enterTheatre();
    await settle();
    panel().focus();
    for (const [code, key] of [
        ['Digit6', '6'],
        ['Digit1', '1']
    ]) {
        await fireEvent.keyDown(panel(), { code, key, bubbles: true, cancelable: true });
        await settle();
    }
    await vi.waitFor(() => expect(get(quizPlayStore)?.rolled).toEqual([6, 1]));
    await vi.waitFor(() => expect(get(selectedMoveStore)).toBe('8/2 6/5'));
    vi.spyOn(Element.prototype, 'getBoundingClientRect').mockReturnValue(new DOMRect(0, 0, W, H));
    await settle();
}

beforeEach(() => {
    vi.clearAllMocks();
    localStorage.clear();
    databasePathStore.set('/tmp/test.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
    /** @type {any} */ (LegalMoves).mockImplementation((/** @type {any} */ pos) => Promise.resolve(pos.dice[0] * pos.dice[1] === 6 ? PLAYS_61 : []));
    /** @type {any} */ (EvaluatePositionImmediate).mockResolvedValue({ moves: RANKED_61 });
    /** @type {any} */ (ApplyTranscriptionGesture).mockImplementation(() => Promise.resolve({ id: 1, annotated: annotated() }));
});

afterEach(async () => {
    vi.restoreAllMocks();
    await exitTheatre();
    fakeVideo.reset();
    cleanup();
    clearTranscription();
    quizPlayStore.set(null);
    selectedMoveStore.set(null);
});

describe('a play on the theatre board, through the panel', () => {
    test('the first step takes the candidate arrows off, and they stay off while the play is in progress', async () => {
        await armedInTheatre();
        // The preselected line draws nothing; `j` points at a line, which does.
        expect(arrows()).toBe(0);
        await fireEvent.keyDown(panel(), { code: 'KeyJ', key: 'j', bubbles: true, cancelable: true });
        await settle();
        expect(arrows()).toBeGreaterThan(0);
        await clickAt(slot(13));
        expect(steps()).toEqual([[13, 7]]);
        expect(arrows()).toBe(0);
        // The panel narrows the candidates to the step and preselects the first left.
        await vi.waitFor(() => expect(get(selectedMoveStore)).toBe('13/7 13/12'));
        await settle();
        expect(arrows()).toBe(0);
    });

    test('a right click on the board takes the play back', async () => {
        await armedInTheatre();
        await clickAt(slot(13));
        expect(steps()).toEqual([[13, 7]]);
        await rightClickAt(slot(6));
        expect(steps()).toEqual([]);
    });
});
