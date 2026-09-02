// filterModel — the declarative table of the search panel's numeric
// min / max / range filters, and the pure functions that turn a filter state
// into command tokens and back into the persisted search params.
//
// Before this module, each of the 20 numeric filters was five hand-written
// `$state` variables (`xxxOption/Min/Max/RangeMin/RangeMax`) re-enumerated in
// SearchPanel's handleSearch / clearFilters / saveSearchState /
// restoreSearchState and once more in searchFilterService's 113-field
// destructuring. Adding a filter meant editing six lists in lockstep; this
// table is the one place a numeric filter is declared.
//
// Each entry:
//   key      — camelCase identifier; prefix of the flat field names the
//              searchParamsStore and buildFilterTokens still use
//              (`${key}Option`, `${key}Min`, …) and of the backend argument
//              name (`${key}Filter`) in onLoadPositionsByFilters / parseFilters.
//   short    — the abbreviated key searchFilterService's parseFilterTokens
//              (`${short}Filter`) and parseSearchCommand (`${short}`) report
//              the recovered token under.
//   label    — canonical (English) filter label, the logic key SearchPanel's
//              filterGroups / filterEnabled / i18n slugs are built on.
//   token    — command-line prefix (`p>12`, `w45,55`), mirrored by
//              commandVocabulary.js / commandProcessor.js.
//   defaults — option/min/max applied on clear (range uses min/max too).
//   bounds   — HTML `min`/`max` attributes of the number inputs (either may
//              be undefined: the pipcount difference and equity are signed
//              and unbounded in the UI).
//   kind     — 'int' (counts, pips) or 'float' (rates, equities): mirrors
//              which backend parser (`ParseIntFilterExpr` /
//              `ParseFloatFilterExpr`) reads the token.

const RATE = { defaults: { option: 'min', min: 0, max: 100 }, bounds: { min: 0, max: 100 }, kind: 'float' };
const COUNT = { defaults: { option: 'min', min: 0, max: 15 }, bounds: { min: 0, max: 15 }, kind: 'int' };

export const NUMERIC_FILTERS = Object.freeze([
    { key: 'pipCount', short: 'pc', label: 'Pipcount Difference', token: 'p', defaults: { option: 'min', min: -375, max: 375 }, bounds: {}, kind: 'int' },
    { key: 'player1AbsolutePipCount', short: 'p1apc', label: 'Player Absolute Pipcount', token: 'P', defaults: { option: 'min', min: 0, max: 375 }, bounds: { min: 0, max: 375 }, kind: 'int' },
    { key: 'equity', short: 'eq', label: 'Equity (millipoints)', token: 'e', defaults: { option: 'min', min: -1000, max: 1000 }, bounds: {}, kind: 'float' },
    { key: 'moveError', short: 'me', label: 'Move Error (millipoints, Player 1)', token: 'E', defaults: { option: 'min', min: 0, max: 1000 }, bounds: { min: 0 }, kind: 'float' },
    { key: 'winRate', short: 'wr', label: 'Win Rate', token: 'w', ...RATE },
    { key: 'gammonRate', short: 'gr', label: 'Gammon Rate', token: 'g', ...RATE },
    { key: 'backgammonRate', short: 'bg', label: 'Backgammon Rate', token: 'b', ...RATE },
    { key: 'player2WinRate', short: 'p2wr', label: 'Opponent Win Rate', token: 'W', ...RATE },
    { key: 'player2GammonRate', short: 'p2gr', label: 'Opponent Gammon Rate', token: 'G', ...RATE },
    { key: 'player2BackgammonRate', short: 'p2bg', label: 'Opponent Backgammon Rate', token: 'B', ...RATE },
    { key: 'player1CheckerOff', short: 'p1co', label: 'Player Checker-Off', token: 'o', ...COUNT },
    { key: 'player2CheckerOff', short: 'p2co', label: 'Opponent Checker-Off', token: 'O', ...COUNT },
    { key: 'player1BackChecker', short: 'p1bc', label: 'Player Back Checker', token: 'k', ...COUNT },
    { key: 'player2BackChecker', short: 'p2bc', label: 'Opponent Back Checker', token: 'K', ...COUNT },
    { key: 'player1CheckerInZone', short: 'p1cz', label: 'Player Checker in the Zone', token: 'z', ...COUNT },
    { key: 'player2CheckerInZone', short: 'p2cz', label: 'Opponent Checker in the Zone', token: 'Z', ...COUNT },
    { key: 'player1OutfieldBlot', short: 'p1ob', label: 'Player Outfield Blot', token: 'bo', ...COUNT },
    { key: 'player2OutfieldBlot', short: 'p2ob', label: 'Opponent Outfield Blot', token: 'BO', ...COUNT },
    { key: 'player1JanBlot', short: 'p1jb', label: 'Player Jan Blot', token: 'bj', ...COUNT },
    { key: 'player2JanBlot', short: 'p2jb', label: 'Opponent Jan Blot', token: 'BJ', ...COUNT }
]);

/** Filter descriptor by canonical label (undefined for non-numeric labels). */
export const NUMERIC_FILTER_BY_LABEL = Object.freeze(Object.fromEntries(NUMERIC_FILTERS.map((f) => [f.label, f])));

/** The five per-filter fields, in the order the flat store names them. */
const FIELDS = ['option', 'min', 'max', 'rangeMin', 'rangeMax'];

/** Flat field name in the searchParamsStore / buildFilterTokens options. */
export function flatName(key, field) {
    return key + field[0].toUpperCase() + field.slice(1);
}

function defaultEntry(filter) {
    const { option, min, max } = filter.defaults;
    return { option, min, max, rangeMin: min, rangeMax: max };
}

/**
 * Fresh per-filter state, keyed by filter key:
 * `{ option, min, max, rangeMin, rangeMax }` at the declared defaults. The
 * component wraps the result in `$state(...)` so the rows can bind into it.
 */
export function createFilterState(model = NUMERIC_FILTERS) {
    return Object.fromEntries(model.map((f) => [f.key, defaultEntry(f)]));
}

/** Reset every filter to its defaults, in place (keeps the reactive proxy). */
export function clear(state, model = NUMERIC_FILTERS) {
    for (const f of model) {
        Object.assign(state[f.key], defaultEntry(f));
    }
    return state;
}

/**
 * Flatten the state into the `${key}Option/Min/Max/RangeMin/RangeMax` fields —
 * the shape searchParamsStore persists and buildFilterTokens reads.
 */
export function toStore(state, model = NUMERIC_FILTERS) {
    const out = {};
    for (const f of model) {
        const entry = state[f.key];
        for (const field of FIELDS) out[flatName(f.key, field)] = entry[field];
    }
    return out;
}

/**
 * Restore the state from a flat store object, in place. A field the saved
 * object lacks falls back to the filter's default rather than becoming
 * `undefined` (the hand-written restore did not guard against that).
 */
export function fromStore(state, saved, model = NUMERIC_FILTERS) {
    for (const f of model) {
        const fallback = defaultEntry(f);
        const entry = state[f.key];
        for (const field of FIELDS) {
            const v = saved?.[flatName(f.key, field)];
            entry[field] = v === undefined ? fallback[field] : v;
        }
    }
    return state;
}

/** Read one filter's five fields from a flat store / options object. */
export function readFlat(filter, flat) {
    return Object.fromEntries(FIELDS.map((field) => [field, flat?.[flatName(filter.key, field)]]));
}

/**
 * Command token for one filter: `X>min`, `X<max` or `Xrmin,rmax` depending on
 * `option`. Values are interpolated as-is (an undefined value prints as
 * `undefined`, exactly as the original inline switch did).
 */
export function numericToken(filter, entry) {
    const { option, min, max, rangeMin, rangeMax } = entry ?? {};
    const t = filter.token;
    return option === 'min' ? `${t}>${min}` : option === 'max' ? `${t}<${max}` : `${t}${rangeMin},${rangeMax}`;
}

/** Token per filter key for the whole state (all filters, active or not). */
export function toTokens(state, model = NUMERIC_FILTERS) {
    return Object.fromEntries(model.map((f) => [f.key, numericToken(f, state[f.key])]));
}
