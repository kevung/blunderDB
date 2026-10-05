/**
 * Le Duel côté bureau : le formulaire devenu `duel.Settings`, les horloges lues dans l'état de
 * l'Arbitre, et ce que le plateau rejoue du Bot.
 */
import { describe, test, expect } from 'vitest';
import { normalizeForm, settingsFromForm, scoreStart, clockView, formatClock, framesBetween, humanSide, DEFAULT_FORM, START } from '../services/duel.js';

const CADENCES = [{ name: 'rapid-3+12', reserve: 180, delay: 12 }];

describe('normalizeForm', () => {
    test('completes and clamps a remembered form', () => {
        const f = normalizeForm({ matchLength: 99, side: 7, level: 'god', away: [0, 40], timeOut: 'x' });
        expect(f.matchLength).toBe(25);
        expect(f.side).toBe(0);
        expect(f.level).toBe(DEFAULT_FORM.level);
        expect(f.away).toEqual([1, 25]);
        expect(f.timeOut).toBe('continue');
        expect(f.record).toBe(true);
    });

    test('survives garbage', () => {
        expect(normalizeForm('nope')).toEqual(normalizeForm(null));
    });
});

describe('settingsFromForm', () => {
    test('the player and the Bot on the chosen Sides', () => {
        const set = settingsFromForm({ ...DEFAULT_FORM, side: 1, player: ' Kévin ', level: 'thorough' }, CADENCES);
        expect(set.sides).toEqual([
            { kind: 'bot', level: 'thorough' },
            { kind: 'external', name: 'Kévin' }
        ]);
        expect(set.matchLength).toBe(7);
        expect(set.start).toBeUndefined();
        expect(set.cadence).toBeUndefined();
    });

    test('a money session keeps Jacoby, a match never', () => {
        expect(settingsFromForm({ ...DEFAULT_FORM, money: true, jacoby: true }, []).jacoby).toBe(true);
        expect(settingsFromForm({ ...DEFAULT_FORM, money: true }, []).matchLength).toBe(0);
        expect(settingsFromForm({ ...DEFAULT_FORM, money: false, jacoby: true }, []).jacoby).toBe(false);
    });

    test('a named Cadence carries the consequence of the time out', () => {
        const set = settingsFromForm({ ...DEFAULT_FORM, cadence: 'rapid-3+12', timeOut: 'lose_match' }, CADENCES);
        expect(set.cadence).toEqual({ name: 'rapid-3+12', reserve: 180, delay: 12, timeOut: 'lose_match' });
    });

    test('the Start: the board without its id, or the opening at a score', () => {
        const board = { id: 42, board: { points: [] }, score: [3, 3] };
        expect(settingsFromForm({ ...DEFAULT_FORM, start: START.BOARD }, [], board).start).toEqual({ board: { points: [] }, score: [3, 3] });
        const atScore = settingsFromForm({ ...DEFAULT_FORM, start: START.SCORE, matchLength: 5, away: [2, 5] }, []).start;
        expect(atScore.score).toEqual([2, 5]);
        expect(atScore.dice).toEqual([0, 0]);
    });

    test('a record left unticked throws the draft away at the end', () => {
        expect(settingsFromForm({ ...DEFAULT_FORM, record: false }, []).discardAtEnd).toBe(true);
    });
});

describe('scoreStart', () => {
    test('the opening board: fifteen checkers each', () => {
        const pos = scoreStart(7, [7, 7]);
        const count = (/** @type {number} */ color) => pos.board.points.filter((p) => p.color === color).reduce((n, p) => n + p.checkers, 0);
        expect(count(0)).toBe(15);
        expect(count(1)).toBe(15);
        expect(pos.cube).toEqual({ owner: -1, value: 0 });
    });
});

describe('clockView', () => {
    const since = '2026-10-05T10:00:00.000Z';
    const t0 = Date.parse(since);
    const state = {
        awaiting: { side: 1, since },
        clock: { cadence: { delay: 12 }, reserve: [180000, 180000], turn: 0, spent: 0 }
    };

    test('the delay is free, then the running reserve goes down', () => {
        const at5 = clockView(state, t0 + 5000);
        expect(at5.reserve).toEqual([180000, 180000]);
        expect(at5.delayLeft).toBe(7000);
        const at20 = clockView(state, t0 + 20000);
        expect(at20.reserve).toEqual([180000, 172000]);
        expect(at20.running).toBe(1);
    });

    test('no Cadence, no clock; an ended Duel runs no clock', () => {
        expect(clockView({ awaiting: state.awaiting }, t0)).toBeNull();
        expect(clockView({ ...state, ended: { matchId: 1 } }, t0 + 60000).running).toBe(-1);
    });

    test('formatClock', () => {
        expect(formatClock(172000)).toBe('2:52');
        expect(formatClock(-1500)).toBe('−0:02');
    });
});

describe('framesBetween and humanSide', () => {
    test('only the other Side’s checker plays are replayed', () => {
        const before = { actions: [{ side: 0, kind: 'checker', has_position: true }] };
        const after = {
            actions: [...before.actions, { side: 1, kind: 'double', has_position: true }, { side: 1, kind: 'checker', has_position: true }, { side: 0, kind: 'checker', has_position: true }]
        };
        expect(framesBetween(before, after, 0)).toEqual([{ side: 1, kind: 'checker', has_position: true }]);
    });

    test('the player is the external Side', () => {
        expect(humanSide({ sides: [{ kind: 'bot' }, { kind: 'external' }] })).toBe(1);
        expect(humanSide({ sides: [{ kind: 'external' }, { kind: 'bot' }] })).toBe(0);
    });
});
