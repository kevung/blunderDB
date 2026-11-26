// @ts-nocheck
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { get } from 'svelte/store';

const handlers = vi.hoisted(() => /** @type {Record<string, (e: any) => void>} */ ({}));
const app = vi.hoisted(() => ({
    StartRollMap: vi.fn().mockResolvedValue(undefined),
    CancelRollMap: vi.fn().mockResolvedValue(undefined),
    PrefetchRollMaps: vi.fn().mockResolvedValue(undefined),
    CancelRollMapPrefetch: vi.fn().mockResolvedValue(undefined),
    LegalMoves: vi.fn()
}));
vi.mock('../../wailsjs/go/gui/App.js', () => app);
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetDisplayPly: vi.fn().mockResolvedValue(2),
    GetGammonNetPruneK: vi.fn().mockResolvedValue(0)
}));
vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    EventsOn: vi.fn((name, fn) => {
        handlers[name] = fn;
        return () => {};
    })
}));
vi.mock('../services/modeMachine.js', () => ({ sendPositionToEval: vi.fn(), toggleEvalMode: vi.fn() }));

import RollMapView from '../components/RollMapView.svelte';
import { positionStore } from '../stores/positionStore.js';
import { rollTrailStore } from '../stores/mapStore.js';
import { sendPositionToEval } from '../services/modeMachine.js';

function opening() {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    for (const [i, n] of [
        [24, 2],
        [13, 5],
        [8, 3],
        [6, 5]
    ])
        points[i] = { checkers: n, color: 0 };
    for (const [i, n] of [
        [1, 2],
        [12, 5],
        [17, 3],
        [19, 5]
    ])
        points[i] = { checkers: n, color: 1 };
    return { id: 0, board: { points, bearoff: [0, 0] }, cube: { owner: -1, value: 0 }, dice: [0, 0], score: [-1, -1], player_on_roll: 0, decision_type: 0, has_jacoby: 0, max_cube: 0, has_beaver: 0 };
}

const ROLLS = [];
for (let a = 1; a <= 6; a++) for (let b = a; b <= 6; b++) ROLLS.push([a, b]);

async function feed(tag, ply = 2) {
    ROLLS.forEach(([die1, die2], index) => {
        handlers['gammonnet-rolls:row']({ tag, index, row: { die1, die2, weight: die1 === die2 ? 1 : 2, move: '24/14', equity: index / 100, ply, marketLost: index < 3 } });
    });
    await Promise.resolve();
}

describe('RollMapView', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        positionStore.set(opening());
        rollTrailStore.set({ root: null, steps: [] });
    });
    afterEach(() => rollTrailStore.set({ root: null, steps: [] }));

    test('starts the map of the position and draws its 21 rows and the grid', async () => {
        const { container, findByText } = render(RollMapView, { active: true });
        await vi.waitFor(() => expect(app.StartRollMap).toHaveBeenCalledTimes(1));
        const tag = app.StartRollMap.mock.calls[0][3];
        expect(app.StartRollMap.mock.calls[0][1]).toBe(2);
        await feed(tag);
        expect(container.querySelectorAll('tbody tr')).toHaveLength(21);
        expect(container.querySelectorAll('.grid .cell')).toHaveLength(36);
        await findByText(/2-ply/);
        // the "market lost" count is a filter
        const market = container.querySelector('.market');
        expect(market.textContent).toContain('5/36');
        await fireEvent.click(market);
        expect(container.querySelectorAll('tbody tr')).toHaveLength(3);
    });

    test('an answer of an older start is dropped', async () => {
        const { container } = render(RollMapView, { active: true });
        await vi.waitFor(() => expect(app.StartRollMap).toHaveBeenCalled());
        await feed(app.StartRollMap.mock.calls[0][3] + 99);
        expect(container.querySelectorAll('tbody tr')).toHaveLength(0);
    });

    test('computes nothing while not shown', async () => {
        render(RollMapView, { active: false });
        await Promise.resolve();
        expect(app.StartRollMap).not.toHaveBeenCalled();
    });

    test('descending plays the roll, moves the board and asks for the replies', async () => {
        const { container, getByText } = render(RollMapView, { active: true });
        await vi.waitFor(() => expect(app.StartRollMap).toHaveBeenCalled());
        await feed(app.StartRollMap.mock.calls[0][3]);
        // 6-4 is the row the engine numbers 8 + ... : pick the 4-6 row through the table
        const row = [...container.querySelectorAll('tbody tr')].find((r) => r.querySelector('.roll').textContent === '4-6');
        await fireEvent.click(row);
        await fireEvent.click(getByText(/^Réponses|^Replies/));
        expect(get(rollTrailStore).steps).toHaveLength(1);
        expect(get(rollTrailStore).steps[0]).toMatchObject({ die1: 4, die2: 6, move: '24/14' });
        expect(sendPositionToEval).toHaveBeenCalled();
        // the replies are computed on the board the trail shows, never on a notation resolved again
        await vi.waitFor(() => expect(app.StartRollMap).toHaveBeenCalledTimes(2));
        const after = get(rollTrailStore).steps[0].after;
        expect(app.StartRollMap.mock.calls[1][0]).toEqual(after);
        expect(after.player_on_roll).toBe(1);
        expect(after.dice).toEqual([0, 0]);
        expect(after.board.points[24].checkers).toBe(1);
        expect(after.board.points[14]).toEqual({ checkers: 1, color: 0 });
        // the trail reads: start, my play, his rolls
        expect(container.querySelectorAll('.trail .crumb')).toHaveLength(3);
        // climbing back
        await fireEvent.click(container.querySelector('.trail .crumb'));
        expect(get(rollTrailStore).steps).toHaveLength(0);
    });

    test('a finished map prefetches its replies, follows the row selected, and stops when left', async () => {
        const { container } = render(RollMapView, { active: true });
        await vi.waitFor(() => expect(app.StartRollMap).toHaveBeenCalledTimes(1));
        const tag = app.StartRollMap.mock.calls[0][3];
        await feed(tag);
        expect(app.PrefetchRollMaps).not.toHaveBeenCalled();
        const rows = ROLLS.map(([die1, die2]) => ({ die1, die2, weight: die1 === die2 ? 1 : 2, move: '24/14', equity: 0, ply: 2 }));
        handlers['gammonnet-rolls:done']({ tag, map: { ply: 2, rows, average: 0, cubeAccess: false } });
        await vi.waitFor(() => expect(app.PrefetchRollMaps).toHaveBeenCalledTimes(1));
        const [, sent, ply, , focus, sentTag] = app.PrefetchRollMaps.mock.calls[0];
        expect(sent).toHaveLength(21);
        expect([ply, focus, sentTag]).toEqual([2, -1, tag]);

        handlers['gammonnet-rolls:prefetched']({ tag, index: 0, ply: 1 });
        handlers['gammonnet-rolls:prefetched']({ tag: tag + 99, index: 1, ply: 1 });
        await Promise.resolve();
        expect(container.querySelectorAll('tbody tr.ready')).toHaveLength(1);

        const row = [...container.querySelectorAll('tbody tr')].find((r) => r.querySelector('.roll').textContent === '4-6');
        await fireEvent.click(row);
        await vi.waitFor(() => expect(app.PrefetchRollMaps).toHaveBeenCalledTimes(2));
        expect(app.PrefetchRollMaps.mock.calls[1][4]).toBe(ROLLS.findIndex(([a, b]) => a === 4 && b === 6));

        positionStore.set({ ...opening(), player_on_roll: 1 });
        await vi.waitFor(() => expect(app.CancelRollMapPrefetch).toHaveBeenCalled());
    });

    test('a view mounted anew takes no answer meant for the one it replaced', async () => {
        const first = render(RollMapView, { active: true });
        await vi.waitFor(() => expect(app.StartRollMap).toHaveBeenCalledTimes(1));
        const old = app.StartRollMap.mock.calls[0][3];
        first.unmount();
        const { container } = render(RollMapView, { active: true });
        await vi.waitFor(() => expect(app.StartRollMap).toHaveBeenCalledTimes(2));
        expect(app.StartRollMap.mock.calls[1][3]).toBeGreaterThan(old);
        await feed(old);
        expect(container.querySelectorAll('tbody tr')).toHaveLength(0);
        await feed(app.StartRollMap.mock.calls[1][3]);
        expect(container.querySelectorAll('tbody tr')).toHaveLength(21);
    });

    test('another play lists the legal plays but the best, told apart by their board', async () => {
        const board = (from, to) => {
            const b = JSON.parse(JSON.stringify(opening().board));
            b.points[from] = { checkers: b.points[from].checkers - 1, color: 0 };
            b.points[to] = { checkers: b.points[to].checkers + 1, color: 0 };
            return b;
        };
        // the best play under another spelling, and one other play
        app.LegalMoves.mockResolvedValue([
            { notation: '24/18 18/14', result: { board: board(24, 14) } },
            { notation: '13/7 13/9', result: { board: board(13, 9) } }
        ]);
        const { container, getByText, findByText } = render(RollMapView, { active: true });
        await vi.waitFor(() => expect(app.StartRollMap).toHaveBeenCalled());
        await feed(app.StartRollMap.mock.calls[0][3]);
        const row = [...container.querySelectorAll('tbody tr')].find((r) => r.querySelector('.roll').textContent === '4-6');
        await fireEvent.click(row);
        await fireEvent.click(getByText(/^Autre coup|^Other play/));
        await findByText('13/7 13/9');
        expect(container.querySelectorAll('.others li')).toHaveLength(1);
        expect(app.LegalMoves.mock.calls[0][0].dice).toEqual([4, 6]);
        await fireEvent.click(getByText('13/7 13/9'));
        const step = get(rollTrailStore).steps[0];
        expect(step.move).toBe('13/7 13/9');
        expect(step.after.board.points[9]).toEqual({ checkers: 1, color: 0 });
        expect(step.after.player_on_roll).toBe(1);
    });
});
