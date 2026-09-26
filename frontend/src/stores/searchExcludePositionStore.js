import { writable, get } from 'svelte/store';

// A fresh, empty position (all points empty) as a checker-structure template; new points array
// on each call.
export function emptySearchBoardPosition() {
    return {
        id: 0,
        board: {
            points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })),
            bearoff: [15, 15]
        },
        cube: { owner: -1, value: 0 },
        dice: [3, 1],
        score: [-1, -1],
        player_on_roll: 0,
        decision_type: 0,
        has_jacoby: 0,
        max_cube: 0,
        has_beaver: 0
    };
}

// Holds the "Sauf" (exclude) checker structure edited in the Search panel.
export const searchExcludePositionStore = writable(emptySearchBoardPosition());

// Which structure the Search board is currently editing: 'include' or 'exclude'.
// Read by the board container to show a red cue while editing the exclude structure.
export const searchStructureModeStore = writable('include');

// True while the Search panel builds a take/pass query: the board edits the cube as a centred
// "offered" cube (owner -1), as take/pass positions are stored.
export const searchOfferedCubeStore = writable(false);

// boardHasCheckers reports whether a position/board template has any checker set.
/** @param {{ board?: { points?: Array<{ checkers: number, color: number } | null> } } | null | undefined} position */
export function boardHasCheckers(position) {
    const points = position?.board?.points;
    if (!points) return false;
    return points.some((p) => p != null && p.checkers > 0 && p.color >= 0);
}

// The exclude board as JSON for search history, or '' without an exclusion structure.
export function excludePositionHistoryJSON() {
    const p = get(searchExcludePositionStore);
    return boardHasCheckers(p) ? JSON.stringify(p) : '';
}
