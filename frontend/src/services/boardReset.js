/**
 * The two resets of the scratch board, shared by Board (Backspace, context menu) and the Eval
 * panel's buttons: clear the board, or set the checkers of a new game. Only the position moves;
 * the Eval panel re-evaluates what positionStore holds, so no analysis outlives the change.
 */
import { positionStore } from '../stores/positionStore';
import { applyStartingCheckers } from '../utils/boardInteractions.js';

/** EDIT's clear: empty board, 7-away score, dice 3-1. */
export function resetBoard() {
    positionStore.update((pos) => {
        pos.board.points.forEach((point) => (point.checkers = 0));
        pos.board.bearoff = [15, 15];
        pos.cube.value = 0;
        pos.cube.owner = -1;
        pos.score = [7, 7];
        pos.dice = [3, 1];
        pos.decision_type = 0;
        pos.player_on_roll = 0;
        return pos;
    });
}

// Eval's clear: blank board with enterEvalMode()'s defaults (money, no dice), not resetBoard()'s
// 7-away score, which the race table would read.
export function resetEvalBoard() {
    positionStore.update((pos) => {
        pos.board.points.forEach((point) => (point.checkers = 0));
        pos.board.bearoff = [15, 15];
        pos.cube.value = 0;
        pos.cube.owner = -1;
        pos.score = [-1, -1];
        pos.dice = [0, 0];
        pos.decision_type = 0;
        pos.player_on_roll = 0;
        return pos;
    });
}

/** The clear of the mode, then the checkers of a new game. @param {string} mode */
export function startingBoard(mode) {
    if (mode === 'EVAL') resetEvalBoard();
    else resetBoard();
    positionStore.update((pos) => applyStartingCheckers(pos));
}
