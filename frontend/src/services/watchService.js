import { get } from 'svelte/store';
import { EventsOn } from '../../wailsjs/runtime/runtime.js';
import { StartFolderWatch, StopFolderWatch, FolderWatchStatus, ImportFilesIn } from '../../wailsjs/go/gui/App.js';
import { GetWatchFolder, SaveWatchFolder } from '../../wailsjs/go/main/Config.js';
import { watchStatusStore, watchImportNoticeStore } from '../stores/watchStore.js';
import { databaseLoadedStore } from '../stores/databaseStore';
import { importWatchedFiles, maybeAutoAnalyzeAfterImport } from './importService.js';
import { activeGroupId, groupIdsStore, groupOnScreen, noteGroupJob } from '../stores/groupActivityStore.js';
import { tMsg } from '../i18n';
import { logger } from '../utils/logger.js';

// Le dossier surveillé, côté interface : le Go regarde, l'interface importe
// par le même chemin qu'un glisser-déposer (doublons, compte rendu, analyse
// automatique). La surveillance appartient au groupe de bibliothèques où elle
// a démarré : ses fichiers vont à ce groupe, affiché ou non.

let subscribed = false;
/** Les fichiers en attente, par groupe (0 : le groupe affiché). @type {Map<number, string[]>} */
const pending = new Map();

// Ce que le pipeline lit seul, sans l'interface : importé en fond dans un groupe
// non affiché. Le reste (une base à fusionner, un texte de position) attend que
// son groupe revienne à l'écran.
const BACKGROUND_EXTENSIONS = ['.xg', '.xgp', '.bgf', '.ogxm', '.sgf', '.mat'];

/**
 * Démarre l'écoute et, si configurée et qu'une base est ouverte, la
 * surveillance. Elle suit la base : sans base, elle marquerait des fichiers
 * vus sans les importer, perdus pour de bon ; ceux arrivés entre-temps
 * attendent dans la file.
 */
export async function initFolderWatch() {
    if (!subscribed) {
        subscribed = true;
        EventsOn('folder-watch:files', (/** @type {{ group: number, files: string[] }} */ event) => {
            void onWatchedFiles(event?.group ?? 0, event?.files ?? []);
        });
        databaseLoadedStore.subscribe((loaded) => {
            void onDatabaseLoadedChanged(loaded);
        });
        // Les fichiers d'un groupe qui revient à l'écran partent.
        activeGroupId.subscribe(() => {
            void drainPending();
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
 * Met en file les fichiers annoncés par la surveillance pour son groupe.
 * @param {number} group
 * @param {string[]} files
 */
async function onWatchedFiles(group, files) {
    if (!Array.isArray(files) || files.length === 0) return;
    pending.set(group, (pending.get(group) ?? []).concat(files));
    await drainPending();
}

/**
 * Vide la file un lot à la fois : jamais deux imports concurrents. Le groupe
 * affiché importe par le chemin d'un glisser-déposer ; un groupe non affiché
 * importe en fond ce que le pipeline lit seul. Sans base ouverte, la file est
 * CONSERVÉE : le dossier n'annoncera plus ces fichiers.
 */
async function drainPending() {
    if (importing || pending.size === 0) return;
    if (!get(databaseLoadedStore)) return;

    importing = true;
    try {
        for (;;) {
            const next = nextBatch();
            if (!next) break;
            if (next.onScreen) {
                const results = await importWatchedFiles(next.files);
                if (results) watchImportNoticeStore.set(results);
            } else {
                await importInBackground(next.group, next.files);
            }
        }
    } catch (error) {
        logger.error('importing watched files failed:', error);
    } finally {
        importing = false;
    }
}

/**
 * Le prochain lot à importer, retiré de la file ; null quand il ne reste que
 * des fichiers qui attendent le retour de leur groupe.
 * @returns {{ group: number, files: string[], onScreen: boolean } | null}
 */
function nextBatch() {
    if (!get(databaseLoadedStore)) return null;
    for (const [group, files] of pending) {
        // Un groupe fermé a emporté sa bibliothèque : ses fichiers n'ont plus où aller.
        if (group && !get(groupIdsStore).includes(group)) {
            pending.delete(group);
            continue;
        }
        if (groupOnScreen(group)) {
            pending.delete(group);
            return { group, files, onScreen: true };
        }
        const inBackground = files.filter((f) => BACKGROUND_EXTENSIONS.some((ext) => f.toLowerCase().endsWith(ext)));
        if (inBackground.length === 0) continue;
        const waiting = files.filter((f) => !inBackground.includes(f));
        if (waiting.length > 0) pending.set(group, waiting);
        else pending.delete(group);
        return { group, files: inBackground, onScreen: false };
    }
    return null;
}

/**
 * Importe en fond, dans la bibliothèque d'un groupe non affiché ; son en-tête
 * porte la pastille, et le compte rendu s'affiche à son retour.
 * @param {number} group
 * @param {string[]} files
 */
async function importInBackground(group, files) {
    noteGroupJob(group, 'running', 'watch');
    let summary;
    try {
        summary = await ImportFilesIn(group, files, files.length === 1 ? files[0] : `${files.length} files`);
    } catch (error) {
        logger.error('importing watched files into a library group failed:', error);
        noteGroupJob(group, 'done', 'watch', tMsg('status.watchImportDone', { succeeded: 0, skipped: 0, failed: files.length }));
        return;
    }
    const counts = { succeeded: summary.succeeded, skipped: summary.skipped, failed: summary.failed };
    // Revenu à l'écran entre-temps : le compte rendu se lit comme d'habitude.
    if (!noteGroupJob(group, 'done', 'watch', tMsg('status.watchImportDone', counts))) {
        if (summary.succeeded > 0) await maybeAutoAnalyzeAfterImport(group);
        return;
    }
    watchImportNoticeStore.set(counts);
    if (summary.succeeded > 0) await maybeAutoAnalyzeAfterImport(group);
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
        // Un statut absent vaut « rien n'est surveillé » : le panneau lit .running sans garde.
        watchStatusStore.set((await FolderWatchStatus()) ?? { running: false, folder: '', intervalSeconds: 0 });
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
