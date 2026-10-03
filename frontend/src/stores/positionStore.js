import { writable } from 'svelte/store';
import { createPositionList } from './positionList.js';

// emptyPosition() is the canonical "nothing loaded yet" position: 26 empty points
// ({checkers, color: -1}), no cube owner, no score. A factory with freshly allocated points (not
// Array(26).fill({...}), which shares one object), so callers mutating in place never collide.
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
// The browsed list (positionList.js). LoadPositionsByIDs is imported lazily, keeping the Wails
// module out of import time (tests mock it per file; only a cache miss needs it).
export const positionsStore = createPositionList({
    loader: async (ids) => {
        const { LoadPositionsByIDs } = await import('../../wailsjs/go/database/Database.js');
        return (await LoadPositionsByIDs(ids)) || [];
    }
});

// The library as a paged list: its length, id windows and ranks come from the storage contract
// (CountPositions, ListPositionIDs, IndexOfPosition), so opening a library of any size reads
// one count and the pages browsed.
const database = () => import('../../wailsjs/go/database/Database.js');
/** @type {import('./positionList.js').IdSource} */
export const librarySource = {
    growsAtEnd: true,
    count: async () => (await (await database()).CountPositions()) || 0,
    window: async (offset, limit) => (await (await database()).ListPositionIDs(offset, limit)) || [],
    indexOf: async (id) => {
        const index = await (await database()).IndexOfPosition(id);
        return Number.isInteger(index) ? index : -1;
    }
};

/**
 * A search result browsed by windows, as the library is: the backend counts it, answers a window
 * of its ids and the rank of one, and the result is never held whole. `payload` is the
 * SearchFilters it replays, kept so the list can be searched within.
 * @param {any} payload
 * @returns {import('./positionList.js').IdSource & { payload: any }}
 */
export function searchSource(payload) {
    return {
        payload,
        count: async () => (await (await database()).CountPositionsByFilters(payload)) || 0,
        window: async (offset, limit) => (await (await database()).SearchPositionIDs(payload, offset, limit)) || [],
        indexOf: async (id) => {
            const index = await (await database()).IndexOfPositionByFilters(payload, id);
            return Number.isInteger(index) ? index : -1;
        }
    };
}

/**
 * Browse the whole library; resolves to its length.
 * @param {{ reset?: boolean }} [options] reset also drops the position cache
 */
export function openLibrary(options = {}) {
    return positionsStore.setSource(librarySource, options);
}

/**
 * Every id of the browsed list other than the library: held, or read whole from a paged search
 * result — for an operation that sends the ids themselves (an export), bounded by that result.
 * @returns {Promise<number[]>}
 */
export async function listedIds() {
    const list = positionsStore.snapshotList();
    const ids = 'source' in list ? await list.source.window(0, list.length) : list.ids;
    return /** @type {number[]} */ (ids.filter((id) => id != null));
}

/** Whether the list browsed is the whole library. */
export function browsingLibrary() {
    return positionsStore.isSource(librarySource);
}

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
