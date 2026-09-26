/**
 * positionLoader.js — routes Stats panel drill-downs to the analysis, match
 * and tournament views.
 *
 * loadPositionsFromSelection passes the ids as `RestrictToPositionIDs` to
 * LoadPositionIDsByFilters, which returns those still existing;
 * positionsStore fetches the positions lazily by window — the mechanism of
 * `ss` and session restore.
 */

import { get } from 'svelte/store';
// NOTE: these UI messages are translated at emission time via the non-reactive
// `translate` helper; already-displayed messages do not retranslate on language change.
import { tMsg } from '../i18n';
import { GetPositionIDsByStatsSelection, GetPositionIDsByTournament, GetPositionIDsByMatch, LoadPositionIDsByFilters } from '../../wailsjs/go/database/Database.js';
import { activeTabStore, statusBarTextStore } from '../stores/uiStore.js';
import { positionsStore } from '../stores/positionStore.js';
import { currentPositionIndexStore } from '../stores/uiStore.js';
import { selectedTournamentStore } from '../stores/tournamentStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { statsFilterStore } from '../stores/statsStore.js';

/**
 * Push a list of position IDs into the analysis view.
 *
 * @param {number[]} ids        - Ordered list of position IDs to display.
 * @param {{ focusIndex?: number }} [options]
 */
export async function loadPositionsFromSelection(ids, { focusIndex = 0 } = {}) {
    if (!get(databasePathStore)) {
        statusBarTextStore.set(tMsg('commands.noDatabaseLoaded'));
        return;
    }
    if (!ids || ids.length === 0) {
        statusBarTextStore.set(tMsg('commands.noPositionsFound'));
        return;
    }

    const restrictToPositionIDs = ids.join(',');
    const resultIds = await LoadPositionIDsByFilters({ filter: {}, restrictToPositionIDs });

    if (!resultIds || resultIds.length === 0) {
        statusBarTextStore.set(tMsg('commands.noPositionsFound'));
        return;
    }

    positionsStore.setIds(Array.isArray(resultIds) ? resultIds : []);

    const clampedIndex = Math.max(0, Math.min(focusIndex, resultIds.length - 1));
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
 * Load the worst blunders (largest error) into the analysis view, under the
 * current stats filter (Stats panel's "top_blunders" selection). Backs the
 * `blunders`/`bl` command.
 *
 * @param {number} [count] - how many to load (the `bl 50` argument). Omitted /
 *   non-positive falls back to the backend default of 10.
 */
export async function loadWorstBlunders(count) {
    const filter = get(statsFilterStore);
    const selection = { Kind: 'top_blunders' };
    if (Number.isInteger(count) && count > 0) {
        selection.LastN = count;
    }
    return loadPositionsFromStatsSelection(filter, selection);
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
