import { get } from 'svelte/store';
import { ImportStudyQueue } from '../../wailsjs/go/database/Database.js';
import { studyQueueStore, studyQueueIndexStore, studyQueueActiveStore, studyQueueCurrentStore } from '../stores/studyQueueStore.js';
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
