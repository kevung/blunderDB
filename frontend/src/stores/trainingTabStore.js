import { writable, derived } from 'svelte/store';

// L'onglet Entraînement (#320, ADR-0040).
//
// L'état vit ici et non dans le composant : TabbedPanel démonte le panneau
// quand on quitte l'onglet et le remonte au retour, donc tout `$state` local
// serait remis à zéro par un aller-retour — et une session d'entraînement qui
// s'efface parce qu'on est allé voir une position n'est pas une session.

/**
 * La session en cours, ou `null` au repos.
 * @type {import('svelte/store').Writable<import('../services/trainingTab.js').TrainingSessionState|null>}
 */
export const trainingSessionStore = writable(null);

/**
 * Le temps affiché par le chronomètre, en millisecondes. Séparé de la session
 * parce qu'il change cinq fois par seconde : le mêler à l'état de la session
 * ferait recalculer la fiche de score à chaque battement.
 */
export const trainingElapsedStore = writable(0);

/**
 * Le journal relu : une liste de sessions et un détail par type de nombre,
 * par exercice. Rempli à l'ouverture de l'onglet et après chaque
 * « Terminer » — les deux seuls moments où il peut avoir changé.
 * @type {import('svelte/store').Writable<Record<string, {sessions: any[], numbers: any[]}>>}
 */
export const trainingJournalStore = writable({});

/**
 * Le pipcount du plateau doit-il être masqué ?
 *
 * Vrai pendant qu'une question de Pions attend sa réponse : le plateau PORTE
 * la réponse, et une question dont la réponse est affichée à côté n'est pas
 * une question. « Révéler » l'affiche. C'est un masque, jamais un réglage : la
 * préférence de l'utilisateur (`showPipcountStore`) n'est pas touchée, donc
 * elle est intacte à la fin de la session.
 */
export const trainingPipMaskStore = derived(trainingSessionStore, ($session) => !!$session && $session.exercise === 'pips' && !!$session.question && !$session.revealed);
