import { get } from 'svelte/store';
import { SimilarPositions } from '../../wailsjs/go/database/Database.js';
import { positionsStore, matchContextStore } from '../stores/positionStore.js';
import { currentPositionIndexStore, activeTabStore, statusBarModeStore } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { logger } from '../utils/logger.js';
import { tMsg } from '../i18n';

// « Des positions comme celle-ci » (#293, fiche J.3), côté interface.
//
// Le résultat REMPLACE la liste parcourue, comme le fait une recherche : c'est
// le même geste — on repart d'un ensemble de positions et on le feuillette —
// et une seconde façon de présenter un ensemble de positions aurait été une
// seconde façon de le parcourir.
//
// Mais ce n'est PAS un filtre, et cela se dit : la similarité classe toute la
// bibliothèque, elle ne la restreint pas. On ne peut donc pas la combiner avec
// les jetons de recherche — « les plus proches PARMI celles qui matchent »
// serait une autre question, et une question qu'on n'a pas posée.

/** Le nombre de voisins demandés par défaut. */
export const SIMILAR_LIMIT = 10;

/**
 * Remplace la liste parcourue par les voisins de la position donnée, du plus
 * proche au plus lointain. Rend le nombre de voisins trouvés.
 * @param {number} positionId @param {number} [limit]
 */
export async function showSimilarPositions(positionId, limit = SIMILAR_LIMIT) {
    if (!get(databasePathStore)) {
        setStatusBarMessage(tMsg('status.noDatabaseOpened'));
        return 0;
    }
    if (!positionId) {
        setStatusBarMessage(tMsg('similar.noPosition'));
        return 0;
    }
    let neighbours;
    try {
        neighbours = (await SimilarPositions(positionId, limit)) || [];
    } catch (error) {
        logger.error('could not find similar positions:', error);
        setStatusBarMessage(tMsg('similar.failed'));
        return 0;
    }
    if (neighbours.length === 0) {
        setStatusBarMessage(tMsg('similar.none'));
        return 0;
    }
    matchContextStore.set({
        isMatchMode: false,
        matchID: null,
        movePositions: [],
        currentIndex: 0,
        player1Name: '',
        player2Name: ''
    });
    positionsStore.set(neighbours.map((n) => n.position));
    currentPositionIndexStore.set(-1);
    currentPositionIndexStore.set(0);
    activeTabStore.set('analysis');
    statusBarModeStore.set('NORMAL');
    setStatusBarMessage(
        tMsg('similar.found', {
            n: neighbours.length,
            nearest: neighbours[0].distance,
            farthest: neighbours[neighbours.length - 1].distance
        })
    );
    return neighbours.length;
}
