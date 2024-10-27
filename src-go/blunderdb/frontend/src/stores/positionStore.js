import { writable } from 'svelte/store';

export const importPositionPathStore = writable(''); // probablement inutile car non utilise en dehors de importPosition
export const pastePositionTextStore = writable('');
export const currentPositionStore = writable('10'); // a remplacer par positionStore
export const listPositionStore = writable('5432'); // dedie a liste de positions

export const positionStore = writable({
    board: {
        points: Array(26).fill({ checkers: 0, color: -1 }), // 24 points + 2 bars
        bearoff: [0, 0],
    },
    cube: {
        owner: -1,
        value: 0,
    },
    dice: [0, 0],
    score: [0, 0],
    player_on_roll: -1,
    decision_type: 0,
    has_jacoby: 0,
    has_beaver: 0,
});
