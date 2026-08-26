import { writable, derived, get } from 'svelte/store';
import { ComputeStats, GetPlayerTable } from '../../wailsjs/go/database/Database.js';
import { databasePathStore } from './databaseStore.js';
import { dbMutationCounterStore } from './uiStore.js';

const defaultFilter = {
    playerName: '',
    tournamentIDs: [],
    dateFrom: '',
    dateTo: '',
    decisionType: -1, // -1 = all, 0 = checker, 1 = cube
    matchLength: []
};

export const statsFilterStore = writable(defaultFilter);
export const statsResultStore = writable(null);
export const statsLoadingStore = writable(false);
export const statsErrorStore = writable(null);

// Toggle global PR / MWC display (persisted via Config.yaml in fiche 09)
// 'pr' | 'mwc'
export const statsMetricStore = writable('pr');

/**
 * Opaque key combining the open database path and the mutation counter.
 * Changes whenever a new database is opened or the data is mutated (import,
 * delete match, etc.). Used by refreshStats to detect stale cache.
 */
export const statsInvalidationKeyStore = derived([databasePathStore, dbMutationCounterStore], ([$path, $mutation]) => `${$path}::${$mutation}`);

/** Cache key of the last successful fetch. */
let _cachedKey = null;

/**
 * Fetch stats from the backend for the current filter and invalidation key.
 * Skips the backend call when the result is already cached for the same
 * filter + database state — prevents redundant recalculation on every
 * tab activation.
 *
 * @param {object} filter          - StatsFilter object
 * @param {string} invalidationKey - value of statsInvalidationKeyStore
 */
export async function refreshStats(filter, invalidationKey) {
    const key = JSON.stringify(filter) + '||' + invalidationKey;
    if (key === _cachedKey && get(statsResultStore) !== null) {
        return; // cache hit — nothing changed
    }
    _cachedKey = key;
    statsLoadingStore.set(true);
    statsErrorStore.set(null);
    try {
        const result = await ComputeStats(filter);
        statsResultStore.set(result);
    } catch (err) {
        _cachedKey = null; // allow retry on error
        statsErrorStore.set(err?.message ?? String(err));
        statsResultStore.set(null);
    } finally {
        statsLoadingStore.set(false);
    }
}

export const playerTableStore = writable(null);
export const playerTableLoadingStore = writable(false);
export const playerTableErrorStore = writable(null);

/** Cache key of the last successful player-table fetch. */
let _cachedPlayerKey = null;

/**
 * Fetch the players table for the current filter and invalidation key.
 *
 * The table only varies with the parts of the filter the backend honours
 * (dates, tournaments, match lengths), so the cache key drops the rest: the
 * player selection and the decision type are ignored by PlayerTable, and
 * including them would refetch an identical table every time the user picks a
 * player in another tab.
 *
 * @param {object} filter          - StatsFilter object
 * @param {string} invalidationKey - value of statsInvalidationKeyStore
 */
export async function refreshPlayerTable(filter, invalidationKey) {
    const key =
        JSON.stringify({
            tournamentIDs: filter.tournamentIDs,
            dateFrom: filter.dateFrom,
            dateTo: filter.dateTo,
            matchLength: filter.matchLength
        }) +
        '||' +
        invalidationKey;
    if (key === _cachedPlayerKey && get(playerTableStore) !== null) {
        return; // cache hit — nothing the table depends on changed
    }
    _cachedPlayerKey = key;
    playerTableLoadingStore.set(true);
    playerTableErrorStore.set(null);
    try {
        const rows = await GetPlayerTable(filter);
        playerTableStore.set(rows ?? []);
    } catch (err) {
        _cachedPlayerKey = null; // allow retry on error
        playerTableErrorStore.set(err?.message ?? String(err));
        playerTableStore.set(null);
    } finally {
        playerTableLoadingStore.set(false);
    }
}
