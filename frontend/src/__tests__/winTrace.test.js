import { describe, it, expect } from 'vitest';
import { winTraceSeries, winTraceRuns } from '../utils/decisionLoss.js';

const moves = [1, 2, 3, 4].map((id) => ({ move_id: id }));

describe('winTraceSeries', () => {
    it('places each decision on its Move and keeps a missing value null', () => {
        const trace = {
            money: false,
            decisions: [
                { move_id: 1, player: 0, mwc_before: 0.5, mwc_after: 0.48, error: false },
                { move_id: 2, player: 1, mwc_before: 0.47, mwc_after: 0.52, error: true },
                { move_id: 3, player: 0, mwc_before: null, mwc_after: null },
                { move_id: 4, player: 1, mwc_before: 0.6, mwc_after: null }
            ]
        };
        const s = winTraceSeries(moves, trace);
        expect(s.any).toBe(true);
        expect(s.items.map((d) => [d.player, d.before, d.after, d.error])).toEqual([
            [0, 0.5, 0.48, false],
            [1, 0.47, 0.52, true],
            [0, null, null, false],
            [1, 0.6, null, false]
        ]);
        // The unanalysed decision breaks the line; an unscored play stays flat.
        expect(winTraceRuns(s.items)).toEqual([
            [
                { x: 0, v: 0.5 },
                { x: 1, v: 0.48 },
                { x: 1, v: 0.47 },
                { x: 2, v: 0.52 }
            ],
            [
                { x: 3, v: 0.6 },
                { x: 4, v: 0.6 }
            ]
        ]);
    });

    it('draws nothing at money or without a trace', () => {
        expect(winTraceSeries(moves, { money: true, decisions: [{ move_id: 1, mwc_before: null }] }).any).toBe(false);
        expect(winTraceSeries(moves, null).any).toBe(false);
    });
});
