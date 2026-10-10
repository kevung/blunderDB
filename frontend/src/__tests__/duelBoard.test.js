/**
 * Le Duel joué au plateau : ce que chaque clic y devient — lancer, videau, prise, et pendant le
 * coup la grammaire commune (services/boardMove.js, testée dans boardMove.test.js). Les coups légaux sont écrits à la main, comme `App.LegalMoves`
 * les rendrait : la légalité reste celle de quizPlay.js.
 */
import { describe, test, expect } from 'vitest';
import { newPlay } from '../services/quizPlay.js';
import { scoreStart } from '../services/duel.js';
import { playClickedChecker } from '../services/boardMove.js';
import { boardPress, canValidateMove, boardContext, boardPrompt, drawnDice } from '../services/duelBoard.js';

/** @param {[number, number][]} pairs */
const steps = (pairs) => pairs.map(([from, to]) => ({ from, to }));

const opening = { ...scoreStart(7, [7, 7]), dice: [3, 1], decision_type: 0 };
// 3-1 at the opening, Black (player 0) moving high to low.
/** @type {any[]} */
const plays31 = [
    {
        steps: steps([
            [8, 5],
            [6, 5]
        ])
    },
    {
        steps: steps([
            [24, 21],
            [24, 23]
        ])
    },
    {
        steps: steps([
            [13, 10],
            [10, 9]
        ])
    },
    {
        steps: steps([
            [13, 10],
            [6, 5]
        ])
    },
    {
        steps: steps([
            [8, 7],
            [8, 5]
        ])
    }
];

/** @param {any} play @param {any} [extra] */
function ctx(play, extra = {}) {
    return { awaiting: { side: 0, kind: 'move', position: opening }, human: 0, animating: false, play, prompt: null, ...extra };
}

describe('boardPress', () => {
    const cube = { awaiting: { side: 0, kind: 'cube', position: opening }, human: 0, animating: false, play: null, prompt: null };

    test('before the roll: the dice roll, the cube asks the on-board confirmation', () => {
        expect(boardPress(cube, { kind: 'die', index: 0 })).toEqual({ type: 'roll' });
        expect(boardPress(cube, { kind: 'cube' })).toEqual({ type: 'offerDouble' });
        // Anywhere on the board rolls, the cube excepted; the frame's outside keeps the Pile gesture.
        expect(boardPress(cube, { kind: 'point', point: 8 })).toEqual({ type: 'roll' });
        expect(boardPress(cube, { kind: 'none' })).toEqual({ type: 'roll' });
        expect(boardPress(cube, { kind: 'outside' })).toBeNull();
        // The confirmation is showing: the board waits for its answer.
        expect(boardPress({ ...cube, prompt: 'double' }, { kind: 'die', index: 0 })).toBeNull();
    });

    test('nothing on the Bot’s turn, while it animates, or on a cube to answer', () => {
        expect(boardPress({ ...cube, human: 1 }, { kind: 'die', index: 0 })).toBeNull();
        expect(boardPress({ ...cube, animating: true }, { kind: 'cube' })).toBeNull();
        expect(boardPress({ ...cube, awaiting: { ...cube.awaiting, kind: 'answer' } }, { kind: 'cube' })).toBeNull();
    });

    test('during a move: the dice swap while the play is unfinished, validate once it is', () => {
        let play = newPlay(opening, plays31);
        expect(boardPress(ctx(play), { kind: 'die', index: 1 })).toEqual({ type: 'swap' });
        play = playClickedChecker(play, 8, [3, 1]);
        // One die left: swapping it changes nothing.
        expect(boardPress(ctx(play), { kind: 'die', index: 1 })).toBeNull();
        play = playClickedChecker(play, 6, [3, 1]);
        expect(boardPress(ctx(play), { kind: 'die', index: 1 })).toEqual({ type: 'validate' });
        expect(boardPress(ctx(play), { kind: 'cube' })).toBeNull();
    });

    test('a checker click carries the next state, read in the order drawn', () => {
        const play = newPlay(opening, plays31);
        expect(drawnDice(ctx({ ...play, swapped: true }))).toEqual([1, 3]);
        const action = /** @type {any} */ (boardPress(ctx({ ...play, swapped: true }), { kind: 'point', point: 8 }));
        expect(action.type).toBe('play');
        expect(action.play.steps).toEqual([{ from: 8, to: 7, die: 1 }]);
        expect(boardPress(ctx(play), { kind: 'point', point: 1 })).toBeNull();
    });
});

describe('canValidateMove', () => {
    const move = (/** @type {any} */ play, /** @type {any} */ over = {}) => ({
        awaiting: { side: 0, kind: 'move', position: opening },
        human: 0,
        animating: false,
        play,
        prompt: null,
        ...over
    });
    test('only once every die is played, on the human’s own turn', () => {
        let play = newPlay(opening, plays31);
        expect(canValidateMove(move(play))).toBe(false);
        play = playClickedChecker(play, 8, [3, 1]);
        expect(canValidateMove(move(play))).toBe(false);
        play = playClickedChecker(play, 6, [3, 1]);
        expect(canValidateMove(move(play))).toBe(true);
        expect(canValidateMove(move(play, { human: 1 }))).toBe(false);
        expect(canValidateMove(move(play, { animating: true }))).toBe(false);
        expect(canValidateMove(move(play, { awaiting: { side: 0, kind: 'cube', position: opening } }))).toBe(false);
        expect(canValidateMove(move(null))).toBe(false);
    });
});

describe('boardContext', () => {
    test('on the board during the move: takes it back; with nothing played, the menu (the dice swap by a click on them)', () => {
        const play = newPlay(opening, plays31);
        expect(boardContext(ctx(play), { kind: 'point', point: 13 })).toEqual({ type: 'menu' });
        expect(boardContext(ctx(play), { kind: 'none' })).toEqual({ type: 'menu' });
        const moved = playClickedChecker(play, 8, [3, 1]);
        expect(boardContext(ctx(moved), { kind: 'point', point: 13 })).toEqual({ type: 'reset' });
        expect(boardContext(ctx(moved), { kind: 'none' })).toEqual({ type: 'reset' });
    });

    test('outside the frame, or at any other moment: the Duel’s menu', () => {
        const play = playClickedChecker(newPlay(opening, plays31), 8, [3, 1]);
        expect(boardContext(ctx(play), { kind: 'outside' })).toEqual({ type: 'menu' });
        expect(boardContext(ctx(play), { kind: 'die', index: 0 })).toEqual({ type: 'menu' });
        expect(boardContext(ctx(play, { human: 1 }), { kind: 'point', point: 8 })).toEqual({ type: 'menu' });
        expect(boardContext(ctx(null, { awaiting: { side: 0, kind: 'cube', position: opening } }), { kind: 'point', point: 8 })).toEqual({ type: 'menu' });
    });
});

describe('boardPrompt', () => {
    test('take or pass when the Bot doubles; double or cancel once the cube is clicked', () => {
        const base = { human: 0, animating: false, play: null, prompt: null };
        expect(boardPrompt({ ...base, awaiting: { side: 0, kind: 'answer' } })).toBe('answer');
        expect(boardPrompt({ ...base, awaiting: { side: 0, kind: 'cube' } })).toBeNull();
        expect(boardPrompt({ ...base, prompt: 'double', awaiting: { side: 0, kind: 'cube' } })).toBe('double');
        expect(boardPrompt({ ...base, animating: true, awaiting: { side: 0, kind: 'answer' } })).toBeNull();
    });
});
