/**
 * Le Duel joué au plateau : quel dé un clic dépense, l'inversion des dés, l'annulation, les dés
 * grisés, le videau et la prise. Les coups légaux sont écrits à la main, comme `App.LegalMoves`
 * les rendrait : la légalité reste celle de quizPlay.js.
 */
import { describe, test, expect } from 'vitest';
import { newPlay, OFF } from '../services/quizPlay.js';
import { scoreStart } from '../services/duel.js';
import { playClickedChecker, usedDice, spentDice, orderedDice, boardPress, canValidateMove, boardContext, boardPrompt, landing } from '../services/duelBoard.js';

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
    return { awaiting: { side: 0, kind: 'move', position: opening }, human: 0, animating: false, play, swapped: false, prompt: null, ...extra };
}

describe('a click on a checker', () => {
    test('plays the left die first, then the right one', () => {
        let play = newPlay(opening, plays31);
        play = playClickedChecker(play, 8, [3, 1]);
        expect(play.steps).toEqual([{ from: 8, to: 5, die: 3 }]);
        expect(usedDice(play, [3, 1])).toEqual([true, false]);
        play = playClickedChecker(play, 6, [3, 1]);
        expect(play.steps.map((/** @type {any} */ s) => [s.from, s.to])).toEqual([
            [8, 5],
            [6, 5]
        ]);
        expect(usedDice(play, [3, 1])).toEqual([true, true]);
    });

    test('swapped dice: the left die is the other one', () => {
        const play = playClickedChecker(newPlay(opening, plays31), 8, orderedDice([3, 1], true));
        expect(play.steps).toEqual([{ from: 8, to: 7, die: 1 }]);
        expect(usedDice(play, [1, 3])).toEqual([true, false]);
    });

    test('plays the other die when the left one cannot move that checker', () => {
        // From the 6-point no legal play moves a 3: the 1 is played.
        const play = playClickedChecker(newPlay(opening, plays31), 6, [3, 1]);
        expect(play.steps).toEqual([{ from: 6, to: 5, die: 1 }]);
    });

    test('a checker no die can move, or an opponent’s, does nothing', () => {
        const play = newPlay(opening, plays31);
        expect(playClickedChecker(play, 1, [3, 1])).toBe(play);
        expect(playClickedChecker(play, 2, [3, 1])).toBe(play);
    });

    test('a double takes four clicks, one die each; each drawn die greys after two', () => {
        const pos = { ...opening, dice: [2, 2] };
        /** @type {any[]} */
        const fours = [
            {
                steps: steps([
                    [24, 22],
                    [24, 22],
                    [13, 11],
                    [13, 11]
                ])
            }
        ];
        let play = newPlay(pos, fours);
        const seen = [];
        for (const point of [24, 24, 13, 13]) {
            play = playClickedChecker(play, point, [2, 2]);
            seen.push(usedDice(play, [2, 2]));
        }
        expect(play.steps).toHaveLength(4);
        expect(seen).toEqual([
            [false, false],
            [true, false],
            [true, false],
            [true, true]
        ]);
    });

    test('bears off past the home board, a die with room to spare included', () => {
        expect(landing(3, 5, 0)).toBe(OFF);
        expect(landing(22, 4, 1)).toBe(OFF);
        // A step played by a drag carries no die: the smallest one that reaches is spent.
        expect(spentDice({ mover: 0, steps: [{ from: 3, to: OFF }] }, [6, 4])).toEqual([4]);
        expect(spentDice({ mover: 1, steps: [{ from: 22, to: OFF }] }, [6, 4])).toEqual([4]);
        expect(spentDice({ mover: 0, steps: [{ from: 3, to: OFF, die: 6 }] }, [6, 4])).toEqual([6]);
    });
});

describe('boardPress', () => {
    const cube = { awaiting: { side: 0, kind: 'cube', position: opening }, human: 0, animating: false, play: null, swapped: false, prompt: null };

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

    test('during a move: the dice swap before any is played, validate once all are', () => {
        let play = newPlay(opening, plays31);
        expect(boardPress(ctx(play), { kind: 'die', index: 1 })).toEqual({ type: 'swap' });
        play = playClickedChecker(play, 8, [3, 1]);
        expect(boardPress(ctx(play), { kind: 'die', index: 1 })).toBeNull();
        play = playClickedChecker(play, 6, [3, 1]);
        expect(boardPress(ctx(play), { kind: 'die', index: 1 })).toEqual({ type: 'validate' });
        expect(boardPress(ctx(play), { kind: 'cube' })).toBeNull();
    });

    test('a checker click carries the next state, read in the order drawn', () => {
        const play = newPlay(opening, plays31);
        const action = /** @type {any} */ (boardPress(ctx(play, { swapped: true }), { kind: 'point', point: 8 }));
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
        swapped: false,
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
    test('on the board during the move: takes it back, or swaps the dice when nothing is played', () => {
        const play = newPlay(opening, plays31);
        expect(boardContext(ctx(play), { kind: 'point', point: 13 })).toEqual({ type: 'swap' });
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
        const base = { human: 0, animating: false, play: null, swapped: false, prompt: null };
        expect(boardPrompt({ ...base, awaiting: { side: 0, kind: 'answer' } })).toBe('answer');
        expect(boardPrompt({ ...base, awaiting: { side: 0, kind: 'cube' } })).toBeNull();
        expect(boardPrompt({ ...base, prompt: 'double', awaiting: { side: 0, kind: 'cube' } })).toBe('double');
        expect(boardPrompt({ ...base, animating: true, awaiting: { side: 0, kind: 'answer' } })).toBeNull();
    });
});
