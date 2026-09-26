/**
 * Ce qui demande au plateau de se repeindre. `drawBoard()` lit ses stores
 * impérativement dans une frame d'animation : écrire un store ne peint rien
 * sans repaint planifié. La liste vit hors de `Board.svelte`, sans test de
 * rendu (canvas absent de jsdom), pour qu'un store oublié se voie en test.
 *
 * `positionStore` n'y figure pas : son abonnement remet à zéro le coup
 * sélectionné sur une vraie navigation avant le repaint, dans le composant.
 */
import { analysisStore, selectedMoveStore } from '../stores/analysisStore.js';
import { searchOfferedCubeStore } from '../stores/searchExcludePositionStore.js';
import { quizPlayStore } from '../stores/quizPlayStore.js';
import { pipcountVisibleStore } from '../stores/uiStore.js';
import { transcriptionBoardSwapStore } from '../stores/transcriptionStore.js';

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
    // Le coup du quiz, construit un pas à la fois.
    Object.freeze({ name: 'quizPlay', store: quizPlayStore }),
    // La visibilité du pipcount : la préférence de l'utilisateur, ou le masque
    // d'une question de Pions.
    Object.freeze({ name: 'pipcountVisible', store: pipcountVisibleStore }),
    // Le sens du plateau pendant une transcription : le joueur 1 en bas, ou le
    // joueur 2. Rien d'autre ne change quand on la bascule — pas même la
    // position — donc rien d'autre ne demanderait le repaint.
    Object.freeze({ name: 'transcriptionBoardSwap', store: transcriptionBoardSwapStore })
]);

/**
 * Abonne `schedule` à chaque déclencheur ; rend le désabonnement.
 * @param {() => void} schedule
 * @returns {() => void}
 */
export function subscribeBoardRedrawTriggers(schedule) {
    const unsubscribers = BOARD_REDRAW_TRIGGERS.map(({ store }) => store.subscribe(() => schedule()));
    return () => unsubscribers.forEach((unsubscribe) => unsubscribe());
}
