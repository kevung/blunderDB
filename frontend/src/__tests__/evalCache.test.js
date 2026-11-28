import { describe, test, expect } from 'vitest';
import { createEvalCache, evalCacheKey, equityDeltas, mergeMoves, plyOf } from '../services/evalCache.js';

const rows = (/** @type {string} */ depth, /** @type {[string, number][]} */ list) => list.map(([move, equity]) => ({ move, equity, analysisDepth: depth }));
const KEY = 'k';

describe('evalCache', () => {
    test('the key ignores provenance and follows what changes the answer', () => {
        const pos = { id: 4, flagged: true, individually_imported: true, dice: [3, 1] };
        expect(evalCacheKey(pos, { pruneK: 12, candidates: 10 })).toBe(evalCacheKey({ ...pos, id: 9, flagged: false }, { pruneK: 12, candidates: 10 }));
        expect(evalCacheKey(pos, { pruneK: 12, candidates: 10 })).not.toBe(evalCacheKey({ ...pos, dice: [6, 5] }, { pruneK: 12, candidates: 10 }));
        expect(evalCacheKey(pos, { pruneK: 12, candidates: 10 })).not.toBe(evalCacheKey(pos, { pruneK: 0, candidates: 10 }));
    });

    test('a shallower whole answer never replaces a deeper one', () => {
        const c = createEvalCache();
        c.put(KEY, { metId: 0, moves: rows('2-ply', [['a', 0.1]]) });
        const e = c.put(KEY, { metId: 0, moves: rows('0-ply', [['a', 0.3]]) });
        expect(e?.ply).toBe(2);
        expect(e?.moves?.[0].equity).toBe(0.1);
    });

    test('a play re-run deeper keeps its depth when the position is evaluated again', () => {
        const c = createEvalCache();
        c.put(KEY, {
            metId: 0,
            moves: rows('2-ply', [
                ['a', 0.1],
                ['b', 0.05]
            ])
        });
        c.put(KEY, { metId: 0, partial: true, moves: rows('4-ply', [['b', 0.2]]) });
        c.put(KEY, { metId: 0, partial: true, moves: rows('1-ply', [['a', 0.9]]) });
        const e = c.put(KEY, {
            metId: 0,
            moves: rows('2-ply', [
                ['a', 0.11],
                ['b', 0.06]
            ])
        });
        expect(e?.moves?.map((m) => [m.move, m.analysisDepth, m.equity])).toEqual([
            ['b', '4-ply', 0.2],
            ['a', '2-ply', 0.11]
        ]);
        expect(e?.moves?.[1].equityError).toBeCloseTo(0.09);
    });

    test('an entry holds under its own equity table only; a refusal is not kept; the oldest goes first', () => {
        const c = createEvalCache(2);
        c.put(KEY, { metId: 1, moves: rows('2-ply', [['a', 0.1]]) });
        expect(c.get(KEY, 2)).toBeUndefined();
        c.put('r', { refused: true });
        expect(c.size).toBe(1);
        c.put('x', { metId: 0, moves: [] });
        c.put('y', { metId: 0, moves: [] });
        expect(c.get(KEY, 1)).toBeUndefined();
    });

    test('deltas name only the plays the run touched; mergeMoves ranks across depths', () => {
        const before = rows('2-ply', [
            ['a', 0.1],
            ['b', 0.0]
        ]);
        const ran = rows('3-ply', [['b', 0.25]]);
        expect([...equityDeltas(before, ran)]).toEqual([['b', 0.25]]);
        expect(mergeMoves(before, ran).map((m) => m.move)).toEqual(['b', 'a']);
        expect(plyOf('3-ply')).toBe(3);
        expect(plyOf('XG Roller+')).toBe(-1);
    });
});
