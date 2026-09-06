import { get } from 'svelte/store';
import { EventsOn } from '../../wailsjs/runtime/runtime.js';
import { StartFolderWatch, StopFolderWatch, FolderWatchStatus } from '../../wailsjs/go/gui/App.js';
import { GetWatchFolder, SaveWatchFolder } from '../../wailsjs/go/main/Config.js';
import { watchStatusStore, watchImportNoticeStore } from '../stores/watchStore.js';
import { databaseLoadedStore } from '../stores/databaseStore';
import { importWatchedFiles } from './importService.js';
import { logger } from '../utils/logger.js';

// Le dossier surveillé, côté interface (#258, fiche I.2).
//
// Le Go REGARDE, l'interface IMPORTE. Les chemins arrivent par un événement et
// repartent dans le même chemin d'import qu'un glisser-déposer : détection des
// doublons, compte rendu, analyse automatique — rien n'est écrit deux fois, et
// un import surveillé est par construction le même import qu'un import manuel.

let subscribed = false;
let pending = [];

/**
 * Démarre l'écoute et, si la configuration le demande et qu'une base est
 * ouverte, la surveillance.
 *
 * La surveillance suit la base : sans base ouverte, il n'y a rien où importer,
 * et une surveillance qui tourne dans le vide marquerait les fichiers comme
 * vus sans les avoir importés — ils seraient perdus pour de bon. Ouvrir une
 * base la démarre, la fermer l'arrête, et les fichiers arrivés entre-temps
 * attendent dans la file plutôt que d'être jetés.
 */
export async function initFolderWatch() {
    if (!subscribed) {
        subscribed = true;
        EventsOn('folder-watch:files', (files) => {
            void onWatchedFiles(files);
        });
        databaseLoadedStore.subscribe((loaded) => {
            void onDatabaseLoadedChanged(loaded);
        });
    }
    await applyConfiguredWatch();
}

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

/**
 * Importe les fichiers annoncés par la surveillance, en file : deux salves
 * rapprochées ne doivent pas lancer deux imports concurrents sur la même base.
 * @param {string[]} files
 */
let importing = false;
async function onWatchedFiles(files) {
    if (!Array.isArray(files) || files.length === 0) return;
    pending = pending.concat(files);
    await drainPending();
}

/**
 * Vide la file, un lot à la fois : deux salves rapprochées ne doivent pas
 * lancer deux imports concurrents sur la même base.
 *
 * Sans base ouverte la file est CONSERVÉE, jamais jetée : le dossier a déjà
 * annoncé ces fichiers et ne les annoncera plus, donc les oublier ici les
 * perdrait définitivement. Ils partent à la prochaine ouverture.
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
 * Enregistre le réglage ET applique-le tout de suite : un réglage qui ne prend
 * effet qu'au prochain démarrage est un réglage dont on doute.
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
