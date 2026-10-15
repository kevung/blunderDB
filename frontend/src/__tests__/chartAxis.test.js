import { describe, test, expect } from 'vitest';
import { indexAt, stepIndex, visibleIndices, nearestVisible, togglePlayer } from '../utils/chartAxis.js';

const mps = [0, 1, 0, 1, 1, 0].map((p) => ({ player_on_roll: p }));

describe('visibleIndices', () => {
    test('keeps only the decisions of the shown players', () => {
        expect(visibleIndices(mps, [true, true])).toEqual([0, 1, 2, 3, 4, 5]);
        expect(visibleIndices(mps, [true, false])).toEqual([0, 2, 5]);
        expect(visibleIndices(mps, [false, true])).toEqual([1, 3, 4]);
    });
});

describe('nearestVisible', () => {
    test('a visible index stays', () => {
        expect(nearestVisible(2, [0, 2, 5])).toBe(2);
    });
    test('a hidden index goes to the closest visible one', () => {
        expect(nearestVisible(1, [0, 5])).toBe(0);
        expect(nearestVisible(4, [0, 5])).toBe(5);
    });
    test('nothing visible gives null', () => {
        expect(nearestVisible(1, [])).toBeNull();
    });
});

describe('indexAt on the visible decisions', () => {
    const at = (x) => ({ clientX: x, currentTarget: { getBoundingClientRect: () => ({ left: 0, width: 60 }) } });
    test('clicking over a hidden decision takes the closest visible one', () => {
        // 6 decisions of 10 px: x=15 is index 1, hidden when only player 0 is shown.
        expect(indexAt(at(15), 6, [0, 2, 5])).toBe(0);
        expect(indexAt(at(25), 6, [0, 2, 5])).toBe(2);
    });
    test('nothing visible gives null', () => {
        expect(indexAt(at(15), 6, [])).toBeNull();
    });
    test('without a filter nothing changes', () => {
        expect(indexAt(at(15), 6)).toBe(1);
    });
});

describe('stepIndex on the visible decisions', () => {
    test('arrows skip the hidden decisions', () => {
        expect(stepIndex('ArrowRight', 0, 6, [0, 2, 5])).toBe(2);
        expect(stepIndex('ArrowRight', 2, 6, [0, 2, 5])).toBe(5);
        expect(stepIndex('ArrowLeft', 5, 6, [0, 2, 5])).toBe(2);
    });
    test('the first arrow lands on an end', () => {
        expect(stepIndex('ArrowRight', null, 6, [1, 3, 4])).toBe(1);
        expect(stepIndex('ArrowLeft', null, 6, [1, 3, 4])).toBe(4);
    });
    test('Home and End go to the visible ends', () => {
        expect(stepIndex('Home', 3, 6, [1, 3, 4])).toBe(1);
        expect(stepIndex('End', 3, 6, [1, 3, 4])).toBe(4);
    });
    test('from a hidden decision, the arrow goes on from where it stands', () => {
        expect(stepIndex('ArrowRight', 1, 6, [0, 2, 5])).toBe(2);
        expect(stepIndex('ArrowLeft', 1, 6, [0, 2, 5])).toBe(0);
    });
    test('past the last visible decision, the arrow stays put', () => {
        expect(stepIndex('ArrowRight', 5, 6, [0, 2, 3])).toBeNull();
        expect(stepIndex('ArrowLeft', 0, 6, [2, 3])).toBeNull();
        expect(stepIndex('ArrowRight', 3, 6, [0, 2, 3])).toBe(3);
    });
    test('nothing visible gives null', () => {
        expect(stepIndex('ArrowRight', null, 6, [])).toBeNull();
    });
    test('unchanged without a filter', () => {
        expect(stepIndex('ArrowRight', 2, 6)).toBe(3);
    });
});

describe('togglePlayer', () => {
    test('hides one player', () => {
        expect(togglePlayer([true, true], 1)).toEqual([true, false]);
    });
    test('shows him again', () => {
        expect(togglePlayer([true, false], 1)).toEqual([true, true]);
    });
    test('the last shown player stays', () => {
        expect(togglePlayer([true, false], 0)).toEqual([true, false]);
    });
});
