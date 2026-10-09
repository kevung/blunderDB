import { writable, derived, get } from 'svelte/store';
import { positionStore } from './positionStore';
import { trainingAnalysisHiddenStore } from './trainingTabStore.js';

// Le défi du panneau Analyse : se prononcer avant de voir l'analyse d'une position parcourue.
// Un masque d'affichage, comme celui de l'entraînement : dévoiler ne touche pas la base.

/** La préférence (bouton de la barre d'outils, touche `m`), gardée pour la session comme le pipcount. */
export const analysisChallengeStore = writable(false);

/**
 * Ce qui identifie la position affichée pour le défi. Le contenu, pas l'objet : un plateau
 * repeint à l'identique (bascule du pipcount) n'est pas une nouvelle position et ne remasque pas.
 *
 * @param {any} position
 */
export function challengeKey(position) {
    if (!position) return '';
    const { id, board, cube, dice, score, player_on_roll, decision_type } = position;
    return JSON.stringify([id, board, cube, dice, score, player_on_roll, decision_type]);
}

/** La clé de la position dévoilée ; toute autre position est masquée. */
const revealedKeyStore = writable(/** @type {string|null} */ (null));

// Le dévoilement vaut pour la position où il a eu lieu, tant qu'on y reste : la quitter l'oublie,
// sans quoi y revenir (k après j, un clic dans une liste) la montrerait dévoilée sans clic.
let shownKey = challengeKey(get(positionStore));
positionStore.subscribe((position) => {
    const key = challengeKey(position);
    if (key === shownKey) return;
    shownKey = key;
    revealedKeyStore.set(null);
});

/**
 * Le défi cache-t-il la réponse de la position affichée ? Indépendant de l'entraînement : la
 * liste des coups d'un match s'y règle aussi, la marque du coup courant se dévoilant avec l'analyse.
 *
 * @type {import('svelte/store').Readable<boolean>}
 */
export const analysisChallengeHiddenStore = derived(
    [analysisChallengeStore, revealedKeyStore, positionStore],
    ([$challenge, $revealedKey, $position]) => $challenge && $revealedKey !== challengeKey($position)
);

/**
 * Le masque du panneau Analyse : `'training'` tant qu'une question d'entraînement attend sa
 * réponse (inerte : le verdict dévoile), `'challenge'` tant que le défi n'est pas relevé sur cette
 * position (un clic dévoile), `null` sinon. Un seul masque : les deux ne se superposent jamais.
 *
 * @type {import('svelte/store').Readable<'training'|'challenge'|null>}
 */
export const analysisMaskStore = derived([trainingAnalysisHiddenStore, analysisChallengeHiddenStore], ([$training, $challengeHidden]) => {
    if ($training) return 'training';
    if ($challengeHidden) return 'challenge';
    return null;
});

/** Dévoile l'analyse de la position affichée, jusqu'au prochain changement de position. */
export function revealAnalysis() {
    revealedKeyStore.set(challengeKey(get(positionStore)));
}

/** Bascule le défi ; l'activer masque aussi la position affichée. */
export function toggleAnalysisChallenge() {
    revealedKeyStore.set(null);
    analysisChallengeStore.update((on) => !on);
}
