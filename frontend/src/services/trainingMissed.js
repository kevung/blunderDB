// Des ratés du journal d'Entraînement à l'étude : les positions mal répondues à
// l'exercice Décision redeviennent un quiz, un paquet Anki ou une collection.
// La liste vient du moteur (la même pour la ligne de commande et le démon) ;
// les trois gestes sont ceux des erreurs récurrentes.

import { LoadTrainingMissed } from '../../wailsjs/go/database/Database.js';
import { tMsg } from '../i18n';
import { statusBarTextStore } from '../stores/uiStore.js';
import { quizOnIds, deckFromIds, collectionFromIds } from './recurringStudy.js';
import { logger } from '../utils/logger.js';

/**
 * Les positions ratées à Décision, la plus récemment ratée d'abord.
 * @returns {Promise<number[]>}
 */
export async function missedDecisionIds() {
    try {
        return (await LoadTrainingMissed(/** @type {any} */ ({ exercise: 'decision', sessionId: 0, limit: 0 }))) ?? [];
    } catch (error) {
        logger.error('could not read the missed training questions:', error);
        statusBarTextStore.set(tMsg('common.errorWithMsg', { msg: error }));
        return [];
    }
}

/** Le nom d'un paquet ou d'une collection de ratés : daté, pour qu'un second ne se confonde pas avec le premier. */
export function missedStudyName(now = new Date()) {
    return tMsg('training.missedStudyName', { date: now.toISOString().slice(0, 10) });
}

/** Reprendre mes ratés : ils deviennent la liste parcourue et l'exercice Décision repart dessus. */
export async function retakeMissed() {
    return quizOnIds(await missedDecisionIds());
}

/** Un paquet Anki des ratés. */
export async function missedToDeck() {
    const ids = await missedDecisionIds();
    if (!ids.length) {
        statusBarTextStore.set(tMsg('commands.noPositionsFound'));
        return false;
    }
    return deckFromIds(missedStudyName(), ids);
}

/** Une collection des ratés. */
export async function missedToCollection() {
    const ids = await missedDecisionIds();
    if (!ids.length) {
        statusBarTextStore.set(tMsg('commands.noPositionsFound'));
        return false;
    }
    return collectionFromIds(missedStudyName(), ids);
}
