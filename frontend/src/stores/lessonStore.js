import { writable, derived } from 'svelte/store';

// La Leçon en cours de lecture (ADR-0066) : une suite d'étapes que l'on déroule. La lecture
// n'enregistre rien, pas même l'étape atteinte (ADR-0007) ; seul le geste « étape faite » écrit
// la progression, dans la base ouverte (ADR-0069).

/** @type {import('svelte/store').Writable<import('../../wailsjs/go/models').domain.Lesson | null>} */
export const lessonStore = writable(null);

/** Étape courante, 0-indexée. */
export const lessonStepIndexStore = writable(0);

/** Étapes marquées faites de la Leçon lue : date du geste par id d'étape. */
/** @type {import('svelte/store').Writable<Record<string, string>>} */
export const lessonDoneStore = writable({});

/** Une autre base s'ouvre : les ids de la Leçon lue ne désignent plus rien dans celle-ci. */
export function resetLessonStores() {
    lessonStore.set(null);
    lessonStepIndexStore.set(0);
    lessonDoneStore.set({});
}

/** L'étape en cours, ou null. */
export const lessonStepStore = derived([lessonStore, lessonStepIndexStore], ([$lesson, $index]) => {
    if (!$lesson || !$lesson.steps || $lesson.steps.length === 0) return null;
    return $lesson.steps[Math.min($index, $lesson.steps.length - 1)] || null;
});

/** La Leçon que l'éditeur ouvre en premier (0 : la liste seule). */
export const lessonEditorTargetStore = writable(0);
