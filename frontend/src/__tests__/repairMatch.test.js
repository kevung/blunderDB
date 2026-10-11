import { describe, expect, it } from 'vitest';
import { repairCounts, repairDiffers, repairDoneKey, repairMatchErrorKey } from '../utils/repairMatch.js';

const same = { moves: 10, positioned: 10, scored: 9 };

describe('repairMatch', () => {
    it('reads a report as Go does', () => {
        expect(repairDiffers({ before: same, after: same, changed_moves: 0 })).toBe(false);
        expect(repairDiffers({ before: same, after: same, changed_moves: 3 })).toBe(true);
        expect(repairDiffers({ before: { ...same, scored: 1 }, after: same, changed_moves: 0 })).toBe(true);
    });
    it('gives the percentages of analysed moves', () => {
        expect(repairCounts({ before: { moves: 4, positioned: 4, scored: 1 }, after: same, changed_moves: 2 })).toEqual({ changed: 2, before: 25, after: 90, kept: 0 });
    });
    it('names the comments a held position keeps', () => {
        expect(repairDoneKey({ before: same, after: same, changed_moves: 2, kept_comments: 3 })).toBe('match.repairedKept');
        expect(repairDoneKey({ before: same, after: same, changed_moves: 2, kept_comments: 0 })).toBe('match.repaired');
    });
    it('names the refusals', () => {
        expect(repairMatchErrorKey(new Error('storage: not found: the source file of the match cannot be read (a.xg)'))).toBe('match.repairNoSource');
        expect(repairMatchErrorKey('storage: invalid: the source file no longer holds this match')).toBe('match.repairSourceChanged');
        expect(repairMatchErrorKey('storage: invalid: the seats of the stored match cannot be matched with its source file')).toBe('match.repairSeatsUnclear');
        expect(repairMatchErrorKey('boom')).toBe('match.errorRepairing');
    });
});
