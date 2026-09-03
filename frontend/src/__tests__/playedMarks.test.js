/**
 * playedMarks.test.js
 *
 * "Was this played?" — the predicates AnalysisPanel and AnkiPanel apply to a
 * stored record to highlight a row (utils/playedMarks.js). Until fiche D.10
 * (#210) these lived here untested directly: analysisRows.js carried its own
 * second copy, with a different, positional-boolean signature, and that copy
 * — not this one — was what analysisRows.test.js exercised. Moved here so the
 * functions AnalysisPanel/AnkiPanel/clipboardService actually call are the
 * ones under test.
 */

import { describe, test, expect } from 'vitest';
import { normalizeMoveString, playedMovePredicate, playedCubeActionPredicate } from '../utils/playedMarks.js';
import { isPlayedOption } from '../utils/analysisRows.js';

describe('normalizeMoveString', () => {
    test("sorts a move's parts so order of writing does not matter", () => {
        expect(normalizeMoveString('24/23 13/10')).toBe(normalizeMoveString('13/10 24/23'));
        expect(normalizeMoveString('')).toBe('');
        expect(normalizeMoveString(undefined)).toBe('');
    });
});

describe('playedMovePredicate — what AnalysisPanel/AnkiPanel highlight', () => {
    test('a move is matched regardless of the order its parts are written in', () => {
        const isPlayed = playedMovePredicate({ playedMove: '13/10 24/23' });
        expect(isPlayed({ move: '24/23 13/10' })).toBe(true);
        expect(isPlayed({ move: '24/21 13/12' })).toBe(false);
        expect(isPlayed({})).toBe(false);
    });

    test('browsing: every recorded play counts; in a match: only the current one', () => {
        const analysis = { playedMove: '8/5 6/5', playedMoves: ['24/23 13/10', '8/5 6/5'] };
        expect(playedMovePredicate(analysis)({ move: '24/23 13/10' })).toBe(true);
        expect(playedMovePredicate(analysis, { matchMode: true })({ move: '24/23 13/10' })).toBe(false);
        expect(playedMovePredicate(analysis, { matchMode: true })({ move: '8/5 6/5' })).toBe(true);
        expect(playedMovePredicate({}, { matchMode: true })({ move: '8/5 6/5' })).toBe(false);
    });

    test('falls back to the deprecated singular field when playedMoves is empty (records written before it was pluralised)', () => {
        const isPlayed = playedMovePredicate({ playedMove: '8/5 6/5', playedMoves: [] });
        expect(isPlayed({ move: '8/5 6/5' })).toBe(true);
    });
});

describe('playedCubeActionPredicate — the union of recorded plays when browsing, the current one in a match', () => {
    test('browsing: any recorded action counts', () => {
        const browsing = playedCubeActionPredicate({ playedCubeAction: 'Double', playedCubeActions: ['Double', 'Take'] });
        expect(browsing('Double')).toBe(true);
        expect(browsing('Take')).toBe(true);
        expect(browsing('Pass')).toBe(false);
        expect(browsing('No Double')).toBe(false);
    });

    test("in a match: only the current match's own action", () => {
        const inMatch = playedCubeActionPredicate({ playedCubeAction: 'No Double', playedCubeActions: ['Double', 'Take'] }, { matchMode: true });
        expect(inMatch('No Double')).toBe(true);
        expect(inMatch('Double')).toBe(false);
        expect(playedCubeActionPredicate({}, { matchMode: true })('Double')).toBe(false);
        expect(playedCubeActionPredicate({})('Double')).toBe(false);
    });

    test('a standalone take/pass response marks the combined double row, as the panel does', () => {
        const isPlayed = playedCubeActionPredicate({ playedCubeAction: 'Take' });
        expect(isPlayedOption('double_take', isPlayed)).toBe(true);
        expect(isPlayedOption('double_pass', isPlayed)).toBe(false);
    });
});
