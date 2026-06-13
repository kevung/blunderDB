import { describe, test, expect, vi } from 'vitest';
import { buildFilterTokens, buildSearchCommand, parseFilterTokens, parseSearchCommand, filterTokenHint } from '../services/searchFilterService.js';

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

describe('parseFilterTokens', () => {
    test('extracts includes-based boolean flags', () => {
        const p = parseFilterTokens(['cube', 'score', 'nc', 'M', 'd']);
        expect(p.incCube).toBe(true);
        expect(p.incScore).toBe(true);
        expect(p.ncFilter).toBe(true);
        expect(p.mirFilter).toBe(true);
        expect(p.dtFilter).toBe(true);
    });

    test('defaults are false / undefined when no tokens match', () => {
        const p = parseFilterTokens([]);
        expect(p.incCube).toBe(false);
        expect(p.ncFilter).toBe(false);
        expect(p.pcFilter).toBeUndefined();
        expect(p.cdFilter).toBeUndefined();
        expect(p.matchIDs).toBe('');
        expect(p.tournamentIDs).toBe('');
        expect(p.drFilter).toBe(false);
        expect(p.drMode).toBe('both');
    });

    test('disambiguates b-prefixed tokens: backgammon-rate vs outfield-blot vs jan-blot', () => {
        const p = parseFilterTokens(['b>2', 'bo>1', 'bj>1', 'B>3', 'BO>1', 'BJ>1']);
        expect(p.bgFilter).toBe('b>2'); // not bo>1 / bj>1
        expect(p.p1obFilter).toBe('bo>1');
        expect(p.p1jbFilter).toBe('bj>1');
        expect(p.p2bgFilter).toBe('B>3'); // not BO>1 / BJ>1
        expect(p.p2obFilter).toBe('BO>1');
        expect(p.p2jbFilter).toBe('BJ>1');
    });

    test('distinguishes lowercase pip-count (p) from uppercase absolute-pip (P)', () => {
        const p = parseFilterTokens(['p>12', 'P>50']);
        expect(p.pcFilter).toBe('p>12');
        expect(p.p1apcFilter).toBe('P>50');
    });

    test('double token D selects both rolls, D1 selects first roll', () => {
        expect(parseFilterTokens(['D']).drFilter).toBe(true);
        expect(parseFilterTokens(['D']).drMode).toBe('both');
        const first = parseFilterTokens(['D1']);
        expect(first.drFilter).toBe(true);
        expect(first.drMode).toBe('first');
    });

    test('strips the 2-char prefix from match (ma) and tournament (tn) id tokens', () => {
        const p = parseFilterTokens(['ma3,4,5', 'tn7']);
        expect(p.matchIDs).toBe('3,4,5');
        expect(p.tournamentIDs).toBe('7');
    });

    test('round-trips with buildFilterTokens for a representative filter set', () => {
        const tokens = buildFilterTokens(['Equity (millipoints)', 'Move Error (millipoints, Player 1)', 'Player Checker-Off'], {
            equityOption: 'range',
            equityRangeMin: 10,
            equityRangeMax: 50,
            moveErrorOption: 'min',
            moveErrorMin: 5,
            player1CheckerOffOption: 'min',
            player1CheckerOffMin: 2
        });
        const p = parseFilterTokens(tokens);
        expect(p.eqFilter).toBe('e10,50');
        expect(p.meFilter).toBe('E>5');
        expect(p.p1coFilter).toBe('o>2');
    });
});

describe('filterTokenHint', () => {
    test('range filters show the three operator forms', () => {
        expect(filterTokenHint('Player Back Checker')).toBe('k>n · k<n · kn,m');
        expect(filterTokenHint('Equity (millipoints)')).toBe('e>n · e<n · en,m');
        expect(filterTokenHint('Opponent Jan Blot')).toBe('BJ>n · BJ<n · BJn,m');
    });

    test('flag filters show the bare token', () => {
        expect(filterTokenHint('No Contact')).toBe('nc');
        expect(filterTokenHint('Include Cube')).toBe('cube');
        expect(filterTokenHint('Mirror Position')).toBe('M');
    });

    test('dice, text and date filters show their special forms', () => {
        expect(filterTokenHint('Include Dice Roll')).toBe('D · D1');
        expect(filterTokenHint('Search Text')).toBe('t"…"');
        expect(filterTokenHint('Creation Date')).toBe('T>YYYY/MM/DD · T<YYYY/MM/DD');
    });

    test('unknown labels return empty string (no tooltip)', () => {
        expect(filterTokenHint('Not A Filter')).toBe('');
        expect(filterTokenHint('')).toBe('');
    });

    test('every hint begins with its declared token', () => {
        // Guards against a prefix/typo drift between the hint and buildFilterTokens.
        const cases = [
            ['Win Rate', 'w'],
            ['Opponent Win Rate', 'W'],
            ['Player Checker in the Zone', 'z'],
            ['Player Outfield Blot', 'bo']
        ];
        for (const [label, tok] of cases) {
            expect(filterTokenHint(label).startsWith(tok)).toBe(true);
        }
    });
});

describe('Player filter token', () => {
    test('buildFilterTokens emits pl"Name" for the Player filter', () => {
        expect(buildFilterTokens(['Player'], { playerName: 'Alice' })).toEqual(['pl"Alice"']);
        expect(buildFilterTokens(['Player'], { playerName: 'Kévin Unger' })).toEqual(['pl"Kévin Unger"']);
    });

    test('parseFilterTokens extracts plFilter without stealing the pipcount token', () => {
        const p = parseFilterTokens(['pl"Bob"', 'p>120']);
        expect(p.plFilter).toBe('pl"Bob"');
        expect(p.pcFilter).toBe('p>120'); // pl"…" must NOT be picked as the pipcount
    });

    test('parseFilterTokens: pl absent leaves plFilter undefined and pipcount intact', () => {
        const p = parseFilterTokens(['p<60']);
        expect(p.plFilter).toBeUndefined();
        expect(p.pcFilter).toBe('p<60');
    });

    test('filterTokenHint shows the quoted-text form for Player', () => {
        expect(filterTokenHint('Player')).toBe('pl"…"');
    });

    test('buildSearchCommand drops an empty pl"" token', () => {
        expect(buildSearchCommand(['cube', 'pl""'])).toBe('s cube');
    });
});

describe('parseSearchCommand — restore path (verbatim from executeSearch)', () => {
    test('bare "s" yields no filters and all flags off', () => {
        const f = parseSearchCommand('s');
        expect(f.cmdFilters).toEqual([]);
        expect(f.nc).toBe(false);
        expect(f.dt).toBe(false);
        expect(f.dr).toBe(false);
        expect(f.mp).toBe(false);
        expect(f.pc).toBeUndefined();
        expect(f.matchIDs).toBe('');
        expect(f.tournamentIDs).toBe('');
    });

    test('flag tokens set their booleans', () => {
        const f = parseSearchCommand('s cube score nc d M');
        expect(f.ic).toBe(true);
        expect(f.is).toBe(true);
        expect(f.nc).toBe(true);
        expect(f.dt).toBe(true);
        expect(f.mp).toBe(true);
    });

    test('range tokens are captured for both players', () => {
        const f = parseSearchCommand('s p>5 w<70 e10,20 W>30 P12');
        expect(f.pc).toBe('p>5');
        expect(f.wr).toBe('w<70');
        expect(f.eq).toBe('e10,20');
        expect(f.p2wr).toBe('W>30');
        expect(f.p1apc).toBe('P12');
    });

    test('single-value checker tokens expand to a min,max pair', () => {
        expect(parseSearchCommand('s o5').p1co).toBe('o5,5');
        expect(parseSearchCommand('s K3').p2bc).toBe('K3,3');
        expect(parseSearchCommand('s z2').p1cz).toBe('z2,2');
    });

    test('checker tokens already carrying a range or operator are left intact', () => {
        expect(parseSearchCommand('s o3,4').p1co).toBe('o3,4');
        expect(parseSearchCommand('s o>5').p1co).toBe('o>5');
        expect(parseSearchCommand('s O<2').p2co).toBe('O<2');
    });

    test('quoted free-text filters survive embedded spaces', () => {
        const f = parseSearchCommand('s m"Best move here" t"some text" pl"John Doe"');
        expect(f.mpf).toBe('m"Best move here"');
        expect(f.st).toBe('t"some text"');
        expect(f.plf).toBe('pl"John Doe"');
    });

    test('pipcount p is not confused with the pl player filter', () => {
        const f = parseSearchCommand('s pl"Jane" p<60');
        expect(f.pc).toBe('p<60');
        expect(f.plf).toBe('pl"Jane"');
    });

    test('match and tournament id tokens join on ";"', () => {
        const f = parseSearchCommand('s ma1 ma2 ma7 tn3 tn4');
        expect(f.matchIDs).toBe('1;2;7');
        expect(f.tournamentIDs).toBe('3;4');
    });

    test('dice roll mode: D1 is first-only, D is both', () => {
        expect(parseSearchCommand('s D1').drMode).toBe('first');
        expect(parseSearchCommand('s D1').dr).toBe(true);
        expect(parseSearchCommand('s D').drMode).toBe('both');
        expect(parseSearchCommand('s D').dr).toBe(true);
    });

    test('backgammon b is distinguished from bo/bj outfield/jan blot tokens', () => {
        const f = parseSearchCommand('s b>10 bo5 bj2');
        expect(f.bg).toBe('b>10');
        expect(f.p1ob).toBe('bo5');
        expect(f.p1jb).toBe('bj2');
    });
});

describe('shared token classification (parseFilterTokens ↔ parseSearchCommand)', () => {
    test('both parsers pick the same token for every shared range/checker filter', () => {
        // Canonical tokens (operator/range form, so no comma-expansion divergence)
        // covering all 20 filters classified by the shared FILTER_TOKEN_MATCHERS.
        const toks = ['p<60', 'w>5', 'g3,8', 'b2,4', 'W>10', 'G1,5', 'B1,3', 'o3,4', 'O2,5', 'k1,2', 'K2,3', 'z2,3', 'Z1,4', 'P12', 'e10,20', 'T>2026/01/01', 'bo1,2', 'BO2,3', 'bj1,2', 'BJ1,2'];
        const ft = parseFilterTokens(toks);
        const sc = parseSearchCommand('s ' + toks.join(' '));
        const pairs = [
            ['pcFilter', 'pc'],
            ['wrFilter', 'wr'],
            ['grFilter', 'gr'],
            ['bgFilter', 'bg'],
            ['p2wrFilter', 'p2wr'],
            ['p2grFilter', 'p2gr'],
            ['p2bgFilter', 'p2bg'],
            ['p1coFilter', 'p1co'],
            ['p2coFilter', 'p2co'],
            ['p1bcFilter', 'p1bc'],
            ['p2bcFilter', 'p2bc'],
            ['p1czFilter', 'p1cz'],
            ['p2czFilter', 'p2cz'],
            ['p1apcFilter', 'p1apc'],
            ['eqFilter', 'eq'],
            ['cdFilter', 'cd'],
            ['p1obFilter', 'p1ob'],
            ['p2obFilter', 'p2ob'],
            ['p1jbFilter', 'p1jb'],
            ['p2jbFilter', 'p2jb']
        ];
        for (const [a, b] of pairs) {
            expect(sc[b]).toBe(ft[a]);
            expect(sc[b]).toBeTruthy(); // each token was actually classified
        }
    });
});
