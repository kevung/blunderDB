import { writable, derived } from 'svelte/store';
import { questionOnBoard } from '../services/trainingTab.js';

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

/**
 * Le refus qui a empêché la dernière session de démarrer, ou la phrase qui
 * accompagne un repli — le code, jamais la phrase : c'est le panneau qui
 * traduit (#321, ADR-0041 règle 3).
 *
 * Il vit à côté de la session et non dedans parce qu'il survit à l'absence de
 * session : un refus se lit DANS LE LANCEUR, là où l'on vient de cliquer
 * « Démarrer », et un message de barre d'état s'efface au geste suivant.
 *
 * @type {import('svelte/store').Writable<string>}
 */
export const trainingRefusalStore = writable('');

/** Les exercices dont la réponse s'affiche dans le panneau Analyse. */
const ANSWER_BEARING = new Set(['decision', 'evaluation']);

/**
 * Le panneau Analyse doit-il être masqué ? Vrai pendant qu'une question de
 * Décision (#323) ou d'Évaluation (#322) attend sa réponse : le panneau PORTE
 * la réponse — l'analyse enregistrée d'une position tirée de la base dit ses
 * chances de gain et son action de videau —, et une question dont la réponse
 * est affichée à côté n'est pas une question. Le
 * verdict rendu, l'analyse redevient visible — c'est la correction.
 *
 * Un masque, comme celui du pipcount : rien n'est touché, tout revient quand
 * la question est jugée ou la session finie. L'onglet et le panneau Analyse ne
 * sont jamais visibles ensemble ; le masque tient pour qui va regarder
 * l'onglet Analyse au milieu d'une question.
 *
 * @type {import('svelte/store').Readable<boolean>}
 */
export const trainingAnalysisHiddenStore = derived(trainingSessionStore, ($session) => !!$session && ANSWER_BEARING.has($session.exercise) && !!$session.question && !$session.revealed);

/**
 * Le plateau appartient-il à une question en ce moment ? Lu par le
 * répartiteur clavier, qui ne fait pas défiler la liste sous une question
 * ouverte (#323, défaut hérité de #321) — voir `questionOnBoard`.
 *
 * @type {import('svelte/store').Readable<boolean>}
 */
export const trainingHoldsBoardStore = derived(trainingSessionStore, ($session) => questionOnBoard($session));
