/**
 * emptyAnalysis() is the single source of the "no analysis" record. App.svelte
 * (library emptied) and databaseService (database closed/reopened) used to
 * carry verbatim copies of the store's initial value; this pins the factory to
 * that shape so the three sites cannot drift apart again.
 */
import { describe, test, expect } from 'vitest';
import { get } from 'svelte/store';
import { analysisStore, emptyAnalysis } from '../stores/analysisStore.js';

describe('emptyAnalysis', () => {
    test('is the store initial value', () => {
        expect(get(analysisStore)).toEqual(emptyAnalysis());
    });

    test('returns a fresh object on each call', () => {
        const a = emptyAnalysis();
        const b = emptyAnalysis();
        expect(a).not.toBe(b);
        expect(a.doublingCubeAnalysis).not.toBe(b.doublingCubeAnalysis);
        expect(a.checkerAnalysis.moves).not.toBe(b.checkerAnalysis.moves);
    });

    test('has the fields the panels read', () => {
        const a = emptyAnalysis();
        expect(a.positionId).toBeNull();
        expect(a.checkerAnalysis.moves).toEqual([]);
        expect(a.allCubeAnalyses).toEqual([]);
        expect(a.playedMoves).toEqual([]);
        expect(a.playedCubeActions).toEqual([]);
        expect(Object.keys(a.doublingCubeAnalysis)).toHaveLength(18);
    });
});
