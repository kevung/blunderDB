import { writable, derived } from 'svelte/store';
import { completedPlay, sources, destinationsFrom } from '../services/quizPlay.js';

// Le coup que l'utilisateur joue sur le plateau, pendant une question de quiz
// (#294, fiche J.4).
//
// Non nul UNIQUEMENT pendant qu'une question de pions attend sa réponse. Tout
// le reste de l'application peut donc se contenter de « ce magasin est-il
// nul ? » pour savoir si le plateau se joue — le mode de l'application, lui,
// ne change pas : le quiz se déroule sur la position de la bibliothèque, pas
// sur un plateau brouillon, et lui inventer un mode aurait fait un sixième
// état à faire dialoguer avec les cinq autres.

/** @type {import('svelte/store').Writable<import('../services/quizPlay.js').PlayState|null>} */
export const quizPlayStore = writable(null);

/** Le coup est-il complet, donc prêt à être jugé ? */
export const quizPlayCompleteStore = derived(quizPlayStore, ($s) => ($s ? completedPlay($s) !== null : false));

/** Les points d'où un pas peut partir — ce que le plateau met en avant. */
export const quizPlaySourcesStore = derived(quizPlayStore, ($s) => ($s ? sources($s) : new Set()));

/** Les destinations offertes par la source choisie, s'il y en a une. */
export const quizPlayTargetsStore = derived(quizPlayStore, ($s) => ($s && $s.selected !== null ? destinationsFrom($s, $s.selected) : new Set()));
