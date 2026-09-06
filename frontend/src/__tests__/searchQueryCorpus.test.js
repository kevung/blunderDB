/**
 * searchQueryCorpus.test.js — #203
 *
 * Runs testdata/search_query_corpus.json (shared with the future Go grammar,
 * B.18) through every JS entry point of the search-token grammar and checks
 * they all agree:
 *   - the "aller" path (commandProcessor.js's parseFilters, fed tokens the
 *     same way handleSearchCommand derives them from a typed command),
 *   - the "retour" path (searchFilterService.js's parseSearchCommand, used
 *     when SearchPanel replays a history/library entry), and
 *   - the shared grammar itself (parseSearchTokens), called directly with the
 *     bare command string.
 *
 * Before #203 the aller and retour paths were two separately maintained
 * copies of this grammar and had drifted: `xD…`, `id…` and the comment
 * presence derived from `co`/`xco` were parsed on one and silently dropped on
 * the other. Running one corpus through all three closes that gap for good —
 * a future regression on any one of them fails here.
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
const corpus = JSON.parse(fs.readFileSync(path.join(__dirname, '../../../testdata/search_query_corpus.json'), 'utf8'));

// Mirrors handleSearchCommand's own tokenizing of a typed `s …` command
// (commandProcessor.js), so the "aller" check below exercises parseFilters
// exactly the way the app really calls it.
function tokenizeAller(command) {
    if (command === 's') return [];
    return stripQuotedTokens(command.slice(1).trim())
        .split(' ')
        .map((f) => f.trim());
}

// Short-key (parseSearchCommand) → long-key (parseSearchTokens/parseFilters)
// field name, so the same `expected` object can check both.
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
    coOrigin: 'commentOriginFilter',
    tags: 'tagFilter'
};
const LONG_TO_SHORT = Object.fromEntries(Object.entries(SHORT_TO_LONG).map(([short, long]) => [long, short]));

describe('search_query_corpus — grammar cross-check', () => {
    test('the corpus has cases to run', () => {
        expect(Array.isArray(corpus.cases)).toBe(true);
        expect(corpus.cases.length).toBeGreaterThan(10);
    });

    test.each(corpus.cases.map((c) => [c.command, c]))('parseSearchTokens("%s") matches its expected fields', (command, { expected }) => {
        const result = parseSearchTokens(command);
        for (const [field, value] of Object.entries(expected)) {
            expect(result[field], `${command} → ${field}`).toBe(value);
        }
    });

    test.each(corpus.cases.map((c) => [c.command, c]))('aller (parseFilters) matches parseSearchTokens for "%s"', (command, { expected }) => {
        const result = parseFilters(tokenizeAller(command), command);
        for (const [field, value] of Object.entries(expected)) {
            expect(result[field], `${command} → ${field}`).toBe(value);
        }
    });

    test.each(corpus.cases.map((c) => [c.command, c]))('retour (parseSearchCommand) matches, under its short keys, for "%s"', (command, { expected }) => {
        const result = parseSearchCommand(command);
        for (const [field, value] of Object.entries(expected)) {
            // `commentFilter` ('' / 'has' / 'none') restores under parseSearchCommand's
            // `commentMode` ('contains' / 'has' / 'none') — different empty spelling,
            // same fields checked in commandProcessor.test.js / searchFilterService.test.js.
            if (field === 'commentFilter') {
                expect(result.commentMode, `${command} → commentMode`).toBe(value === 'has' ? 'has' : value === 'none' ? 'none' : 'contains');
                continue;
            }
            // `excludeStructure` ('x') has no short-key equivalent: SearchPanel's
            // retour path restores the exclude ("Sauf") board straight from the
            // stored excludePosition (restoreExcludeStructure), never by
            // re-deriving it from the 'x' token — nothing to cross-check here.
            if (field === 'excludeStructure') continue;
            const shortKey = LONG_TO_SHORT[field];
            expect(shortKey, `no short-key mapping declared for ${field} — add one to SHORT_TO_LONG`).toBeDefined();
            expect(result[shortKey], `${command} → ${shortKey} (${field})`).toBe(value);
        }
    });
});
