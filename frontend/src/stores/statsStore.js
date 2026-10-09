import { writable, derived, get } from 'svelte/store';
import { ComputeStats, ComputeRecurringErrors, ComputeStudyPlan, ComputeStudyEffect, ComputeDirectionalBiases, ComputeTrainingStats, GetPlayerTable } from '../../wailsjs/go/database/Database.js';
import { databasePathStore } from './databaseStore.js';
import { dbMutationCounterStore } from './uiStore.js';

/** @typedef {import('../../wailsjs/go/models').database.StatsResult} StatsResult the engine result; consumers narrow it field by field */

const defaultFilter = {
    playerName: '',
    tournamentIDs: [],
    dateFrom: '',
    dateTo: '',
    decisionType: -1, // -1 = all, 0 = checker, 1 = cube
    matchLength: []
};

export const statsFilterStore = writable(defaultFilter);
export const statsResultStore = writable(/** @type {any} */ (null));
export const statsLoadingStore = writable(false);
export const statsErrorStore = writable(/** @type {any} */ (null));
/** The filter statsResultStore was computed under: a tab drilling into the result asks with it. */
export const statsResultFilterStore = writable(/** @type {any} */ (null));

// Toggle global PR / MWC display (persisted via Config.yaml in fiche 09)
// 'pr' | 'mwc' | 'mwc7'
export const statsMetricStore = writable('pr');

/** Opaque key (database path + mutation counter) that refreshStats uses to detect a stale cache. */
export const statsInvalidationKeyStore = derived([databasePathStore, dbMutationCounterStore], ([$path, $mutation]) => `${$path}::${$mutation}`);

/** Cache key of the last successful fetch. */
/** @type {string | null} */
let _cachedKey = null;
/**
 * Sequence number of the latest stats request. Opening the panel computes the default filter,
 * then the restored one: when the older, wider reply lands last it must not overwrite the newer.
 */
let _statsSeq = 0;

/**
 * Fetch stats for the filter, skipping the backend when already cached for the same filter and
 * database state (no recalculation on every tab activation).
 *
 * @param {any} filter          - StatsFilter object
 * @param {string} invalidationKey - value of statsInvalidationKeyStore
 */
export async function refreshStats(filter, invalidationKey) {
    const key = JSON.stringify(filter) + '||' + invalidationKey;
    if (key === _cachedKey && get(statsResultStore) !== null) {
        return; // cache hit — nothing changed
    }
    _cachedKey = key;
    const seq = ++_statsSeq;
    statsLoadingStore.set(true);
    statsErrorStore.set(null);
    try {
        const result = await ComputeStats(filter);
        if (seq !== _statsSeq) return;
        statsResultFilterStore.set(filter);
        statsResultStore.set(result);
    } catch (err) {
        if (seq !== _statsSeq) return;
        _cachedKey = null; // allow retry on error
        statsErrorStore.set(/** @type {any} */ (err)?.message ?? String(err));
        statsResultFilterStore.set(null);
        statsResultStore.set(null);
    } finally {
        if (seq === _statsSeq) statsLoadingStore.set(false);
    }
}

export const playerTableStore = writable(/** @type {any} */ (null));
export const playerTableLoadingStore = writable(false);
export const playerTableErrorStore = writable(/** @type {any} */ (null));

/** Cache key of the last successful player-table fetch. */
/** @type {string | null} */
let _cachedPlayerKey = null;
/** Sequence number of the latest player-table request: a slower, older reply is dropped. */
let _playerSeq = 0;

/**
 * Fetch the players table. The cache key keeps only the filter parts the backend honours (dates,
 * tournaments, match lengths): picking a player elsewhere must not refetch an identical table.
 *
 * @param {any} filter          - StatsFilter object
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
    const seq = ++_playerSeq;
    playerTableLoadingStore.set(true);
    playerTableErrorStore.set(null);
    try {
        const rows = await GetPlayerTable(filter);
        if (seq !== _playerSeq) return;
        playerTableStore.set(rows ?? []);
    } catch (err) {
        if (seq !== _playerSeq) return;
        _cachedPlayerKey = null; // allow retry on error
        playerTableErrorStore.set(/** @type {any} */ (err)?.message ?? String(err));
        playerTableStore.set(null);
    } finally {
        if (seq === _playerSeq) playerTableLoadingStore.set(false);
    }
}

export const recurringErrorsStore = writable(/** @type {any} */ (null));
export const recurringErrorsLoadingStore = writable(false);
export const recurringErrorsErrorStore = writable(/** @type {any} */ (null));

/** Cache key of the last successful recurring-errors fetch. */
/** @type {string | null} */
let _cachedRecurringKey = null;
/** Sequence number of the latest recurring-errors request: a slower, older reply is dropped. */
let _recurringSeq = 0;

/**
 * Fetch the recurring errors of the filter: its errors grouped by plan of play and theme. Fetched
 * apart from ComputeStats because classifying each error replays its analysis — a cost the other
 * tabs must not pay.
 *
 * @param {any} filter          - StatsFilter object
 * @param {string} invalidationKey - value of statsInvalidationKeyStore
 */
export async function refreshRecurringErrors(filter, invalidationKey) {
    const key = JSON.stringify(filter) + '||' + invalidationKey;
    if (key === _cachedRecurringKey && get(recurringErrorsStore) !== null) {
        return;
    }
    _cachedRecurringKey = key;
    const seq = ++_recurringSeq;
    recurringErrorsLoadingStore.set(true);
    recurringErrorsErrorStore.set(null);
    try {
        const recurring = await ComputeRecurringErrors(filter);
        if (seq !== _recurringSeq) return;
        recurringErrorsStore.set(recurring);
    } catch (err) {
        if (seq !== _recurringSeq) return;
        _cachedRecurringKey = null; // allow retry on error
        recurringErrorsErrorStore.set(/** @type {any} */ (err)?.message ?? String(err));
        recurringErrorsStore.set(null);
    } finally {
        if (seq === _recurringSeq) recurringErrorsLoadingStore.set(false);
    }
}

export const studyPlanStore = writable(/** @type {any} */ (null));
export const studyPlanLoadingStore = writable(false);
export const studyPlanErrorStore = writable(/** @type {any} */ (null));

/** Cache key of the last successful study-plan fetch. */
/** @type {string | null} */
let _cachedStudyPlanKey = null;
/** Sequence number of the latest study-plan request: a slower, older reply is dropped. */
let _studyPlanSeq = 0;

/**
 * Fetch the study plan of the filter (ADR-0077): the error families ranked by recoverable MWC.
 * It replays each error's analysis, like the recurring errors: fetched only while the dashboard,
 * which shows it, is open.
 * @param {any} filter
 * @param {number} invalidationKey
 */
export async function refreshStudyPlan(filter, invalidationKey) {
    const key = JSON.stringify(filter) + '||' + invalidationKey;
    if (key === _cachedStudyPlanKey && get(studyPlanStore) !== null) {
        return;
    }
    _cachedStudyPlanKey = key;
    const seq = ++_studyPlanSeq;
    studyPlanLoadingStore.set(true);
    studyPlanErrorStore.set(null);
    try {
        const plan = await ComputeStudyPlan(filter);
        if (seq !== _studyPlanSeq) return;
        studyPlanStore.set(plan);
    } catch (err) {
        if (seq !== _studyPlanSeq) return;
        _cachedStudyPlanKey = null; // allow retry on error
        studyPlanErrorStore.set(/** @type {any} */ (err)?.message ?? String(err));
        studyPlanStore.set(null);
    } finally {
        if (seq === _studyPlanSeq) studyPlanLoadingStore.set(false);
    }
}

export const studyEffectStore = writable(/** @type {any} */ (null));
export const biasesStore = writable(/** @type {any} */ (null));
export const studyLoopLoadingStore = writable(false);
export const studyLoopErrorStore = writable(/** @type {any} */ (null));

/** Cache key of the last successful before/after and biases fetch. */
/** @type {string | null} */
let _cachedStudyLoopKey = null;
/** Sequence number of the latest before/after and biases request: a slower, older reply is dropped. */
let _studyLoopSeq = 0;

/**
 * Fetch the before/after measure of the studied families and the signed biases of the filter
 * (ADR-0079). Both replay analyses: fetched only while the dashboard, which shows them, is open.
 * @param {any} filter
 * @param {number} invalidationKey
 */
export async function refreshStudyLoop(filter, invalidationKey) {
    const key = JSON.stringify(filter) + '||' + invalidationKey;
    if (key === _cachedStudyLoopKey && get(studyEffectStore) !== null && get(biasesStore) !== null) {
        return;
    }
    _cachedStudyLoopKey = key;
    const seq = ++_studyLoopSeq;
    studyLoopLoadingStore.set(true);
    studyLoopErrorStore.set(null);
    try {
        const [effect, biases] = await Promise.all([ComputeStudyEffect(filter), ComputeDirectionalBiases(filter)]);
        if (seq !== _studyLoopSeq) return;
        studyEffectStore.set(effect);
        biasesStore.set(biases);
    } catch (err) {
        if (seq !== _studyLoopSeq) return;
        _cachedStudyLoopKey = null; // allow retry on error
        studyLoopErrorStore.set(/** @type {any} */ (err)?.message ?? String(err));
        studyEffectStore.set(null);
        biasesStore.set(null);
    } finally {
        if (seq === _studyLoopSeq) studyLoopLoadingStore.set(false);
    }
}

export const trainingStatsStore = writable(/** @type {any} */ (null));
export const trainingStatsLoadingStore = writable(false);
export const trainingStatsErrorStore = writable(/** @type {any} */ (null));

/** The calendar window the training series are folded by: `week` or `month`. */
export const trainingWindowStore = writable('week');

/** Cache key of the last successful training-stats fetch. */
/** @type {string | null} */
let _cachedTrainingKey = null;
/** Sequence number of the latest training-stats request: a slower, older reply is dropped. */
let _trainingSeq = 0;

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
 * @param {any} filter          - StatsFilter object (it restricts the matches only)
 * @param {string} invalidationKey - value of statsInvalidationKeyStore
 * @param {string} window          - `week` or `month`
 */
export async function refreshTrainingStats(filter, invalidationKey, window) {
    const key = JSON.stringify(filter) + '||' + invalidationKey + '||' + window;
    if (key === _cachedTrainingKey && get(trainingStatsStore) !== null) {
        return;
    }
    _cachedTrainingKey = key;
    const seq = ++_trainingSeq;
    trainingStatsLoadingStore.set(true);
    trainingStatsErrorStore.set(null);
    try {
        const training = await ComputeTrainingStats(filter, window);
        if (seq !== _trainingSeq) return;
        trainingStatsStore.set(training);
    } catch (err) {
        if (seq !== _trainingSeq) return;
        _cachedTrainingKey = null; // allow retry on error
        trainingStatsErrorStore.set(/** @type {any} */ (err)?.message ?? String(err));
        trainingStatsStore.set(null);
    } finally {
        if (seq === _trainingSeq) trainingStatsLoadingStore.set(false);
    }
}
