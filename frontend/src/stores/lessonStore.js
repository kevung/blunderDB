import { writable, derived } from 'svelte/store';

// La Leçon en cours de lecture (ADR-0066) : une suite d'étapes que l'on déroule. Rien n'est
// enregistré de la lecture, pas même l'étape atteinte : une base reçue reste celle de l'auteur
// (ADR-0007).

/** @type {import('svelte/store').Writable<import('../../wailsjs/go/models').domain.Lesson | null>} */
export const lessonStore = writable(null);

/** Étape courante, 0-indexée. */
export const lessonStepIndexStore = writable(0);

/** Une autre base s'ouvre : les ids de la Leçon lue ne désignent plus rien dans celle-ci. */
export function resetLessonStores() {
    lessonStore.set(null);
    lessonStepIndexStore.set(0);
}

/** L'étape en cours, ou null. */
export const lessonStepStore = derived([lessonStore, lessonStepIndexStore], ([$lesson, $index]) => {
    if (!$lesson || !$lesson.steps || $lesson.steps.length === 0) return null;
    return $lesson.steps[Math.min($index, $lesson.steps.length - 1)] || null;
});
