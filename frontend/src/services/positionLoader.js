/**
 * positionLoader.js
 *
 * Routes drill-down actions from the Stats panel to the existing
 * analysis/match/tournament UI infrastructure.
 *
 * loadPositionsFromSelection(ids, options)
 *   Loads a pre-resolved list of position IDs into the main analysis view.
 *   The IDs are passed as `RestrictToPositionIDs` to LoadPositionsByFilters
 *   (a comma-separated string), which returns the full Position objects in the
 *   same order. This is the same mechanism used by the search sub-filter (`ss`
 *   command) and session restore.
 */

import { get } from 'svelte/store';
import {
    GetPositionIDsByStatsSelection,
    GetPositionIDsByTournament,
    GetPositionIDsByMatch,
    LoadPositionsByFilters
} from '../../wailsjs/go/main/Database.js';
import { activeTabStore, statusBarTextStore } from '../stores/uiStore.js';
import { positionsStore } from '../stores/positionStore.js';
import { currentPositionIndexStore } from '../stores/uiStore.js';
import { selectedTournamentStore } from '../stores/tournamentStore.js';
import { databasePathStore } from '../stores/databaseStore.js';

/**
 * Push a list of position IDs into the analysis view.
 *
 * @param {number[]} ids        - Ordered list of position IDs to display.
 * @param {{ focusIndex?: number }} [options]
 */
export async function loadPositionsFromSelection(ids, { focusIndex = 0 } = {}) {
    if (!get(databasePathStore)) {
        statusBarTextStore.set('No database loaded');
        return;
    }
    if (!ids || ids.length === 0) {
        statusBarTextStore.set('No positions found');
        return;
    }

    const restrictToPositionIDs = ids.join(',');
    const positions = await LoadPositionsByFilters({ filter: {}, restrictToPositionIDs });

    if (!positions || positions.length === 0) {
        statusBarTextStore.set('No positions found');
        return;
    }

    positionsStore.set(Array.isArray(positions) ? positions : []);

    const clampedIndex = Math.max(0, Math.min(focusIndex, positions.length - 1));
    // Force a re-render even if the index was already at the same value.
    currentPositionIndexStore.set(-1);
    currentPositionIndexStore.set(clampedIndex);

    activeTabStore.set('analysis');
}

/**
 * Resolve positions from a StatsFilter + SelectionSpec, then open them.
 *
 * @param {object} filter    - StatsFilter (see db_stats.go).
 * @param {object} selection - SelectionSpec (see db_stats.go).
 */
export async function loadPositionsFromStatsSelection(filter, selection) {
    const ids = await GetPositionIDsByStatsSelection(filter, selection);
    return loadPositionsFromSelection(ids);
}

/**
 * Load all positions for a given tournament ID.
 *
 * @param {number} tournamentID
 */
export async function loadPositionsFromTournament(tournamentID) {
    const ids = await GetPositionIDsByTournament(tournamentID);
    return loadPositionsFromSelection(ids);
}

/**
 * Load all positions for a given match ID.
 *
 * @param {number} matchID
 */
export async function loadPositionsFromMatch(matchID) {
    const ids = await GetPositionIDsByMatch(matchID);
    return loadPositionsFromSelection(ids);
}

/**
 * Navigate the UI to the Tournament panel and highlight the given tournament.
 *
 * @param {number} tournamentID
 */
export function openTournamentInPanel(tournamentID) {
    selectedTournamentStore.set(tournamentID);
    activeTabStore.set('tournaments');
}

/**
 * Navigate the UI to the Match panel and show the given match.
 *
 * @param {number} matchID
 */
export function openMatchInPanel(matchID) {
    // matchID is passed for future use (e.g. scroll-into-view within the panel).
    // Currently we just switch to the matches tab; the panel can pick up the ID
    // from the store added in a later sheet.
    void matchID;
    activeTabStore.set('matches');
}
