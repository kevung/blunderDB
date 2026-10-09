// duelBoard.js — what a click on the board means during a Duel (ADR-0072: the Duel holds the
// board, and the board is where it is played). Pure: no store, no binding, no DOM. The checker
// legality is quizPlay.js's, itself fed by `App.LegalMoves`: nothing here decides a rule, it only
// picks which die a click spends and lets `playHop` accept or refuse it.

import { completedPlay } from './quizPlay.js';
import { orderedDice, playClickedChecker } from './boardMove.js';

// The grammar itself is boardMove.js's, shared by every mode that takes a checker play.
export { stepDistance, landing, orderedDice, spentDice, usedDice, playClickedChecker } from './boardMove.js';

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
        // Any click on the board rolls, the cube excepted (it proposes the double); the frame's
        // outside keeps the Pile gesture.
        if (hit.kind === 'cube') return { type: 'offerDouble' };
        if (hit.kind === 'outside') return null;
        return { type: 'roll' };
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
 * Space: validates the move once every playable die is played; nothing else, nowhere else.
 * @param {DuelBoardContext} ctx
 */
export function canValidateMove(ctx) {
    return isMine(ctx) && !ctx.prompt && ctx.awaiting.kind === 'move' && !!ctx.play && !!completedPlay(ctx.play);
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
