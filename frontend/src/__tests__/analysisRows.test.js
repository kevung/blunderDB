/**
 * analysisRows.test.js
 *
 * The one place a number of an analysis becomes a cell (utils/analysisRows.js).
 * Two DOM tables and the copied image read these rows, so the formatting
 * rules are pinned here once rather than observed three times.
 */

import { describe, test, expect } from 'vitest';
import {
    formatEquity,
    formatChance,
    cubeRows,
    cubeInfoRows,
    cubeFactRows,
    checkerRows,
    orderMoveTokens,
    condenseMoveTokens,
    moveLabel,
    isPlayedOption,
    DASH,
    HIDDEN,
    CHECKER_COLUMNS
} from '../utils/analysisRows.js';
import { cubeDecision, DECISION_STATE } from '../utils/cubeDecision.js';

// Labels come back as their key: the tests read the key, the app reads the
// translation, and both go through the same parameter.
const t = (key) => key;

describe('formatEquity — the single equity rule', () => {
    test.each([
        [0.123456, '+0.123'],
        [-0.5, '-0.500'],
        [0, '+0.000'],
        [1, '+1.000'],
        [-1.0004, '-1.000']
    ])('%s → %s', (value, expected) => {
        expect(formatEquity(value)).toBe(expected);
    });

    test('an absent value is null, never +0.000 — the surface decides what absence means', () => {
        expect(formatEquity(null)).toBeNull();
        expect(formatEquity(undefined)).toBeNull();
        expect(formatEquity(NaN)).toBeNull();
    });
});

describe('formatChance — the single probability rule', () => {
    test('two decimals on the scale the value arrives in (no ×100, ADR-0019)', () => {
        expect(formatChance(65.4321)).toBe('65.43');
        expect(formatChance(0)).toBe('0.00');
        expect(formatChance(null)).toBeNull();
        expect(formatChance(undefined)).toBeNull();
    });
});

const storedCube = {
    analysisDepth: 'XG Roller++',
    analysisEngine: 'eXtreme Gammon 2.19',
    playerWinChances: 72.15,
    playerGammonChances: 20.5,
    playerBackgammonChances: 1.02,
    opponentWinChances: 27.85,
    opponentGammonChances: 5.1,
    opponentBackgammonChances: 0,
    cubelessNoDoubleEquity: 0.612,
    cubelessDoubleEquity: 1.224,
    cubefulNoDoubleEquity: 0.85,
    cubefulNoDoubleError: -0.15,
    cubefulDoubleTakeEquity: 1.0,
    cubefulDoubleTakeError: 0,
    cubefulDoublePassEquity: 1.0,
    cubefulDoublePassError: 0,
    bestCubeAction: 'Double/Take'
};

describe('cubeRows', () => {
    test("a stored record: the three named rows, formatted, and the record's own verdict verbatim", () => {
        const block = cubeRows(cubeDecision({ cubeAnalysis: storedCube, stored: true }), { t });
        expect(block.header).toEqual(['analysis.decision', 'analysis.equity', 'analysis.error']);
        expect(block.rows.map((r) => r.key)).toEqual(['no_double', 'double_take', 'double_pass']);
        expect(block.rows.map((r) => r.label)).toEqual(['analysis.noDouble', 'analysis.doubleTake', 'analysis.doublePass']);
        expect(block.rows.map((r) => r.cells)).toEqual([
            ['+0.850', '-0.150'],
            ['+1.000', '+0.000'],
            ['+1.000', '+0.000']
        ]);
        expect(block.rows.map((r) => r.best)).toEqual([false, true, false]);
        expect(block.verdict).toEqual({ label: 'analysis.bestAction', text: 'Double/Take', unavailable: false });
    });

    test('a zero error is +0.000 (a measured result), an absent one is empty (ADR-0020 rule 2)', () => {
        const live = cubeDecision({ cubeAnalysis: { ...storedCube, cubefulNoDoubleEquity: 0.4, cubefulDoubleTakeEquity: 0.55, cubefulDoublePassEquity: 1 }, verdictKey: 'double_take' });
        const block = cubeRows(live, { t });
        expect(block.rows[1].cells).toEqual(['+0.550', '']); // the best: nothing to lose
        expect(block.rows[0].cells).toEqual(['+0.400', '-0.150']);
        expect(block.rows[2].cells).toEqual(['+1.000', '+0.450']);
        expect(block.verdict.text).toBe('cube.verdicts.double_take');
    });

    test('"too good" is a verdict on the no-double row, and is translated by key', () => {
        const live = cubeDecision({ cubeAnalysis: { ...storedCube, cubefulNoDoubleEquity: 1.2, cubefulDoubleTakeEquity: 1.1, cubefulDoublePassEquity: 1 }, verdictKey: 'too_good' });
        const block = cubeRows(live, { t });
        expect(block.verdict.text).toBe('cube.verdicts.too_good');
        expect(block.rows.find((r) => r.best).key).toBe('no_double');
    });

    test('a turned cube relabels the options as redoubles (cubeValue is the log2 exponent)', () => {
        const decision = cubeDecision({ cubeAnalysis: storedCube, stored: true });
        expect(cubeRows(decision, { t, cubeValue: 0 }).rows.map((r) => r.label)).toEqual(['analysis.noDouble', 'analysis.doubleTake', 'analysis.doublePass']);
        expect(cubeRows(decision, { t, cubeValue: 1 }).rows.map((r) => r.label)).toEqual(['analysis.noRedouble', 'analysis.redoubleTake', 'analysis.redoublePass']);
    });

    // ADR-0016 point 6 / #190/C.3 point 4: the equity column states its own
    // referential — the one thing that told a reader the scale had changed
    // (money points vs normalised match equity, ADR-0019) — instead of the
    // same silent "Équité" either way.
    test('the equity header states the referential when the caller has one, and stays plain otherwise', () => {
        const decision = cubeDecision({ cubeAnalysis: storedCube, stored: true });
        expect(cubeRows(decision, { t }).header[1]).toBe('analysis.equity');
        expect(cubeRows(decision, { t, isMoney: true }).header[1]).toBe('analysis.equityMoney');
        expect(cubeRows(decision, { t, isMoney: false }).header[1]).toBe('analysis.equityMatch');
    });

    test('the played action is highlighted through the legacy action vocabulary', () => {
        const decision = cubeDecision({ cubeAnalysis: storedCube, stored: true });
        const played = new Set(['Double', 'Pass']);
        const block = cubeRows(decision, { t, isPlayedCubeAction: (a) => played.has(a) });
        expect(block.rows.map((r) => r.highlight)).toEqual([false, false, true]);
        expect(isPlayedOption('no_double', (a) => a === 'No Double')).toBe(true);
        expect(isPlayedOption('double_take', (a) => a === 'Double')).toBe(false);
    });

    test('where doubling is not an option, errors are empty and the verdict names the state', () => {
        const decision = cubeDecision({ cubeAnalysis: storedCube, stored: true, turnability: DECISION_STATE.CUBE_OPPONENT });
        const block = cubeRows(decision, { t });
        expect(block.rows.map((r) => r.cells[1])).toEqual(['', '', '']);
        expect(block.rows.map((r) => r.cells[0])).toEqual(['+0.850', '+1.000', '+1.000']);
        expect(block.verdict).toEqual({ label: 'analysis.bestAction', text: 'cube.cubeOpponent', unavailable: true });
        expect(cubeRows(cubeDecision({ cubeAnalysis: storedCube, turnability: DECISION_STATE.CRAWFORD }), { t }).verdict.text).toBe('cube.crawford');
        expect(cubeRows(cubeDecision({ refused: true }), { t }).verdict.text).toBe('cube.refused');
        expect(cubeRows(cubeDecision({ isRace: true, race: {}, settled: true }), { t }).verdict.text).toBe('cube.noDecision');
    });

    test('pending, or no decision at all: the structure stays, every value cell is empty', () => {
        for (const decision of [cubeDecision({}), null, undefined]) {
            const block = cubeRows(decision, { t });
            expect(block.rows).toHaveLength(3);
            expect(block.rows.every((r) => r.cells.join('') === '' && !r.best && !r.highlight)).toBe(true);
            expect(block.verdict).toEqual({ label: 'analysis.bestAction', text: '', unavailable: false });
        }
    });

    test('masked (Défi): every figure and the verdict hidden, no best, no played mark', () => {
        const decision = cubeDecision({ cubeAnalysis: storedCube, stored: true });
        const block = cubeRows(decision, { t, isPlayedCubeAction: () => true, masked: true });
        expect(block.rows.every((r) => r.cells.every((c) => c === HIDDEN) && !r.best && !r.highlight)).toBe(true);
        expect(block.verdict.text).toBe(HIDDEN);
    });
});

describe('cubeInfoRows / cubeFactRows', () => {
    test('depth and engine, the engine falling back to the record-wide version', () => {
        expect(cubeInfoRows(storedCube, { t })).toEqual([
            { label: 'analysis.analysisDepth', cells: ['XG Roller++'] },
            { label: 'analysis.engine', cells: ['eXtreme Gammon 2.19'] }
        ]);
        expect(cubeInfoRows({ analysisDepth: '3-ply' }, { t, engineFallback: 'GNU 1.08' })[1].cells).toEqual(['GNU 1.08']);
        expect(cubeInfoRows(null, { t }).map((r) => r.cells)).toEqual([[''], ['']]);
    });

    test('the facts grid: chances at two decimals, equities at three, a dash for what is missing', () => {
        const facts = cubeFactRows(storedCube);
        expect(facts.header).toEqual(['', 'P', 'O']);
        expect(facts.rows.map((r) => [r.label, ...r.cells])).toEqual([
            ['W', '72.15', '27.85'],
            ['G', '20.50', '5.10'],
            ['B', '1.02', '0.00'],
            ['ND Eq', '+0.612'],
            ['D Eq', '+1.224']
        ]);
        expect(cubeFactRows({}).rows.map((r) => r.cells)).toEqual([[DASH, DASH], [DASH, DASH], [DASH, DASH], [DASH], [DASH]]);
    });
});

const moves = [
    {
        index: 0,
        move: '24/23 13/10',
        analysisDepth: '4-ply',
        analysisEngine: 'XG',
        equity: 0.123,
        playerWinChance: 55.123,
        playerGammonChance: 12.5,
        playerBackgammonChance: 0.4,
        opponentWinChance: 44.877,
        opponentGammonChance: 9.99,
        opponentBackgammonChance: 0
    },
    {
        index: 1,
        move: '24/21 13/12',
        analysisDepth: '4-ply',
        analysisEngine: 'XG',
        equity: -0.02,
        equityError: -0.143,
        playerWinChance: 50,
        playerGammonChance: 10,
        playerBackgammonChance: 0.1,
        opponentWinChance: 50,
        opponentGammonChance: 11,
        opponentBackgammonChance: 0.5
    }
];

describe('orderMoveTokens — a play reads from the back', () => {
    test('the least advanced checker moves first, whatever order the producer wrote', () => {
        expect(orderMoveTokens('18/14 24/18')).toBe('24/18 18/14');
        expect(orderMoveTokens('13/11 24/22 22/20')).toBe('24/22 22/20 13/11');
    });

    test('the bar comes before every point', () => {
        expect(orderMoveTokens('13/11 bar/24')).toBe('bar/24 13/11');
        expect(orderMoveTokens('6/2 bar/21*')).toBe('bar/21* 6/2');
    });

    test('hits, repetitions and bear-offs travel with their token', () => {
        expect(orderMoveTokens('6/off(2) 13/7*')).toBe('13/7* 6/off(2)');
        expect(orderMoveTokens('8/3 24/18/13')).toBe('24/18/13 8/3');
    });

    test('a tie keeps the producer order — nothing to decide between two checkers from the same point', () => {
        expect(orderMoveTokens('8/5 8/3')).toBe('8/5 8/3');
        expect(orderMoveTokens('8/3 8/5')).toBe('8/3 8/5');
    });

    test('what does not parse as this notation is returned untouched', () => {
        expect(orderMoveTokens('Cannot move')).toBe('Cannot move');
        expect(orderMoveTokens('')).toBe('');
        expect(orderMoveTokens(undefined)).toBe('');
    });
});

describe('condenseMoveTokens — one checker, one displacement', () => {
    test('the steps of a single checker become the move they are', () => {
        expect(condenseMoveTokens('24/18 18/14')).toBe('24/14');
        expect(condenseMoveTokens('24/18/14')).toBe('24/14');
        expect(condenseMoveTokens('bar/24 24/18')).toBe('bar/18');
        expect(condenseMoveTokens('13/7 7/1 1/off')).toBe('13/off');
    });

    test('a hit on arrival travels with the condensed move', () => {
        expect(condenseMoveTokens('24/18 18/14*')).toBe('24/14*');
    });

    test('a hit on the way through keeps the staging point', () => {
        expect(condenseMoveTokens('24/18* 18/14')).toBe('24/18* 18/14');
        expect(condenseMoveTokens('24/18*/14')).toBe('24/18*/14');
        expect(condenseMoveTokens('24/20 20/16* 16/12')).toBe('24/16* 16/12');
        expect(condenseMoveTokens('24/20/16*/12')).toBe('24/16*/12');
    });

    test('the two halves of a journey need not be neighbours', () => {
        expect(condenseMoveTokens('18/14 24/18')).toBe('24/14');
        expect(condenseMoveTokens('13/11 12/10 11/9')).toBe('13/9 12/10');
    });

    test('checkers that stay apart are left alone', () => {
        expect(condenseMoveTokens('8/5 6/5')).toBe('8/5 6/5');
        expect(condenseMoveTokens('24/18 13/8')).toBe('24/18 13/8');
    });

    test('a chain is joined only when the same checkers made both halves', () => {
        expect(condenseMoveTokens('24/18(2) 18/14(2)')).toBe('24/14(2)');
        expect(condenseMoveTokens('24/18(2) 18/14')).toBe('24/18(2) 18/14');
    });

    test('what does not parse as this notation is returned untouched', () => {
        expect(condenseMoveTokens('Cannot move')).toBe('Cannot move');
        expect(condenseMoveTokens('')).toBe('');
        expect(condenseMoveTokens(undefined)).toBe('');
    });
});

describe('moveLabel — condensed, then read from the back', () => {
    test('the play is condensed before it is ordered', () => {
        expect(moveLabel('18/14 24/18')).toBe('24/14');
        expect(moveLabel('8/3 24/18 18/14')).toBe('24/14 8/3');
        expect(moveLabel('8/3 24/18* 18/14')).toBe('24/18* 18/14 8/3');
    });
});

describe('checkerRows', () => {
    test('the header follows CHECKER_COLUMNS, with and without provenance', () => {
        expect(checkerRows(moves, { t }).header).toEqual(CHECKER_COLUMNS.map(() => expect.stringMatching(/^analysis\./)));
        expect(checkerRows(moves, { t }).header).toHaveLength(11);
        const bare = checkerRows(moves, { t, showProvenance: false });
        expect(bare.columns).toEqual(CHECKER_COLUMNS.slice(0, 9));
        expect(bare.header.at(-1)).toBe('analysis.opponentBackgammon');
        expect(bare.rows[0].cells).toHaveLength(8);
    });

    // ADR-0016 point 6 / #190/C.3 point 4, same rule as cubeRows above: only
    // the equity column's key changes with the referential, never the rest
    // of the header.
    test('the equity column states the referential; the other columns are unaffected', () => {
        expect(checkerRows(moves, { t }).header[1]).toBe('analysis.equity');
        expect(checkerRows(moves, { t, isMoney: true }).header[1]).toBe('analysis.equityMoney');
        expect(checkerRows(moves, { t, isMoney: false }).header[1]).toBe('analysis.equityMatch');
        expect(checkerRows(moves, { t, isMoney: true }).header[0]).toBe('analysis.move');
        expect(checkerRows(moves, { t, isMoney: true }).header[2]).toBe('analysis.error');
    });

    test('every figure formatted once: equity and error at three decimals, chances at two', () => {
        const block = checkerRows(moves, { t });
        expect(block.rows[1].label).toBe('24/21 13/12');
        expect(block.rows[1].cells).toEqual(['-0.020', '-0.143', '50.00', '10.00', '0.10', '50.00', '11.00', '0.50', '4-ply', 'XG']);
        expect(block.rows[1].key).toBe(1);
        expect(block.rows[1].move).toBe(moves[1]);
    });

    test('the best move of a stored record carries no error at all: a zero by construction, +0.000', () => {
        const block = checkerRows(moves, { t });
        expect(block.rows[0].cells[1]).toBe('+0.000');
        expect(block.rows[0].cells[0]).toBe('+0.123');
    });

    test('every other absent value is a dash — never measured, never a zero', () => {
        expect(checkerRows([{ move: '8/5 6/5' }], { t }).rows[0].cells).toEqual([DASH, '+0.000', DASH, DASH, DASH, DASH, DASH, DASH, '', '']);
    });

    test('a zero equity or error is a measured +0.000', () => {
        expect(checkerRows([{ move: 'x', equity: 0, equityError: 0 }], { t }).rows[0].cells.slice(0, 2)).toEqual(['+0.000', '+0.000']);
    });

    test('the played move is highlighted, the order given is the order returned', () => {
        const block = checkerRows([...moves].reverse(), { t, isPlayedMove: (m) => m.index === 1 });
        expect(block.rows.map((r) => r.label)).toEqual(['24/21 13/12', '24/23 13/10']);
        expect(block.rows.map((r) => r.highlight)).toEqual([true, false]);
    });

    test('the move cell is rewritten for reading, the move object is not', () => {
        const row = checkerRows([{ move: '18/14 24/18', index: 0 }], { t }).rows[0];
        expect(row.label).toBe('24/14');
        expect(row.move.move).toBe('18/14 24/18');
    });

    test('the baseline row: no error figure (ADR-0018 rule 3), a dash for a missing equity', () => {
        const baseline = { cubelessEquity: 0.05, playerWinChance: 52, playerGammonChance: 8, playerBackgammonChance: 0, opponentWinChance: 48, opponentGammonChance: 7, opponentBackgammonChance: 0 };
        const block = checkerRows(moves, { t, baseline });
        expect(block.baseline.label).toBe('eval.baseline');
        expect(block.baseline.cells).toEqual(['+0.050', '', '52.00', '8.00', '0.00', '48.00', '7.00', '0.00', '', '']);
        expect(checkerRows(moves, { t, baseline: { ...baseline, cubelessEquity: null }, showProvenance: false }).baseline.cells).toEqual([DASH, '', '52.00', '8.00', '0.00', '48.00', '7.00', '0.00']);
        expect(checkerRows(moves, { t }).baseline).toBeNull();
    });

    test('no moves: a header and nothing under it', () => {
        expect(checkerRows([], { t }).rows).toEqual([]);
        expect(checkerRows(undefined, { t }).rows).toEqual([]);
    });
});

// The "played" predicates themselves (playedMovePredicate/playedCubeActionPredicate,
// including isPlayedOption's use of one) moved to playedMarks.test.js — they live in
// utils/playedMarks.js, not here (fiche D.10, #210).
