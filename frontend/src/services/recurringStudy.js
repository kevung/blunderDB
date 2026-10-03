// De l'erreur récurrente à l'étude : le quiz Décision, un paquet Anki ou une
// collection sur les positions d'un groupe. Ce module ne décide de rien : les
// groupes viennent du moteur de statistiques, le quiz et le paquet des services
// qui les portent déjà ; il ne fait que les relier.

import { CreateCollection, AddPositionsToCollection } from '../../wailsjs/go/database/Database.js';
import { tMsg } from '../i18n';
import { activeTabStore, statusBarTextStore } from '../stores/uiStore.js';
import { loadPositionsFromSelection } from './positionLoader.js';
import { startTrainingSession } from './trainingTabService.js';
import { createDeck } from './ankiService.js';
import { logger } from '../utils/logger.js';

/** Combien de groupes forment « mes pires groupes », et combien de positions le quiz en tire. */
export const WORST_GROUPS = 3;
export const WORST_QUIZ_SIZE = 20;

/**
 * Les positions distinctes de ces groupes, dans l'ordre du classement.
 * @param {Array<{PositionIDs?: number[]}>} groups
 * @returns {number[]}
 */
export function idsOfGroups(groups) {
    const seen = new Set();
    for (const g of groups) for (const id of g.PositionIDs ?? []) seen.add(id);
    return [...seen];
}

/**
 * Au plus `size` éléments tirés au hasard, sans remise ; l'entrée n'est pas modifiée.
 * @template T
 * @param {T[]} items @param {number} size @param {() => number} [random]
 * @returns {T[]}
 */
export function drawSample(items, size, random = Math.random) {
    const pool = [...items];
    for (let i = pool.length - 1; i > 0; i--) {
        const j = Math.floor(random() * (i + 1));
        [pool[i], pool[j]] = [pool[j], pool[i]];
    }
    return pool.slice(0, size);
}

/**
 * Lance l'exercice Décision sur ces positions : elles deviennent la liste
 * parcourue, d'où l'exercice tire ses questions.
 * @param {number[]} ids
 * @returns {Promise<boolean>} faux quand il n'y a rien à poser
 */
export async function quizOnIds(ids) {
    if (!ids.length) {
        statusBarTextStore.set(tMsg('commands.noPositionsFound'));
        return false;
    }
    await loadPositionsFromSelection(ids);
    activeTabStore.set('training');
    return startTrainingSession({ exercise: 'decision', seedSource: 'library' });
}

/**
 * Le quiz des trois pires groupes : vingt positions au hasard parmi les leurs.
 * @param {Array<{PositionIDs?: number[]}>} groups les groupes, du plus coûteux au moins coûteux
 */
export function quizOnWorstGroups(groups) {
    return quizOnIds(drawSample(idsOfGroups(groups.slice(0, WORST_GROUPS)), WORST_QUIZ_SIZE));
}

/**
 * Un paquet Anki de ces positions, rangé comme un paquet de recherche : les ids
 * restent avec lui.
 * @param {string} name @param {number[]} ids
 */
export async function deckFromIds(name, ids) {
    try {
        await createDeck({ name, sourceType: 'search', sourceId: 0, positionIds: ids });
        statusBarTextStore.set(tMsg('stats.recurringDeckCreated', { name, n: ids.length }));
        return true;
    } catch (error) {
        logger.error('could not make a deck from a recurring-error group:', error);
        statusBarTextStore.set(tMsg('common.errorWithMsg', { msg: error }));
        return false;
    }
}

/**
 * Une collection de ces positions.
 * @param {string} name @param {number[]} ids
 */
export async function collectionFromIds(name, ids) {
    try {
        const id = await CreateCollection(name, '');
        await AddPositionsToCollection(id, ids);
        statusBarTextStore.set(tMsg('stats.recurringCollectionCreated', { name, n: ids.length }));
        return true;
    } catch (error) {
        logger.error('could not make a collection from a recurring-error group:', error);
        statusBarTextStore.set(tMsg('common.errorWithMsg', { msg: error }));
        return false;
    }
}
