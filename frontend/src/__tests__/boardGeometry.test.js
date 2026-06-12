import { describe, test, expect } from 'vitest';
import { parseMoveNotation, mirrorPosition, computePipCount } from '../utils/boardGeometry.js';

describe('parseMoveNotation', () => {
    test('parses a simple two-move string', () => {
        expect(parseMoveNotation('24/23 13/11')).toEqual([
            { from: 24, to: 23, index: 0 },
            { from: 13, to: 11, index: 0 }
        ]);
    });

    test('expands the (n) multiplier into n moves', () => {
        expect(parseMoveNotation('13/11(2)')).toEqual([
            { from: 13, to: 11, index: 0 },
            { from: 13, to: 11, index: 1 }
        ]);
    });

    test('maps bar to 0 and off to -1', () => {
        expect(parseMoveNotation('bar/23')).toEqual([{ from: 0, to: 23, index: 0 }]);
        expect(parseMoveNotation('6/off(2)')).toEqual([
            { from: 6, to: -1, index: 0 },
            { from: 6, to: -1, index: 1 }
        ]);
    });

    test('returns [] for empty, bare "bar", or "cannot move"', () => {
        expect(parseMoveNotation('')).toEqual([]);
        expect(parseMoveNotation(null)).toEqual([]);
        expect(parseMoveNotation('bar')).toEqual([]);
        expect(parseMoveNotation('Cannot Move')).toEqual([]);
    });

    test('ignores unparseable fragments', () => {
        expect(parseMoveNotation('24/23 garbage')).toEqual([{ from: 24, to: 23, index: 0 }]);
    });
});

// Minimal position factory: 26 points (0..25), bearoff/score/cube.
function makePosition(overrides = {}) {
    const points = Array.from({ length: 26 }, () => ({ color: -1, checkers: 0 }));
    return {
        board: { points, bearoff: [0, 0] },
        player_on_roll: 0,
        score: [0, 0],
        cube: { owner: -1, value: 0 },
        ...overrides
    };
}

describe('mirrorPosition', () => {
    test('reverses points (i ↔ 25-i) and swaps colours', () => {
        const pos = makePosition();
        pos.board.points[1] = { color: 0, checkers: 2 };
        pos.board.points[24] = { color: 1, checkers: 3 };
        const m = mirrorPosition(pos);
        // point 1 (color 0) moves to 24 with colour flipped to 1
        expect(m.board.points[24]).toEqual({ color: 1, checkers: 2 });
        // point 24 (color 1) moves to 1 with colour flipped to 0
        expect(m.board.points[1]).toEqual({ color: 0, checkers: 3 });
    });

    test('leaves empty points (-1) untouched in colour', () => {
        const m = mirrorPosition(makePosition());
        expect(m.board.points.every((p) => p.color === -1)).toBe(true);
    });

    test('swaps bearoff, scores, on-roll player, and owned cube', () => {
        const pos = makePosition({
            board: { points: Array.from({ length: 26 }, () => ({ color: -1, checkers: 0 })), bearoff: [5, 9] },
            player_on_roll: 0,
            score: [3, 7],
            cube: { owner: 0, value: 2 }
        });
        const m = mirrorPosition(pos);
        expect(m.board.bearoff).toEqual([9, 5]);
        expect(m.score).toEqual([7, 3]);
        expect(m.player_on_roll).toBe(1);
        expect(m.cube.owner).toBe(1);
    });

    test('keeps a centred cube centred (owner -1)', () => {
        const m = mirrorPosition(makePosition({ cube: { owner: -1, value: 0 } }));
        expect(m.cube.owner).toBe(-1);
    });

    test('does not mutate the input and is its own inverse', () => {
        const pos = makePosition();
        pos.board.points[3] = { color: 0, checkers: 4 };
        pos.board.points[20] = { color: 1, checkers: 1 };
        pos.score = [2, 5];
        pos.player_on_roll = 0;
        const snapshot = JSON.stringify(pos);
        const back = mirrorPosition(mirrorPosition(pos));
        expect(JSON.stringify(pos)).toBe(snapshot); // input untouched
        expect(back).toEqual(pos); // involution
    });
});

describe('computePipCount', () => {
    test('color-0 counts index, color-1 counts 25 - index', () => {
        const pos = makePosition();
        pos.board.points[6] = { color: 0, checkers: 2 }; // 2 * 6 = 12
        pos.board.points[24] = { color: 1, checkers: 3 }; // 3 * (25-24) = 3
        expect(computePipCount(pos)).toEqual({ pipCount1: 12, pipCount2: 3 });
    });

    test('empty board is zero/zero', () => {
        expect(computePipCount(makePosition())).toEqual({ pipCount1: 0, pipCount2: 0 });
    });
});
