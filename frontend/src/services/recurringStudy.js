// De l'erreur récurrente à l'étude : le quiz Décision, un paquet Anki ou une
// collection sur les positions d'un groupe. Ce module ne décide de rien : les
// groupes viennent du moteur de statistiques, le quiz et le paquet des services
// qui les portent déjà ; il ne fait que les relier.

import { get } from 'svelte/store';
import { CreateCollection, AddPositionsToCollection, CreateStudyDeck, StudyPositionIDs, StudyPlanPositionIDs } from '../../wailsjs/go/database/Database.js';
import { tMsg } from '../i18n';
import { activeTabStore, statusBarTextStore } from '../stores/uiStore.js';
import { loadPositionsFromSelection } from './positionLoader.js';
import { startTrainingSession } from './trainingTabService.js';
import { loadDecks } from './ankiService.js';
import { statsFilterStore } from '../stores/statsStore.js';
import { logger } from '../utils/logger.js';

/** Combien de positions le quiz des pires groupes en tire. */
export const WORST_QUIZ_SIZE = 20;

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
 * Le quiz des pires groupes du filtre : le tirage est celui du moteur, le même pour la ligne de
 * commande et le démon.
 */
export async function quizOnWorstGroups() {
    return quizOnIds((await StudyPositionIDs(get(statsFilterStore), 0, WORST_QUIZ_SIZE)) ?? []);
}

/**
 * Le quiz du plan d'étude : rang 0 pour les trois premières familles, n pour la n-ième seule.
 * @param {number} rank
 */
export async function quizOnPlan(rank) {
    return quizOnIds((await StudyPlanPositionIDs(get(statsFilterStore), rank, WORST_QUIZ_SIZE)) ?? []);
}

/**
 * Un paquet Anki de ces positions, rangé comme un paquet de recherche : les ids
 * restent avec lui.
 * @param {string} name @param {number[]} ids
 */
export async function deckFromIds(name, ids) {
    try {
        await CreateStudyDeck(name, ids);
        await loadDecks();
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
