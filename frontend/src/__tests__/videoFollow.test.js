/**
 * videoFollow.test.js — the stretch of video each timed Action covers, and when the
 * Cursor that follows it moves (ADR-0082 rule 3).
 */

import { describe, test, expect } from 'vitest';
import { actionSpans, coveringAt, createCursorFollower, isTyping } from '../services/videoFollow.js';

const checker = (/** @type {number | undefined} */ roll, /** @type {number | undefined} */ tick) => ({
    kind: 'checker',
    roll_tick_ms: roll,
    tick_ms: tick,
    before: { dice: [3, 1] },
    notation: '8/5 6/5'
});

describe('actionSpans', () => {
    test('an Action covers from its roll to the next one’s start; the last ends at its action', () => {
        expect(actionSpans([checker(1000, 5000), checker(8000, 12000)])).toEqual([
            { index: 0, start: 1000, end: 8000, closed: true },
            { index: 1, start: 8000, end: 12000, closed: true }
        ]);
    });

    test('a cube action starts at the previous Action’s instant; an Action without Repère covers nothing', () => {
        const spans = actionSpans([checker(1000, 5000), { kind: 'double', tick_ms: 7000 }, { kind: 'take' }, checker(9000, undefined)]);
        expect(spans).toEqual([
            { index: 0, start: 1000, end: 5000, closed: true },
            { index: 1, start: 5000, end: 9000, closed: true },
            { index: 3, start: 9000, end: 9000, closed: false }
        ]);
        expect(coveringAt(spans, 500)).toBeNull();
        expect(coveringAt(spans, 6000)).toBe(1);
        // Past the last Repère: the part not transcribed yet.
        expect(coveringAt(spans, 9000)).toBe('end');
    });

    test('a play timed by its action only starts where the previous one ended', () => {
        expect(actionSpans([checker(1000, 5000), checker(undefined, 9000)])[1]).toEqual({ index: 1, start: 5000, end: 9000, closed: true });
    });
});

describe('createCursorFollower', () => {
    const ann = (/** @type {number} */ cursor) => ({ actions: [checker(1000, 5000), checker(8000, 12000)], cursor });

    test('moves on a transition only, and only while the clock runs', () => {
        const f = createCursorFollower();
        expect(f.step(2000, ann(2), false)).toBeNull();
        expect(f.step(2000, ann(2), false)).toBeNull();
        expect(f.step(2500, ann(2), false)).toBe(0);
        expect(f.step(3000, ann(0), false)).toBeNull();
        expect(f.step(9000, ann(0), false)).toBe(1);
        expect(f.step(13000, ann(1), false)).toBe(2);
        expect(f.step(14000, ann(2), false)).toBeNull();
    });

    test('a placement by hand holds until the playback leaves the Action', () => {
        const f = createCursorFollower();
        f.step(13000, ann(2), false);
        // Placed on 0 by hand; the video, a second ahead of its roll, plays through it.
        expect(f.step(13000, ann(0), false)).toBeNull();
        expect(f.step(500, ann(0), false)).toBeNull();
        expect(f.step(4000, ann(0), false)).toBeNull();
        expect(f.step(9000, ann(0), false)).toBeNull();
        expect(f.step(10500, ann(0), false)).toBe(1);
    });

    test('a video the panel placed at opening starts no transition: the Cursor stays at the end', () => {
        const f = createCursorFollower({ startMs: 9000 });
        // Reopened a little before the last Repère, the Cursor at the end where the user types.
        expect(f.step(9000, ann(2), false)).toBeNull();
        expect(f.step(9500, ann(2), false)).toBeNull();
        expect(f.step(11000, ann(2), false)).toBeNull();
        expect(f.step(13000, ann(2), false)).toBeNull();
        // Sent back by hand into an earlier stretch, the video is followed again.
        expect(f.step(3000, ann(2), false)).toBe(0);
    });

    test('a first instant away from where the panel placed the video is followed as before', () => {
        const f = createCursorFollower({ startMs: 9000 });
        expect(f.step(2000, ann(2), false)).toBeNull();
        expect(f.step(2500, ann(2), false)).toBe(0);
    });

    test('a quiet step never moves, and the transition it saw is spent', () => {
        const f = createCursorFollower();
        f.step(13000, ann(2), false);
        expect(f.step(2000, ann(2), true)).toBeNull();
        expect(f.step(2500, ann(2), false)).toBeNull();
    });
});

describe('isTyping', () => {
    const actions = [checker(1000, 5000)];

    test('an Action loaded as it is written is not typing; a new roll or a touched list is', () => {
        expect(isTyping({ actions, entry: { at: 0, replacing: true, dice: [1, 3], notation: '8/5 6/5' } }, { phase: 'roll' })).toBe(false);
        expect(isTyping({ actions, entry: { at: 0, replacing: true, dice: [5, 2], notation: '' } }, { phase: 'roll' })).toBe(true);
        expect(isTyping({ actions, entry: { at: 0, replacing: true, dice: [3, 1], notation: '8/5 6/5' } }, { phase: 'candidate' })).toBe(true);
        expect(isTyping({ actions, entry: { at: 1, replacing: false, dice: [6, 0] } }, { phase: 'die1' })).toBe(true);
        expect(isTyping({ actions, entry: null }, { phase: 'dice' })).toBe(false);
    });
});
