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
