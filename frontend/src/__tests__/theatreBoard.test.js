import { describe, test, expect } from 'vitest';
import { theatreScene } from '../services/theatreBoard.js';

const position = (/** @type {number} */ onRoll) => ({
    board: { points: Array.from({ length: 26 }, (_, i) => ({ checkers: i === 6 ? 5 : 0, color: i === 6 ? 0 : -1 })), bearoff: [0, 0] },
    cube: { owner: -1, value: 0 },
    dice: [3, 1],
    score: [7, 7],
    player_on_roll: onRoll,
    decision_type: 0
});

describe('the theatre board draws what the main board draws', () => {
    test('no position, nothing to draw', () => {
        expect(theatreScene({ position: null })).toBeNull();
    });

    test('the arrows of the selected candidate, in the numbering of the side on roll', () => {
        expect(theatreScene({ position: position(0), selectedMove: '8/5 6/5' })?.moves).toEqual([
            { from: 8, to: 5, index: 0 },
            { from: 6, to: 5, index: 0 }
        ]);
        const other = theatreScene({ position: position(1), selectedMove: '8/5 bar/22' });
        expect(other?.flip).toBe(true);
        expect(other?.moves).toEqual([
            { from: 17, to: 20, index: 0 },
            { from: 25, to: 3, index: 0 }
        ]);
    });

    test('the first step played on the board takes the arrows away, as on the main board', () => {
        const play = { board: position(0).board, steps: [] };
        expect(theatreScene({ position: position(0), play, selectedMove: '8/5 6/5' })?.moves).toHaveLength(2);
        expect(theatreScene({ position: position(0), play: { ...play, steps: [{ from: 6, to: 3 }] }, selectedMove: '8/5 6/5' })?.moves).toEqual([]);
    });

    test('an armed play draws its dice in the chosen order', () => {
        const play = { board: position(0).board, steps: [], rolled: [3, 1], swapped: true };
        expect(theatreScene({ position: position(0), play, armed: true })?.position.dice).toEqual([1, 3]);
        expect(theatreScene({ position: position(0), play })?.position.dice).toEqual([3, 1]);
    });

    test('the move being played on the board replaces the checkers only', () => {
        const board = { points: [], bearoff: [1, 0] };
        const scene = theatreScene({ position: position(0), play: { board } });
        expect(scene?.position.board).toBe(board);
        expect(scene?.position.dice).toEqual([3, 1]);
    });

    test('the swapped board is mirrored, as on the main board', () => {
        const scene = theatreScene({ position: position(0), swap: true });
        expect(scene?.position.board.points[19].checkers).toBe(5);
    });
});
