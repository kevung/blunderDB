// searchFilterService — shared logic that turns the search UI's active filter
// labels + their option/min/max/range state into the backend command tokens
// (`cube`, `p>12`, `e10,50`, `t"foo"`, `T>2026/01/01`, …).
//
// This was duplicated verbatim as a 31-case switch inside both
// `SearchModal.svelte` and `SearchPanel.svelte`. Extracting it removes the
// duplication and makes the mapping unit-testable. It is the inverse of
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
        tournamentIDsInput
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
        if (token !== 't""' && token !== 'm""') {
            commandParts.push(token);
        }
    });
    return commandParts.join(' ');
}
