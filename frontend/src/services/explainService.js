import { ExplainDecision } from '../../wailsjs/go/database/Database.js';
import { normalizeCubeAction } from '../utils/cubeAction.js';
import { logger } from '../utils/logger.js';

// Expliquer un blunder en une phrase. Le backend rend un THÈME et ses écarts
// mesurés ; la phrase vient d'un gabarit traduit ici (neuf langues). Sans
// règle confiante, le thème est vide et rien ne s'affiche, comme gnubg.

/**
 * L'action de videau jouée, dans les trois jetons que le moteur accepte.
 * @param {string} action
 */
export function cubeActionToken(action) {
    const parts = normalizeCubeAction(action);
    if (parts.includes('nodouble')) return 'nd';
    if (parts.includes('take')) return 'dt';
    if (parts.includes('pass')) return 'dp';
    return '';
}

/**
 * Le coup ou l'action réellement joués, ou '' si le record n'en connaît aucun.
 * @param {{doublingCubeAnalysis?: object, playedCubeAction?: string,
 *          playedCubeActions?: string[], playedMove?: string,
 *          playedMoves?: string[]}|null} analysis
 */
export function playedFromAnalysis(analysis) {
    if (!analysis) return '';
    if (analysis.doublingCubeAnalysis) {
        const action = analysis.playedCubeAction || (analysis.playedCubeActions || [])[0] || '';
        return cubeActionToken(action);
    }
    return analysis.playedMove || (analysis.playedMoves || [])[0] || '';
}

/**
 * Demande l'explication d'une décision ; null (le cas voulu et fréquent) quand
 * il n'y a rien à dire.
 * @param {number} positionId @param {string} played
 */
export async function explainDecision(positionId, played) {
    if (!positionId || !played) return null;
    try {
        const explanation = await ExplainDecision(positionId, played);
        return explanation && explanation.theme ? explanation : null;
    } catch (error) {
        logger.error('could not explain the decision:', error);
        return null;
    }
}
