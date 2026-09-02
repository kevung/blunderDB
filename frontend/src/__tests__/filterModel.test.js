/**
 * filterModel.test.js
 *
 * The declarative table of numeric search filters replaced five hand-written
 * state variables per filter, re-enumerated in four SearchPanel functions and
 * in searchFilterService. These tests pin the table's integrity (unique keys,
 * tokens, labels; defaults inside the bounds) and the pure round-trips the
 * panel relies on: state → flat store → state, and state → token → the field
 * every parser (parseFilterTokens, parseSearchCommand, commandProcessor's
 * parseFilters) reports it under.
 */

import { describe, test, expect, vi } from 'vitest';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    SaveComment: vi.fn().mockResolvedValue(undefined),
    SaveSearchHistory: vi.fn().mockResolvedValue(undefined),
    ClearCommandHistory: vi.fn().mockResolvedValue(undefined)
}));

import { NUMERIC_FILTERS, NUMERIC_FILTER_BY_LABEL, createFilterState, clear, toStore, fromStore, toTokens, numericToken, readFlat, flatName } from '../services/filterModel.js';
import { buildFilterTokens, parseFilterTokens, parseSearchCommand, filterTokenHint } from '../services/searchFilterService.js';
import { parseFilters } from '../commandProcessor.js';

const FIELDS = ['option', 'min', 'max', 'rangeMin', 'rangeMax'];

function unique(values) {
    return new Set(values).size === values.length;
}

describe('NUMERIC_FILTERS — table integrity', () => {
    test('declares the 20 numeric filters of the search panel', () => {
        expect(NUMERIC_FILTERS).toHaveLength(20);
    });

    test('keys, short names, labels and tokens are all unique', () => {
        expect(unique(NUMERIC_FILTERS.map((f) => f.key))).toBe(true);
        expect(unique(NUMERIC_FILTERS.map((f) => f.short))).toBe(true);
        expect(unique(NUMERIC_FILTERS.map((f) => f.label))).toBe(true);
        expect(unique(NUMERIC_FILTERS.map((f) => f.token))).toBe(true);
    });

    test.each(NUMERIC_FILTERS.map((f) => [f.label, f]))('%s — defaults sit inside the bounds, kind is int|float', (_label, f) => {
        const { min, max, option } = f.defaults;
        expect(['min', 'max', 'range']).toContain(option);
        expect(min).toBeLessThanOrEqual(max);
        if (f.bounds.min !== undefined) expect(min).toBeGreaterThanOrEqual(f.bounds.min);
        if (f.bounds.max !== undefined) expect(max).toBeLessThanOrEqual(f.bounds.max);
        expect(['int', 'float']).toContain(f.kind);
    });

    test('NUMERIC_FILTER_BY_LABEL indexes every entry and nothing else', () => {
        expect(Object.keys(NUMERIC_FILTER_BY_LABEL)).toHaveLength(NUMERIC_FILTERS.length);
        for (const f of NUMERIC_FILTERS) expect(NUMERIC_FILTER_BY_LABEL[f.label]).toBe(f);
        expect(NUMERIC_FILTER_BY_LABEL['Creation Date']).toBeUndefined();
        expect(NUMERIC_FILTER_BY_LABEL['Comment']).toBeUndefined();
    });

    test('the hover hint of every numeric filter is built on its token', () => {
        for (const f of NUMERIC_FILTERS) {
            expect(filterTokenHint(f.label)).toBe(`${f.token}>n · ${f.token}<n · ${f.token}n,m`);
        }
    });
});

describe('createFilterState / clear', () => {
    test('fresh state carries the defaults, range mirroring min/max', () => {
        const state = createFilterState();
        expect(Object.keys(state)).toEqual(NUMERIC_FILTERS.map((f) => f.key));
        for (const f of NUMERIC_FILTERS) {
            expect(state[f.key]).toEqual({ option: f.defaults.option, min: f.defaults.min, max: f.defaults.max, rangeMin: f.defaults.min, rangeMax: f.defaults.max });
        }
    });

    test('clear resets every field in place', () => {
        const state = createFilterState();
        const entry = state.winRate;
        Object.assign(entry, { option: 'range', min: 7, max: 8, rangeMin: 9, rangeMax: 10 });
        state.pipCount.option = 'max';
        expect(clear(state)).toBe(state);
        expect(state.winRate).toBe(entry); // same object: bindings stay attached
        expect(state).toEqual(createFilterState());
    });
});

describe('toStore / fromStore', () => {
    test('flat names follow the historical <key>Option/Min/Max/RangeMin/RangeMax convention', () => {
        expect(flatName('winRate', 'option')).toBe('winRateOption');
        expect(flatName('winRate', 'rangeMin')).toBe('winRateRangeMin');
        const flat = toStore(createFilterState());
        expect(Object.keys(flat)).toHaveLength(NUMERIC_FILTERS.length * FIELDS.length);
        expect(flat.pipCountMin).toBe(-375);
        expect(flat.player2JanBlotRangeMax).toBe(15);
    });

    test('state → store → state is the identity', () => {
        const state = createFilterState();
        let n = 1;
        for (const f of NUMERIC_FILTERS) {
            state[f.key].option = ['min', 'max', 'range'][n % 3];
            for (const field of FIELDS.slice(1)) state[f.key][field] = n++;
        }
        const restored = fromStore(createFilterState(), toStore(state));
        expect(restored).toEqual(state);
    });

    test('a missing saved field falls back to the default instead of undefined', () => {
        const state = fromStore(createFilterState(), { winRateOption: 'max', winRateMax: 42 });
        expect(state.winRate).toEqual({ option: 'max', min: 0, max: 42, rangeMin: 0, rangeMax: 100 });
        expect(state.equity).toEqual(createFilterState().equity);
        expect(fromStore(createFilterState(), null)).toEqual(createFilterState());
    });

    test('readFlat picks one filter out of a flat object', () => {
        const flat = { equityOption: 'range', equityMin: 1, equityMax: 2, equityRangeMin: 3, equityRangeMax: 4, winRateMin: 99 };
        expect(readFlat(NUMERIC_FILTER_BY_LABEL['Equity (millipoints)'], flat)).toEqual({ option: 'range', min: 1, max: 2, rangeMin: 3, rangeMax: 4 });
    });
});

describe('tokens', () => {
    test('numericToken follows the option', () => {
        const f = NUMERIC_FILTER_BY_LABEL['Player Outfield Blot'];
        const entry = { option: 'min', min: 1, max: 4, rangeMin: 2, rangeMax: 3 };
        expect(numericToken(f, entry)).toBe('bo>1');
        expect(numericToken(f, { ...entry, option: 'max' })).toBe('bo<4');
        expect(numericToken(f, { ...entry, option: 'range' })).toBe('bo2,3');
    });

    test('toTokens covers every filter and agrees with buildFilterTokens on the flat shape', () => {
        const state = createFilterState();
        state.winRate.option = 'range';
        state.winRate.rangeMin = 45;
        state.winRate.rangeMax = 55;
        const tokens = toTokens(state);
        expect(Object.keys(tokens)).toEqual(NUMERIC_FILTERS.map((f) => f.key));
        expect(tokens.winRate).toBe('w45,55');
        const viaService = buildFilterTokens(
            NUMERIC_FILTERS.map((f) => f.label),
            toStore(state)
        );
        expect(viaService).toEqual(NUMERIC_FILTERS.map((f) => tokens[f.key]));
    });

    // Every parser must file the token under the name the panel reads it back
    // from: `${short}Filter` (parseFilterTokens), `${short}` (parseSearchCommand)
    // and `${key}Filter` (commandProcessor.parseFilters / onLoadPositionsByFilters).
    test.each(NUMERIC_FILTERS.map((f) => [f.label, f]))('%s — token round-trips through the three parsers', (_label, f) => {
        const entry = { option: 'range', min: 1, max: 9, rangeMin: 2, rangeMax: 7 };
        const tok = numericToken(f, entry);
        expect(parseFilterTokens([tok])[`${f.short}Filter`]).toBe(tok);
        expect(parseSearchCommand(`s ${tok}`)[f.short]).toBe(tok);
        expect(parseFilters([tok], `s ${tok}`)[`${f.key}Filter`]).toBe(tok);
    });
});
