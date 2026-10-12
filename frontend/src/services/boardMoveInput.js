/**
 * boardMoveInput.js — les gestes d'un coup au plateau selon la grammaire d'ADR-0086, posés sur
 * les magasins du coup. Chaque plateau qui dessine le coup armé (Board.svelte par
 * utils/boardInteractions.js, le mini-plateau du théâtre) fait son propre test de touche, puis
 * appelle ces gestes : la grammaire (services/boardMove.js) n'a qu'un point d'entrée par geste.
 *
 * `stores` : { quizPlay, quizPlayValidate, position } — passés plutôt qu'importés, pour que le
 * plateau principal reste testable sur des magasins nus.
 */

import { get } from 'svelte/store';
import { boardRightClick, diceClick, orderedDice, playClickedChecker } from './boardMove.js';
import { dragStep, resetBoardPlay } from './transcriptionPlay.js';

/**
 * @typedef {{ quizPlay?: import('svelte/store').Writable<any>, quizPlayValidate?: import('svelte/store').Readable<(() => void)|null>, position: import('svelte/store').Readable<any> }} MoveStores
 */

/**
 * Le coup armé selon la grammaire (`quizPlayValidate` posé par son mode), ou null.
 * @param {MoveStores} stores
 */
export function armedPlay(stores) {
    const play = stores.quizPlay ? get(stores.quizPlay) : null;
    if (!play || !stores.quizPlayValidate) return null;
    return get(stores.quizPlayValidate) ? play : null;
}

/**
 * Le jet dans l'ordre dessiné : le jet saisi du coup, sinon les dés de la position.
 * @param {any} play
 * @param {any} position
 */
export function drawnDice(play, position) {
    return orderedDice(play.rolled ?? position?.dice, !!play.swapped);
}

/**
 * Clic sur les dés : valide le coup achevé, sinon intervertit les dés restant à jouer.
 * @param {MoveStores} stores
 */
export function pressDice(stores) {
    const play = armedPlay(stores);
    if (!play || !stores.quizPlay) return;
    const action = diceClick(play, drawnDice(play, get(stores.position)));
    if (action === 'validate') stores.quizPlayValidate && get(stores.quizPlayValidate)?.();
    else if (action === 'swap') stores.quizPlay.update((/** @type {any} */ s) => (s ? { ...s, swapped: !s.swapped } : s));
}

/**
 * Relâchement d'un pion pressé sur `from` : sur place (ou hors du damier), c'est un clic, le pion
 * avance du premier dé non joué ; ailleurs, un glissé vers `target`.
 * @param {MoveStores} stores
 * @param {number} from point du modèle
 * @param {number|null} target point du modèle, ou null
 */
export function releaseChecker(stores, from, target) {
    if (!stores.quizPlay) return;
    stores.quizPlay.update((/** @type {any} */ s) => {
        if (!s) return s;
        if (target === null || target === from) return playClickedChecker(s, from, drawnDice(s, get(stores.position)));
        return dragStep(s, from, target);
    });
}

/**
 * Clic droit sur le damier : reprend tous les pas du coup en cours (jet saisi, origine et ordre
 * des dés gardés). Rend `true` quand il y avait un pas à reprendre.
 * @param {MoveStores} stores
 */
export function takeBackPlay(stores) {
    const play = armedPlay(stores);
    if (!play || !stores.quizPlay || boardRightClick(play) !== 'reset') return false;
    stores.quizPlay.update((/** @type {any} */ s) => (s ? { ...resetBoardPlay(s, get(stores.position)), swapped: !!s.swapped } : s));
    return true;
}

/**
 * Les flèches d'un candidat se retirent dès le premier pas joué au plateau : elles se
 * superposeraient au coup qu'on saisit. Elles reviennent quand le coup est validé ou repris.
 * @param {any} play l'état du coup au plateau, ou null
 */
export function arrowsHiddenBy(play) {
    return !!play && Array.isArray(play.steps) && play.steps.length > 0;
}
