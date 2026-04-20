import { get } from 'svelte/store';
import { SaveSessionState, LoadSessionState, LoadAllPositions } from '../../wailsjs/go/main/Database.js';

import { databasePathStore } from '../stores/databaseStore.js';
import { positionsStore } from '../stores/positionStore.js';
import { currentPositionIndexStore } from '../stores/uiStore.js';
import { lastSearchStore } from '../stores/searchHistoryStore.js';
import { viewStore } from '../stores/viewStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { getSearchState, setSearchState } from './positionService.js';
import { logger } from '../utils/logger.js';

export async function saveSessionState() {
    if (!get(databasePathStore)) return;

    try {
        const positions = get(positionsStore);
        const currentPositionIndex = get(currentPositionIndexStore);
        const searchState = getSearchState();

        const positionIds = positions.map((pos) => pos.id);
        const sessionState = {
            lastSearchCommand: searchState.lastSearchCommand,
            lastSearchPosition: searchState.lastSearchPosition ? JSON.stringify(searchState.lastSearchPosition) : '',
            lastPositionIndex: currentPositionIndex,
            lastPositionIds: positionIds,
            hasActiveSearch: searchState.hasActiveSearch,
            viewsJSON: viewStore.serialize()
        };

        await SaveSessionState(sessionState);
        logger.log('Session state saved');
    } catch (error) {
        logger.error('Error saving session state:', error);
    }
}

export async function restoreSessionState() {
    try {
        const sessionState = await LoadSessionState();
        logger.log('Loaded session state:', sessionState);

        // Try restoring view tabs first
        if (sessionState && sessionState.viewsJSON) {
            const viewsRestored = await viewStore.deserialize(sessionState.viewsJSON, LoadAllPositions);
            if (viewsRestored) {
                setSearchState({
                    lastSearchCommand: sessionState.lastSearchCommand || '',
                    lastSearchPosition: sessionState.lastSearchPosition ? JSON.parse(sessionState.lastSearchPosition) : null,
                    hasActiveSearch: sessionState.hasActiveSearch || false
                });
                setStatusBarMessage('Session restored with views');
                logger.log('Session restored with views');
                return;
            }
        }

        if (sessionState && sessionState.hasActiveSearch && sessionState.lastPositionIds && sessionState.lastPositionIds.length > 0) {
            setSearchState({
                lastSearchCommand: sessionState.lastSearchCommand || '',
                lastSearchPosition: sessionState.lastSearchPosition ? JSON.parse(sessionState.lastSearchPosition) : null,
                hasActiveSearch: true
            });

            const allPositions = await LoadAllPositions();
            const positionIdSet = new Set(sessionState.lastPositionIds);
            const restoredPositions = allPositions.filter((pos) => positionIdSet.has(pos.id));

            const positionMap = new Map(restoredPositions.map((pos) => [pos.id, pos]));
            const orderedPositions = sessionState.lastPositionIds.map((id) => positionMap.get(id)).filter((pos) => pos !== undefined);

            if (orderedPositions.length > 0) {
                positionsStore.set(orderedPositions);

                let indexToRestore = sessionState.lastPositionIndex || 0;
                if (indexToRestore < 0) indexToRestore = 0;
                if (indexToRestore >= orderedPositions.length) indexToRestore = orderedPositions.length - 1;

                currentPositionIndexStore.set(-1);
                currentPositionIndexStore.set(indexToRestore);

                setStatusBarMessage(`Session restored: ${orderedPositions.length} positions, showing #${indexToRestore + 1}`);
                logger.log(`Session restored with ${orderedPositions.length} positions at index ${indexToRestore}`);
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
