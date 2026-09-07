import { writable, derived } from 'svelte/store';
import { completedPlay, sources, destinationsFrom } from '../services/quizPlay.js';

// Le coup que l'utilisateur joue SUR LE PLATEAU.
//
// Deux surfaces l'écrivent, et c'est le même geste : la question de pions d'un
// quiz (#294, fiche J.4) et le coup d'une transcription, dont les dés se
// déduisent des pas joués (T2.3) ou qui est déplacé librement parce qu'il est
// illégal (T2.4). Un seul magasin, parce qu'un second aurait redemandé au
// plateau, au dessin et au clic de choisir lequel des deux ils écoutent — pour
// un état qui est le même, écrit par le même réducteur.
//
// Non nul UNIQUEMENT pendant qu'un coup se joue au plateau. Tout le reste de
// l'application peut donc se contenter de « ce magasin est-il nul ? » pour
// savoir si le plateau se joue — le mode de l'application, lui, ne change pas :
// le quiz se déroule sur la position de la bibliothèque, la transcription sur
// celle du Cursor, et leur inventer un mode aurait fait un sixième état à faire
// dialoguer avec les cinq autres.

/** @type {import('svelte/store').Writable<import('../services/quizPlay.js').PlayState|null>} */
export const quizPlayStore = writable(null);

/** Le coup est-il complet, donc prêt à être jugé (ou enregistré) ? */
export const quizPlayCompleteStore = derived(quizPlayStore, ($s) => ($s ? completedPlay($s) !== null : false));

/**
 * Les points d'où un pas peut partir — ce que le plateau met en avant
 * (`drawPlayHighlights`, un anneau par point). Vide en déplacement libre :
 * aucun coup légal ne s'y offre, et tout point qui porte un pion est bon.
 */
export const quizPlaySourcesStore = derived(quizPlayStore, ($s) => ($s ? sources($s) : new Set()));

/** Les destinations offertes par la source choisie, s'il y en a une. */
export const quizPlayTargetsStore = derived(quizPlayStore, ($s) => ($s && $s.selected !== null ? destinationsFrom($s, $s.selected) : new Set()));
