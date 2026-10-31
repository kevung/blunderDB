// Scripted table scenes for the automata, each with its ground truth: what happened and when,
// as an annotator would mark it frame by frame. The journals under this directory are these
// scenes played through `simulate`; the truth is what F1 and the median offset are measured on.

import { along } from './board.js';

/** @typedef {import('./board.js').SceneState} SceneState */
/** @typedef {{ type: string, mediaTime: number }} Truth */

const START = [
    { x: 274, y: 154 },
    { x: 274, y: 142 },
    { x: 192, y: 26 },
    { x: 192, y: 38 },
    { x: 192, y: 50 },
    { x: 46, y: 26 },
    { x: 46, y: 38 },
    { x: 120, y: 154 },
    { x: 120, y: 142 }
];

/** Faces change on every video frame while a die tumbles. */
const tumbling = (/** @type {number} */ t, /** @type {number} */ k) => 1 + ((Math.floor(t * 30) + k * 3) % 6);

/**
 * A die thrown at `t0`, tumbling along `path` until `t1`, then resting at the path's end.
 * @param {number} t
 * @param {number} t0
 * @param {number} t1
 * @param {[number, number, number][]} path
 * @param {number} k
 */
function die(t, t0, t1, path, k) {
    const p = along(path, t);
    return { ...p, face: t < t1 ? tumbling(t, k) : 1 + k };
}

/**
 * Two turns: a throw on the right half, a move, a pickup, then the opponent's throw on the left.
 * @param {number} t
 * @returns {SceneState}
 */
function twoTurns(t) {
    /** @type {SceneState} */
    const s = { checkers: START.map((c) => ({ ...c })), dice: [], hands: [] };
    const hands = /** @type {NonNullable<SceneState['hands']>} */ (s.hands);
    const dice = /** @type {NonNullable<SceneState['dice']>} */ (s.dice);
    const checkers = /** @type {NonNullable<SceneState['checkers']>} */ (s.checkers);
    // Throw 1: the hand comes in from below, shakes, lets go at 3.8 and is gone by 4.1.
    if (t >= 3.0 && t < 4.1)
        hands.push(
            along(
                [
                    [3.0, 200, 175],
                    [3.4, 200, 100],
                    [3.5, 206, 96],
                    [3.6, 198, 102],
                    [3.7, 206, 96],
                    [3.8, 200, 110],
                    [4.1, 200, 180]
                ],
                t
            )
        );
    // The dice, from 3.8 to their rest at 4.6, until the pickup at 10.0.
    if (t >= 3.8 && t < 10.0)
        dice.push(
            die(
                t,
                3.8,
                4.6,
                [
                    [3.8, 212, 112],
                    [4.6, 236, 88]
                ],
                0
            ),
            die(
                t,
                3.8,
                4.6,
                [
                    [3.8, 222, 116],
                    [4.6, 252, 98]
                ],
                1
            )
        );
    // The move, 7.0–9.0: the hand sweeps over the right half and carries two checkers.
    if (t >= 7.0 && t < 9.0) {
        const phase = (t - 7.0) * 6;
        hands.push({ x: 180 + 30 * Math.sin(phase), y: Math.max(110, 175 - (t - 7.0) * 160) + 8 * Math.cos(phase * 1.3) });
    }
    if (t >= 7.5) checkers[0] = { x: 212, y: 154 };
    if (t >= 8.3) checkers[1] = { x: 212, y: 142 };
    // The pickup, 9.5–10.3.
    if (t >= 9.5 && t < 10.3)
        hands.push(
            along(
                [
                    [9.5, 236, 175],
                    [9.8, 226, 80],
                    [9.9, 232, 84],
                    [10.0, 226, 80],
                    [10.3, 236, 180]
                ],
                t
            )
        );
    // Throw 2, on the left half: the hand comes in from above at 13.0, lets go at 13.7.
    if (t >= 13.0 && t < 13.95)
        hands.push(
            along(
                [
                    [13.0, 90, -30],
                    [13.35, 90, 40],
                    [13.45, 96, 44],
                    [13.55, 86, 40],
                    [13.65, 94, 46],
                    [13.95, 90, -40]
                ],
                t
            )
        );
    if (t >= 13.7)
        dice.push(
            die(
                t,
                13.7,
                14.3,
                [
                    [13.7, 100, 70],
                    [14.3, 84, 82]
                ],
                2
            ),
            die(
                t,
                13.7,
                14.3,
                [
                    [13.7, 110, 72],
                    [14.3, 108, 92]
                ],
                3
            )
        );
    return s;
}

/**
 * The opening roll: one die per half, thrown together.
 * @param {number} t
 * @returns {SceneState}
 */
function opening(t) {
    /** @type {SceneState} */
    const s = { checkers: START.map((c) => ({ ...c })), dice: [], hands: [] };
    const dice = /** @type {NonNullable<SceneState['dice']>} */ (s.dice);
    if (t >= 2.6)
        dice.push(
            die(
                t,
                2.6,
                3.2,
                [
                    [2.6, 150, 90],
                    [3.2, 110, 86]
                ],
                0
            ),
            die(
                t,
                2.6,
                3.2,
                [
                    [2.6, 170, 92],
                    [3.2, 220, 94]
                ],
                1
            )
        );
    return s;
}

/**
 * A game ends and the board is set up again: hands collect and replace the checkers for four
 * seconds; the analysis started on the initial position.
 * @param {number} t
 * @returns {SceneState}
 */
function newGame(t) {
    /** @type {SceneState} */
    const s = { checkers: START.map((c) => ({ ...c })), dice: [], hands: [] };
    const hands = /** @type {NonNullable<SceneState['hands']>} */ (s.hands);
    // Mid-game from 1 s: most checkers are elsewhere (the game was played off-camera).
    if (t >= 1.0 && t < 8.5) s.checkers = START.map((c, i) => ({ x: c.x + (i % 2 ? 30 : -24), y: 90 + ((i * 13) % 50) }));
    // A last throw on the right half.
    if (t >= 3.0 && t < 5.5)
        /** @type {NonNullable<SceneState['dice']>} */ (s.dice).push(
            die(
                t,
                3.0,
                3.6,
                [
                    [3.0, 200, 100],
                    [3.6, 240, 90]
                ],
                0
            )
        );
    // The reset: two hands sweep over the board from 5.5 to 9.0; dice and checkers go back.
    if (t >= 5.5 && t < 9.0) {
        const phase = (t - 5.5) * 5;
        hands.push({ x: 70 + 40 * Math.sin(phase), y: 70 + 30 * Math.cos(phase) }, { x: 200 + 40 * Math.cos(phase * 1.1), y: 60 + 30 * Math.sin(phase) });
    }
    return s;
}

/**
 * The new game again, with a camera cut at 2.0–2.4: the backgrounds are reset on the cut, the
 * start template is lost, and the game automaton must stay silent.
 * @param {number} t
 */
function cutThenReset(t) {
    return { ...newGame(t), cut: t >= 2.0 && t < 2.4 };
}

/** Twenty seconds of a still table: nothing to report, and the analysis stays coarse. */
function still() {
    return { checkers: START.map((c) => ({ ...c })) };
}

/** @type {Record<string, { script: (t: number) => SceneState, duration: number, description: string, truth: Truth[] }>} */
export const SCENARIOS = {
    'two-turns': {
        script: twoTurns,
        duration: 17,
        description: 'Throw on the right half (hand 3.0-4.1, dice 3.8-4.6), a move 7.0-9.0, a pickup 9.5-10.3, the opponent throws on the left (hand 13.0-13.95, dice 13.7-14.3).',
        truth: [
            { type: 'hands.in', mediaTime: 3.0 },
            { type: 'dice.thrown', mediaTime: 3.8 },
            { type: 'hands.out', mediaTime: 4.1 },
            { type: 'dice.rest', mediaTime: 4.6 },
            { type: 'hands.in', mediaTime: 7.0 },
            { type: 'hands.out', mediaTime: 9.0 },
            { type: 'hands.in', mediaTime: 9.5 },
            { type: 'dice.pickup', mediaTime: 9.5 },
            { type: 'hands.out', mediaTime: 10.3 },
            { type: 'hands.in', mediaTime: 13.0 },
            { type: 'dice.thrown', mediaTime: 13.7 },
            { type: 'hands.out', mediaTime: 13.95 },
            { type: 'dice.rest', mediaTime: 14.3 }
        ]
    },
    opening: {
        script: opening,
        duration: 6,
        description: 'Opening roll: one die per half, thrown at 2.6, at rest at 3.2.',
        truth: [
            { type: 'dice.thrown', mediaTime: 2.6 },
            { type: 'dice.rest', mediaTime: 3.2 }
        ]
    },
    'new-game': {
        script: newGame,
        duration: 12,
        description: 'The analysis starts on the initial position; mid-game board from 1 s, last throw 3.0-3.6, hands reset the board 5.5-9.0, initial position from 9.0 (checkers back at 8.5).',
        truth: [
            { type: 'dice.thrown', mediaTime: 3.0 },
            { type: 'dice.rest', mediaTime: 3.6 },
            { type: 'hands.in', mediaTime: 5.5 },
            { type: 'game.end', mediaTime: 5.5 },
            { type: 'hands.out', mediaTime: 9.0 },
            { type: 'game.start', mediaTime: 9.0 }
        ]
    },
    'cut-then-reset': {
        script: cutThenReset,
        duration: 12,
        description: 'The new game with a camera cut at 2.0-2.4: the start template is lost, no game event may follow.',
        truth: [
            { type: 'frame.uncertain', mediaTime: 2.0 },
            { type: 'frame.uncertain', mediaTime: 2.4 },
            { type: 'dice.thrown', mediaTime: 3.0 },
            { type: 'dice.rest', mediaTime: 3.6 },
            { type: 'hands.in', mediaTime: 5.5 },
            { type: 'hands.out', mediaTime: 9.0 }
        ]
    },
    still: {
        script: still,
        duration: 20,
        description: 'A still table for twenty seconds.',
        truth: []
    }
};
