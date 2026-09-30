/**
 * searchQuery.property.test.js — the JS search grammar on generated commands.
 *
 * The corpus test checks known commands; this one builds thousands of commands
 * from the corpus's own tokens, mutated (a character dropped, doubled or
 * swapped for a quote, a digit, a separator), and holds the three JS readers
 * to what the corpus only samples:
 *   - none of them throws, whatever the user typed;
 *   - the typed-command path (parseFilters, fed as handleSearchCommand feeds
 *     it) and the shared grammar (parseSearchTokens) agree on every field;
 *   - the replay path (parseSearchCommand) agrees with them on every field
 *     it reports.
 * The generator is seeded, so a failure names a command that reproduces.
 */

import { describe, test, expect, vi } from 'vitest';
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    SaveComment: vi.fn(),
    ClearCommandHistory: vi.fn(),
    SaveSearchHistory: vi.fn()
}));

import { parseFilters, stripQuotedTokens } from '../commandProcessor.js';
import { parseSearchTokens, parseSearchCommand } from '../services/searchFilterService.js';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
/** @type {{ cases: { command: string }[] }} */
const corpus = JSON.parse(fs.readFileSync(path.join(__dirname, '../../../testdata/search_query_corpus.json'), 'utf8'));

/** @param {string} command */
function tokenizeAller(command) {
    if (command === 's') return [];
    return stripQuotedTokens(command.slice(1).trim())
        .split(' ')
        .map((f) => f.trim());
}

/** mulberry32: a small seeded PRNG, so a failing command reproduces. */
/** @param {number} seed */
function prng(seed) {
    let a = seed >>> 0;
    return () => {
        a = (a + 0x6d2b79f5) >>> 0;
        let t = a;
        t = Math.imul(t ^ (t >>> 15), t | 1);
        t ^= t + Math.imul(t ^ (t >>> 7), t | 61);
        return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
    };
}

const tokens = [...new Set(corpus.cases.flatMap((c) => c.command.replace(/^ss? /, '').split(' ')).filter((t) => t && t !== 's' && t !== 'ss'))];
const noise = ['"', "'", '<', '>', ',', ';', '*', '#', '-', '.', '0', '9', '99999999999999999999', 'x', 'é'];

/**
 * @param {() => number} rnd
 * @param {string} tok
 */
function mutate(rnd, tok) {
    const i = Math.floor(rnd() * (tok.length + 1));
    const n = noise[Math.floor(rnd() * noise.length)];
    switch (Math.floor(rnd() * 4)) {
        case 0:
            return tok;
        case 1:
            return tok.slice(0, i) + tok.slice(i + 1);
        case 2:
            return tok.slice(0, i) + n + tok.slice(i);
        default:
            return tok + tok.slice(i);
    }
}

/** @param {() => number} rnd */
function command(rnd) {
    const count = Math.floor(rnd() * 6);
    const parts = [];
    for (let k = 0; k < count; k++) {
        parts.push(mutate(rnd, tokens[Math.floor(rnd() * tokens.length)]));
    }
    return ['s', ...parts].join(' ');
}

// Short-key (parseSearchCommand) → long-key names, as in the corpus test.
/** @type {Record<string, string>} */
const SHORT_TO_LONG = {
    ic: 'includeCube',
    is: 'includeScore',
    nc: 'noContactFilter',
    dt: 'decisionTypeFilter',
    dr: 'diceRollFilter',
    drMode: 'diceRollMode',
    mp: 'mirrorPositionFilter',
    ii: 'individuallyImportedFilter',
    fl: 'flaggedFilter',
    pc: 'pipCountFilter',
    wr: 'winRateFilter',
    gr: 'gammonRateFilter',
    bg: 'backgammonRateFilter',
    p2wr: 'player2WinRateFilter',
    p2gr: 'player2GammonRateFilter',
    p2bg: 'player2BackgammonRateFilter',
    p1co: 'player1CheckerOffFilter',
    p2co: 'player2CheckerOffFilter',
    p1bc: 'player1BackCheckerFilter',
    p2bc: 'player2BackCheckerFilter',
    p1cz: 'player1CheckerInZoneFilter',
    p2cz: 'player2CheckerInZoneFilter',
    p1apc: 'player1AbsolutePipCountFilter',
    eq: 'equityFilter',
    cd: 'dateFilter',
    mpf: 'movePatternFilter',
    st: 'searchText',
    plf: 'playerFilter',
    p1ob: 'player1OutfieldBlotFilter',
    p2ob: 'player2OutfieldBlotFilter',
    p1jb: 'player1JanBlotFilter',
    p2jb: 'player2JanBlotFilter',
    me: 'moveErrorFilter',
    matchIDs: 'matchIDsFilter',
    tournamentIDs: 'tournamentIDsFilter',
    xd: 'exceptDiceFilter',
    posIds: 'positionIDsFilter',
    ph: 'gamePhaseFilter',
    gt: 'gameTypeFilter',
    coOrigin: 'commentOriginFilter',
    tags: 'tagFilter'
};

const RUNS = 500;

describe('search grammar — generated commands', () => {
    test('the generator has tokens to draw from', () => {
        expect(tokens.length).toBeGreaterThan(20);
    });

    test(`${RUNS} generated commands: no reader throws, and all three agree`, () => {
        const rnd = prng(0x5eed);
        for (let run = 0; run < RUNS; run++) {
            const cmd = command(rnd);
            /** @type {Record<string, unknown>} */
            let shared;
            /** @type {Record<string, unknown>} */
            let aller;
            /** @type {Record<string, unknown>} */
            let retour;
            try {
                shared = /** @type {Record<string, unknown>} */ (parseSearchTokens(cmd));
                aller = /** @type {Record<string, unknown>} */ (parseFilters(tokenizeAller(cmd), cmd));
                retour = /** @type {Record<string, unknown>} */ (parseSearchCommand(cmd));
            } catch (e) {
                throw new Error(`a reader threw on ${JSON.stringify(cmd)}: ${e}`, { cause: e });
            }
            for (const [field, value] of Object.entries(shared)) {
                expect(aller[field], `aller vs shared on ${JSON.stringify(cmd)} → ${field}`).toStrictEqual(value);
            }
            for (const [short, long] of Object.entries(SHORT_TO_LONG)) {
                if (!(short in retour)) continue;
                expect(retour[short], `retour vs shared on ${JSON.stringify(cmd)} → ${long}`).toStrictEqual(shared[long]);
            }
        }
    }, 30000);

    test('a saved search with an apostrophe tag stays valid, and a stray quote token claims nothing', () => {
        const saved = /** @type {Record<string, unknown>} */ (parseSearchTokens("s cube #l'ouverture"));
        expect(saved.includeCube).toBe(true);
        expect(saved.tagFilter).toBe("#l'ouverture");
        const stray = /** @type {Record<string, unknown>} */ (parseSearchTokens('s Bt" id0;'));
        expect(stray.player2BackgammonFilter ?? stray.player2BackgammonRateFilter).toBeUndefined();
        expect(stray.positionIDsFilter).toBe('');
    });
});
