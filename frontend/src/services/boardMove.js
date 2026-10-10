// boardMove.js — the one grammar a checker play is entered with on the board (ADR-0086): a click
// on a checker moves it by the first die still to play, a click on the dice validates a finished
// play or swaps the dice left, a right click on the board takes the play back. Pure: no store, no
// binding, no DOM. Legality stays quizPlay.js's (`playHop`, fed by `App.LegalMoves`): nothing here
// decides a rule, it only picks which die a click spends and lets `playHop` accept or refuse it.
// A free play (ADR-0052 §4, out of the rules) moves by the same die without `playHop`.

import { OFF, playHop, completedPlay, applyStep } from './quizPlay.js';

const BLACK = 0;

/**
 * The pips a step covers, a bear-off counting from its point.
 * @param {{from: number, to: number}} step
 * @param {number} mover
 */
function stepDistance(step, mover) {
    if (step.to === OFF) return mover === BLACK ? step.from : 25 - step.from;
    return mover === BLACK ? step.from - step.to : step.to - step.from;
}

/**
 * Where a checker on `point` lands with `die`: past the home board is off.
 * @param {number} point
 * @param {number} die
 * @param {number} mover
 */
export function landing(point, die, mover) {
    const to = mover === BLACK ? point - die : point + die;
    return to <= 0 || to >= 25 ? OFF : to;
}

/**
 * The roll in the order drawn: as rolled, or swapped. The order is only the next click's: the
 * recorded roll is a pair and never rewritten.
 * @param {number[]} dice
 * @param {boolean} swapped
 */
export function orderedDice(dice, swapped) {
    const d = [dice?.[0] ?? 0, dice?.[1] ?? 0];
    return swapped ? [d[1], d[0]] : d;
}

/**
 * The die each played step spent, in the order played. A step played by a click carries its die;
 * one played by a drag is read from its distance, a bear-off with room to spare taking the
 * smallest die that reaches. A free step no die covers is left out.
 * @param {any} play a quizPlay state
 * @param {number[]} dice the roll
 * @returns {number[]}
 */
export function spentDice(play, dice) {
    const left = dice[0] === dice[1] ? [dice[0], dice[0], dice[0], dice[0]] : [dice[0], dice[1]];
    /** @type {number[]} */
    const spent = [];
    for (const step of play?.steps ?? []) {
        const distance = stepDistance(step, play.mover);
        let i = step.die ? left.indexOf(step.die) : left.indexOf(distance);
        if (i < 0 && step.to === OFF) {
            const reaching = left.filter((d) => d >= distance).sort((a, b) => a - b);
            i = reaching.length ? left.indexOf(reaching[0]) : -1;
        }
        if (i < 0) continue;
        spent.push(left[i]);
        left.splice(i, 1);
    }
    return spent;
}

/**
 * The dice still to play, in the order drawn.
 * @param {any} play
 * @param {number[]} drawn
 * @returns {number[]}
 */
export function diceLeft(play, drawn) {
    if (!(drawn[0] >= 1)) return [];
    const spent = spentDice(play, drawn);
    const all = drawn[0] === drawn[1] ? [drawn[0], drawn[0], drawn[0], drawn[0]] : [...drawn];
    for (const die of spent) all.splice(all.indexOf(die), 1);
    return all;
}

/**
 * Is the play finished, so that a click on the dice validates it? A legal play: when the engine
 * has a play of exactly the steps played — a forced partial play and a dance (no legal play)
 * included. A free play: when no die is left, or when a step no die covers was dropped (its dice
 * can no longer be read).
 * @param {any} play
 * @param {number[]} drawn
 */
export function playIsDone(play, drawn) {
    if (!play) return false;
    if (play.free) {
        if (play.steps.length === 0) return false;
        return diceLeft(play, drawn).length === 0 || spentDice(play, drawn).length < play.steps.length;
    }
    return (play.plays?.length ?? 0) === 0 || completedPlay(play) !== null;
}

/**
 * How much of each drawn die is spent, 0 to 1: the drawing's veil. A single die greys when its
 * step is played; each die of a double stands for two steps and is half veiled after the first.
 * Once the play is done every die is grey — a forced partial play's unplayable dice and a dance's
 * two included: grey dice always say "a click validates".
 * @param {any} play
 * @param {number[]} drawn the roll in the order drawn
 * @returns {number[]}
 */
export function diceShade(play, drawn) {
    if (!play || !(drawn[0] >= 1)) return [0, 0];
    if (playIsDone(play, drawn)) return [1, 1];
    if (drawn[0] === drawn[1]) {
        const n = spentDice(play, drawn).length;
        return [Math.min(n, 2) / 2, Math.max(0, Math.min(n - 2, 2)) / 2];
    }
    const shade = [0, 0];
    for (const die of spentDice(play, drawn)) {
        const i = drawn.findIndex((d, k) => d === die && !shade[k]);
        if (i >= 0) shade[i] = 1;
    }
    return shade;
}

/**
 * One click on a checker: it moves by the first die still to play, or by the next one when that
 * die cannot move it; nothing when no die can. `playHop` keeps every step inside a legal play —
 * the larger-die and play-as-much-as-possible rules included. A free play moves by the first die
 * left, wherever it lands.
 * @param {any} play a quizPlay state
 * @param {number} point the model point clicked
 * @param {number[]} drawn the roll in the order drawn
 */
export function playClickedChecker(play, point, drawn) {
    if (!play || point === OFF) return play;
    const left = diceLeft(play, drawn);
    if (play.free) return freeAdvance(play, point, left[0]);
    const tried = new Set();
    for (const die of left) {
        if (tried.has(die)) continue;
        tried.add(die);
        const next = playHop({ ...play, selected: null }, point, landing(point, die, play.mover));
        if (next.steps.length > play.steps.length) {
            const steps = next.steps.map((s, i) => (i === next.steps.length - 1 ? { ...s, die } : s));
            return { ...next, steps };
        }
    }
    return play;
}

/**
 * A free play's click: the checker moves by `die`, no rule asked.
 * @param {any} play
 * @param {number} point
 * @param {number|undefined} die
 */
function freeAdvance(play, point, die) {
    const p = play.board?.points?.[point];
    if (!die || !p || p.checkers <= 0 || p.color !== play.mover) return play;
    const step = { from: point, to: landing(point, die, play.mover) };
    const board = applyStep(play.board, step, play.mover);
    return { ...play, board, steps: [...play.steps, { ...step, die }], selected: null };
}

/**
 * A click on the dice: `validate` a finished play, else `swap` the two dice still to play. Nothing
 * when the swap would change nothing — a double, or a single die left.
 * @param {any} play
 * @param {number[]} drawn the roll in the order drawn
 * @returns {'validate'|'swap'|null}
 */
export function diceClick(play, drawn) {
    if (!play) return null;
    if (playIsDone(play, drawn)) return 'validate';
    const left = diceLeft(play, drawn);
    return left.length === 2 && left[0] !== left[1] ? 'swap' : null;
}

/**
 * A right click on the board: `reset` takes back every step of the play in progress; with
 * nothing played, null — the board's menu opens, as outside any play.
 * @param {any} play
 * @returns {'reset'|null}
 */
export function boardRightClick(play) {
    return play && play.steps.length > 0 ? 'reset' : null;
}
