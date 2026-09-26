import { writable, derived } from 'svelte/store';
import { questionOnBoard } from '../services/trainingTab.js';

// L'onglet Entraînement (ADR-0040). L'état vit ici car TabbedPanel démonte le panneau à chaque
// changement d'onglet : une session ne doit pas s'effacer parce qu'on est allé voir une position.

/**
 * La session en cours, ou `null` au repos.
 * @type {import('svelte/store').Writable<import('../services/trainingTab.js').TrainingSessionState|null>}
 */
export const trainingSessionStore = writable(null);

/**
 * Le temps du chronomètre, en ms. Séparé de la session : il change cinq fois par seconde et ferait
 * recalculer la fiche de score à chaque battement.
 */
export const trainingElapsedStore = writable(0);

/**
 * Le journal relu (sessions et détail par type de nombre, par exercice), rempli à l'ouverture de
 * l'onglet et après chaque « Terminer », seuls moments où il change.
 * @type {import('svelte/store').Writable<Record<string, {sessions: any[], numbers: any[]}>>}
 */
export const trainingJournalStore = writable({});

/**
 * Le pipcount pendant une question de Pions : `false` question ouverte, `true` révélée, `null`
 * sinon (la préférence décide). Un masque, jamais un réglage (ADR-0040) : `showPipcountStore`
 * n'est pas touché. La révélation IMPOSE l'affichage, même pipcount masqué par `p`, sans quoi la
 * réponse serait invérifiable.
 *
 * @type {import('svelte/store').Readable<boolean|null>}
 */
export const trainingPipOverrideStore = derived(trainingSessionStore, ($session) => {
    if (!$session || $session.exercise !== 'pips' || !$session.question) return null;
    return $session.revealed;
});

/**
 * Le code du refus qui a empêché la dernière session de démarrer, ou du repli — le panneau
 * traduit (ADR-0041 règle 3). Hors de la session car il survit à son absence : il se lit dans le
 * lanceur, là où l'on vient de cliquer.
 *
 * @type {import('svelte/store').Writable<string>}
 */
export const trainingRefusalStore = writable('');

/** Les exercices dont la réponse s'affiche dans le panneau Analyse. */
const ANSWER_BEARING = new Set(['decision', 'evaluation']);

/**
 * Le panneau Analyse est-il masqué ? Vrai tant qu'une question de Décision ou d'Évaluation attend
 * sa réponse : l'analyse enregistrée PORTE la réponse. Un masque comme celui du pipcount : tout
 * revient au verdict ou en fin de session, y compris pour qui ouvre l'onglet Analyse en cours.
 *
 * @type {import('svelte/store').Readable<boolean>}
 */
export const trainingAnalysisHiddenStore = derived(trainingSessionStore, ($session) => !!$session && ANSWER_BEARING.has($session.exercise) && !!$session.question && !$session.revealed);

/**
 * Le plateau appartient-il à une question ? Le répartiteur clavier ne fait alors pas défiler la
 * liste (voir `questionOnBoard`).
 *
 * @type {import('svelte/store').Readable<boolean>}
 */
export const trainingHoldsBoardStore = derived(trainingSessionStore, ($session) => questionOnBoard($session));
