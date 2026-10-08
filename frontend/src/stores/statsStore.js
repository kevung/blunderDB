import { writable, derived, get } from 'svelte/store';
import { ComputeStats, ComputeRecurringErrors, ComputeStudyPlan, ComputeTrainingStats, GetPlayerTable } from '../../wailsjs/go/database/Database.js';
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
// 'pr' | 'mwc' | 'mwc7'
export const statsMetricStore = writable('pr');

/** Opaque key (database path + mutation counter) that refreshStats uses to detect a stale cache. */
export const statsInvalidationKeyStore = derived([databasePathStore, dbMutationCounterStore], ([$path, $mutation]) => `${$path}::${$mutation}`);

/** Cache key of the last successful fetch. */
let _cachedKey = null;

/**
 * Fetch stats for the filter, skipping the backend when already cached for the same filter and
 * database state (no recalculation on every tab activation).
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
 * Fetch the players table. The cache key keeps only the filter parts the backend honours (dates,
 * tournaments, match lengths): picking a player elsewhere must not refetch an identical table.
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

export const recurringErrorsStore = writable(null);
export const recurringErrorsLoadingStore = writable(false);
export const recurringErrorsErrorStore = writable(null);

/** Cache key of the last successful recurring-errors fetch. */
let _cachedRecurringKey = null;

/**
 * Fetch the recurring errors of the filter: its errors grouped by plan of play and theme. Fetched
 * apart from ComputeStats because classifying each error replays its analysis — a cost the other
 * tabs must not pay.
 *
 * @param {object} filter          - StatsFilter object
 * @param {string} invalidationKey - value of statsInvalidationKeyStore
 */
export async function refreshRecurringErrors(filter, invalidationKey) {
    const key = JSON.stringify(filter) + '||' + invalidationKey;
    if (key === _cachedRecurringKey && get(recurringErrorsStore) !== null) {
        return;
    }
    _cachedRecurringKey = key;
    recurringErrorsLoadingStore.set(true);
    recurringErrorsErrorStore.set(null);
    try {
        recurringErrorsStore.set(await ComputeRecurringErrors(filter));
    } catch (err) {
        _cachedRecurringKey = null; // allow retry on error
        recurringErrorsErrorStore.set(err?.message ?? String(err));
        recurringErrorsStore.set(null);
    } finally {
        recurringErrorsLoadingStore.set(false);
    }
}

export const studyPlanStore = writable(null);
export const studyPlanLoadingStore = writable(false);
export const studyPlanErrorStore = writable(null);

/** Cache key of the last successful study-plan fetch. */
let _cachedStudyPlanKey = null;

/**
 * Fetch the study plan of the filter (ADR-0077): the error families ranked by recoverable MWC.
 * It replays each error's analysis, like the recurring errors: fetched only while the dashboard,
 * which shows it, is open.
 * @param {object} filter
 * @param {number} invalidationKey
 */
export async function refreshStudyPlan(filter, invalidationKey) {
    const key = JSON.stringify(filter) + '||' + invalidationKey;
    if (key === _cachedStudyPlanKey && get(studyPlanStore) !== null) {
        return;
    }
    _cachedStudyPlanKey = key;
    studyPlanLoadingStore.set(true);
    studyPlanErrorStore.set(null);
    try {
        studyPlanStore.set(await ComputeStudyPlan(filter));
    } catch (err) {
        _cachedStudyPlanKey = null; // allow retry on error
        studyPlanErrorStore.set(err?.message ?? String(err));
        studyPlanStore.set(null);
    } finally {
        studyPlanLoadingStore.set(false);
    }
}

export const trainingStatsStore = writable(null);
export const trainingStatsLoadingStore = writable(false);
export const trainingStatsErrorStore = writable(null);

/** The calendar window the training series are folded by: `week` or `month`. */
export const trainingWindowStore = writable('week');

/** Cache key of the last successful training-stats fetch. */
let _cachedTrainingKey = null;

/**
 * Un quiz terminé ou une carte révisée change la série sans toucher à aucune clé de la requête :
 * la prochaine lecture doit recalculer.
 */
export function invalidateTrainingStats() {
    _cachedTrainingKey = null;
}

/**
 * Fetch the training series: the Decision quiz PR and the Anki retention folded by calendar
 * window, with the real PR of the filter's matches on the same windows. Fetched apart from
 * ComputeStats, and only while its tab is open.
 *
 * @param {object} filter          - StatsFilter object (it restricts the matches only)
 * @param {string} invalidationKey - value of statsInvalidationKeyStore
 * @param {string} window          - `week` or `month`
 */
export async function refreshTrainingStats(filter, invalidationKey, window) {
    const key = JSON.stringify(filter) + '||' + invalidationKey + '||' + window;
    if (key === _cachedTrainingKey && get(trainingStatsStore) !== null) {
        return;
    }
    _cachedTrainingKey = key;
    trainingStatsLoadingStore.set(true);
    trainingStatsErrorStore.set(null);
    try {
        trainingStatsStore.set(await ComputeTrainingStats(filter, window));
    } catch (err) {
        _cachedTrainingKey = null; // allow retry on error
        trainingStatsErrorStore.set(err?.message ?? String(err));
        trainingStatsStore.set(null);
    } finally {
        trainingStatsLoadingStore.set(false);
    }
}
