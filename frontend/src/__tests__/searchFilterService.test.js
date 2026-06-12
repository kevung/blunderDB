import { describe, test, expect, vi } from 'vitest';
import { buildFilterTokens, buildSearchCommand } from '../services/searchFilterService.js';

// buildFilterTokens/buildSearchCommand are pure (no Wails imports), but the
// round-trip block below imports commandProcessor's parseFilters, which pulls in
// the Database bindings — mock them the way commandProcessor.test.js does.
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    SaveComment: vi.fn().mockResolvedValue(undefined),
    SaveSearchHistory: vi.fn().mockResolvedValue(undefined),
    ClearCommandHistory: vi.fn().mockResolvedValue(undefined)
}));

import { parseFilters } from '../commandProcessor.js';

// A baseline options object with distinct min/max/range values per family so a
// wrong field surfaces immediately. Range-type filters read <name>Option plus
// <name>Min / <name>Max / <name>RangeMin / <name>RangeMax.
function baseOptions(overrides = {}) {
    return {
        diceRollOption: 'both',
        pipCountOption: 'min',
        pipCountMin: 5,
        pipCountMax: 50,
        pipCountRangeMin: 10,
        pipCountRangeMax: 20,
        player1AbsolutePipCountOption: 'min',
        player1AbsolutePipCountMin: 60,
        player1AbsolutePipCountMax: 160,
        player1AbsolutePipCountRangeMin: 80,
        player1AbsolutePipCountRangeMax: 120,
        equityOption: 'min',
        equityMin: 100,
        equityMax: 900,
        equityRangeMin: 200,
        equityRangeMax: 800,
        moveErrorOption: 'min',
        moveErrorMin: 10,
        moveErrorMax: 90,
        moveErrorRangeMin: 20,
        moveErrorRangeMax: 80,
        winRateOption: 'min',
        winRateMin: 40,
        winRateMax: 60,
        winRateRangeMin: 45,
        winRateRangeMax: 55,
        gammonRateOption: 'min',
        gammonRateMin: 1,
        gammonRateMax: 9,
        gammonRateRangeMin: 2,
        gammonRateRangeMax: 8,
        backgammonRateOption: 'min',
        backgammonRateMin: 1,
        backgammonRateMax: 5,
        backgammonRateRangeMin: 2,
        backgammonRateRangeMax: 4,
        player2WinRateOption: 'min',
        player2WinRateMin: 41,
        player2WinRateMax: 61,
        player2WinRateRangeMin: 46,
        player2WinRateRangeMax: 56,
        player2GammonRateOption: 'min',
        player2GammonRateMin: 3,
        player2GammonRateMax: 7,
        player2GammonRateRangeMin: 4,
        player2GammonRateRangeMax: 6,
        player2BackgammonRateOption: 'min',
        player2BackgammonRateMin: 1,
        player2BackgammonRateMax: 3,
        player2BackgammonRateRangeMin: 1,
        player2BackgammonRateRangeMax: 2,
        player1CheckerOffOption: 'min',
        player1CheckerOffMin: 1,
        player1CheckerOffMax: 15,
        player1CheckerOffRangeMin: 3,
        player1CheckerOffRangeMax: 12,
        player2CheckerOffOption: 'min',
        player2CheckerOffMin: 2,
        player2CheckerOffMax: 14,
        player2CheckerOffRangeMin: 4,
        player2CheckerOffRangeMax: 11,
        player1BackCheckerOption: 'min',
        player1BackCheckerMin: 1,
        player1BackCheckerMax: 6,
        player1BackCheckerRangeMin: 2,
        player1BackCheckerRangeMax: 5,
        player2BackCheckerOption: 'min',
        player2BackCheckerMin: 1,
        player2BackCheckerMax: 7,
        player2BackCheckerRangeMin: 2,
        player2BackCheckerRangeMax: 6,
        player1CheckerInZoneOption: 'min',
        player1CheckerInZoneMin: 1,
        player1CheckerInZoneMax: 10,
        player1CheckerInZoneRangeMin: 3,
        player1CheckerInZoneRangeMax: 9,
        player2CheckerInZoneOption: 'min',
        player2CheckerInZoneMin: 2,
        player2CheckerInZoneMax: 11,
        player2CheckerInZoneRangeMin: 4,
        player2CheckerInZoneRangeMax: 8,
        searchText: 'hello',
        movePattern: '24/18 13/11',
        creationDateOption: 'min',
        creationDateMin: '2026-01-15',
        creationDateMax: '2026-12-31',
        creationDateRangeMin: '2026-03-01',
        creationDateRangeMax: '2026-06-30',
        player1OutfieldBlotOption: 'min',
        player1OutfieldBlotMin: 1,
        player1OutfieldBlotMax: 4,
        player1OutfieldBlotRangeMin: 1,
        player1OutfieldBlotRangeMax: 3,
        player2OutfieldBlotOption: 'min',
        player2OutfieldBlotMin: 1,
        player2OutfieldBlotMax: 5,
        player2OutfieldBlotRangeMin: 2,
        player2OutfieldBlotRangeMax: 4,
        player1JanBlotOption: 'min',
        player1JanBlotMin: 1,
        player1JanBlotMax: 3,
        player1JanBlotRangeMin: 1,
        player1JanBlotRangeMax: 2,
        player2JanBlotOption: 'min',
        player2JanBlotMin: 1,
        player2JanBlotMax: 4,
        player2JanBlotRangeMin: 2,
        player2JanBlotRangeMax: 3,
        matchIDsInput: '12',
        tournamentIDsInput: '7',
        ...overrides
    };
}

const token = (label, opts) => buildFilterTokens([label], baseOptions(opts))[0];

describe('buildFilterTokens — flag filters', () => {
    test.each([
        ['Include Cube', 'cube'],
        ['Include Score', 'score'],
        ['Include Decision Type', 'd'],
        ['No Contact', 'nc'],
        ['Mirror Position', 'M']
    ])('%s → %s', (label, expected) => {
        expect(token(label)).toBe(expected);
    });

    test('Include Dice Roll honours the first-die option', () => {
        expect(token('Include Dice Roll', { diceRollOption: 'both' })).toBe('D');
        expect(token('Include Dice Roll', { diceRollOption: 'first' })).toBe('D1');
    });

    test('unknown label maps to empty string', () => {
        expect(token('Not A Real Filter')).toBe('');
    });
});

// Every range-style filter: prefix + the three modes (min → `>`, max → `<`,
// range → `rmin,rmax`). <prefix>Option drives the branch.
describe('buildFilterTokens — range filters', () => {
    const RANGE = [
        ['Pipcount Difference', 'p', 'pipCount'],
        ['Player Absolute Pipcount', 'P', 'player1AbsolutePipCount'],
        ['Equity (millipoints)', 'e', 'equity'],
        ['Move Error (millipoints, Player 1)', 'E', 'moveError'],
        ['Win Rate', 'w', 'winRate'],
        ['Gammon Rate', 'g', 'gammonRate'],
        ['Backgammon Rate', 'b', 'backgammonRate'],
        ['Opponent Win Rate', 'W', 'player2WinRate'],
        ['Opponent Gammon Rate', 'G', 'player2GammonRate'],
        ['Opponent Backgammon Rate', 'B', 'player2BackgammonRate'],
        ['Player Checker-Off', 'o', 'player1CheckerOff'],
        ['Opponent Checker-Off', 'O', 'player2CheckerOff'],
        ['Player Back Checker', 'k', 'player1BackChecker'],
        ['Opponent Back Checker', 'K', 'player2BackChecker'],
        ['Player Checker in the Zone', 'z', 'player1CheckerInZone'],
        ['Opponent Checker in the Zone', 'Z', 'player2CheckerInZone'],
        ['Player Outfield Blot', 'bo', 'player1OutfieldBlot'],
        ['Opponent Outfield Blot', 'BO', 'player2OutfieldBlot'],
        ['Player Jan Blot', 'bj', 'player1JanBlot'],
        ['Opponent Jan Blot', 'BJ', 'player2JanBlot']
    ];

    test.each(RANGE)('%s (%s) — min/max/range', (label, prefix, name) => {
        const o = baseOptions();
        const min = o[`${name}Min`];
        const max = o[`${name}Max`];
        const rmin = o[`${name}RangeMin`];
        const rmax = o[`${name}RangeMax`];
        expect(token(label, { [`${name}Option`]: 'min' })).toBe(`${prefix}>${min}`);
        expect(token(label, { [`${name}Option`]: 'max' })).toBe(`${prefix}<${max}`);
        expect(token(label, { [`${name}Option`]: 'range' })).toBe(`${prefix}${rmin},${rmax}`);
    });
});

describe('buildFilterTokens — text, date, id filters', () => {
    test('Search Text and Best Move are quoted', () => {
        expect(token('Search Text', { searchText: 'foo bar' })).toBe('t"foo bar"');
        expect(token('Best Move or Cube Decision', { movePattern: '24/18' })).toBe('m"24/18"');
    });

    test('Creation Date converts yyyy-mm-dd to yyyy/mm/dd', () => {
        expect(token('Creation Date', { creationDateOption: 'min', creationDateMin: '2026-01-15' })).toBe('T>2026/01/15');
        expect(token('Creation Date', { creationDateOption: 'max', creationDateMax: '2026-12-31' })).toBe('T<2026/12/31');
        expect(token('Creation Date', { creationDateOption: 'range', creationDateRangeMin: '2026-03-01', creationDateRangeMax: '2026-06-30' })).toBe('T2026/03/01,2026/06/30');
    });

    test('Match/Tournament IDs are empty when the input is blank', () => {
        expect(token('Match IDs', { matchIDsInput: '12' })).toBe('ma12');
        expect(token('Match IDs', { matchIDsInput: '' })).toBe('');
        expect(token('Tournament IDs', { tournamentIDsInput: '7' })).toBe('tn7');
        expect(token('Tournament IDs', { tournamentIDsInput: '' })).toBe('');
    });
});

describe('buildSearchCommand', () => {
    test('prefixes with s and keeps tokens in order', () => {
        expect(buildSearchCommand(['p>5', 'cube'])).toBe('s p>5 cube');
    });

    test('no filters yields a bare s', () => {
        expect(buildSearchCommand([])).toBe('s');
    });

    test('drops empty text/move placeholders only', () => {
        expect(buildSearchCommand(['t""', 'cube', 'm""'])).toBe('s cube');
        // a non-empty text token survives
        expect(buildSearchCommand(['t"x"'])).toBe('s t"x"');
    });
});

// The service is the inverse of parseFilters: tokens produced here, fed back
// through parseFilters, must recover the same filter values. This locks the two
// search components against commandProcessor.
describe('round-trip against parseFilters', () => {
    test('a multi-filter selection recovers its tokens', () => {
        const labels = ['Include Cube', 'Pipcount Difference', 'Win Rate', 'Search Text', 'Creation Date'];
        const opts = baseOptions({
            pipCountOption: 'min',
            pipCountMin: 7,
            winRateOption: 'range',
            winRateRangeMin: 45,
            winRateRangeMax: 55,
            searchText: 'blitz',
            creationDateOption: 'min',
            creationDateMin: '2026-02-01'
        });
        const tokens = buildFilterTokens(labels, opts);
        const command = buildSearchCommand(tokens);

        const parsed = parseFilters(tokens, command);
        expect(parsed.includeCube).toBe(true);
        expect(parsed.pipCountFilter).toBe('p>7');
        expect(parsed.winRateFilter).toBe('w45,55');
        expect(parsed.searchText).toBe('t"blitz"');
        expect(parsed.dateFilter).toBe('T>2026/02/01');
    });

    test('opponent rates and blots are not confused with player ones', () => {
        const labels = ['Backgammon Rate', 'Player Outfield Blot', 'Player Jan Blot', 'Opponent Backgammon Rate'];
        const opts = baseOptions({
            backgammonRateOption: 'min',
            backgammonRateMin: 2,
            player1OutfieldBlotOption: 'min',
            player1OutfieldBlotMin: 1,
            player1JanBlotOption: 'min',
            player1JanBlotMin: 1,
            player2BackgammonRateOption: 'min',
            player2BackgammonRateMin: 3
        });
        const tokens = buildFilterTokens(labels, opts);
        const parsed = parseFilters(tokens, buildSearchCommand(tokens));
        expect(parsed.backgammonRateFilter).toBe('b>2');
        expect(parsed.player1OutfieldBlotFilter).toBe('bo>1');
        expect(parsed.player1JanBlotFilter).toBe('bj>1');
        expect(parsed.player2BackgammonRateFilter).toBe('B>3');
    });
});
