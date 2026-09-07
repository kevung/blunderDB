/**
 * Ce qui demande au plateau de se repeindre.
 *
 * `drawBoard()` lit ses stores IMPÉRATIVEMENT, dans une frame d'animation :
 * écrire un store ne peint donc rien tant que personne n'a planifié un
 * repaint. La liste ci-dessous est cette demande, et elle vit hors de
 * `Board.svelte` pour une raison précise — le composant n'a pas de test de
 * rendu (two.js, un canvas que jsdom n'implémente pas), si bien qu'un store
 * oublié ici ne se voyait nulle part. Le pipcount de l'exercice Pions (#320)
 * est arrivé ainsi : le masque se calculait, et l'écran ne bougeait pas.
 *
 * `positionStore` n'y figure pas : son abonnement porte en plus une règle
 * métier (remettre à zéro le coup sélectionné sur une VRAIE navigation) qui
 * doit s'exécuter avant le repaint, et il reste donc écrit dans le composant.
 */
import { analysisStore, selectedMoveStore } from '../stores/analysisStore.js';
import { searchOfferedCubeStore } from '../stores/searchExcludePositionStore.js';
import { quizPlayStore } from '../stores/quizPlayStore.js';
import { pipcountVisibleStore } from '../stores/uiStore.js';

/**
 * Les stores dont tout changement rend le plateau sale. Nommés, parce qu'un
 * échec de test doit dire LEQUEL manque.
 * @type {readonly {name: string, store: import('svelte/store').Readable<any>}[]}
 */
export const BOARD_REDRAW_TRIGGERS = Object.freeze([
    // Le coup choisi ou survolé dans le panneau Analyse : ses flèches.
    Object.freeze({ name: 'selectedMove', store: selectedMoveStore }),
    // L'analyse chargée : le videau offert d'une décision de prise.
    Object.freeze({ name: 'analysis', store: analysisStore }),
    // La bascule « videau offert » : le videau change de place.
    Object.freeze({ name: 'offeredCube', store: searchOfferedCubeStore }),
    // Le coup du quiz, construit un pas à la fois (#294).
    Object.freeze({ name: 'quizPlay', store: quizPlayStore }),
    // La visibilité du pipcount : la préférence de l'utilisateur, ou le masque
    // d'une question de Pions (#320).
    Object.freeze({ name: 'pipcountVisible', store: pipcountVisibleStore })
]);

/**
 * Abonne `schedule` à chacun des déclencheurs et rend la fonction de
 * désabonnement.
 * @param {() => void} schedule
 * @returns {() => void}
 */
export function subscribeBoardRedrawTriggers(schedule) {
    const unsubscribers = BOARD_REDRAW_TRIGGERS.map(({ store }) => store.subscribe(() => schedule()));
    return () => unsubscribers.forEach((unsubscribe) => unsubscribe());
}
