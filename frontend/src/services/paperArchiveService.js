import { SaveBoardImageDialog, OpenPaperArchiveDialog, StartPaperArchiveExport, StartPaperArchiveImport, IsBlunderDBArchive } from '../../wailsjs/go/gui/App.js';
import { runLongOp } from './longOpService.js';
import { setStatusBarMessage } from './databaseService.js';
import { openPage, sheetPDFAvailable, pdfName } from './printService.js';
import { logger } from '../utils/logger.js';
import { tMsg } from '../i18n';
import { matchListStore } from '../stores/matchListStore.js';

// L'archive papier dans l'interface : les mêmes ExportPaperArchive et
// ImportPaperArchive que `archive export|import` et le serveur (ADR-0119),
// lancés comme tâches du panneau Calculs (progression, Annuler) : la boîte se
// ferme dès le lancement (`onLaunch`), l'issue s'écrit dans la barre d'état.

/**
 * Écrit l'archive des matchs (aucun : toute la bibliothèque) en PDF — en page
 * HTML là où ce rendu manque —, là où l'utilisateur le choisit, puis l'ouvre.
 * @param {number[]} matchIds
 * @param {{ onLaunch?: () => void }} [hooks] appelé une fois la tâche lancée
 * @returns {Promise<string|null>} le chemin écrit, ou null.
 */
export async function exportPaperArchive(matchIds, hooks = {}) {
    try {
        const name = matchIds.length === 1 ? `blunderdb-archive-${matchIds[0]}.html` : 'blunderdb-archive.html';
        const pdf = await sheetPDFAvailable();
        const path = await SaveBoardImageDialog(pdf ? 'pdf' : 'html', pdf ? pdfName(name) : name);
        if (!path) return null;
        setStatusBarMessage(tMsg('paperArchive.exporting'));
        const outcome = await runLongOp('paper-archive', async () => {
            const id = await StartPaperArchiveExport(matchIds, path);
            hooks.onLaunch?.();
            return id;
        });
        if (outcome.status === 'cancelled') {
            setStatusBarMessage(tMsg('paperArchive.cancelled'));
            return null;
        }
        if (outcome.status === 'error') throw outcome.message;
        const res = outcome.result;
        setStatusBarMessage(tMsg('paperArchive.exported', { path, matches: res?.matches ?? 0, codes: res?.codes ?? 0, skipped: res?.skipped?.length ?? 0 }));
        openPage(path);
        return path;
    } catch (err) {
        logger.error('could not export the paper archive:', err);
        setStatusBarMessage(tMsg('paperArchive.failed', { err }));
        return null;
    }
}

// The backend's words for a PDF that blunderDB did not print, or printed
// without reimport codes (ingest.ErrNoPDFPayload).
const NO_PDF_PAYLOAD = 'carries no blunderDB archive';

/**
 * Lit des pages d'archive (le fichier exporté, son PDF, une feuille ou une
 * liste imprimée en PDF, des scans ou des photos) et importe leurs matchs et
 * leurs positions par l'import ordinaire. Sans chemins, les demande.
 * @param {string[]} [given] des fichiers déjà choisis (un dépôt sur la fenêtre)
 * @param {{ onLaunch?: () => void }} [hooks] appelé une fois la tâche lancée
 * @returns {Promise<import('../../wailsjs/go/models').database.PaperArchiveImport|null>}
 */
export async function importPaperArchive(given, hooks = {}) {
    try {
        const paths = given?.length ? given : (await OpenPaperArchiveDialog()) || [];
        if (!paths.length) return null;
        setStatusBarMessage(tMsg('paperArchive.importing'));
        const outcome = await runLongOp('paper-archive', async () => {
            const id = await StartPaperArchiveImport(paths);
            hooks.onLaunch?.();
            return id;
        });
        if (outcome.status === 'cancelled') {
            setStatusBarMessage(tMsg('paperArchive.cancelled'));
            return null;
        }
        if (outcome.status === 'error') throw outcome.message;
        /** @type {import('../../wailsjs/go/models').database.PaperArchiveImport} */
        const res = outcome.result;
        setStatusBarMessage(
            tMsg('paperArchive.imported', {
                imported: res?.imported?.length ?? 0,
                duplicates: res?.duplicates ?? 0,
                positions: res?.positions ?? 0,
                knownPositions: res?.knownPositions ?? 0,
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
        if (String(err).includes(NO_PDF_PAYLOAD)) setStatusBarMessage(tMsg('paperArchive.noPdfPayload'));
        else setStatusBarMessage(tMsg('paperArchive.failed', { err }));
        return null;
    }
}

/**
 * Le fichier (PDF ou HTML) est-il une feuille, une liste ou une archive que
 * blunderDB a produite, codes de réimport compris ? La seule question que
 * posent le dépôt, Ctrl-I et l'assistant d'import de document.
 * @param {string} path
 * @returns {Promise<boolean>}
 */
export async function isBlunderDBArchive(path) {
    try {
        return (await IsBlunderDBArchive(path)) === true;
    } catch (err) {
        logger.error('could not read the file:', err);
        return false;
    }
}

/**
 * Range des PDF apportés (dépôt, Ctrl-I) : ceux que blunderDB a imprimés,
 * codes de réimport joints, s'importent en archive papier ; un autre PDF est
 * sans doute un livre de positions, confié à l'assistant d'import de document,
 * qui en lit un à la fois.
 * @param {string[]} pdfs
 */
export async function importPDFs(pdfs) {
    /** @type {string[]} */
    const archives = [];
    /** @type {string[]} */
    const documents = [];
    for (const p of pdfs) (await isBlunderDBArchive(p)) ? archives.push(p) : documents.push(p);
    if (archives.length) await importPaperArchive(archives);
    if (documents.length) {
        if (documents.length > 1) logger.warn('one document at a time; left aside:', documents.slice(1));
        const { startDocumentImport } = await import('./documentImportService.js');
        await startDocumentImport(documents[0]);
    }
}
