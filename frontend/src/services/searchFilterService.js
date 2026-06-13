// searchFilterService — shared logic that turns the search UI's active filter
// labels + their option/min/max/range state into the backend command tokens
// (`cube`, `p>12`, `e10,50`, `t"foo"`, `T>2026/01/01`, …).
//
// This was originally duplicated verbatim as a 31-case switch inside two search
// components (the now-removed SearchModal and the live SearchPanel). Extracting
// it removed the duplication and makes the mapping unit-testable. It is the inverse of
// `parseFilters()` in `commandProcessor.js` (which parses tokens back into
// filter flags), so the two are covered by a round-trip test.
//
// `options` is a plain object carrying every option/min/max/range field the
// switch reads; both components already hold these as identically-named state,
// so they pass them with object shorthand. Missing fields simply produce
// `undefined` in the token, exactly as the original inline switch did.

/**
 * Map a list of active filter labels to their backend command tokens.
 * @param {string[]} activeFilters - the selected filter labels, in order.
 * @param {object} options - the option/min/max/range state fields.
 * @returns {string[]} one token per filter (empty string for unknown labels).
 */
export function buildFilterTokens(activeFilters, options) {
    const {
        diceRollOption,
        pipCountOption,
        pipCountMin,
        pipCountMax,
        pipCountRangeMin,
        pipCountRangeMax,
        player1AbsolutePipCountOption,
        player1AbsolutePipCountMin,
        player1AbsolutePipCountMax,
        player1AbsolutePipCountRangeMin,
        player1AbsolutePipCountRangeMax,
        equityOption,
        equityMin,
        equityMax,
        equityRangeMin,
        equityRangeMax,
        moveErrorOption,
        moveErrorMin,
        moveErrorMax,
        moveErrorRangeMin,
        moveErrorRangeMax,
        winRateOption,
        winRateMin,
        winRateMax,
        winRateRangeMin,
        winRateRangeMax,
        gammonRateOption,
        gammonRateMin,
        gammonRateMax,
        gammonRateRangeMin,
        gammonRateRangeMax,
        backgammonRateOption,
        backgammonRateMin,
        backgammonRateMax,
        backgammonRateRangeMin,
        backgammonRateRangeMax,
        player2WinRateOption,
        player2WinRateMin,
        player2WinRateMax,
        player2WinRateRangeMin,
        player2WinRateRangeMax,
        player2GammonRateOption,
        player2GammonRateMin,
        player2GammonRateMax,
        player2GammonRateRangeMin,
        player2GammonRateRangeMax,
        player2BackgammonRateOption,
        player2BackgammonRateMin,
        player2BackgammonRateMax,
        player2BackgammonRateRangeMin,
        player2BackgammonRateRangeMax,
        player1CheckerOffOption,
        player1CheckerOffMin,
        player1CheckerOffMax,
        player1CheckerOffRangeMin,
        player1CheckerOffRangeMax,
        player2CheckerOffOption,
        player2CheckerOffMin,
        player2CheckerOffMax,
        player2CheckerOffRangeMin,
        player2CheckerOffRangeMax,
        player1BackCheckerOption,
        player1BackCheckerMin,
        player1BackCheckerMax,
        player1BackCheckerRangeMin,
        player1BackCheckerRangeMax,
        player2BackCheckerOption,
        player2BackCheckerMin,
        player2BackCheckerMax,
        player2BackCheckerRangeMin,
        player2BackCheckerRangeMax,
        player1CheckerInZoneOption,
        player1CheckerInZoneMin,
        player1CheckerInZoneMax,
        player1CheckerInZoneRangeMin,
        player1CheckerInZoneRangeMax,
        player2CheckerInZoneOption,
        player2CheckerInZoneMin,
        player2CheckerInZoneMax,
        player2CheckerInZoneRangeMin,
        player2CheckerInZoneRangeMax,
        searchText,
        movePattern,
        creationDateOption,
        creationDateMin,
        creationDateMax,
        creationDateRangeMin,
        creationDateRangeMax,
        player1OutfieldBlotOption,
        player1OutfieldBlotMin,
        player1OutfieldBlotMax,
        player1OutfieldBlotRangeMin,
        player1OutfieldBlotRangeMax,
        player2OutfieldBlotOption,
        player2OutfieldBlotMin,
        player2OutfieldBlotMax,
        player2OutfieldBlotRangeMin,
        player2OutfieldBlotRangeMax,
        player1JanBlotOption,
        player1JanBlotMin,
        player1JanBlotMax,
        player1JanBlotRangeMin,
        player1JanBlotRangeMax,
        player2JanBlotOption,
        player2JanBlotMin,
        player2JanBlotMax,
        player2JanBlotRangeMin,
        player2JanBlotRangeMax,
        matchIDsInput,
        tournamentIDsInput,
        playerName
    } = options;

    return activeFilters.map((filter) => {
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
            case 'Pipcount Difference':
                return pipCountOption === 'min' ? `p>${pipCountMin}` : pipCountOption === 'max' ? `p<${pipCountMax}` : `p${pipCountRangeMin},${pipCountRangeMax}`;
            case 'Player Absolute Pipcount':
                return player1AbsolutePipCountOption === 'min'
                    ? `P>${player1AbsolutePipCountMin}`
                    : player1AbsolutePipCountOption === 'max'
                      ? `P<${player1AbsolutePipCountMax}`
                      : `P${player1AbsolutePipCountRangeMin},${player1AbsolutePipCountRangeMax}`;
            case 'Equity (millipoints)':
                return equityOption === 'min' ? `e>${equityMin}` : equityOption === 'max' ? `e<${equityMax}` : `e${equityRangeMin},${equityRangeMax}`;
            case 'Move Error (millipoints, Player 1)':
                return moveErrorOption === 'min' ? `E>${moveErrorMin}` : moveErrorOption === 'max' ? `E<${moveErrorMax}` : `E${moveErrorRangeMin},${moveErrorRangeMax}`;
            case 'Win Rate':
                return winRateOption === 'min' ? `w>${winRateMin}` : winRateOption === 'max' ? `w<${winRateMax}` : `w${winRateRangeMin},${winRateRangeMax}`;
            case 'Gammon Rate':
                return gammonRateOption === 'min' ? `g>${gammonRateMin}` : gammonRateOption === 'max' ? `g<${gammonRateMax}` : `g${gammonRateRangeMin},${gammonRateRangeMax}`;
            case 'Backgammon Rate':
                return backgammonRateOption === 'min' ? `b>${backgammonRateMin}` : backgammonRateOption === 'max' ? `b<${backgammonRateMax}` : `b${backgammonRateRangeMin},${backgammonRateRangeMax}`;
            case 'Opponent Win Rate':
                return player2WinRateOption === 'min' ? `W>${player2WinRateMin}` : player2WinRateOption === 'max' ? `W<${player2WinRateMax}` : `W${player2WinRateRangeMin},${player2WinRateRangeMax}`;
            case 'Opponent Gammon Rate':
                return player2GammonRateOption === 'min'
                    ? `G>${player2GammonRateMin}`
                    : player2GammonRateOption === 'max'
                      ? `G<${player2GammonRateMax}`
                      : `G${player2GammonRateRangeMin},${player2GammonRateRangeMax}`;
            case 'Opponent Backgammon Rate':
                return player2BackgammonRateOption === 'min'
                    ? `B>${player2BackgammonRateMin}`
                    : player2BackgammonRateOption === 'max'
                      ? `B<${player2BackgammonRateMax}`
                      : `B${player2BackgammonRateRangeMin},${player2BackgammonRateRangeMax}`;
            case 'Player Checker-Off':
                return player1CheckerOffOption === 'min'
                    ? `o>${player1CheckerOffMin}`
                    : player1CheckerOffOption === 'max'
                      ? `o<${player1CheckerOffMax}`
                      : `o${player1CheckerOffRangeMin},${player1CheckerOffRangeMax}`;
            case 'Opponent Checker-Off':
                return player2CheckerOffOption === 'min'
                    ? `O>${player2CheckerOffMin}`
                    : player2CheckerOffOption === 'max'
                      ? `O<${player2CheckerOffMax}`
                      : `O${player2CheckerOffRangeMin},${player2CheckerOffRangeMax}`;
            case 'Player Back Checker':
                return player1BackCheckerOption === 'min'
                    ? `k>${player1BackCheckerMin}`
                    : player1BackCheckerOption === 'max'
                      ? `k<${player1BackCheckerMax}`
                      : `k${player1BackCheckerRangeMin},${player1BackCheckerRangeMax}`;
            case 'Opponent Back Checker':
                return player2BackCheckerOption === 'min'
                    ? `K>${player2BackCheckerMin}`
                    : player2BackCheckerOption === 'max'
                      ? `K<${player2BackCheckerMax}`
                      : `K${player2BackCheckerRangeMin},${player2BackCheckerRangeMax}`;
            case 'Player Checker in the Zone':
                return player1CheckerInZoneOption === 'min'
                    ? `z>${player1CheckerInZoneMin}`
                    : player1CheckerInZoneOption === 'max'
                      ? `z<${player1CheckerInZoneMax}`
                      : `z${player1CheckerInZoneRangeMin},${player1CheckerInZoneRangeMax}`;
            case 'Opponent Checker in the Zone':
                return player2CheckerInZoneOption === 'min'
                    ? `Z>${player2CheckerInZoneMin}`
                    : player2CheckerInZoneOption === 'max'
                      ? `Z<${player2CheckerInZoneMax}`
                      : `Z${player2CheckerInZoneRangeMin},${player2CheckerInZoneRangeMax}`;
            case 'Search Text':
                return `t"${searchText}"`;
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
            case 'Player Outfield Blot':
                return player1OutfieldBlotOption === 'min'
                    ? `bo>${player1OutfieldBlotMin}`
                    : player1OutfieldBlotOption === 'max'
                      ? `bo<${player1OutfieldBlotMax}`
                      : `bo${player1OutfieldBlotRangeMin},${player1OutfieldBlotRangeMax}`;
            case 'Opponent Outfield Blot':
                return player2OutfieldBlotOption === 'min'
                    ? `BO>${player2OutfieldBlotMin}`
                    : player2OutfieldBlotOption === 'max'
                      ? `BO<${player2OutfieldBlotMax}`
                      : `BO${player2OutfieldBlotRangeMin},${player2OutfieldBlotRangeMax}`;
            case 'Player Jan Blot':
                return player1JanBlotOption === 'min'
                    ? `bj>${player1JanBlotMin}`
                    : player1JanBlotOption === 'max'
                      ? `bj<${player1JanBlotMax}`
                      : `bj${player1JanBlotRangeMin},${player1JanBlotRangeMax}`;
            case 'Opponent Jan Blot':
                return player2JanBlotOption === 'min'
                    ? `BJ>${player2JanBlotMin}`
                    : player2JanBlotOption === 'max'
                      ? `BJ<${player2JanBlotMax}`
                      : `BJ${player2JanBlotRangeMin},${player2JanBlotRangeMax}`;
            case 'Match IDs':
                return matchIDsInput ? `ma${matchIDsInput}` : '';
            case 'Tournament IDs':
                return tournamentIDsInput ? `tn${tournamentIDsInput}` : '';
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

/**
 * Pick the individual backend filter arguments back out of a token list.
 *
 * `onLoadPositionsByFilters` (in App.svelte) takes ~30 positional filter
 * arguments, each of which SearchPanel derived from the `buildFilterTokens`
 * output by prefix-matching the token array. That dense block of `find`/
 * `includes` calls is pure (it depends only on the tokens), so it lives here
 * where it can be unit-tested rather than inline in the component. Prefix
 * collisions are disambiguated exactly as the original inline code did:
 *   - `b…` is the player-1 backgammon-rate token, but NOT the outfield-blot
 *     (`bo…`) or jan-blot (`bj…`) tokens; likewise `B…` vs `BO…`/`BJ…`.
 *   - the double token may be `D` (both rolls) or `D1` (first roll only),
 *     which also selects `drMode`.
 * Match/tournament id tokens (`ma…`, `tn…`) are returned with their 2-char
 * prefix stripped, as the backend expects.
 *
 * @param {string[]} tokens - the output of {@link buildFilterTokens}.
 * @returns {object} the named filter arguments consumed by onLoadPositionsByFilters.
 */
export function parseFilterTokens(tokens) {
    const matchIDToken = tokens.find((f) => f.startsWith('ma'));
    const tournamentIDToken = tokens.find((f) => f.startsWith('tn'));
    return {
        incCube: tokens.includes('cube'),
        incScore: tokens.includes('score'),
        ncFilter: tokens.includes('nc'),
        mirFilter: tokens.includes('M'),
        pcFilter: tokens.find((f) => f.startsWith('p') && !f.startsWith('pl')),
        plFilter: tokens.find((f) => f.startsWith('pl')),
        wrFilter: tokens.find((f) => f.startsWith('w')),
        grFilter: tokens.find((f) => f.startsWith('g')),
        bgFilter: tokens.find((f) => f.startsWith('b') && !f.startsWith('bo') && !f.startsWith('bj')),
        p2wrFilter: tokens.find((f) => f.startsWith('W')),
        p2grFilter: tokens.find((f) => f.startsWith('G')),
        p2bgFilter: tokens.find((f) => f.startsWith('B') && !f.startsWith('BO') && !f.startsWith('BJ')),
        p1coFilter: tokens.find((f) => f.startsWith('o')),
        p2coFilter: tokens.find((f) => f.startsWith('O')),
        p1bcFilter: tokens.find((f) => f.startsWith('k')),
        p2bcFilter: tokens.find((f) => f.startsWith('K')),
        p1czFilter: tokens.find((f) => f.startsWith('z')),
        p2czFilter: tokens.find((f) => f.startsWith('Z')),
        p1apcFilter: tokens.find((f) => f.startsWith('P')),
        eqFilter: tokens.find((f) => f.startsWith('e')),
        meFilter: tokens.find((f) => f.startsWith('E')),
        p1obFilter: tokens.find((f) => f.startsWith('bo')),
        p2obFilter: tokens.find((f) => f.startsWith('BO')),
        p1jbFilter: tokens.find((f) => f.startsWith('bj')),
        p2jbFilter: tokens.find((f) => f.startsWith('BJ')),
        matchIDs: matchIDToken ? matchIDToken.slice(2) : '',
        tournamentIDs: tournamentIDToken ? tournamentIDToken.slice(2) : '',
        dtFilter: tokens.includes('d'),
        drFilter: tokens.includes('D') || tokens.includes('D1'),
        drMode: tokens.includes('D1') ? 'first' : 'both',
        cdFilter: tokens.find((f) => f.startsWith('T'))
    };
}

/**
 * Parse a persisted `s …` search command string back into the flat set of
 * filter values SearchPanel hands to its `onLoadPositionsByFilters` callback
 * when replaying a saved/library search.
 *
 * This is the inverse of buildSearchCommand for the *restore* path and is
 * deliberately more complete than parseFilterTokens (which parses freshly built
 * tokens at save time): it
 *   - expands the shorthand single-value checker tokens (`o5` → `o5,5`) for the
 *     two-sided range filters, and
 *   - pulls the quoted free-text filters (`m"…"`, `t"…"`, `pl"…"`) straight from
 *     the raw command so embedded spaces survive the whitespace split.
 *
 * Logic is lifted verbatim from SearchPanel.executeSearch; unifying it with
 * parseFilterTokens is left as a follow-up because their predicates differ.
 *
 * @param {string} command - a command starting with `s ` (or the bare `s`).
 * @returns {object} the parsed filter values, keyed by short name.
 */
export function parseSearchCommand(command) {
    const cmdFilters =
        command === 's'
            ? []
            : command
                  .slice(2)
                  .trim()
                  .split(' ')
                  .map((f) => f.trim());

    // Single-value checker tokens (e.g. `o5`) restore as a `min,max` pair.
    const expandPair = (tok) => {
        if (tok && !tok.includes(',') && !tok.includes('>') && !tok.includes('<')) {
            return `${tok},${tok.slice(1)}`;
        }
        return tok;
    };

    const matchTokens = (re) => cmdFilters.filter((f) => typeof f === 'string' && re.test(f));
    const quoted = (prefix) => {
        const m = command.match(new RegExp(`${prefix}["'][^"']*["']`));
        return m ? m[0] : '';
    };

    const maTokens = matchTokens(/^ma\d/);
    const tnTokens = matchTokens(/^tn\d/);

    return {
        cmdFilters,
        ic: cmdFilters.includes('cube') || cmdFilters.includes('cu') || cmdFilters.includes('c') || cmdFilters.includes('cub'),
        is: cmdFilters.includes('score') || cmdFilters.includes('sco') || cmdFilters.includes('sc') || cmdFilters.includes('s'),
        nc: cmdFilters.includes('nc'),
        dt: cmdFilters.includes('d'),
        dr: cmdFilters.includes('D') || cmdFilters.includes('D1'),
        drMode: cmdFilters.includes('D1') ? 'first' : 'both',
        mp: cmdFilters.includes('M'),
        pc: cmdFilters.find((f) => typeof f === 'string' && !f.startsWith('pl') && (f.startsWith('p>') || f.startsWith('p<') || f.startsWith('p'))),
        wr: cmdFilters.find((f) => typeof f === 'string' && (f.startsWith('w>') || f.startsWith('w<') || f.startsWith('w'))),
        gr: cmdFilters.find((f) => typeof f === 'string' && (f.startsWith('g>') || f.startsWith('g<') || f.startsWith('g'))),
        bg: cmdFilters.find((f) => typeof f === 'string' && (f.startsWith('b>') || f.startsWith('b<') || (f.startsWith('b') && !f.startsWith('bo'))) && !f.startsWith('bj')),
        p2wr: cmdFilters.find((f) => typeof f === 'string' && (f.startsWith('W>') || f.startsWith('W<') || f.startsWith('W'))),
        p2gr: cmdFilters.find((f) => typeof f === 'string' && (f.startsWith('G>') || f.startsWith('G<') || f.startsWith('G'))),
        p2bg: cmdFilters.find((f) => typeof f === 'string' && (f.startsWith('B>') || f.startsWith('B<') || (f.startsWith('B') && !f.startsWith('BO'))) && !f.startsWith('BJ')),
        p1co: expandPair(cmdFilters.find((f) => typeof f === 'string' && (f.startsWith('o>') || f.startsWith('o<') || f.startsWith('o')))),
        p2co: expandPair(cmdFilters.find((f) => typeof f === 'string' && (f.startsWith('O>') || f.startsWith('O<') || f.startsWith('O')))),
        p1bc: expandPair(cmdFilters.find((f) => typeof f === 'string' && (f.startsWith('k>') || f.startsWith('k<') || f.startsWith('k')))),
        p2bc: expandPair(cmdFilters.find((f) => typeof f === 'string' && (f.startsWith('K>') || f.startsWith('K<') || f.startsWith('K')))),
        p1cz: expandPair(cmdFilters.find((f) => typeof f === 'string' && (f.startsWith('z>') || f.startsWith('z<') || f.startsWith('z')))),
        p2cz: expandPair(cmdFilters.find((f) => typeof f === 'string' && (f.startsWith('Z>') || f.startsWith('Z<') || f.startsWith('Z')))),
        p1apc: cmdFilters.find((f) => typeof f === 'string' && (f.startsWith('P>') || f.startsWith('P<') || f.startsWith('P'))),
        eq: cmdFilters.find((f) => typeof f === 'string' && (f.startsWith('e>') || f.startsWith('e<') || f.startsWith('e'))),
        cd: cmdFilters.find((f) => typeof f === 'string' && (f.startsWith('T>') || f.startsWith('T<') || f.startsWith('T'))),
        mpf: quoted('m'),
        st: quoted('t'),
        plf: quoted('pl'),
        p1ob: cmdFilters.find((f) => typeof f === 'string' && (f.startsWith('bo>') || f.startsWith('bo<') || f.startsWith('bo'))),
        p2ob: cmdFilters.find((f) => typeof f === 'string' && (f.startsWith('BO>') || f.startsWith('BO<') || f.startsWith('BO'))),
        p1jb: cmdFilters.find((f) => typeof f === 'string' && (f.startsWith('bj>') || f.startsWith('bj<') || f.startsWith('bj'))),
        p2jb: cmdFilters.find((f) => typeof f === 'string' && (f.startsWith('BJ>') || f.startsWith('BJ<') || f.startsWith('BJ'))),
        me: cmdFilters.find((f) => typeof f === 'string' && (f.startsWith('E>') || f.startsWith('E<') || (f.startsWith('E') && /^E\d/.test(f)))),
        matchIDs: maTokens.length > 0 ? maTokens.map((t) => t.slice(2)).join(';') : '',
        tournamentIDs: tnTokens.length > 0 ? tnTokens.map((t) => t.slice(2)).join(';') : ''
    };
}

// Command-line token for each search filter, keyed by its canonical (English)
// label — the same labels SearchPanel's filterGroups use. Single source of
// truth for the in-UI token hint shown on hover; the prefixes mirror the
// buildFilterTokens switch above. `type` drives how filterTokenHint renders the
// usage forms:
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
    'Pipcount Difference': { token: 'p', type: 'range' },
    'Player Absolute Pipcount': { token: 'P', type: 'range' },
    'Equity (millipoints)': { token: 'e', type: 'range' },
    'Move Error (millipoints, Player 1)': { token: 'E', type: 'range' },
    'Win Rate': { token: 'w', type: 'range' },
    'Gammon Rate': { token: 'g', type: 'range' },
    'Backgammon Rate': { token: 'b', type: 'range' },
    'Opponent Win Rate': { token: 'W', type: 'range' },
    'Opponent Gammon Rate': { token: 'G', type: 'range' },
    'Opponent Backgammon Rate': { token: 'B', type: 'range' },
    'Player Checker-Off': { token: 'o', type: 'range' },
    'Opponent Checker-Off': { token: 'O', type: 'range' },
    'Player Back Checker': { token: 'k', type: 'range' },
    'Opponent Back Checker': { token: 'K', type: 'range' },
    'Player Checker in the Zone': { token: 'z', type: 'range' },
    'Opponent Checker in the Zone': { token: 'Z', type: 'range' },
    'Player Outfield Blot': { token: 'bo', type: 'range' },
    'Opponent Outfield Blot': { token: 'BO', type: 'range' },
    'Player Jan Blot': { token: 'bj', type: 'range' },
    'Opponent Jan Blot': { token: 'BJ', type: 'range' },
    'Search Text': { token: 't', type: 'text' },
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
        case 'date':
            return `${token}>YYYY/MM/DD · ${token}<YYYY/MM/DD`;
        case 'dice':
            return `${token} · ${token}1`;
        case 'flag':
        default:
            return token;
    }
}
