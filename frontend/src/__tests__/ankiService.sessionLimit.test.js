/**
 * ankiService.sessionLimit.test.js — how many cards one sitting serves
 * (ADR-0026 rules 2 and 3).
 *
 * The whole point of these cases is that "no limit" and "a limit of zero" are
 * DIFFERENT states. A single number would collapse them — and collapsing them
 * is the known Anki trap, where 0 quietly means "none" while everyone reads it
 * as "unlimited".
 */
import { describe, test, expect } from 'vitest';
import { sessionLimitOf, sessionLimitReached, canStudy } from '../services/ankiService.js';

const deck = (sessionLimit) => ({ id: 1, sessionLimit });

describe('sessionLimitOf', () => {
    test('null and undefined are no limit', () => {
        expect(sessionLimitOf(deck(null))).toBeNull();
        expect(sessionLimitOf(deck(undefined))).toBeNull();
        expect(sessionLimitOf(undefined)).toBeNull();
    });

    test('zero is a limit, not the absence of one', () => {
        expect(sessionLimitOf(deck(0))).toBe(0);
    });

    test('a number is that number', () => {
        expect(sessionLimitOf(deck(20))).toBe(20);
    });
});

describe('sessionLimitReached', () => {
    test('a deck without a limit never reaches one', () => {
        expect(sessionLimitReached(deck(null), 0)).toBe(false);
        expect(sessionLimitReached(deck(null), 9999)).toBe(false);
    });

    test('a limit of zero is reached before the first card', () => {
        expect(sessionLimitReached(deck(0), 0)).toBe(true);
    });

    test('the limit bites once the count catches up with it', () => {
        expect(sessionLimitReached(deck(3), 2)).toBe(false);
        expect(sessionLimitReached(deck(3), 3)).toBe(true);
        expect(sessionLimitReached(deck(3), 4)).toBe(true);
    });

    test('cram is never bounded: free drill schedules nothing, so nothing needs pacing', () => {
        expect(sessionLimitReached(deck(0), 0, { cram: true })).toBe(false);
        expect(sessionLimitReached(deck(3), 10, { cram: true })).toBe(false);
    });
});

describe('canStudy', () => {
    const stats = (dueCount) => ({ dueCount, totalCount: 10 });

    test('needs due cards, as before', () => {
        expect(canStudy(stats(0), deck(null))).toBe(false);
        expect(canStudy(stats(2), deck(null))).toBe(true);
        expect(canStudy(null, deck(null))).toBe(false);
    });

    test('a deck limited to zero cannot be studied even with cards due', () => {
        // Otherwise the button stays active, the click does nothing, and the
        // user discovers the setting by bumping into it.
        expect(canStudy(stats(5), deck(0))).toBe(false);
    });

    test('a positive limit does not block starting', () => {
        expect(canStudy(stats(5), deck(1))).toBe(true);
    });
});
