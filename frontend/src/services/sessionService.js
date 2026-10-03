import { tMsg } from '../i18n';
import { get } from 'svelte/store';
import { SaveSessionState, LoadSessionState, LoadPositionIDsByFilters, RankPositionIDsByFilters } from '../../wailsjs/go/database/Database.js';
import { GetLikeLimit } from '../../wailsjs/go/main/Config.js';

import { databasePathStore } from '../stores/databaseStore.js';
import { currentPositionIndexStore } from '../stores/uiStore.js';
import { lastSearchStore } from '../stores/searchHistoryStore.js';
import { viewStore } from '../stores/viewStore.js';
import { librarySource, searchSource, positionsStore } from '../stores/positionStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { getSearchState, setSearchState } from './positionService.js';
import { logger } from '../utils/logger.js';

// The session persists a descriptor (the search that produced each list, the id of the position
// shown), never the ids: its size does not depend on the number of positions.
export async function saveSessionState() {
    if (!get(databasePathStore)) return;

    try {
        const currentPositionIndex = get(currentPositionIndexStore);
        const searchState = getSearchState();

        const sessionState = {
            lastSearchCommand: searchState.lastSearchCommand,
            lastSearchPosition: searchState.lastSearchPosition ? JSON.stringify(searchState.lastSearchPosition) : '',
            lastPositionIndex: currentPositionIndex,
            lastPositionIds: [],
            hasActiveSearch: searchState.hasActiveSearch,
            viewsJSON: viewStore.serialize()
        };

        await SaveSessionState(sessionState);
        logger.log('Session state saved');
    } catch (error) {
        logger.error('Error saving session state:', error);
    }
}

// Replays a list origin into a positionList snapshot: the library as a paged list (its length
// only), a plain search as a paged list opened on its first window — as a search shows it, its
// count left to the view once on screen (viewStore settle) — a ranked or restricted search as its
// ids, bounded by the ranking limit or by the list it searched within. An origin that no longer
// yields anything (positions deleted, search now empty) falls back to the library.
async function resolveOriginList(origin) {
    if (origin && origin.kind === 'search' && origin.payload) {
        try {
            const payload = origin.payload;
            if (!payload.likeFilter && !payload.restrictToPositionIDs) {
                const source = searchSource(payload);
                const firstPage = await source.window(0, positionsStore.firstPageSize());
                // A short window is the whole result: its length is known.
                const provisional = firstPage.length >= positionsStore.firstPageSize();
                if (firstPage.length > 0) return { source, length: firstPage.length, provisional, firstPage };
            } else {
                const ids = payload.likeFilter ? ((await RankPositionIDsByFilters(payload, (await GetLikeLimit()) || 0)) || []).map((n) => n.id) : await LoadPositionIDsByFilters(payload);
                if (ids && ids.length > 0) return { ids };
            }
        } catch (error) {
            logger.error('Error replaying the saved search:', error);
        }
    }
    return { source: librarySource, length: await librarySource.count() };
}

export async function restoreSessionState() {
    try {
        const sessionState = await LoadSessionState();
        logger.log('Loaded session state:', sessionState);

        if (sessionState && sessionState.viewsJSON) {
            const viewsRestored = await viewStore.deserialize(sessionState.viewsJSON, resolveOriginList);
            if (viewsRestored) {
                setSearchState({
                    lastSearchCommand: sessionState.lastSearchCommand || '',
                    lastSearchPosition: sessionState.lastSearchPosition ? JSON.parse(sessionState.lastSearchPosition) : null,
                    hasActiveSearch: sessionState.hasActiveSearch || false
                });
                setStatusBarMessage(tMsg('status.sessionRestoredViews'));
                logger.log('Session restored with views');
                return;
            }
        }

        // No session to restore - load all positions
        setSearchState({
            lastSearchCommand: '',
            lastSearchPosition: null,
            hasActiveSearch: false
        });
        lastSearchStore.set(null);
        const { loadAllPositions } = await import('./positionService.js');
        await loadAllPositions();
    } catch (error) {
        logger.error('Error restoring session state:', error);
        const { loadAllPositions } = await import('./positionService.js');
        await loadAllPositions();
    }
}
