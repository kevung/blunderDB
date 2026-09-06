import { writable } from 'svelte/store';
import { createPositionList } from './positionList.js';

// emptyPosition() is the one canonical "nothing loaded yet" position: 26
// empty points ({checkers, color: -1} — the shape Board.svelte and the rest
// of the app expect everywhere else), no cube owner, no score. Every point
// object is freshly allocated (not Array(26).fill({...}), which would give
// every slot the *same* object reference) so a future per-point mutation
// can't silently corrupt all 26 points at once.
//
// This is a factory, not a shared constant, so each caller gets its own
// object graph — callers that then mutate the result in place (as
// positionStore's own set() consumers do) never see each other's edits.
export function emptyPosition() {
    return {
        id: 0,
        board: {
            points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), // 24 points + 2 bars
            bearoff: [15, 15]
        },
        cube: {
            owner: -1,
            value: 0
        },
        dice: [3, 1],
        score: [-1, -1],
        player_on_roll: 0,
        decision_type: 0,
        has_jacoby: 0,
        max_cube: 0,
        has_beaver: 0
    };
}

export const pastePositionTextStore = writable('');
export const positionStore = writable(emptyPosition());
// The browsed list: ids plus a window cache (see positionList.js). Positions
// are fetched through LoadPositionsByIDs; the binding is imported lazily so
// this store stays free of the Wails module at import time (tests mock it
// per file, and a cache miss is the only path that needs it).
export const positionsStore = createPositionList({
    loader: async (ids) => {
        const { LoadPositionsByIDs } = await import('../../wailsjs/go/database/Database.js');
        return (await LoadPositionsByIDs(ids)) || [];
    }
});
export const positionBeforeFilterLibraryStore = writable(null); // Store position before opening filter library
export const positionIndexBeforeFilterLibraryStore = writable(-1); // Store position index before opening filter library

// Match context store - stores match move positions and current index
export const matchContextStore = writable({
    isMatchMode: false, // Whether we're in match mode
    matchID: null, // Current match ID
    movePositions: [], // Array of MatchMovePosition objects
    currentIndex: 0, // Current position index
    player1Name: '', // Player 1 name
    player2Name: '' // Player 2 name
});

// Last visited match store - remembers the last match and position viewed
export const lastVisitedMatchStore = writable({
    matchID: null, // Last visited match ID
    currentIndex: 0, // Last position index in that match
    gameNumber: 1 // Last game number viewed
});

// Internal clipboard for copy/paste position to search board
export const clipboardPositionStore = writable(null);
