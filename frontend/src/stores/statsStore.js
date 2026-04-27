import { writable, derived, get } from 'svelte/store';
import { ComputeStats } from '../../wailsjs/go/main/Database.js';
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
export const statsInvalidationKeyStore = derived(
    [databasePathStore, dbMutationCounterStore],
    ([$path, $mutation]) => `${$path}::${$mutation}`
);

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
