import { writable, derived } from 'svelte/store';
import { completedPlay, destinationsFrom } from '../services/quizPlay.js';

// Le coup joué SUR LE PLATEAU, par une question de pions de quiz ou une transcription (dés déduits
// des pas, ou déplacement libre d'un coup illégal, ADR-0052) : un seul magasin, un seul réducteur.
// Non nul UNIQUEMENT pendant un coup au plateau — c'est le signal ; le mode de l'application ne
// change pas.

/** @type {import('svelte/store').Writable<(import('../services/quizPlay.js').PlayState & {free?: boolean, rolled?: number[]|null, origin?: any})|null>} */
export const quizPlayStore = writable(null);

/** Le coup est-il complet, donc prêt à être jugé (ou enregistré) ? */
export const quizPlayCompleteStore = derived(quizPlayStore, ($s) => ($s ? completedPlay($s) !== null : false));

/** Les destinations offertes par la source choisie, s'il y en a une. */
export const quizPlayTargetsStore = derived(quizPlayStore, ($s) => ($s && !$s.free && $s.selected !== null ? destinationsFrom($s, $s.selected) : new Set()));
