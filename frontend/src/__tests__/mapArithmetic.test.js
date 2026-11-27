// @ts-nocheck
import { describe, test, expect } from 'vitest';
import { positionAfterPlay } from '../utils/moveNotation.js';
import { rollStats, sortRows, rollIndex, deviationScale, weightedMean } from '../utils/rollMap.js';
import { playGroups, gridCells, tint, positionCell, positionAtScore, rawAway, cellMargin } from '../utils/scoreMap.js';

function opening(dice = [6, 4]) {
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
    return { id: 7, board: { points, bearoff: [0, 0] }, cube: { owner: -1, value: 0 }, dice, score: [5, 3], player_on_roll: 0, decision_type: 0, has_jacoby: 0, max_cube: 0, has_beaver: 0 };
}

describe('positionAfterPlay', () => {
    test('plays the move, hands the roll over and clears the dice', () => {
        const after = positionAfterPlay(opening(), 6, 4, '24/14');
        expect(after.player_on_roll).toBe(1);
        expect(after.dice).toEqual([0, 0]);
        expect(after.id).toBe(0);
        expect(after.board.points[24].checkers).toBe(1);
        expect(after.board.points[14]).toEqual({ checkers: 1, color: 0 });
        expect(after.score).toEqual([5, 3]);
    });
    test('refuses a play that does not exist', () => {
        expect(positionAfterPlay(opening(), 6, 4, '24/23')).toBeNull();
    });
    test('a dance keeps the board', () => {
        const before = opening();
        const after = positionAfterPlay(before, 6, 4, '');
        expect(after.board).toEqual(before.board);
        expect(after.player_on_roll).toBe(1);
    });
});

describe('roll map arithmetic', () => {
    const rows = [
        { index: 0, weight: 1, equity: 0.2, move: 'b' },
        { index: 1, weight: 2, equity: -0.1, move: 'a', marketLost: true },
        { index: 2, weight: 1, equity: 0.0, move: 'c' }
    ];
    test('mean, spread, best and worst weigh the rolls', () => {
        expect(weightedMean(rows)).toBeCloseTo(0, 10);
        const stats = rollStats(rows);
        expect(stats.best.index).toBe(0);
        expect(stats.worst.index).toBe(1);
        expect(stats.marketLost).toBe(2);
        expect(stats.stdev).toBeCloseTo(Math.sqrt((0.04 + 2 * 0.01) / 4), 10);
    });
    test('sorting keeps the engine order on ties', () => {
        expect(sortRows(rows, 'equity', true, 0).map((r) => r.index)).toEqual([0, 2, 1]);
        expect(sortRows(rows, 'move', false, 0).map((r) => r.move)).toEqual(['a', 'b', 'c']);
    });
    test('the grid index follows the engine order of 21 rolls', () => {
        expect(rollIndex(1, 1)).toBe(0);
        expect(rollIndex(6, 1)).toBe(5);
        expect(rollIndex(2, 2)).toBe(6);
        expect(rollIndex(6, 6)).toBe(20);
    });
    test('the deviation scale is bounded', () => {
        expect(deviationScale(1, 0, 0.5)).toBe(1);
        expect(deviationScale(-1, 0, 0.5)).toBe(-1);
        expect(deviationScale(0.1, 0, 0)).toBe(0);
    });
});

describe('score map arithmetic', () => {
    const cell = (move, gap) => ({ move, gap, awayOnRoll: 1, awayOpponent: 1, equity: 0 });
    test('plays are lettered by frequency, three get a colour, the rest share the grey', () => {
        const cells = [cell('a', 0.1), cell('b', 0.1), cell('b', 0.1), cell('c', 0.1), cell('c', 0.1), cell('c', 0.1), cell('d', 0.1), cell('d', 0.1)];
        const groups = playGroups(cells, 'checker');
        expect(groups.map((g) => g.key)).toEqual(['c', 'b', 'd', 'a']);
        expect(groups.map((g) => g.letter)).toEqual(['A', 'B', 'C', 'D']);
        expect(groups.map((g) => g.slot)).toEqual([0, 1, 2, -1]);
    });
    test('the margin sets the tint and the toss-up mark', () => {
        expect(tint(0.005)).toBe('close');
        expect(tint(0.02)).toBe('soft');
        expect(tint(0.2)).toBe('strong');
        expect(tint(null)).toBe('strong');
        expect(cellMargin({ noDouble: 0.5, doubleTake: 0.7, doublePass: 1 }, 'cube')).toBeCloseTo(0.2, 10);
        // a double whose take and pass are close is a close cell, however clear the double
        expect(cellMargin({ noDouble: 0.2, doubleTake: 0.995, doublePass: 1 }, 'cube')).toBeCloseTo(0.005, 10);
        // not doubling: the take/pass gap does not count
        expect(cellMargin({ noDouble: 0.5, doubleTake: 0.4, doublePass: 0.41 }, 'cube')).toBeCloseTo(0.1, 10);
        expect(cellMargin({ verdict: 'dmp', noDouble: 0, doubleTake: 0, doublePass: 0 }, 'cube')).toBeNull();
    });
    test('the grid opens on the 1-away lines', () => {
        const cells = Array.from({ length: 9 }, (_, i) => ({ awayOnRoll: Math.floor(i / 3) + 1, awayOpponent: (i % 3) + 1 }));
        const grid = gridCells(cells, 3);
        expect(grid[0][0]).toMatchObject({ awayOnRoll: 1, awayOpponent: 1 });
        expect(grid[2][2]).toMatchObject({ awayOnRoll: 3, awayOpponent: 3 });
        expect(grid[0][2]).toMatchObject({ awayOnRoll: 1, awayOpponent: 3 });
    });
    test('the position stands on its own score and a cell becomes a score again', () => {
        const pos = opening();
        expect(positionCell(pos, 9)).toEqual({ mine: 5, theirs: 3, crawford: false });
        expect(positionCell({ ...pos, score: [-1, -1] }, 9)).toBeNull();
        expect(rawAway(1, true)).toBe(1);
        expect(rawAway(1, false)).toBe(0);
        const at = positionAtScore(pos, { awayOnRoll: 4, awayOpponent: 1, crawford: true });
        expect(at.score).toEqual([4, 1]);
        expect(at.player_on_roll).toBe(0);
    });
    test('the Crawford game and post-Crawford stand on different cells', () => {
        const pos = opening();
        // on roll 1-away in the Crawford game (stored 1), opponent 3-away
        expect(positionCell({ ...pos, score: [1, 3] }, 9)).toEqual({ mine: 1, theirs: 3, crawford: true });
        // the same distances after the Crawford game (stored 0)
        expect(positionCell({ ...pos, score: [0, 3] }, 9)).toEqual({ mine: 1, theirs: 3, crawford: false });
        expect(positionCell({ ...pos, score: [3, 1] }, 9)).toEqual({ mine: 3, theirs: 1, crawford: true });
        // DMP is never the Crawford game
        expect(positionCell({ ...pos, score: [0, 0] }, 9)).toEqual({ mine: 1, theirs: 1, crawford: false });
    });

    test('an engine notation is read with its hits marked, as Go reads it', () => {
        const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
        points[6] = { checkers: 2, color: 0 };
        points[2] = { checkers: 1, color: 1 };
        points[20] = { checkers: 2, color: 1 };
        const pos = { id: 0, board: { points, bearoff: [13, 12] }, cube: { owner: -1, value: 0 }, dice: [4, 1], score: [-1, -1], player_on_roll: 0 };
        // 6/1 at 4-1 goes by 5 or, hitting, by 2: two boards when hits are not written
        expect(positionAfterPlay(pos, 4, 1, '6/1', false)).toBeNull();
        // unstarred in a dialect that marks its hits, it hits nothing
        const quiet = positionAfterPlay(pos, 4, 1, '6/1');
        expect(quiet.board.points[2]).toEqual({ checkers: 1, color: 1 });
        // the hit, marked, is resolved to the board it leaves
        const hit = positionAfterPlay(pos, 4, 1, '6/2* 2/1');
        expect(hit.board.points[2]).toEqual({ checkers: 0, color: -1 });
        expect(hit.board.points[0]).toEqual({ checkers: 1, color: 1 });
        expect(hit.board.points[1]).toEqual({ checkers: 1, color: 0 });
        expect(hit.player_on_roll).toBe(1);
    });
});
