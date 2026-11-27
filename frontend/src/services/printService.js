import { SaveBoardImageDialog, SaveBoardSVG, OpenLocalPage, SheetPDFAvailable, WriteSheetPDF } from '../../wailsjs/go/gui/App.js';
import { setStatusBarMessage } from './databaseService.js';
import { logger } from '../utils/logger.js';
import { tMsg } from '../i18n';

// Le dernier pas de toute impression : la page HTML autonome que le moteur a
// construite (package printsheet ou arkqr) devient un PDF A4, rendu hors
// écran par le moteur web du système, enregistré là où l'utilisateur le
// choisit puis ouvert dans la visionneuse, d'où il s'imprime ; le PDF porte
// les codes de réimport de la page. Une version sans ce rendu garde la page
// HTML, ouverte dans le navigateur.

/** @type {Promise<boolean>|null} */
let pdfAvailable = null;

/** @returns {Promise<boolean>} */
export function sheetPDFAvailable() {
    if (!pdfAvailable) {
        pdfAvailable = Promise.resolve()
            .then(() => SheetPDFAvailable())
            .then((ok) => ok === true)
            .catch(() => false);
    }
    return pdfAvailable;
}

/** @param {string} filename */
export function pdfName(filename) {
    return filename.replace(/\.html?$/i, '') + '.pdf';
}

/**
 * @param {string} html
 * @param {string} filename le nom proposé
 * @param {{ saved: string, failed: string }} keys les messages de la barre d'état
 * @returns {Promise<string|null>} le chemin écrit, ou null.
 */
export async function saveAndOpenPage(html, filename, keys) {
    try {
        const pdf = await sheetPDFAvailable();
        const path = await SaveBoardImageDialog(pdf ? 'pdf' : 'html', pdf ? pdfName(filename) : filename);
        if (!path) return null;
        if (pdf) await WriteSheetPDF(path, html);
        else await SaveBoardSVG(path, html);
        setStatusBarMessage(tMsg(keys.saved, { path }));
        openPage(path);
        return path;
    } catch (err) {
        logger.error('could not save the page:', err);
        setStatusBarMessage(tMsg(keys.failed, { err }));
        return null;
    }
}

/** @param {string} path */
export async function openPage(path) {
    try {
        await OpenLocalPage(path);
    } catch (err) {
        logger.error('could not open the page:', err);
    }
}
