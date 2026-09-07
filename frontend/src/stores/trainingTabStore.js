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
 * Le pipcount du plateau, pendant une question de Pions : `false` tant que la
 * question est ouverte, `true` une fois révélée, `null` le reste du temps —
 * c'est-à-dire « la préférence de l'utilisateur décide ».
 *
 * Un masque, jamais un réglage (ADR-0040) : `showPipcountStore` n'est pas
 * touché, donc la préférence est intacte à la fin de la session, sans rien à
 * restaurer. Et la révélation IMPOSE l'affichage, même à qui a masqué le
 * pipcount avec `p` : un masque qui, retiré, laisse l'écran vide n'a pas rendu
 * ce qu'il avait pris, et l'exercice deviendrait invérifiable — on ne peut pas
 * comparer sa réponse à une vérité qui ne s'affiche pas.
 *
 * @type {import('svelte/store').Readable<boolean|null>}
 */
export const trainingPipOverrideStore = derived(trainingSessionStore, ($session) => {
    if (!$session || $session.exercise !== 'pips' || !$session.question) return null;
    return $session.revealed;
});
