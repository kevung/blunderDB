import { ExplainDecision } from '../../wailsjs/go/database/Database.js';
import { normalizeCubeAction } from '../utils/cubeAction.js';
import { logger } from '../utils/logger.js';

// Expliquer un blunder en une phrase (#298, fiche J.8), côté interface.
//
// Le backend rend un THÈME et les écarts mesurés qui le justifient ; la phrase
// est écrite ici, à partir d'un gabarit traduit. C'est la séparation que les
// outils qui expliquent sans LLM appliquent tous (rapport P18) : des gabarits
// à trous alimentés par une base de règles. Une phrase rendue par le moteur
// aurait été une phrase française dans un logiciel qui parle neuf langues.
//
// Et la règle qui compte est de SE TAIRE : quand aucune règle n'est confiante,
// le thème est vide et rien ne s'affiche. gnubg fait exactement cela — son
// menu d'analyse reste vide quand il n'y avait rien à redire.

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
 * Le coup ou l'action réellement joués, tels que l'enregistrement les porte.
 * Rend '' quand le record n'en connaît aucun — une position consultée hors
 * match n'a pas toujours été jouée.
 * @param {object} analysis
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
 * Demande l'explication d'une décision. Rend null quand il n'y a rien à dire —
 * ce qui est le cas le plus fréquent, et voulu.
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
