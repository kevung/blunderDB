import { writable, derived } from 'svelte/store';

// La file d'étude post-import (#259, fiche I.3).
//
// Le compte rendu d'import répond à « que vient-il de se passer ? ». La file
// répond à la question qui suit — « qu'est-ce que je regarde maintenant ? » —
// et elle y répond UNE FOIS : une liste ordonnée, parcourue, une décision
// prise sur chaque position.
//
// Rien n'est enregistré, et rien ne note qu'une position a été vue. Ce que
// l'utilisateur en fait — un commentaire, une collection, une carte — EST la
// trace, et il n'y a rien d'autre à garder. C'est la même retenue que
// l'ADR-0006 énonce à propos des marques.

/** @typedef {{positionId: number, matchId: number, reason: string, label: string, errorMp: number, isCube: boolean}} StudyQueueEntry */

/** @type {import('svelte/store').Writable<StudyQueueEntry[]>} */
/** @type {import('svelte/store').Writable<import('../../wailsjs/go/models').domain.StudyQueueEntry[]>} */
export const studyQueueStore = writable([]);

/** Position courante dans la file, 0-indexée. */
export const studyQueueIndexStore = writable(0);

/** La file est-elle en cours de parcours ? */
export const studyQueueActiveStore = writable(false);

/** L'entrée en cours, ou null. */
export const studyQueueCurrentStore = derived([studyQueueStore, studyQueueIndexStore, studyQueueActiveStore], ([$queue, $index, $active]) => {
    if (!$active || $index < 0 || $index >= $queue.length) return null;
    return $queue[$index];
});
