// duelBoard.js — what a click on the board means during a Duel (ADR-0072: the Duel holds the
// board, and the board is where it is played). Pure: no store, no binding, no DOM. The checker
// legality is quizPlay.js's, itself fed by `App.LegalMoves`: nothing here decides a rule, it only
// picks which die a click spends and lets `playHop` accept or refuse it.

import { OFF, playHop, completedPlay } from './quizPlay.js';

const BLACK = 0;

/**
 * What the click fell on: a die (0 left, 1 right, as drawn), the cube, a point of the model
 * (0..25, OFF for the bear-off tray), the frame's outside, or nothing.
 * @typedef {{ kind: 'die', index: number } | { kind: 'cube' } | { kind: 'point', point: number } | { kind: 'outside' } | { kind: 'none' }} BoardHit
 */

/**
 * The Duel as the board reads it.
 * @typedef {{ awaiting: any, human: number, animating: boolean, play: any, swapped: boolean, prompt: string|null }} DuelBoardContext
 */

/**
 * The pips a step covers, a bear-off counting from its point.
 * @param {{from: number, to: number}} step
 * @param {number} mover
 */
export function stepDistance(step, mover) {
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
 * The roll in the order drawn: the Arbiter's order, or swapped.
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
 * smallest die that reaches.
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
 * Which drawn die is spent, left to right. A double is drawn as two dice of two moves each: the
 * left one is spent after two moves, the right one after four.
 * @param {any} play
 * @param {number[]} drawn the roll in the order drawn
 * @returns {boolean[]}
 */
export function usedDice(play, drawn) {
    if (!play || !(drawn[0] >= 1)) return [false, false];
    const spent = spentDice(play, drawn);
    if (drawn[0] === drawn[1]) return [spent.length >= 2, spent.length >= 4];
    const used = [false, false];
    for (const die of spent) {
        const i = drawn.findIndex((d, k) => d === die && !used[k]);
        if (i >= 0) used[i] = true;
    }
    return used;
}

/**
 * The dice still to play, left first.
 * @param {any} play
 * @param {number[]} drawn
 */
function diceLeft(play, drawn) {
    const spent = spentDice(play, drawn);
    const all = drawn[0] === drawn[1] ? [drawn[0], drawn[0], drawn[0], drawn[0]] : [...drawn];
    for (const die of spent) all.splice(all.indexOf(die), 1);
    return all;
}

/**
 * One click on a checker: it moves by the leftmost die still to play, or by the next one when
 * that die cannot move it; nothing when no die can. `playHop` keeps every step inside a legal
 * play — the larger-die and play-as-much-as-possible rules included.
 * @param {any} play a quizPlay state
 * @param {number} point the model point clicked
 * @param {number[]} drawn the roll in the order drawn
 */
export function playClickedChecker(play, point, drawn) {
    if (!play || point === OFF) return play;
    const tried = new Set();
    for (const die of diceLeft(play, drawn)) {
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
 * Is it the player's turn on the board, nothing animating?
 * @param {DuelBoardContext} ctx
 */
export function isMine(ctx) {
    return !!ctx.awaiting && ctx.awaiting.side === ctx.human && !ctx.animating;
}

/**
 * A left click on the board, as an action for the Duel to carry out, or null.
 * `roll`, `offerDouble` (asks the on-board confirmation), `swap` (the dice order), `validate`,
 * `play` (with the next quizPlay state).
 * @param {DuelBoardContext} ctx
 * @param {BoardHit} hit
 * @returns {{ type: 'roll'|'offerDouble'|'swap'|'validate' } | { type: 'play', play: any } | null}
 */
export function boardPress(ctx, hit) {
    if (!isMine(ctx) || ctx.prompt) return null;
    const kind = ctx.awaiting.kind;
    if (kind === 'cube') {
        // The Arbiter asks for a cube decision only when doubling is open: rolls without one
        // are its own (ADR-0072 rule 9).
        if (hit.kind === 'die') return { type: 'roll' };
        if (hit.kind === 'cube') return { type: 'offerDouble' };
        return null;
    }
    if (kind !== 'move' || !ctx.play) return null;
    if (hit.kind === 'die') {
        if (ctx.play.steps.length === 0) return { type: 'swap' };
        // The dice spent: a click on them confirms a finished move, as Enter does.
        return completedPlay(ctx.play) ? { type: 'validate' } : null;
    }
    if (hit.kind === 'point') {
        const next = playClickedChecker(ctx.play, hit.point, orderedDice(ctx.awaiting.position?.dice, ctx.swapped));
        return next === ctx.play ? null : { type: 'play', play: next };
    }
    return null;
}

/**
 * A right click: on the board during the player's move it takes the move back, or swaps the
 * dice when nothing is played; anywhere else, and at any other moment, the Duel's menu.
 * @param {DuelBoardContext} ctx
 * @param {BoardHit} hit
 * @returns {{ type: 'menu'|'reset'|'swap' }}
 */
export function boardContext(ctx, hit) {
    const onBoard = hit.kind === 'point' || hit.kind === 'none';
    if (!onBoard || !isMine(ctx) || ctx.prompt || ctx.awaiting.kind !== 'move' || !ctx.play) return { type: 'menu' };
    return ctx.play.steps.length > 0 ? { type: 'reset' } : { type: 'swap' };
}

/**
 * The confirmation the board shows: `double` (Double / Cancel), `answer` (Take / Pass), or null.
 * @param {DuelBoardContext} ctx
 */
export function boardPrompt(ctx) {
    if (!isMine(ctx)) return null;
    if (ctx.awaiting.kind === 'answer') return 'answer';
    if (ctx.awaiting.kind === 'cube' && ctx.prompt === 'double') return 'double';
    return null;
}
