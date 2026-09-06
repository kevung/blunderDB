import { writable, derived } from 'svelte/store';

// Les micro-entraînements (#273, fiche I.17).
//
// Une session est une suite de questions posées SUR LE PLATEAU de
// l'application, pas dans une fenêtre à part : la question d'un exercice de
// comptage EST la position affichée, et la recopier ailleurs aurait fait un
// second plateau à tenir en phase avec le premier.
//
// Comme la file d'étude (#259), rien n'est écrit tant que la session n'est pas
// finie, et ce qui est écrit alors est un résumé — pas la trace question par
// question. Un module d'entraînement qui tient le registre de chaque réponse
// devient un registre, et le registre n'est pas ce qu'on a promis.

/** @typedef {{drill: string, positionId: number|null, truth: number, prompt: string}} TrainingQuestion */
/** @typedef {{correct: boolean, error: number|null, ms: number}} TrainingAnswer */

/** L'exercice en cours ('pips' | 'epc' | 'takepoint'), ou null hors session. */
export const trainingDrillStore = writable(null);

/** Les questions de la session. */
/** @type {import('svelte/store').Writable<TrainingQuestion[]>} */
export const trainingQuestionsStore = writable([]);

/** L'index de la question courante, 0-indexé. */
export const trainingIndexStore = writable(0);

/** Les réponses déjà données. */
/** @type {import('svelte/store').Writable<TrainingAnswer[]>} */
export const trainingAnswersStore = writable([]);

/** La session est-elle en cours ? */
export const trainingActiveStore = writable(false);

/** Le verdict de la dernière réponse, affiché jusqu'à la question suivante. */
export const trainingVerdictStore = writable(null);

/** La question courante, ou null. */
export const trainingCurrentStore = derived([trainingQuestionsStore, trainingIndexStore, trainingActiveStore], ([$questions, $index, $active]) => {
    if (!$active || $index < 0 || $index >= $questions.length) return null;
    return $questions[$index];
});

/** L'analyse doit-elle être masquée ? Vrai pendant qu'une question de quiz
 *  attend sa réponse : le panneau Analyse PORTE la réponse, et une question
 *  dont la réponse est affichée à côté n'est pas une question. Les exercices
 *  de calcul (#273) ne masquent rien — ils portent sur le damier, pas sur
 *  l'évaluation. */
export const trainingMaskStore = derived(
    [trainingActiveStore, trainingCurrentStore, trainingVerdictStore],
    ([$active, $question, $verdict]) => $active && !!$question && $question.drill === 'quiz' && !$verdict
);
