// @ts-nocheck
import { describe, test, expect, vi, beforeEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { get } from 'svelte/store';

const handlers = vi.hoisted(() => /** @type {Record<string, (e: any) => void>} */ ({}));
const app = vi.hoisted(() => ({ StartScoreMap: vi.fn().mockResolvedValue(undefined), CancelScoreMap: vi.fn().mockResolvedValue(undefined) }));
vi.mock('../../wailsjs/go/gui/App.js', () => app);
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetDisplayPly: vi.fn().mockResolvedValue(2),
    GetGammonNetPruneK: vi.fn().mockResolvedValue(0),
    GetMapMatchLength: vi.fn().mockResolvedValue(3)
}));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    EventsOn: vi.fn((name, fn) => {
        handlers[name] = fn;
        return () => {};
    })
}));
vi.mock('../services/modeMachine.js', () => ({ sendPositionToEval: vi.fn(), toggleEvalMode: vi.fn() }));

import ScoreMapView from '../components/ScoreMapView.svelte';
import { positionStore, emptyPosition } from '../stores/positionStore.js';
import { evalViewStore, mapMatchLengthStore } from '../stores/mapStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { sendPositionToEval } from '../services/modeMachine.js';

function cells(n) {
    const out = [];
    for (let mine = 1; mine <= n; mine++)
        for (let theirs = 1; theirs <= n; theirs++) {
            const move = mine === theirs ? '8/5 6/5' : '24/21 13/11';
            out.push({ awayOnRoll: mine, awayOpponent: theirs, ply: 2, crawford: mine === 1 && theirs > 1, move, equity: 0.1, secondMove: '13/10', secondEquity: 0.05, gap: mine === 2 ? 0.005 : 0.1 });
        }
    return out;
}

describe('ScoreMapView', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        mapMatchLengthStore.set(null);
        evalViewStore.set('scores');
        selectedMoveStore.set(null);
        positionStore.set({ ...emptyPosition(), score: [2, 3], dice: [3, 1] });
    });

    test('draws an N by N grid lettered by play, with a legend and the score framed', async () => {
        const { container } = render(ScoreMapView, { active: true });
        await vi.waitFor(() => expect(app.StartScoreMap).toHaveBeenCalled());
        const [, length, ply, , tag] = app.StartScoreMap.mock.calls.at(-1);
        expect([length, ply]).toEqual([3, 2]);
        cells(3).forEach((cell, index) => handlers['gammonnet-scoremap:cell']({ tag, index, cell }));
        await Promise.resolve();
        expect(container.querySelectorAll('.cell:not(.blank)')).toHaveLength(9);
        expect(container.querySelectorAll('.legend .key')).toHaveLength(2);
        expect(container.querySelectorAll('.cell.here')).toHaveLength(1);
        expect(container.textContent).toContain('≈');
        // the letter, never the colour alone
        expect(container.querySelector('.cell .glyph').textContent.trim()).toMatch(/^[AB]$/);
    });

    test('hovering a cell shows its play, a click opens its detail and Evaluate goes back to the decision', async () => {
        const { container, getByText } = render(ScoreMapView, { active: true });
        await vi.waitFor(() => expect(app.StartScoreMap).toHaveBeenCalled());
        const tag = app.StartScoreMap.mock.calls.at(-1)[4];
        cells(3).forEach((cell, index) => handlers['gammonnet-scoremap:cell']({ tag, index, cell }));
        await Promise.resolve();
        const first = container.querySelector('.cell:not(.blank)');
        await fireEvent.mouseEnter(first);
        expect(get(selectedMoveStore)).toBe('8/5 6/5');
        await fireEvent.mouseLeave(first);
        expect(get(selectedMoveStore)).toBeNull();
        await fireEvent.click(first);
        await fireEvent.click(getByText(/^Évaluer|^Evaluate/));
        expect(sendPositionToEval).toHaveBeenCalled();
        expect(get(evalViewStore)).toBe('decision');
    });

    test('a refusal is shown, not a grid', async () => {
        const { container } = render(ScoreMapView, { active: true });
        await vi.waitFor(() => expect(app.StartScoreMap).toHaveBeenCalled());
        handlers['gammonnet-scoremap:error']({ tag: app.StartScoreMap.mock.calls.at(-1)[4], message: 'nope' });
        await Promise.resolve();
        expect(container.querySelector('[role="alert"]').textContent).toContain('nope');
    });

    test('a view mounted anew takes no answer meant for the one it replaced', async () => {
        const first = render(ScoreMapView, { active: true });
        await vi.waitFor(() => expect(app.StartScoreMap).toHaveBeenCalledTimes(1));
        const old = app.StartScoreMap.mock.calls[0][4];
        first.unmount();
        const { container } = render(ScoreMapView, { active: true });
        await vi.waitFor(() => expect(app.StartScoreMap).toHaveBeenCalledTimes(2));
        expect(app.StartScoreMap.mock.calls[1][4]).toBeGreaterThan(old);
        cells(3).forEach((cell, index) => handlers['gammonnet-scoremap:cell']({ tag: old, index, cell }));
        await Promise.resolve();
        expect(container.querySelectorAll('.cell:not(.blank)')).toHaveLength(0);
    });

    test('an impossible cell opens its detail without Evaluate', async () => {
        const { container, queryByText } = render(ScoreMapView, { active: true });
        await vi.waitFor(() => expect(app.StartScoreMap).toHaveBeenCalled());
        const tag = app.StartScoreMap.mock.calls.at(-1)[4];
        cells(3).forEach((cell, index) => handlers['gammonnet-scoremap:cell']({ tag, index, cell: { ...cell, refused: true, reason: 'unreachable' } }));
        await Promise.resolve();
        await fireEvent.click(container.querySelector('.cell:not(.blank)'));
        expect(container.querySelector('.detail')).not.toBeNull();
        expect(queryByText(/^Évaluer|^Evaluate/)).toBeNull();
    });

    test('a cube map marks its 1-away lines post-Crawford, not Crawford, and frames no post-Crawford score on a Crawford cell', async () => {
        positionStore.set({ ...emptyPosition(), score: [0, 3], dice: [0, 0] });
        const { container } = render(ScoreMapView, { active: true });
        await vi.waitFor(() => expect(app.StartScoreMap).toHaveBeenCalled());
        const tag = app.StartScoreMap.mock.calls.at(-1)[4];
        const cube = cells(3).map((c) => ({ ...c, crawford: false, verdict: 'no_double', noDouble: 0.1, doubleTake: 0.3, doublePass: 1, move: undefined }));
        cube.forEach((cell, index) => handlers['gammonnet-scoremap:cell']({ tag, index, cell }));
        await Promise.resolve();
        const heads = [...container.querySelectorAll('th')].map((th) => th.textContent.trim());
        expect(heads).toContain('1 PC');
        expect(heads).not.toContain('1 C');
        expect([...container.querySelectorAll('th[scope="col"]')].map((th) => th.textContent.trim())).toEqual(['1 PC', '2', '3']);
        expect(container.querySelectorAll('.cell.here')).toHaveLength(1);
    });

    test('a post-Crawford position is not framed on the Crawford cell of a checker map', async () => {
        positionStore.set({ ...emptyPosition(), score: [0, 3], dice: [3, 1] });
        const { container } = render(ScoreMapView, { active: true });
        await vi.waitFor(() => expect(app.StartScoreMap).toHaveBeenCalled());
        const tag = app.StartScoreMap.mock.calls.at(-1)[4];
        cells(3).forEach((cell, index) => handlers['gammonnet-scoremap:cell']({ tag, index, cell }));
        await Promise.resolve();
        expect(container.querySelectorAll('.cell.here')).toHaveLength(0);
        expect([...container.querySelectorAll('th[scope="row"]')].some((th) => th.textContent.includes('C'))).toBe(true);
    });
});
