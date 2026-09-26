// filterLibraryService — the saved-filter library outside the search panel:
// loading, pinning, running a saved filter from anywhere.
//
// Pinned filters: chips in the search panel and Alt+1…9 elsewhere, stored per
// database (Database.SetFilterPinned), in library order (Alt+1 = oldest).
//
// A run asks exactly what the panel's double-click asks (replaySearchArgs,
// stored board, "Sauf" structure). The structure the panel gets from EDIT mode
// is passed explicitly (queryBoard), so a run while browsing keeps it.

import { get } from 'svelte/store';
import { LoadFilters, SetFilterPinned, LoadEditPosition, LoadExcludePosition } from '../../wailsjs/go/database/Database.js';
import { filterLibraryStore } from '../stores/filterLibraryStore.js';
import { positionStore } from '../stores/positionStore.js';
import { searchExcludePositionStore, searchStructureModeStore, emptySearchBoardPosition } from '../stores/searchExcludePositionStore.js';
import { statusBarModeStore, statusBarTextStore } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { replaySearchArgs } from './searchFilterService.js';
import { loadPositionsByFilters } from './positionService.js';
import { tMsg } from '../i18n';
import { logger } from '../utils/logger.js';

/** @typedef {{ id: number, name: string, command: string, pinned?: boolean }} SavedFilter */

/**
 * Reload the library into filterLibraryStore and return it.
 * @returns {Promise<SavedFilter[]>}
 */
export async function loadFilterLibrary() {
    try {
        const lib = (await LoadFilters()) || [];
        filterLibraryStore.set(lib);
        return lib;
    } catch (_error) {
        filterLibraryStore.set([]);
        return [];
    }
}

/**
 * The pinned filters of a library, in library order.
 * @param {SavedFilter[]} library
 * @returns {SavedFilter[]}
 */
export function pinnedFilters(library) {
    return (library || []).filter((f) => f.pinned);
}

/**
 * Pin or unpin a filter, then reload the library.
 * @param {SavedFilter} filter
 * @param {boolean} pinned
 */
export async function setFilterPinned(filter, pinned) {
    try {
        await SetFilterPinned(filter.id, pinned);
    } catch (error) {
        logger.error('Error pinning filter:', error);
    }
    await loadFilterLibrary();
}

/**
 * Run a saved filter the way the panel's double-click does, from any NORMAL
 * or EDIT screen.
 * @param {SavedFilter} filter
 */
export async function runSavedFilter(filter) {
    const mode = get(statusBarModeStore);
    if (mode !== 'NORMAL' && mode !== 'EDIT') {
        statusBarTextStore.set(tMsg('commands.searchRequiresMode'));
        return;
    }
    const replay = replaySearchArgs(filter.command);
    if (!replay) return;

    const editPosition = await LoadEditPosition(filter.name);
    const excludePosition = await LoadExcludePosition(filter.name);
    const board = editPosition ? JSON.parse(editPosition) : null;

    // A bare `like` ranks against the board it was saved with; a filter
    // that did not keep one has no target left.
    if (replay.f.likeFilter && !replay.f.likeTargetId && !board) {
        statusBarTextStore.set(tMsg('similar.noPosition'));
        return;
    }

    // In EDIT the board on screen is the structure being edited: show the
    // filter's, as the panel does. Elsewhere the board shows a position and
    // the results about to replace it; the structure travels as queryBoard.
    if (mode === 'EDIT') {
        if (board) positionStore.set(board);
        // The board was showing the "Sauf" structure: it must not become the
        // included one when the mode switches back.
        else if (get(searchStructureModeStore) === 'exclude') positionStore.set(emptySearchBoardPosition());
    }
    searchStructureModeStore.set('include');
    let exclude = emptySearchBoardPosition();
    if (excludePosition) {
        try {
            exclude = JSON.parse(excludePosition);
        } catch (_e) {
            /* keep the empty board */
        }
    }
    searchExcludePositionStore.set(exclude);

    await loadPositionsByFilters({ ...replay.args, ...(board ? { queryBoard: board } : {}) });
}

/**
 * Alt+n: run the n-th pinned filter (1-based). The library is read fresh, so
 * the latest pin wins.
 * @param {number} n
 */
export async function runPinnedFilter(n) {
    if (!get(databasePathStore)) {
        statusBarTextStore.set(tMsg('commands.noDatabaseOpened'));
        return;
    }
    const pinned = pinnedFilters(await loadFilterLibrary());
    const filter = pinned[n - 1];
    if (!filter) {
        statusBarTextStore.set(tMsg('search.noPinnedAt', { n }));
        return;
    }
    await runSavedFilter(filter);
}
