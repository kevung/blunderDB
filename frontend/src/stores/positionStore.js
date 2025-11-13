import { writable } from 'svelte/store';

export const importPositionPathStore = writable(''); // probablement inutile car non utilise en dehors de importPosition
export const pastePositionTextStore = writable('');
export const positionStore = writable({
    id: 0, // Add ID field
    board: {
        points: Array(26).fill({ checkers: 0, color: -1 }), // 24 points + 2 bars
        bearoff: [15, 15],
    },
    cube: {
        owner: -1,
        value: 0,
    },
    dice: [3, 1],
    score: [-1, -1],
    player_on_roll: 0,
    decision_type: 0,
    has_jacoby: 0,
    has_beaver: 0,
});
export const positionsStore = writable([]); // Add positions store
export const positionBeforeFilterLibraryStore = writable(null); // Store position before opening filter library
export const positionIndexBeforeFilterLibraryStore = writable(-1); // Store position index before opening filter library

// Match context store - stores match move positions and current index
export const matchContextStore = writable({
    isMatchMode: false,         // Whether we're in match mode
    matchID: null,              // Current match ID
    movePositions: [],          // Array of MatchMovePosition objects
    currentIndex: 0,            // Current position index
    player1Name: '',            // Player 1 name
    player2Name: '',            // Player 2 name
});

// Last visited match store - remembers the last match and position viewed
export const lastVisitedMatchStore = writable({
    matchID: null,              // Last visited match ID
    currentIndex: 0,            // Last position index in that match
    gameNumber: 1,              // Last game number viewed
});
