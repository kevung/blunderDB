import { describe, it, expect } from 'vitest';
import { lossSeries, fmtLoss, niceCeil } from '../utils/decisionLoss.js';

const mps = [
    { move_id: 1, player_on_roll: 0 },
    { move_id: 2, player_on_roll: 1 },
    { move_id: 3, player_on_roll: 0 },
    { move_id: 4, player_on_roll: 1 }
];

describe('lossSeries', () => {
    it('accumulates per player and ends on the player total', () => {
        const s = lossSeries(mps, [
            { move_id: 1, mwc_loss: 0.01 },
            { move_id: 2, mwc_loss: 0.02 },
            { move_id: 3, mwc_loss: 0.03 },
            { move_id: 4, mwc_loss: 0.005 }
        ]);
        expect(s.totals[0]).toBeCloseTo(0.04);
        expect(s.totals[1]).toBeCloseTo(0.025);
        expect(s.items[2].cum).toBeCloseTo(0.04);
        expect(s.scored).toEqual([2, 2]);
    });

    it('keeps an unscored decision null, not zero', () => {
        const s = lossSeries(mps, [
            { move_id: 1, mwc_loss: 0.01 },
            { move_id: 2, mwc_loss: null }
        ]);
        expect(s.items[1].loss).toBeNull();
        expect(s.items[2].loss).toBeNull();
        expect(s.scored).toEqual([1, 0]);
        expect(s.any).toBe(true);
    });

    it('is empty of analysis when nothing is scored', () => {
        expect(lossSeries(mps, []).any).toBe(false);
        expect(lossSeries(mps, null).any).toBe(false);
    });
});

describe('fmtLoss', () => {
    it('shows a percentage and nothing when unscored', () => {
        expect(fmtLoss(0.01234)).toBe('1.23 %');
        expect(fmtLoss(null)).toBe('');
    });
});

describe('niceCeil', () => {
    it('rounds up to a round figure', () => {
        expect(niceCeil(0.0123)).toBeCloseTo(0.02);
        expect(niceCeil(0.31)).toBeCloseTo(0.5);
        expect(niceCeil(0.1)).toBeCloseTo(0.1);
        expect(niceCeil(0)).toBe(1);
    });
});
