import { writable, derived } from 'svelte/store';
import { completedPlay } from '../services/quizPlay.js';

// Le coup joué SUR LE PLATEAU, par une question de pions de quiz ou une transcription (pas joués selon la
// grammaire d'ADR-0086, ou déplacement libre d'un coup illégal, ADR-0052) : un seul magasin, un seul réducteur.
// Non nul UNIQUEMENT pendant un coup au plateau — c'est le signal ; le mode de l'application ne
// change pas.

/** @type {import('svelte/store').Writable<(import('../services/quizPlay.js').PlayState & {free?: boolean, swapped?: boolean, rolled?: number[]|null, origin?: any})|null>} */
export const quizPlayStore = writable(null);

/** Le coup est-il complet, donc prêt à être jugé (ou enregistré) ? */
export const quizPlayCompleteStore = derived(quizPlayStore, ($s) => ($s ? completedPlay($s) !== null : false));

// Le rappel de validation du mode qui a armé le coup : il fait suivre au plateau la grammaire
// d'ADR-0086 (services/boardMove.js : clic sur un pion, clic sur les dés, clic droit). Le coup
// désarmé l'efface : un mode ne reçoit pas celui d'un autre.
/** @type {import('svelte/store').Writable<(() => void)|null>} */
export const quizPlayValidateStore = writable(null);

quizPlayStore.subscribe(($s) => {
    if (!$s) quizPlayValidateStore.set(null);
});

/**
 * Arme un coup au plateau selon la grammaire d'ADR-0086.
 * @param {any} play l'état de quizPlay (le jet est `rolled`, sinon les dés de la position)
 * @param {() => void} validate ce que fait le clic sur les dés quand le coup est achevé
 */
export function armBoardMove(play, validate) {
    quizPlayStore.set(play);
    quizPlayValidateStore.set(validate);
}
