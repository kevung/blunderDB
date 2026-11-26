import { ExportPaperArchive, ImportPaperArchive } from '../../wailsjs/go/database/Database.js';
import { SaveBoardImageDialog, OpenPaperArchiveDialog } from '../../wailsjs/go/gui/App.js';
import { setStatusBarMessage } from './databaseService.js';
import { openPage } from './printService.js';
import { logger } from '../utils/logger.js';
import { tMsg } from '../i18n';
import { matchListStore } from '../stores/matchListStore.js';

// L'archive papier dans l'interface : les mêmes ExportPaperArchive et
// ImportPaperArchive que `archive export|import` et le serveur (ADR-0119).

/**
 * Écrit la page d'archive des matchs (aucun : toute la bibliothèque), là où
 * l'utilisateur le choisit, puis l'ouvre dans le navigateur.
 * @param {number[]} matchIds
 * @returns {Promise<string|null>} le chemin écrit, ou null.
 */
export async function exportPaperArchive(matchIds) {
    try {
        const name = matchIds.length === 1 ? `blunderdb-archive-${matchIds[0]}.html` : 'blunderdb-archive.html';
        const path = await SaveBoardImageDialog('html', name);
        if (!path) return null;
        setStatusBarMessage(tMsg('paperArchive.exporting'));
        const res = await ExportPaperArchive(matchIds, path, '');
        setStatusBarMessage(tMsg('paperArchive.exported', { path, matches: res?.matches ?? 0, codes: res?.codes ?? 0, skipped: res?.skipped?.length ?? 0 }));
        openPage(path);
        return path;
    } catch (err) {
        logger.error('could not export the paper archive:', err);
        setStatusBarMessage(tMsg('paperArchive.failed', { err }));
        return null;
    }
}

/**
 * Lit des pages d'archive (le fichier exporté, des scans ou des photos) et
 * importe leurs matchs par l'import ordinaire.
 * @returns {Promise<import('../../wailsjs/go/models').database.PaperArchiveImport|null>}
 */
export async function importPaperArchive() {
    try {
        const paths = (await OpenPaperArchiveDialog()) || [];
        if (!paths.length) return null;
        setStatusBarMessage(tMsg('paperArchive.importing'));
        const res = await ImportPaperArchive(paths);
        setStatusBarMessage(
            tMsg('paperArchive.imported', {
                imported: res?.imported?.length ?? 0,
                duplicates: res?.duplicates ?? 0,
                missing: res?.missing?.length ?? 0,
                failed: res?.failed?.length ?? 0
            })
        );
        // Every entry to the import (match menu, empty panel, command mode)
        // shows the matches it brought without its caller asking.
        if (res?.imported?.length) matchListStore.reload();
        return res;
    } catch (err) {
        logger.error('could not import the paper archive:', err);
        setStatusBarMessage(tMsg('paperArchive.failed', { err }));
        return null;
    }
}
