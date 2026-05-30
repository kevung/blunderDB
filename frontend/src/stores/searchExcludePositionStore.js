import { writable, get } from 'svelte/store';

// emptySearchBoardPosition returns a fresh, empty position usable as a checker
// structure template (all points empty). Each call builds a new points array so
// callers never share point references.
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
        has_beaver: 0
    };
}

// Holds the "Sauf" (exclude) checker structure edited in the Search panel.
export const searchExcludePositionStore = writable(emptySearchBoardPosition());

// Which structure the Search board is currently editing: 'include' or 'exclude'.
// Read by the board container to show a red cue while editing the exclude structure.
export const searchStructureModeStore = writable('include');

// boardHasCheckers reports whether a position/board template has any checker set.
export function boardHasCheckers(position) {
    const points = position?.board?.points;
    if (!points) return false;
    return points.some((p) => p && p.checkers > 0 && p.color >= 0);
}

// excludePositionHistoryJSON returns the current exclude board as JSON for storing
// in search history, or '' when no exclusion structure is set.
export function excludePositionHistoryJSON() {
    const p = get(searchExcludePositionStore);
    return boardHasCheckers(p) ? JSON.stringify(p) : '';
}
