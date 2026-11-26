import { SaveBoardImageDialog, SaveBoardSVG, OpenLocalPage } from '../../wailsjs/go/gui/App.js';
import { setStatusBarMessage } from './databaseService.js';
import { logger } from '../utils/logger.js';
import { tMsg } from '../i18n';

// Le dernier pas de toute impression : la page HTML autonome que le moteur a
// construite (package printsheet ou arkqr) est enregistrée là où l'utilisateur
// le choisit, puis ouverte dans le navigateur, d'où elle s'imprime.

/**
 * @param {string} html
 * @param {string} filename le nom proposé
 * @param {{ saved: string, failed: string }} keys les messages de la barre d'état
 * @returns {Promise<string|null>} le chemin écrit, ou null.
 */
export async function saveAndOpenPage(html, filename, keys) {
    try {
        const path = await SaveBoardImageDialog('html', filename);
        if (!path) return null;
        await SaveBoardSVG(path, html);
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
