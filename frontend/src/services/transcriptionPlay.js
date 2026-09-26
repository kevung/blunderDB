/**
 * transcriptionPlay.js — le coup joué SUR LE PLATEAU pendant une transcription,
 * les dés qu'il déduit, et le coup hors des règles posé d'un glissé (ADR-0052).
 *
 * Le réducteur est celui de `quizPlay.js` (ADR-0040 : « rebound unchanged »),
 * appelé et non recopié ; les coups légaux viennent de `App.LegalMoves`.
 *
 * Le jet n'est pas connu : le transcripteur suit les pions sans taper de
 * chiffre. La liste de départ est donc l'UNION des coups légaux des 21 jets,
 * chacun portant le sien ; le filtre du réducteur la réduit comme une liste
 * d'un seul jet, et les jets possibles se réduisent avec.
 *
 * Les dés se déduisent des pips, sauf ambiguïté réelle (sortie « 2/off » jouée
 * d'un 2 comme d'un 6, dé injouable). Règle, sans jamais deviner :
 *   1. un seul jet compatible avec un coup achevé → dés déduits ([deducedDice]) ;
 *   2. plusieurs jets avec un coup achevé → le triangle ne garde qu'eux et
 *      attend un clic ([choosableRolls]) ;
 *   3. aucun coup achevé → rien n'est enregistré.
 * Deviner écrirait un jet que personne n'a vu ; demander à chaque coup
 * coûterait le budget d'ux.md §4.1.
 *
 * Jet saisi (ADR-0052) : la liste de départ est ce seul jet (`rolled`), et un
 * glissé qu'aucun coup légal n'offre pose le pion où il est lâché
 * ([dragStep]) : l'état passe `free` et chaque pas s'applique tel quel
 * (`applyStep`). Le plateau obtenu devient `board_after`, gardé par le moteur
 * seulement si aucun coup légal ne l'atteint (transcript.validate). Sans jet
 * saisi, pas de coup libre : il n'y aurait aucun jet à écrire.
 */

import { newPlay, alivePlays, applyStep, barOf, playHop, resetPlay, OFF } from './quizPlay.js';
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

/** Les vingt et un jets distincts, dé fort d'abord — l'ordre du triangle. */
export const ROLLS = Object.freeze([1, 2, 3, 4, 5, 6].flatMap((high) => [1, 2, 3, 4, 5, 6].filter((low) => low <= high).map((low) => Object.freeze([high, low]))));

/**
 * La clé d'un jet, dé fort d'abord (3-1 = 1-3), étiquette de la case du triangle.
 * @param {number[]} dice
 */
export function rollKey(dice) {
    const [a, b] = dice ?? [0, 0];
    return a >= b ? `${a}${b}` : `${b}${a}`;
}

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
 * Choisir le pion à déplacer en coup libre (barre comprise) ; un second clic
 * sur le même point le désélectionne.
 * @param {any} state
 * @param {number} point
 */
export function freeSelect(state, point) {
    if (state.selected === point) return { ...state, selected: null };
    if (!hasMoverChecker(state, point)) return state;
    return { ...state, selected: point };
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
 * Le clic d'un coup libre : choisir une source, ou déplacer le pion choisi.
 * Même forme que le clic du quiz, pour une seule branche au plateau.
 * @param {any} state
 * @param {number} point
 */
export function freeClick(state, point) {
    if (state.selected === null) return freeSelect(state, point);
    const moved = freeStep(state, state.selected, point);
    return moved === state ? freeSelect(state, point) : moved;
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
 * Les jets encore compatibles avec ce qui a été joué, clés triées.
 *
 * @param {any} state
 */
export function compatibleRolls(state) {
    if (!state || state.free) return [];
    const keys = new Set();
    for (const play of /** @type {BoardPlay[]} */ (alivePlays(state))) keys.add(rollKey(play.roll));
    return [...keys].sort();
}

/**
 * Les jets dont un coup est ACHEVÉ par les pas joués, à désigner quand le
 * plateau ne tranche pas.
 * @param {any} state
 */
export function choosableRolls(state) {
    if (!state || state.free || state.steps.length === 0) return [];
    const keys = new Set();
    for (const play of /** @type {BoardPlay[]} */ (alivePlays(state))) {
        if (play.steps.length === state.steps.length) keys.add(rollKey(play.roll));
    }
    return [...keys].sort();
}

/**
 * Les deux dés déduits : un seul jet compatible avec un coup achevé. Sinon
 * `null`, et le panneau attend un pas ou un clic sur le triangle.
 *
 * @param {any} state
 * @returns {number[]|null} le jet, dé fort d'abord
 */
export function deducedDice(state) {
    if (!state || state.free || state.steps.length === 0) return null;
    const alive = /** @type {BoardPlay[]} */ (alivePlays(state));
    if (alive.length === 0) return null;
    const keys = new Set(alive.map((play) => rollKey(play.roll)));
    if (keys.size !== 1) return null;
    if (!alive.some((play) => play.steps.length === state.steps.length)) return null;
    const [high, low] = alive[0].roll;
    return [high, low];
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
