/**
 * transcriptionPlay.js — le coup joué SUR LE PLATEAU pendant une transcription,
 * et les dés qu'il déduit (T2.3, puis le déplacement libre de T2.4).
 *
 * # Ce qui n'est pas écrit ici
 *
 * Le réducteur. Il existe, il s'appelle `services/quizPlay.js`, il a été écrit
 * pour le quiz (#294) et l'ADR-0040 le dit « rebound unchanged » : ce fichier
 * l'appelle et n'en recopie pas une ligne. Aucune règle du backgammon n'est
 * donc écrite ici non plus — les coups légaux viennent de `App.LegalMoves`,
 * comme partout ailleurs.
 *
 * # Ce qui est neuf : le jet n'est pas connu
 *
 * Le quiz pose une question, donc un jet, donc une liste de coups légaux. Le
 * transcripteur qui regarde une vidéo, lui, suit les pions : il joue le coup et
 * ne tape jamais de chiffre. La liste de départ est donc l'UNION des coups
 * légaux des vingt et un jets, chaque coup portant le sien.
 *
 * Et c'est tout ce qu'il faut : le filtre du réducteur — un coup reste vivant
 * tant que les pas joués sont contenus dans les siens, multiplicités comprises —
 * réduit cette union exactement comme il réduit une liste d'un seul jet. Les
 * jets encore possibles sont ceux des coups encore vivants ; ils se réduisent
 * seuls, sans une ligne de code de plus.
 *
 * # Les dés déduits, et l'ambiguïté
 *
 * Un pas dit ses pips, donc son dé — sauf à la sortie, où « 2/off » se joue
 * avec un 2 comme avec un 6 quand les points hauts sont vides, et sauf quand un
 * dé n'est pas jouable et que le coup s'arrête plus tôt. L'ambiguïté est donc
 * RÉELLE, rare, et concentrée en fin de partie.
 *
 * La règle tient en trois cas et ne devine jamais :
 *
 *   1. un seul jet reste compatible et l'un de ses coups est achevé → les deux
 *      dés sont déduits, l'Action part sans une touche ([deducedDice]) ;
 *   2. plusieurs jets restent compatibles et au moins un d'entre eux a un coup
 *      achevé → le triangle des jets (T2.1) ne garde que ceux-là et attend un
 *      clic ([choosableRolls]). Un clic, seulement dans le cas où le plateau ne
 *      peut pas répondre à la place de l'utilisateur ;
 *   3. aucun coup achevé → il reste des pas à jouer, rien n'est enregistré.
 *
 * Deviner à la place de l'utilisateur aurait écrit dans le match un jet que
 * personne n'a vu ; demander à chaque coup aurait coûté le budget d'ux.md §4.1.
 *
 * # Le déplacement libre (T2.4)
 *
 * Un coup illégal a tenu à la table : il se transcrit. L'état porte alors
 * `free`, aucune liste de coups ne le contraint, et un pas est appliqué tel
 * quel — par `applyStep` de quizPlay, encore lui, qui sait déjà retirer un
 * pion, frapper un blot et sortir. Le plateau obtenu est ce qui s'est passé :
 * c'est lui qui devient `board_after` sur l'Action, et le moteur le garde
 * seulement si aucun coup légal ne l'atteint (transcript.validate).
 */

import { newPlay, alivePlays, applyStep, barOf, playHop, resetPlay, OFF } from './quizPlay.js';
import { parseMoveNotation } from '../utils/boardGeometry.js';

const WHITE = 1;

/** Les vingt et un jets distincts, dé fort d'abord — l'ordre du triangle. */
export const ROLLS = Object.freeze([1, 2, 3, 4, 5, 6].flatMap((high) => [1, 2, 3, 4, 5, 6].filter((low) => low <= high).map((low) => Object.freeze([high, low]))));

/**
 * La clé d'un jet, dé fort d'abord : 3-1 et 1-3 sont le même jet, et c'est
 * l'étiquette que porte la case du triangle.
 * @param {number[]} dice
 */
export function rollKey(dice) {
    const [a, b] = dice ?? [0, 0];
    return a >= b ? `${a}${b}` : `${b}${a}`;
}

/**
 * L'état de départ d'un coup joué au plateau : l'union des coups légaux des
 * jets donnés, chacun portant le sien.
 *
 * `byRoll` est ce que le panneau a demandé au moteur — une entrée par jet,
 * `{ dice, plays }` — et un jet sans aucun coup légal (une danse) n'y apporte
 * rien, ce qui l'écarte de lui-même.
 *
 * @param {any} position la position d'où le coup part, camp au trait posé
 * @param {{dice: number[], plays: any[]}[]} byRoll
 */
export function newBoardPlay(position, byRoll) {
    const plays = [];
    for (const entry of byRoll ?? []) {
        const [a, b] = entry?.dice ?? [0, 0];
        const roll = a >= b ? [a, b] : [b, a];
        for (const play of entry?.plays ?? []) plays.push({ ...play, roll });
    }
    return { ...newPlay(position, plays), free: false, origin: position };
}

/** L'état de départ d'un déplacement LIBRE : aucun coup ne le contraint. */
export function newFreePlay(position) {
    return { ...newPlay(position, []), free: true, origin: position };
}

/**
 * Remet le plateau tel que le tour le pose, en gardant ce qui n'appartient pas
 * au réducteur — le mode libre et la position d'origine.
 *
 * `resetPlay` rend un état NEUF de quizPlay, donc sans eux : c'est ici qu'ils
 * lui sont rendus, plutôt que dans le réducteur, qui ne les connaît pas.
 *
 * @param {any} state
 * @param {any} fallback la position à défaut d'origine (une question de quiz)
 */
export function resetBoardPlay(state, fallback) {
    if (!state) return state;
    const base = state.origin ?? fallback;
    return { ...resetPlay(state, base), free: state.free === true, origin: state.origin };
}

/** Le point porte-t-il un pion du camp qui joue ? */
function hasMoverChecker(state, point) {
    const p = state?.board?.points?.[point];
    return !!p && p.checkers > 0 && p.color === state.mover;
}

/**
 * Choisir le pion à déplacer, en mode libre : n'importe quel point qui porte un
 * pion du camp au trait, la barre comprise. Un second clic sur le même point le
 * déselectionne.
 * @param {any} state
 * @param {number} point
 */
export function freeSelect(state, point) {
    if (state.selected === point) return { ...state, selected: null };
    if (!hasMoverChecker(state, point)) return state;
    return { ...state, selected: point };
}

/**
 * Déplacer un pion sans rien vérifier : c'est le coup qui a été joué à la
 * table, et il n'est pas jugé (ADR-0044). Seule condition, physique : il faut
 * un pion à prendre.
 * @param {any} state
 * @param {number} from
 * @param {number} to
 */
export function freeStep(state, from, to) {
    if (from === to || !hasMoverChecker(state, from)) return state;
    const board = applyStep(state.board, { from, to }, state.mover);
    return { ...state, board, steps: [...state.steps, { from, to }], selected: null };
}

/**
 * Le clic du mode libre : il choisit une source, ou déplace le pion choisi.
 * Même forme que le clic du quiz, pour que le plateau n'ait qu'une branche.
 * @param {any} state
 * @param {number} point
 */
export function freeClick(state, point) {
    if (state.selected === null) return freeSelect(state, point);
    const moved = freeStep(state, state.selected, point);
    return moved === state ? freeSelect(state, point) : moved;
}

/**
 * Annule le dernier pas, en rejouant les autres : la correction d'un clic
 * manqué, sans reprendre le coup au début.
 *
 * `undoLast` de quizPlay ferait la même chose, mais par `newPlay`, qui rend un
 * état neuf du réducteur — donc sans le mode libre ni la position d'origine.
 * Le rejeu passe ici par le même chemin que le clic, libre ou contraint.
 *
 * @param {any} state
 */
export function undoBoardStep(state) {
    if (!state || state.steps.length === 0) return state;
    const kept = state.steps.slice(0, -1);
    let next = resetBoardPlay(state, state.origin);
    for (const s of kept) next = state.free ? freeStep(next, s.from, s.to) : playHop(next, s.from, s.to);
    return next;
}

/** Les jets encore compatibles avec ce qui a été joué, clés triées. */
export function compatibleRolls(state) {
    if (!state || state.free) return [];
    const keys = new Set();
    for (const play of alivePlays(state)) keys.add(rollKey(play.roll));
    return [...keys].sort();
}

/**
 * Les jets dont un coup est ACHEVÉ par les pas joués — ceux que l'utilisateur
 * peut désigner quand le plateau ne peut plus trancher seul.
 * @param {any} state
 */
export function choosableRolls(state) {
    if (!state || state.free || state.steps.length === 0) return [];
    const keys = new Set();
    for (const play of alivePlays(state)) {
        if (play.steps.length === state.steps.length) keys.add(rollKey(play.roll));
    }
    return [...keys].sort();
}

/**
 * Les deux dés que les pas joués déduisent, ou `null`.
 *
 * Un seul jet compatible, et un de ses coups achevé : il n'y a rien à demander.
 * Tout le reste — plusieurs jets, ou un coup en cours — rend `null`, et c'est
 * le panneau qui décide s'il attend un pas de plus ou un clic sur le triangle.
 *
 * @param {any} state
 * @returns {number[]|null} le jet, dé fort d'abord
 */
export function deducedDice(state) {
    if (!state || state.free || state.steps.length === 0) return null;
    const alive = alivePlays(state);
    if (alive.length === 0) return null;
    const keys = new Set(alive.map((play) => rollKey(play.roll)));
    if (keys.size !== 1) return null;
    if (!alive.some((play) => play.steps.length === state.steps.length)) return null;
    const [high, low] = alive[0].roll;
    return [high, low];
}

/**
 * Un point de la notation, rendu dans le repère ABSOLU du plateau.
 *
 * La notation est mover-relative — 24 nomme toujours les pions arrière du camp
 * qui joue — et `domain.pointLabel` la produit en miroitant les points du camp
 * blanc. Ceci en est l'inverse exact, et rien d'autre.
 */
function absolutePoint(relative, mover) {
    return mover === WHITE ? 25 - relative : relative;
}

/**
 * Les pas qu'un texte de notation décrit : `13/7 8/7*`, `bar/22`, `6/off(2)`.
 *
 * Le parseur n'est pas récrit : c'est `parseMoveNotation`, celui des flèches du
 * plateau, qui rend déjà `bar` → 0, `off` → −1 et développe les `(n)`. Ce qui
 * est ajouté ici est le passage au repère absolu, la seule chose que le
 * parseur ne pouvait pas savoir : il ne connaît pas le camp qui joue.
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
 * Le plateau que ces pas laissent, appliqués l'un après l'autre au plateau
 * donné. Rien n'est jugé : un pas dont la source est vide ne prend simplement
 * aucun pion, et le Replay dira l'incohérence.
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
