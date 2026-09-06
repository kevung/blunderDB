import { emptySearchBoardPosition } from '../stores/searchExcludePositionStore.js';
import { NUMERIC_FILTERS, NUMERIC_FILTER_BY_LABEL, numericToken, readFlat } from './filterModel.js';

// searchFilterService — shared logic that turns the search UI's active filter
// labels + their option/min/max/range state into the backend command tokens
// (`cube`, `p>12`, `e10,50`, `t"foo"`, `T>2026/01/01`, …), and back.
//
// This was originally duplicated verbatim as a 31-case switch inside two search
// components (the now-removed SearchModal and the live SearchPanel). Extracting
// it removed the duplication and makes the mapping unit-testable.
//
// `options` is a flat object carrying the non-numeric fields the switch reads
// plus, for each numeric filter, its `<key>Option/Min/Max/RangeMin/RangeMax`
// fields (the shape `filterModel.toStore` produces). The numeric filters are
// no longer spelled out here: they come from the NUMERIC_FILTERS table.
// Missing fields simply produce `undefined` in the token, exactly as the
// original inline switch did.
//
// `parseSearchTokens` (below) is the single grammar that reads command tokens
// back into filter values. It used to be forked in two: `commandProcessor.js`
// had its own copy for the "aller" path (a command the user types or a saved
// search replayed straight through it, e.g. the Anki deck sync in
// `ankiService.js`), and this file had a second, less complete copy
// (`parseSearchCommand`) for the "retour" path (double-clicking a search
// history or filter-library entry in `SearchPanel.svelte`). The two diverged
// silently: `xD…` (exclude-dice), `id…` (position id) and the derived
// comment-presence mode were parsed on the aller path and dropped on the
// retour path, so replaying `s D xD65` from history brought the 6-5 roll
// back (#203). `parseFilters` (commandProcessor.js) and `parseSearchCommand`
// (below) are now both thin adapters over `parseSearchTokens`, keeping their
// existing return shapes (long field names / short abbreviated keys
// respectively) so no caller had to change — only the parsing itself is
// shared. `testdata/search_query_corpus.json` cross-checks both adapters
// against the same cases (shared with the future Go grammar, B.18).

/**
 * Map a list of active filter labels to their backend command tokens.
 * @param {string[]} activeFilters - the selected filter labels, in order.
 * @param {object} options - the option/min/max/range state fields.
 * @returns {string[]} one token per filter (empty string for unknown labels).
 */
export function buildFilterTokens(activeFilters, options) {
    const {
        diceRollOption,
        searchText,
        commentMode = 'contains',
        movePattern,
        creationDateOption,
        creationDateMin,
        creationDateMax,
        creationDateRangeMin,
        creationDateRangeMax,
        matchIDsSelected,
        tournamentIDsSelected,
        playerName
    } = options;

    return activeFilters.map((filter) => {
        const numeric = NUMERIC_FILTER_BY_LABEL[filter];
        if (numeric) return numericToken(numeric, readFlat(numeric, options));
        switch (filter) {
            case 'Include Cube':
                return 'cube';
            case 'Include Score':
                return 'score';
            case 'Include Decision Type':
                return 'd';
            case 'Include Dice Roll':
                return diceRollOption === 'first' ? 'D1' : 'D';
            case 'No Contact':
                return 'nc';
            case 'Mirror Position':
                return 'M';
            case 'Individually Imported':
                return 'i';
            case 'Flagged':
                return 'fl';
            // One filter, three modes: `t"…"` searches comment content, `co` /
            // `xco` ask only whether a comment is there at all. The modes are
            // mutually exclusive here, so the token is never ambiguous.
            case 'Comment':
                return commentMode === 'has' ? 'co' : commentMode === 'none' ? 'xco' : `t"${searchText}"`;
            case 'Player':
                return `pl"${playerName}"`;
            case 'Best Move or Cube Decision':
                return `m"${movePattern}"`;
            case 'Creation Date': {
                const formatDate = (date) => date.replace(/-/g, '/'); // Convert date format to yyyy/mm/dd
                return creationDateOption === 'min'
                    ? `T>${formatDate(creationDateMin)}`
                    : creationDateOption === 'max'
                      ? `T<${formatDate(creationDateMax)}`
                      : `T${formatDate(creationDateRangeMin)},${formatDate(creationDateRangeMax)}`;
            }
            case 'Match IDs':
                return matchIDsSelected && matchIDsSelected.length ? `ma${matchIDsSelected.join(';')}` : '';
            case 'Tournament IDs':
                return tournamentIDsSelected && tournamentIDsSelected.length ? `tn${tournamentIDsSelected.join(';')}` : '';
            default:
                return '';
        }
    });
}

/**
 * Assemble the `s …` search command string from filter tokens, dropping the
 * empty text/move placeholders exactly as the inline code did.
 * @param {string[]} tokens
 * @returns {string}
 */
export function buildSearchCommand(tokens) {
    const commandParts = ['s'];
    tokens.forEach((token) => {
        if (token !== 't""' && token !== 'm""' && token !== 'pl""') {
            commandParts.push(token);
        }
    });
    return commandParts.join(' ');
}

// Quoted filter values — pl"…" (player), m"…" (move pattern) and t"…" (search
// text / comment) — may contain spaces. A naive whitespace split tears a
// multi-word value into loose words (`t"big win"` → `t"big`, `win"`) and those
// words get misclassified as range filters (`win"` → win-rate, the bare `b` →
// backgammon-rate), silently corrupting the result set. Strip the whole quoted
// region before splitting so no interior word survives, on both the aller and
// retour paths. Both quote styles are supported. Moved here (from
// `commandProcessor.js`, which re-exports it) so `parseSearchTokens` below can
// tokenize a raw command the same way regardless of caller.
export function stripQuotedTokens(str) {
    return str.replace(/(?:pl|m|t)["'][^"']*["']/g, ' ');
}

/**
 * The single grammar behind every search-token parser in the app: reads
 * filter tokens back into the full `SearchFilters` shape (long field names,
 * matching what `onLoadPositionsByFilters` / `buildSearchFilterPayload`
 * expect). Accepts either an already-split token array (paired with the
 * source command, needed to recover quoted values) or a bare command string,
 * from which tokens are derived the same way `stripQuotedTokens` +
 * whitespace-split does everywhere else.
 *
 * `parseFilters` (`commandProcessor.js`, the "aller" path: a typed or
 * replayed-verbatim command) and `parseSearchCommand` (below, the "retour"
 * path: SearchPanel replaying a history/library entry) are both thin adapters
 * over this function — see the module doc comment for why that split existed
 * and what it silently dropped (#203).
 *
 * @param {string[]|string} filtersOrCommand - filter tokens (no leading `s`), or the full command.
 * @param {string} [command] - the raw command, used to recover quoted values; required when
 *        `filtersOrCommand` is already a token array, derived automatically otherwise.
 * @returns {object} the parsed filter values, under their long (backend) field names.
 */
export function parseSearchTokens(filtersOrCommand, command) {
    let filters;
    let cmd;
    if (Array.isArray(filtersOrCommand)) {
        filters = filtersOrCommand;
        cmd = command ?? '';
    } else {
        cmd = filtersOrCommand ?? '';
        filters =
            cmd === 's' || cmd === ''
                ? []
                : stripQuotedTokens(cmd.slice(2).trim())
                      .split(' ')
                      .map((f) => f.trim());
    }

    const includeCube = filters.includes('cube') || filters.includes('cu') || filters.includes('c') || filters.includes('cub');
    const includeScore = filters.includes('score') || filters.includes('sco') || filters.includes('sc') || filters.includes('s');
    const noContactFilter = filters.includes('nc');
    const decisionTypeFilter = filters.includes('d');
    const diceRollFilter = filters.includes('D') || filters.includes('D1');
    const diceRollMode = filters.includes('D1') ? 'first' : 'both';
    // `xD65` excludes the 6-5 roll (order-insensitive); repeatable (`xD65 xD54`).
    // Unlike `D`, the value is inline in the token, not read from the board.
    // Joined into a ";"-separated string for the backend (ExceptDiceFilter).
    const exceptDiceFilter = filters
        .filter((f) => typeof f === 'string' && /^xD[1-6][1-6]$/.test(f))
        .map((f) => f.slice(2))
        .join(';');
    const mirrorPositionFilter = filters.includes('M');
    // Positions the user imported on their own rather than inside a match.
    // An exact match, so it does not collide with the id<ids> token.
    const individuallyImportedFilter = filters.includes('i');
    // Positions the user marked for study in the tool the match came from
    // (eXtreme Gammon flags). An exact match, like 'i'.
    const flaggedFilter = filters.includes('fl');
    // 'x' marks that an exclusion ("Sauf") structure is active. The structure
    // itself is carried by the exclude board (store), like the include structure.
    const excludeStructure = filters.includes('x');
    // Comment presence: `co` (has one) / `xco` (has none). Exact matches, so
    // they collide neither with each other nor with the `co` alias of the
    // `comment` command — filter tokens only exist after the `s ` prefix.
    // Asking for both is contradictory rather than ambiguous; 'none' wins and
    // the search comes back empty, which is the honest answer.
    const commentFilter = filters.includes('xco') ? 'none' : filters.includes('co') ? 'has' : '';
    // Derived game phase: `ph:race`, repeatable (`ph:race ph:bearoff`), joined
    // into a ";"-separated string for the backend (GamePhaseFilter, ADR-0035).
    // A closed vocabulary, so the token names its value rather than quoting it.
    const gamePhaseFilter = filters
        .filter((f) => typeof f === 'string' && /^ph:[a-z]+$/.test(f))
        .map((f) => f.slice(3))
        .join(';');
    // Comment provenance: `co:user`, repeatable, joined the same way
    // (CommentOriginFilter, #263). Distinct from the bare `co` above, which
    // asks about presence only — an exact match, so the two never collide.
    const commentOriginFilter = filters
        .filter((f) => typeof f === 'string' && /^co:[a-z]+$/.test(f))
        .map((f) => f.slice(3))
        .join(';');
    // Tags: `#prime`, repeatable, joined the same way (TagFilter, #265). A tag
    // names itself, so there is no letter prefix to strip and nothing else can
    // claim the token. Several tags narrow TOGETHER — a position has many
    // tags, so naming two means "both", unlike the two closed lists above
    // where naming two can only mean "either".
    const tagFilter = filters
        .filter((f) => typeof f === 'string' && /^#[^\s#]+$/.test(f))
        .map((f) => f.toLowerCase())
        .join(';');
    // Exclude `pl"…"` (player filter) and `ph:…` (phase) — both start with 'p'
    // and neither is a pipcount.
    const pipCountFilter = filters.find((f) => typeof f === 'string' && !f.startsWith('pl') && !f.startsWith('ph') && (f.startsWith('p>') || f.startsWith('p<') || f.startsWith('p')));
    const winRateFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('w>') || f.startsWith('w<') || f.startsWith('w')));
    const gammonRateFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('g>') || f.startsWith('g<') || f.startsWith('g')));
    const backgammonRateFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('b>') || f.startsWith('b<') || (f.startsWith('b') && !f.startsWith('bo'))) && !f.startsWith('bj'));
    const player2WinRateFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('W>') || f.startsWith('W<') || f.startsWith('W')));
    const player2GammonRateFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('G>') || f.startsWith('G<') || f.startsWith('G')));
    const player2BackgammonRateFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('B>') || f.startsWith('B<') || (f.startsWith('B') && !f.startsWith('BO'))) && !f.startsWith('BJ'));
    let player1CheckerOffFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('o>') || f.startsWith('o<') || f.startsWith('o')));
    if (player1CheckerOffFilter && !player1CheckerOffFilter.includes(',') && !player1CheckerOffFilter.includes('>') && !player1CheckerOffFilter.includes('<')) {
        player1CheckerOffFilter = `${player1CheckerOffFilter},${player1CheckerOffFilter.slice(1)}`;
    }
    let player2CheckerOffFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('O>') || f.startsWith('O<') || f.startsWith('O')));
    if (player2CheckerOffFilter && !player2CheckerOffFilter.includes(',') && !player2CheckerOffFilter.includes('>') && !player2CheckerOffFilter.includes('<')) {
        player2CheckerOffFilter = `${player2CheckerOffFilter},${player2CheckerOffFilter.slice(1)}`;
    }
    let player1BackCheckerFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('k>') || f.startsWith('k<') || f.startsWith('k')));
    if (player1BackCheckerFilter && !player1BackCheckerFilter.includes(',') && !player1BackCheckerFilter.includes('>') && !player1BackCheckerFilter.includes('<')) {
        player1BackCheckerFilter = `${player1BackCheckerFilter},${player1BackCheckerFilter.slice(1)}`;
    }
    let player2BackCheckerFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('K>') || f.startsWith('K<') || f.startsWith('K')));
    if (player2BackCheckerFilter && !player2BackCheckerFilter.includes(',') && !player2BackCheckerFilter.includes('>') && !player2BackCheckerFilter.includes('<')) {
        player2BackCheckerFilter = `${player2BackCheckerFilter},${player2BackCheckerFilter.slice(1)}`;
    }
    let player1CheckerInZoneFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('z>') || f.startsWith('z<') || f.startsWith('z')));
    if (player1CheckerInZoneFilter && !player1CheckerInZoneFilter.includes(',') && !player1CheckerInZoneFilter.includes('>') && !player1CheckerInZoneFilter.includes('<')) {
        player1CheckerInZoneFilter = `${player1CheckerInZoneFilter},${player1CheckerInZoneFilter.slice(1)}`;
    }
    let player2CheckerInZoneFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('Z>') || f.startsWith('Z<') || f.startsWith('Z')));
    if (player2CheckerInZoneFilter && !player2CheckerInZoneFilter.includes(',') && !player2CheckerInZoneFilter.includes('>') && !player2CheckerInZoneFilter.includes('<')) {
        player2CheckerInZoneFilter = `${player2CheckerInZoneFilter},${player2CheckerInZoneFilter.slice(1)}`;
    }
    const player1AbsolutePipCountFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('P>') || f.startsWith('P<') || f.startsWith('P')));
    const equityFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('e>') || f.startsWith('e<') || f.startsWith('e')));
    const dateFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('T>') || f.startsWith('T<') || f.startsWith('T')));
    const movePatternMatch = cmd.match(/m["'][^"']*["']/);
    const movePatternFilter = movePatternMatch ? movePatternMatch[0] : '';
    const searchTextMatch = cmd.match(/t["'][^"']*["']/);
    const searchText = searchTextMatch ? searchTextMatch[0] : '';
    // Player filter `pl"Name"` — matched on the raw command so names with spaces
    // survive (the space-split `filters` array would break them).
    const playerMatch = cmd.match(/pl["'][^"']*["']/);
    const playerFilter = playerMatch ? playerMatch[0] : '';
    const player1OutfieldBlotFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('bo>') || f.startsWith('bo<') || f.startsWith('bo')));
    const player2OutfieldBlotFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('BO>') || f.startsWith('BO<') || f.startsWith('BO')));
    const player1JanBlotFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('bj>') || f.startsWith('bj<') || f.startsWith('bj')));
    const player2JanBlotFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('BJ>') || f.startsWith('BJ<') || f.startsWith('BJ')));
    const moveErrorFilter = filters.find((f) => typeof f === 'string' && (f.startsWith('E>') || f.startsWith('E<') || (f.startsWith('E') && /^E\d/.test(f))));

    const matchIDTokens = filters.filter((f) => typeof f === 'string' && /^ma\d/.test(f));
    let matchIDsFilter = '';
    if (matchIDTokens.length > 0) {
        const parts = matchIDTokens.map((token) => token.slice(2));
        matchIDsFilter = parts.join(';');
    }

    const tournamentIDTokens = filters.filter((f) => typeof f === 'string' && /^tn\d/.test(f));
    let tournamentIDsFilter = '';
    if (tournamentIDTokens.length > 0) {
        const parts = tournamentIDTokens.map((token) => token.slice(2));
        tournamentIDsFilter = parts.join(';');
    }

    // Position-id filter: `id12`, `id5,10` (range 5..10), or several `id` tokens
    // joined as an explicit list (e.g. `id5 id10`). Mirrors the ma/tn convention.
    const positionIDTokens = filters.filter((f) => typeof f === 'string' && /^id\d/.test(f));
    let positionIDsFilter = '';
    if (positionIDTokens.length > 0) {
        const parts = positionIDTokens.map((token) => token.slice(2));
        positionIDsFilter = parts.join(';');
    }

    return {
        tokens: filters,
        includeCube,
        includeScore,
        noContactFilter,
        decisionTypeFilter,
        diceRollFilter,
        diceRollMode,
        exceptDiceFilter,
        mirrorPositionFilter,
        individuallyImportedFilter,
        flaggedFilter,
        excludeStructure,
        pipCountFilter,
        winRateFilter,
        gammonRateFilter,
        backgammonRateFilter,
        player2WinRateFilter,
        player2GammonRateFilter,
        player2BackgammonRateFilter,
        player1CheckerOffFilter,
        player2CheckerOffFilter,
        player1BackCheckerFilter,
        player2BackCheckerFilter,
        player1CheckerInZoneFilter,
        player2CheckerInZoneFilter,
        player1AbsolutePipCountFilter,
        equityFilter,
        dateFilter,
        movePatternFilter,
        searchText,
        commentFilter,
        commentOriginFilter,
        gamePhaseFilter,
        tagFilter,
        player1OutfieldBlotFilter,
        player2OutfieldBlotFilter,
        player1JanBlotFilter,
        player2JanBlotFilter,
        moveErrorFilter,
        matchIDsFilter,
        tournamentIDsFilter,
        playerFilter,
        positionIDsFilter
    };
}

/**
 * Pick the individual backend filter arguments back out of a freshly built
 * token list (the "save" path: SearchPanel's checkboxes → `buildFilterTokens`
 * → this function → `onLoadPositionsByFilters`). A thin adapter over
 * {@link parseSearchTokens}, reporting the same fields under the short
 * abbreviated keys this call site (and its tests) have always used.
 *
 * @param {string[]} tokens - the output of {@link buildFilterTokens}.
 * @returns {object} the named filter arguments consumed by onLoadPositionsByFilters.
 */
export function parseFilterTokens(tokens) {
    // Quoted values (pl"…"/m"…"/t"…") are recovered by parseSearchTokens from the
    // raw command text, not the split tokens — rebuild a command-shaped string
    // so a freshly built pl"Name"/m"…"/t"…" token is still found.
    const p = parseSearchTokens(tokens, tokens.join(' '));
    return {
        incCube: p.includeCube,
        incScore: p.includeScore,
        ncFilter: p.noContactFilter,
        mirFilter: p.mirrorPositionFilter,
        iiFilter: p.individuallyImportedFilter,
        flFilter: p.flaggedFilter,
        pcFilter: p.pipCountFilter,
        plFilter: p.playerFilter || undefined,
        wrFilter: p.winRateFilter,
        grFilter: p.gammonRateFilter,
        bgFilter: p.backgammonRateFilter,
        p2wrFilter: p.player2WinRateFilter,
        p2grFilter: p.player2GammonRateFilter,
        p2bgFilter: p.player2BackgammonRateFilter,
        p1coFilter: p.player1CheckerOffFilter,
        p2coFilter: p.player2CheckerOffFilter,
        p1bcFilter: p.player1BackCheckerFilter,
        p2bcFilter: p.player2BackCheckerFilter,
        p1czFilter: p.player1CheckerInZoneFilter,
        p2czFilter: p.player2CheckerInZoneFilter,
        p1apcFilter: p.player1AbsolutePipCountFilter,
        eqFilter: p.equityFilter,
        meFilter: p.moveErrorFilter,
        p1obFilter: p.player1OutfieldBlotFilter,
        p2obFilter: p.player2OutfieldBlotFilter,
        p1jbFilter: p.player1JanBlotFilter,
        p2jbFilter: p.player2JanBlotFilter,
        matchIDs: p.matchIDsFilter,
        tournamentIDs: p.tournamentIDsFilter,
        xdFilter: p.exceptDiceFilter,
        posIdsFilter: p.positionIDsFilter,
        phFilter: p.gamePhaseFilter,
        coOriginFilter: p.commentOriginFilter,
        tagFilter: p.tagFilter,
        dtFilter: p.decisionTypeFilter,
        drFilter: p.diceRollFilter,
        drMode: p.diceRollMode,
        // Comment-presence mode; 'contains' is the text-search mode, whose value
        // travels separately as the t"…" token.
        commentMode: p.commentFilter === 'none' ? 'none' : p.commentFilter === 'has' ? 'has' : 'contains',
        cdFilter: p.dateFilter
    };
}

/**
 * Parse a persisted `s …` search command string back into the flat set of
 * filter values SearchPanel hands to its `onLoadPositionsByFilters` callback
 * when replaying a saved/library search (the "retour" path). A thin adapter
 * over {@link parseSearchTokens}, reporting the same fields under the short
 * abbreviated keys this call site (and its tests) have always used — plus
 * `xd`/`posIds`/`commentMode`, the three fields the pre-#203 version of this
 * function silently dropped because it parsed the command on its own instead
 * of sharing `parseFilters`' (commandProcessor.js) grammar.
 *
 * @param {string} command - a command starting with `s ` (or the bare `s`).
 * @returns {object} the parsed filter values, keyed by short name.
 */
export function parseSearchCommand(command) {
    const p = parseSearchTokens(command);
    return {
        cmdFilters: p.tokens,
        ic: p.includeCube,
        is: p.includeScore,
        nc: p.noContactFilter,
        dt: p.decisionTypeFilter,
        dr: p.diceRollFilter,
        drMode: p.diceRollMode,
        mp: p.mirrorPositionFilter,
        ii: p.individuallyImportedFilter,
        fl: p.flaggedFilter,
        pc: p.pipCountFilter,
        wr: p.winRateFilter,
        gr: p.gammonRateFilter,
        bg: p.backgammonRateFilter,
        p2wr: p.player2WinRateFilter,
        p2gr: p.player2GammonRateFilter,
        p2bg: p.player2BackgammonRateFilter,
        p1co: p.player1CheckerOffFilter,
        p2co: p.player2CheckerOffFilter,
        p1bc: p.player1BackCheckerFilter,
        p2bc: p.player2BackCheckerFilter,
        p1cz: p.player1CheckerInZoneFilter,
        p2cz: p.player2CheckerInZoneFilter,
        p1apc: p.player1AbsolutePipCountFilter,
        eq: p.equityFilter,
        cd: p.dateFilter,
        mpf: p.movePatternFilter,
        st: p.searchText,
        plf: p.playerFilter,
        p1ob: p.player1OutfieldBlotFilter,
        p2ob: p.player2OutfieldBlotFilter,
        p1jb: p.player1JanBlotFilter,
        p2jb: p.player2JanBlotFilter,
        me: p.moveErrorFilter,
        matchIDs: p.matchIDsFilter,
        tournamentIDs: p.tournamentIDsFilter,
        // Previously dropped on replay (#203): no panel checkbox drives these
        // (documented as command-line only, cmd_mode.rst), so a saved/history
        // command carrying them silently lost them here.
        xd: p.exceptDiceFilter,
        posIds: p.positionIDsFilter,
        ph: p.gamePhaseFilter,
        coOrigin: p.commentOriginFilter,
        tags: p.tagFilter,
        commentMode: p.commentFilter === 'none' ? 'none' : p.commentFilter === 'has' ? 'has' : 'contains'
    };
}

// Command-line token for each search filter, keyed by its canonical (English)
// label — the same labels SearchPanel's filterGroups use. Single source of
// truth for the in-UI token hint shown on hover; the range entries come from
// the NUMERIC_FILTERS table, the others mirror the buildFilterTokens switch
// above. `type` drives how filterTokenHint renders the usage forms:
//   flag  — the bare token (cube, nc, M, d)
//   range — three forms: X>n, X<n, Xn,m
//   text  — quoted free text: t"…"
//   date  — T>YYYY/MM/DD …
//   dice  — D (both rolls) / D1 (first roll only)
const FILTER_TOKENS = {
    'Include Cube': { token: 'cube', type: 'flag' },
    'Include Score': { token: 'score', type: 'flag' },
    'Include Decision Type': { token: 'd', type: 'flag' },
    'Include Dice Roll': { token: 'D', type: 'dice' },
    'No Contact': { token: 'nc', type: 'flag' },
    'Mirror Position': { token: 'M', type: 'flag' },
    'Individually Imported': { token: 'i', type: 'flag' },
    Flagged: { token: 'fl', type: 'flag' },
    ...Object.fromEntries(NUMERIC_FILTERS.map((f) => [f.label, { token: f.token, type: 'range' }])),
    // Three modes, so the hint spells all three rather than the `t"…"` form alone.
    Comment: { token: 't', type: 'comment' },
    'Best Move or Cube Decision': { token: 'm', type: 'text' },
    Player: { token: 'pl', type: 'text' },
    'Creation Date': { token: 'T', type: 'date' }
};

/**
 * The command-line token hint for a filter label, shown as the filter's `title`
 * (hover tooltip) in SearchPanel so the cryptic `s` tokens are discoverable
 * without leaving the UI. Returns '' for unknown labels. The string is
 * deliberately word-free — only the token and its operator forms — so it needs
 * no translation.
 *
 * @param {string} label - the canonical (English) filter label.
 * @returns {string}
 */
export function filterTokenHint(label) {
    const entry = FILTER_TOKENS[label];
    if (!entry) return '';
    const { token, type } = entry;
    switch (type) {
        case 'range':
            return `${token}>n · ${token}<n · ${token}n,m`;
        case 'text':
            return `${token}"…"`;
        case 'comment':
            return `${token}"…" · co · xco`;
        case 'date':
            return `${token}>YYYY/MM/DD · ${token}<YYYY/MM/DD`;
        case 'dice':
            return `${token} · ${token}1`;
        case 'flag':
        default:
            return token;
    }
}

/**
 * Build the SearchFilters object the LoadPositionsByFilters binding expects.
 *
 * The binding takes exactly one argument, so every filter has to travel inside a
 * single object. Passing them positionally — as AnkiPanel did — leaves all of
 * them behind and hands the backend a Position where it expects a SearchFilters:
 * that deserialises to an all-zero struct, i.e. no filter at all, and the search
 * answers with the whole database (#111).
 *
 * `position` is sent as-is. Callers pass the board recorded in lastSearchStore,
 * which positionService already normalised and mirrored; normalising it a second
 * time would flip an already-mirrored board.
 *
 * @param {object} position - the search board, already normalised.
 * @param {object} [pf] - parsed filter flags, as returned by `parseFilters`.
 * @param {string[]} [filters] - raw filter tokens, for the token-derived fields.
 * @returns {object} the SearchFilters payload.
 */
export function buildSearchFilterPayload(position, pf = {}, filters = []) {
    const tokens = Array.isArray(filters) ? filters : [];
    return {
        filter: position,
        excludeFilter: emptySearchBoardPosition(),
        includeCube: pf.includeCube || false,
        includeScore: pf.includeScore || false,
        pipCountFilter: pf.pipCountFilter || '',
        winRateFilter: pf.winRateFilter || '',
        gammonRateFilter: pf.gammonRateFilter || '',
        backgammonRateFilter: pf.backgammonRateFilter || '',
        player2WinRateFilter: pf.player2WinRateFilter || '',
        player2GammonRateFilter: pf.player2GammonRateFilter || '',
        player2BackgammonRateFilter: pf.player2BackgammonRateFilter || '',
        player1CheckerOffFilter: pf.player1CheckerOffFilter || '',
        player2CheckerOffFilter: pf.player2CheckerOffFilter || '',
        player1BackCheckerFilter: pf.player1BackCheckerFilter || '',
        player2BackCheckerFilter: pf.player2BackCheckerFilter || '',
        player1CheckerInZoneFilter: pf.player1CheckerInZoneFilter || '',
        player2CheckerInZoneFilter: pf.player2CheckerInZoneFilter || '',
        searchText: pf.searchText || '',
        commentFilter: pf.commentFilter || '',
        commentOriginFilter: pf.commentOriginFilter || '',
        gamePhaseFilter: pf.gamePhaseFilter || '',
        tagFilter: pf.tagFilter || '',
        player1AbsolutePipCountFilter: pf.player1AbsolutePipCountFilter || '',
        equityFilter: pf.equityFilter || '',
        decisionTypeFilter: pf.decisionTypeFilter || false,
        // Derived from the tokens, exactly as positionService does.
        cubeResponseFilter: tokens.includes('dr') ? 'takepass' : tokens.includes('dd') ? 'double' : '',
        diceRollFilter: pf.diceRollFilter || false,
        diceRollMode: pf.diceRollMode || 'both',
        exceptDiceFilter: pf.exceptDiceFilter || '',
        movePatternFilter: pf.movePatternFilter || '',
        dateFilter: pf.dateFilter || '',
        player1OutfieldBlotFilter: pf.player1OutfieldBlotFilter || '',
        player2OutfieldBlotFilter: pf.player2OutfieldBlotFilter || '',
        player1JanBlotFilter: pf.player1JanBlotFilter || '',
        player2JanBlotFilter: pf.player2JanBlotFilter || '',
        noContactFilter: pf.noContactFilter || false,
        mirrorFilter: pf.mirrorPositionFilter || false,
        individuallyImportedFilter: pf.individuallyImportedFilter || false,
        flaggedFilter: pf.flaggedFilter || false,
        moveErrorFilter: pf.moveErrorFilter || '',
        matchIDsFilter: pf.matchIDsFilter || '',
        tournamentIDsFilter: pf.tournamentIDsFilter || '',
        playerFilter: pf.playerFilter || '',
        positionIDsFilter: pf.positionIDsFilter || '',
        restrictToPositionIDs: ''
    };
}
