/**
 * cubeDecision.test.js
 *
 * The cube Decision's one shape (ADR-0020), tested where it is worth testing:
 * as a pure function, with no DOM. Five sources reach it — an exact race's
 * money block, an evaluated one, a live DoublingCubeAnalysis, a stored record,
 * and nothing at all — and it has to tell four kinds of "no answer" apart, which
 * is the whole point of rule 4: an empty verdict cell means "still computing"
 * and nothing else.
 */

import { describe, test, expect } from 'vitest';
import { cubeDecision, cubeTurnability, isMoneyPosition, CUBE_OPTIONS, DECISION_STATE } from '../utils/cubeDecision.js';

const money = { cubeless: 0.4, no_double: 0.55, double_take: 0.71, double_pass: 1.0, verdict: 'double_take' };

const liveCube = {
    cubefulNoDoubleEquity: 0.055,
    cubefulDoubleTakeEquity: -0.256,
    cubefulDoublePassEquity: 1.0,
    cubefulNoDoubleError: 0,
    cubefulDoubleTakeError: -0.311,
    cubefulDoublePassError: 0,
    bestCubeAction: 'No Double'
};

const keys = (d) => d.options.map((o) => o.key);

describe('cubeDecision — shape', () => {
    test('always the three canonical options, in canonical order, whatever the source', () => {
        expect(keys(cubeDecision({}))).toEqual(CUBE_OPTIONS);
        expect(keys(cubeDecision({ isRace: true, race: { money } }))).toEqual(CUBE_OPTIONS);
        expect(keys(cubeDecision({ cubeAnalysis: liveCube }))).toEqual(CUBE_OPTIONS);
        expect(keys(cubeDecision({ refused: true }))).toEqual(CUBE_OPTIONS);
    });

    test('the order never follows the equities — the best option can be any row', () => {
        // no-double best here, double/take best in the race block above: same
        // order both times. Sorting would permute rows mid-escalation.
        expect(cubeDecision({ cubeAnalysis: liveCube }).best).toBe('no_double');
        expect(cubeDecision({ isRace: true, race: { money } }).best).toBe('double_take');
        expect(keys(cubeDecision({ cubeAnalysis: liveCube }))).toEqual(keys(cubeDecision({ isRace: true, race: { money } })));
    });
});

describe('cubeDecision — errors', () => {
    test('each option is its equity minus the best, and the best carries none', () => {
        const d = cubeDecision({ isRace: true, race: { money } });
        // best = max(no_double, min(take, pass)) = max(0.55, 0.71) = 0.71
        const byKey = Object.fromEntries(d.options.map((o) => [o.key, o]));
        expect(byKey.double_take.error).toBeNull();
        expect(byKey.no_double.error).toBeCloseTo(0.55 - 0.71, 10);
        expect(byKey.double_pass.error).toBeCloseTo(1.0 - 0.71, 10);
    });

    test('a stored record keeps its own errors verbatim, best action included', () => {
        // A source declaring "No Double" best while giving it a non-zero error
        // is exactly what rounding produces: Analysis reports, it does not
        // correct.
        const inconsistent = { ...liveCube, cubefulNoDoubleError: -0.004 };
        const d = cubeDecision({ cubeAnalysis: inconsistent, stored: true });
        const byKey = Object.fromEntries(d.options.map((o) => [o.key, o]));
        expect(byKey.no_double.error).toBe(-0.004);
        expect(byKey.double_take.error).toBe(-0.311);
        expect(d.best).toBe('no_double'); // the record's declared best, not ours
    });

    test('a stored record with no best action marks no row rather than guessing', () => {
        const d = cubeDecision({ cubeAnalysis: { ...liveCube, bestCubeAction: '' }, stored: true });
        expect(d.best).toBeNull();
    });
});

describe('cubeDecision — the four ways there is no verdict', () => {
    test('nothing yet is pending', () => {
        const d = cubeDecision({});
        expect(d.state).toBe(DECISION_STATE.PENDING);
        expect(d.options.every((o) => o.equity === null)).toBe(true);
    });

    test('an estimated race is "no decision", not pending — it will never arrive', () => {
        const d = cubeDecision({ isRace: true, race: { regime: 'estimated', win_prob: 0.7 } });
        expect(d.state).toBe(DECISION_STATE.NO_DECISION);
    });

    test('…but not before gammonNet has answered: the fast race path lands first', () => {
        // updateEPC fills the estimated block well before the evaluated regime
        // does. Claiming "no decision" in that window would flash a settled
        // state at a position still being computed.
        const d = cubeDecision({ isRace: true, race: { regime: 'estimated', win_prob: 0.7 }, settled: false });
        expect(d.state).toBe(DECISION_STATE.PENDING);
    });

    test('a refusal is a state, and it drops the previous numbers', () => {
        const d = cubeDecision({ refused: true, cubeAnalysis: liveCube });
        expect(d.state).toBe(DECISION_STATE.REFUSED);
        expect(d.options.every((o) => o.equity === null)).toBe(true);
    });

    test('a cube that cannot be turned keeps the equities and drops every error', () => {
        const d = cubeDecision({ cubeAnalysis: liveCube, turnability: DECISION_STATE.CUBE_OPPONENT });
        expect(d.state).toBe(DECISION_STATE.CUBE_OPPONENT);
        expect(d.options.map((o) => o.equity)).toEqual([0.055, -0.256, 1.0]);
        expect(d.options.every((o) => o.error === null)).toBe(true);
        expect(d.best).toBeNull(); // nothing is "best" among options nobody can take
        expect(d.verdict).toBeNull();
    });
});

describe('cubeDecision — verdict source', () => {
    test('the live path carries a key, so the panel can translate it', () => {
        expect(cubeDecision({ cubeAnalysis: liveCube, verdictKey: 'too_good' }).verdict).toBe('too_good');
        expect(cubeDecision({ isRace: true, race: { money } }).verdict).toBe('double_take');
    });

    test('the stored path carries the record’s own words, never a key', () => {
        const d = cubeDecision({ cubeAnalysis: { ...liveCube, bestCubeAction: 'ダブル' }, stored: true });
        expect(d.verdict).toBeNull();
        expect(d.verdictText).toBe('ダブル');
    });
});

describe('cubeTurnability', () => {
    const at = (score, owner, onRoll = 0) => cubeTurnability({ score, cube: { owner, value: 0 }, player_on_roll: onRoll });

    test('money play with a centred cube: nothing in the way', () => {
        expect(at([-1, -1], -1)).toBeNull();
    });

    test('the opponent owning the cube removes every option', () => {
        expect(at([-1, -1], 1, 0)).toBe(DECISION_STATE.CUBE_OPPONENT);
        expect(at([-1, -1], 0, 0)).toBeNull(); // owned by the player on roll
    });

    test('the Crawford game has no cube in play, by rule', () => {
        expect(at([1, 5], -1)).toBe(DECISION_STATE.CRAWFORD);
        expect(at([5, 1], -1)).toBe(DECISION_STATE.CRAWFORD);
    });

    test('post-Crawford is not Crawford: the away=0 sentinel is a different game', () => {
        expect(at([0, 7], -1)).toBeNull();
    });

    test('a 1-away money score is not Crawford — money play has no match at all', () => {
        expect(at([-1, -1], -1)).toBeNull();
    });
});

// #190/C.3 point 2: THE money/match predicate on the frontend's own position
// shape, so cubeTurnability above and EPCPanel's own hasScore read the exact
// same rule instead of two independently-written forms that only agreed on a
// well-formed score.
describe('isMoneyPosition', () => {
    test('both sides at the money sentinel is money play', () => {
        expect(isMoneyPosition({ score: [-1, -1] })).toBe(true);
    });

    test('a real away score on both sides is match play', () => {
        expect(isMoneyPosition({ score: [5, 7] })).toBe(false);
        expect(isMoneyPosition({ score: [0, 3] })).toBe(false); // post-Crawford sentinel
        expect(isMoneyPosition({ score: [1, 3] })).toBe(false); // Crawford sentinel
    });

    test('exactly one side at the money sentinel is malformed, and is NOT money', () => {
        expect(isMoneyPosition({ score: [-1, 5] })).toBe(false);
        expect(isMoneyPosition({ score: [5, -1] })).toBe(false);
    });

    test('a missing position or score defaults to money, like cubeTurnability', () => {
        expect(isMoneyPosition(undefined)).toBe(true);
        expect(isMoneyPosition({})).toBe(true);
    });
});
