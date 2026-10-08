import { get } from 'svelte/store';
import { ImportStudyQueue, StudyBacklog, SetPositionStudied, StudyPlanQueue } from '../../wailsjs/go/database/Database.js';
import { studyQueueStore, studyQueueIndexStore, studyQueueActiveStore, studyQueueCurrentStore, studyQueueBacklogStore, studyQueueLastMarkedStore } from '../stores/studyQueueStore.js';
import { showImportedPosition } from './importService.js';
import { setStatusBarMessage } from './databaseService.js';
import { activeTabStore } from '../stores/uiStore';
import { fileImportReportStore, showFileImportModalStore, fileImportModeStore } from '../stores/importModalStore.js';
import { logger } from '../utils/logger.js';
import { tMsg } from '../i18n';

// La file d'étude post-import : amène une position à la fois sur le plateau,
// sans s'approprier la liste, pour que les gestes existants (commenter,
// collection, carte) restent disponibles pendant le parcours.

/**
 * Démarre la file d'un lot d'import, ou dit qu'il n'a rien à revoir plutôt
 * que d'ouvrir une file vide.
 * @param {number} batchId
 */
export async function startStudyQueue(batchId) {
    try {
        const entries = (await ImportStudyQueue(batchId, 0)) || [];
        if (entries.length === 0) {
            setStatusBarMessage(tMsg('studyQueue.empty'));
            return false;
        }
        studyQueueBacklogStore.set(false);
        studyQueueStore.set(entries);
        studyQueueIndexStore.set(0);
        studyQueueActiveStore.set(true);
        await showCurrent();
        return true;
    } catch (error) {
        logger.error('could not build the study queue:', error);
        setStatusBarMessage(tMsg('studyQueue.failed'));
        return false;
    }
}

/**
 * Démarre la file transversale : les blunders du joueur de référence, tous lots
 * confondus, que rien n'a encore traités (ni commentaire, ni carte, ni collection,
 * ni marque « vu »), du plus coûteux au moins coûteux.
 */
export async function startStudyBacklog() {
    try {
        const entries = (await StudyBacklog(0)) || [];
        if (entries.length === 0) {
            setStatusBarMessage(tMsg('studyQueue.backlogEmpty'));
            return false;
        }
        studyQueueBacklogStore.set(true);
        studyQueueLastMarkedStore.set(0);
        studyQueueStore.set(entries);
        studyQueueIndexStore.set(0);
        studyQueueActiveStore.set(true);
        await showCurrent();
        return true;
    } catch (error) {
        logger.error('could not build the study backlog:', error);
        setStatusBarMessage(tMsg('studyQueue.failed'));
        return false;
    }
}

/**
 * Démarre la file d'une famille du plan d'étude (ADR-0077) : ses positions, l'excès sur le
 * joueur de référence le plus grand d'abord. Le rang 0 prend les trois premières familles.
 * @param {object} filter le filtre des statistiques, celui qui a produit le plan
 * @param {number} rank
 */
export async function startStudyPlanQueue(filter, rank) {
    try {
        const entries = (await StudyPlanQueue(filter, rank)) || [];
        if (entries.length === 0) {
            setStatusBarMessage(tMsg('studyQueue.planEmpty'));
            return false;
        }
        studyQueueBacklogStore.set(false);
        studyQueueStore.set(entries);
        studyQueueIndexStore.set(0);
        studyQueueActiveStore.set(true);
        await showCurrent();
        return true;
    } catch (error) {
        logger.error('could not build the study-plan queue:', error);
        setStatusBarMessage(tMsg('studyQueue.failed'));
        return false;
    }
}

/** Marque la position courante « vue » (geste explicite, réversible), puis passe à la suivante. */
export async function markCurrentStudied() {
    const entry = get(studyQueueCurrentStore);
    if (!entry) return;
    try {
        await SetPositionStudied(entry.positionId, true);
        studyQueueLastMarkedStore.set(entry.positionId);
    } catch (error) {
        logger.error('could not mark the position studied:', error);
        setStatusBarMessage(tMsg('studyQueue.markFailed'));
        return;
    }
    await nextInQueue();
}

/** Retire la dernière marque posée : la position revient dans la file transversale. */
export async function unmarkLastStudied() {
    const id = get(studyQueueLastMarkedStore);
    if (!id) return;
    try {
        await SetPositionStudied(id, false);
        studyQueueLastMarkedStore.set(0);
        setStatusBarMessage(tMsg('studyQueue.unmarked'));
    } catch (error) {
        logger.error('could not withdraw the studied mark:', error);
        setStatusBarMessage(tMsg('studyQueue.markFailed'));
    }
}

/** Passe à la position suivante ; termine la file à la dernière. */
export async function nextInQueue() {
    const queue = get(studyQueueStore);
    const index = get(studyQueueIndexStore);
    if (index + 1 >= queue.length) {
        stopStudyQueue({ finished: true });
        return;
    }
    studyQueueIndexStore.set(index + 1);
    await showCurrent();
}

/** Revient à la position précédente. Une file se parcourt une fois, mais
 *  revenir d'un cran n'est pas la reparcourir : c'est corriger un clic. */
export async function previousInQueue() {
    const index = get(studyQueueIndexStore);
    if (index <= 0) return;
    studyQueueIndexStore.set(index - 1);
    await showCurrent();
}

/** @param {{finished?: boolean}} [opts] */
export function stopStudyQueue({ finished = false } = {}) {
    const total = get(studyQueueStore).length;
    studyQueueActiveStore.set(false);
    studyQueueBacklogStore.set(false);
    studyQueueStore.set([]);
    studyQueueIndexStore.set(0);
    setStatusBarMessage(finished ? tMsg('studyQueue.finished', { n: total }) : tMsg('studyQueue.stopped'));
}

/**
 * Ouvre l'onglet où se prend la décision, sans quitter la file.
 * @param {string} tab
 */
export function actOnCurrent(tab) {
    activeTabStore.set(tab);
}

async function showCurrent() {
    const entry = get(studyQueueCurrentStore);
    if (!entry) return;
    await showImportedPosition(entry.positionId);
}

/**
 * Démarre la file du lot affiché par le compte rendu : ferme la fenêtre, puis
 * amène la première position.
 */
export async function beginStudyQueueFromReport() {
    const report = get(fileImportReportStore);
    if (!report?.id) return;
    showFileImportModalStore.set(false);
    fileImportModeStore.set('idle');
    await startStudyQueue(report.id);
}
