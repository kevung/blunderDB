import { writable } from 'svelte/store';
import { ComputeStats } from '../../wailsjs/go/main/Database.js';

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
 * Fetch stats from the backend for the current filter and update the stores.
 * Called manually from StatsPanel on mount and on filter changes.
 * @param {object} filter - StatsFilter object
 */
export async function refreshStats(filter) {
    statsLoadingStore.set(true);
    statsErrorStore.set(null);
    try {
        const result = await ComputeStats(filter);
        statsResultStore.set(result);
    } catch (err) {
        statsErrorStore.set(err?.message ?? String(err));
        statsResultStore.set(null);
    } finally {
        statsLoadingStore.set(false);
    }
}
