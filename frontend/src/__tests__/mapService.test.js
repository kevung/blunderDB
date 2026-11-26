// @ts-nocheck
import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../services/modeMachine.js', () => ({ sendPositionToEval: vi.fn() }));

import { descend, climbTo } from '../services/mapService.js';
import { rollTrailStore } from '../stores/mapStore.js';
import { positionStore } from '../stores/positionStore.js';
import { sendPositionToEval } from '../services/modeMachine.js';

function position(from) {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    points[from] = { checkers: 2, color: 0 };
    points[1] = { checkers: 2, color: 1 };
    return { id: 0, board: { points, bearoff: [13, 13] }, cube: { owner: -1, value: 0 }, dice: [0, 0], score: [-1, -1], player_on_roll: 0 };
}

describe('mapService trail', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        rollTrailStore.set({ root: null, steps: [] });
    });

    test('back at the start, the trail forgets its root: a later descent leaves from the board as it then is', () => {
        positionStore.set(position(24));
        expect(descend(6, 5, '24/13')).toBe(true);
        climbTo(0);
        expect(get(rollTrailStore)).toEqual({ root: null, steps: [] });
        expect(sendPositionToEval).toHaveBeenLastCalledWith(position(24));

        // another position comes to the board, then a descent from it
        positionStore.set(position(20));
        expect(descend(6, 5, '20/9')).toBe(true);
        const trail = get(rollTrailStore);
        expect(trail.root.board.points[20].checkers).toBe(2);
        expect(trail.root.board.points[24].checkers).toBe(0);
    });

    test('climbing to a link keeps the root', () => {
        positionStore.set(position(24));
        descend(6, 5, '24/13');
        descend(2, 1, '1/3 1/2');
        climbTo(1);
        const trail = get(rollTrailStore);
        expect(trail.steps).toHaveLength(1);
        expect(trail.root.board.points[24].checkers).toBe(2);
    });

    test('a play given with its board takes that board, the notation a label only', () => {
        positionStore.set(position(24));
        const board = JSON.parse(JSON.stringify(position(24).board));
        board.points[24] = { checkers: 1, color: 0 };
        board.points[13] = { checkers: 1, color: 0 };
        expect(descend(6, 5, 'any spelling', board)).toBe(true);
        const step = get(rollTrailStore).steps[0];
        expect(step.move).toBe('any spelling');
        expect(step.after.board).toEqual(board);
        expect(step.after.player_on_roll).toBe(1);
    });
});
