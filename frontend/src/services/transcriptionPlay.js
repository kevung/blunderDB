/**
 * transcriptionPlay.js — le coup joué SUR LE PLATEAU pendant une transcription
 * et le coup hors des règles posé d'un glissé (ADR-0052, ADR-0086).
 *
 * Le réducteur est celui de `quizPlay.js`, appelé et non recopié ; les coups
 * légaux viennent de `App.LegalMoves`. Les dés sont saisis d'abord (clavier ou
 * triangle) : la liste de départ est celle du seul jet saisi (`rolled`), et un
 * glissé qu'aucun coup légal n'offre pose le pion où il est lâché ([dragStep]) :
 * l'état passe `free` et chaque pas s'applique tel quel (`applyStep`). Le
 * plateau obtenu devient `board_after`, gardé par le moteur seulement si aucun
 * coup légal ne l'atteint (transcript.validate).
 */

import { newPlay, applyStep, barOf, playHop, resetPlay, OFF } from './quizPlay.js';
import { parseMoveNotation } from '../utils/boardGeometry.js';

const WHITE = 1;

/**
 * Un coup légal de l'union, portant le jet qui le permet.
 *
 * @typedef {import('./quizPlay.js').Play & {roll: number[]}} BoardPlay
 */

/**
 * L'état du réducteur, plus le mode libre, le jet saisi (`null` sinon) et la
 * position d'origine.
 *
 * @typedef {import('./quizPlay.js').PlayState & {free: boolean, rolled: number[]|null, origin: any}} BoardPlayState
 */

/**
 * L'état de départ : l'union des coups légaux des jets de `byRoll` (réponse du
 * moteur, `{ dice, plays }` par jet ; une danse n'apporte rien). `rolled`, le
 * jet saisi, ouvre le glissé hors des règles.
 *
 * @param {any} position la position d'où le coup part, camp au trait posé
 * @param {{dice: readonly number[], plays: any[]}[]} byRoll
 * @param {{rolled?: number[]|null}} [options]
 * @returns {BoardPlayState}
 */
export function newBoardPlay(position, byRoll, { rolled = null } = {}) {
    const plays = [];
    for (const entry of byRoll ?? []) {
        const [a, b] = entry?.dice ?? [0, 0];
        const roll = a >= b ? [a, b] : [b, a];
        for (const play of entry?.plays ?? []) plays.push({ ...play, roll });
    }
    return { ...newPlay(position, plays), free: false, rolled: rolled ? [rolled[0], rolled[1]] : null, origin: position };
}

/**
 * Remet le plateau au début du tour, coup de nouveau contraint. `resetPlay`
 * rend un état neuf de quizPlay : le jet saisi et l'origine lui sont rendus ici.
 *
 * @param {any} state
 * @param {any} fallback la position à défaut d'origine (une question de quiz)
 */
export function resetBoardPlay(state, fallback) {
    if (!state) return state;
    const base = state.origin ?? fallback;
    return { ...resetPlay(state, base), free: false, rolled: state.rolled ?? null, origin: state.origin };
}

/**
 * Le coup peut-il sortir des règles ? Seulement jet connu (ADR-0052), ou déjà
 * sorti.
 *
 * @param {any} state
 */
export function canPlayFree(state) {
    return !!state && (state.free === true || Array.isArray(state.rolled));
}

/**
 * Le point porte-t-il un pion du camp qui joue ?
 *
 * @param {any} state
 * @param {number} point
 */
export function hasMoverChecker(state, point) {
    const p = state?.board?.points?.[point];
    return !!p && p.checkers > 0 && p.color === state.mover;
}

/**
 * Déplacer un pion sans rien vérifier (ADR-0044) ; seule condition, un pion à
 * prendre. Le coup reste libre jusqu'au bout.
 * @param {any} state
 * @param {number} from
 * @param {number} to
 */
export function freeStep(state, from, to) {
    if (from === to || !hasMoverChecker(state, from)) return state;
    const board = applyStep(state.board, { from, to }, state.mover);
    return { ...state, board, steps: [...state.steps, { from, to }], selected: null, free: true };
}

/**
 * Le pion lâché sur `to` après un glissé depuis `from` (ADR-0052). Un pas légal
 * reste contraint ; sinon, jet connu seulement ([canPlayFree]), le pion est
 * posé et le coup devient libre. Sans jet saisi, rien.
 *
 * @param {any} state
 * @param {number} from
 * @param {number} to
 */
export function dragStep(state, from, to) {
    if (!state || from === to) return state;
    if (!state.free) {
        const played = playHop(state, from, to);
        if (played !== state || !canPlayFree(state)) return played;
    }
    return freeStep(state, from, to);
}

/**
 * Annule le dernier pas en rejouant les autres. Pas `undoLast` de quizPlay,
 * qui perdrait le jet saisi et l'origine. Chaque pas est rejoué contraint si un
 * coup légal l'offre encore, libre sinon : défaire le seul pas hors des règles
 * rend un coup contraint.
 *
 * @param {any} state
 */
export function undoBoardStep(state) {
    if (!state || state.steps.length === 0) return state;
    const kept = state.steps.slice(0, -1);
    let next = resetBoardPlay(state, state.origin);
    for (const s of kept) next = dragStep(next, s.from, s.to);
    return next;
}

/**
 * Un point de notation (relatif au camp qui joue) dans le repère absolu :
 * l'inverse exact de `domain.pointLabel`.
 *
 * @param {number} relative
 * @param {number} mover
 */
function absolutePoint(relative, mover) {
    return mover === WHITE ? 25 - relative : relative;
}

/**
 * Les pas d'une notation (`13/7 8/7*`, `bar/22`, `6/off(2)`), par
 * `parseMoveNotation` (bar → 0, off → −1, `(n)` développés) puis passage au
 * repère absolu, que le parseur ignore.
 *
 * @param {string} text
 * @param {number} mover
 * @returns {{from: number, to: number}[]}
 */
export function stepsFromNotation(text, mover) {
    return parseMoveNotation(text).map(({ from, to }) => ({
        from: from === 0 ? barOf(mover) : absolutePoint(from, mover),
        to: to === -1 ? OFF : absolutePoint(to, mover)
    }));
}

/**
 * Le plateau laissé par ces pas, sans jugement : un pas à source vide ne
 * prend rien, et le Replay dira l'incohérence.
 *
 * @param {any} board
 * @param {{from: number, to: number}[]} steps
 * @param {number} mover
 */
export function boardAfterSteps(board, steps, mover) {
    let out = board;
    for (const step of steps ?? []) out = applyStep(out, step, mover);
    return out;
}
