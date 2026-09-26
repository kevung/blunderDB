import { writable, derived } from 'svelte/store';

// La file d'étude post-import : « qu'est-ce que je regarde maintenant ? », une liste ordonnée
// parcourue une fois. Rien n'est enregistré, pas même qu'une position a été vue : ce que
// l'utilisateur en fait est la trace (même retenue que l'ADR-0006).

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
