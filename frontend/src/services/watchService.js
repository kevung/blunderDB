import { get } from 'svelte/store';
import { EventsOn } from '../../wailsjs/runtime/runtime.js';
import { StartFolderWatch, StopFolderWatch, FolderWatchStatus } from '../../wailsjs/go/gui/App.js';
import { GetWatchFolder, SaveWatchFolder } from '../../wailsjs/go/main/Config.js';
import { watchStatusStore, watchImportNoticeStore } from '../stores/watchStore.js';
import { databaseLoadedStore } from '../stores/databaseStore';
import { importWatchedFiles } from './importService.js';
import { logger } from '../utils/logger.js';

// Le dossier surveillé, côté interface : le Go regarde, l'interface importe
// par le même chemin qu'un glisser-déposer (doublons, compte rendu, analyse
// automatique).

let subscribed = false;
/** @type {string[]} */
let pending = [];

/**
 * Démarre l'écoute et, si configurée et qu'une base est ouverte, la
 * surveillance. Elle suit la base : sans base, elle marquerait des fichiers
 * vus sans les importer, perdus pour de bon ; ceux arrivés entre-temps
 * attendent dans la file.
 */
export async function initFolderWatch() {
    if (!subscribed) {
        subscribed = true;
        EventsOn('folder-watch:files', (/** @type {string[]} */ files) => {
            void onWatchedFiles(files);
        });
        databaseLoadedStore.subscribe((loaded) => {
            void onDatabaseLoadedChanged(loaded);
        });
    }
    await applyConfiguredWatch();
}

/** @type {boolean|null} */
let lastLoaded = null;

/** @param {boolean} loaded */
async function onDatabaseLoadedChanged(loaded) {
    if (loaded === lastLoaded) return;
    lastLoaded = loaded;
    if (loaded) {
        await applyConfiguredWatch();
        void drainPending();
    } else {
        await stopWatch();
    }
}

async function applyConfiguredWatch() {
    if (!get(databaseLoadedStore)) {
        await refreshWatchStatus();
        return;
    }
    try {
        const { on, path: folder, intervalSeconds } = await GetWatchFolder();
        if (on && folder) {
            await startWatch(folder, intervalSeconds);
        } else {
            await refreshWatchStatus();
        }
    } catch (error) {
        logger.error('could not read the watched-folder setting:', error);
    }
}

let importing = false;

/**
 * Met en file les fichiers annoncés par la surveillance.
 * @param {string[]} files
 */
async function onWatchedFiles(files) {
    if (!Array.isArray(files) || files.length === 0) return;
    pending = pending.concat(files);
    await drainPending();
}

/**
 * Vide la file un lot à la fois : jamais deux imports concurrents sur la même
 * base. Sans base ouverte, la file est CONSERVÉE : le dossier n'annoncera plus
 * ces fichiers.
 */
async function drainPending() {
    if (importing || pending.length === 0) return;
    if (!get(databaseLoadedStore)) return;

    importing = true;
    try {
        while (pending.length > 0 && get(databaseLoadedStore)) {
            const batch = pending;
            pending = [];
            const results = await importWatchedFiles(batch);
            if (results) watchImportNoticeStore.set(results);
        }
    } catch (error) {
        logger.error('importing watched files failed:', error);
    } finally {
        importing = false;
    }
}

/** @param {string} folder @param {number} intervalSeconds */
export async function startWatch(folder, intervalSeconds) {
    try {
        const status = await StartFolderWatch(folder, intervalSeconds || 0);
        watchStatusStore.set(status);
        return status;
    } catch (error) {
        logger.error('could not start watching the folder:', error);
        watchStatusStore.set({ running: false, folder: '', intervalSeconds: 0 });
        throw error;
    }
}

export async function stopWatch() {
    try {
        await StopFolderWatch();
    } catch (error) {
        logger.error('could not stop the folder watch:', error);
    }
    await refreshWatchStatus();
}

export async function refreshWatchStatus() {
    try {
        watchStatusStore.set(await FolderWatchStatus());
    } catch (error) {
        logger.error('could not read the folder-watch status:', error);
    }
}

/**
 * Enregistre le réglage et l'applique aussitôt.
 * @param {boolean} on @param {string} folder @param {number} intervalSeconds
 */
export async function saveWatchSetting(on, folder, intervalSeconds) {
    await SaveWatchFolder(on, folder, intervalSeconds || 0);
    if (on && folder) {
        await startWatch(folder, intervalSeconds);
    } else {
        await stopWatch();
    }
}
