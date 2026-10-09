import { describe, test, expect } from 'vitest';
import { placeTip } from '../utils/chartTip.js';
import { gameSpans } from '../utils/chartAxis.js';

const frame = { w: 300, h: 150 };
const tip = { w: 120, h: 60 };

describe('a chart tooltip stays inside its frame', () => {
    test('beside and below the pointer when there is room', () => {
        expect(placeTip({ x: 20, y: 10 }, tip, frame)).toEqual({ left: 30, top: 20 });
    });

    test('turned to the left at the right edge, above at the bottom', () => {
        expect(placeTip({ x: 295, y: 145 }, tip, frame)).toEqual({ left: 165, top: 75 });
    });

    test('in a frame too narrow for either side, pushed back inside', () => {
        const { left, top } = placeTip({ x: 100, y: 75 }, tip, { w: 180, h: 100 });
        expect(left).toBeGreaterThanOrEqual(0);
        expect(left + tip.w).toBeLessThanOrEqual(180);
        expect(top).toBeGreaterThanOrEqual(0);
        expect(top + tip.h).toBeLessThanOrEqual(100);
    });

    test('larger than its frame, it starts at the frame start', () => {
        expect(placeTip({ x: 50, y: 20 }, { w: 400, h: 60 }, frame).left).toBe(0);
    });
});

describe('the games along the decision axis', () => {
    test('one span per game, in match order', () => {
        const moves = [1, 1, 2, 2, 2, 3].map((game_number) => ({ game_number }));
        expect(gameSpans(moves)).toEqual([
            { game: 1, from: 0, to: 2 },
            { game: 2, from: 2, to: 5 },
            { game: 3, from: 5, to: 6 }
        ]);
        expect(gameSpans([])).toEqual([]);
    });
});
