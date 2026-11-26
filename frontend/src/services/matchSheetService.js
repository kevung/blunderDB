import { get } from 'svelte/store';
import { MatchSheetHTML } from '../../wailsjs/go/database/Database.js';
import { SaveBoardImageDialog, SaveBoardSVG, OpenLocalPage } from '../../wailsjs/go/gui/App.js';
import { setStatusBarMessage } from './databaseService.js';
import { logger } from '../utils/logger.js';
import { language, tMsg } from '../i18n';

// La feuille de match imprimable : un fichier HTML autonome que le navigateur
// imprime ou enregistre en PDF. Le document est construit par le moteur
// (package printsheet), le même que pour `match sheet` et matches.sheet ;
// l'écran ne fournit que les choix de la boîte d'impression.

/** Les choix par défaut de la boîte, ceux de printsheet.DefaultMatchOptions. */
export const DEFAULT_SHEET_OPTIONS = Object.freeze({
    opponentBlunders: false,
    analysis: true,
    comments: true,
    expanded: false,
    reimportCode: true
});

/**
 * Construit la feuille des matchs, propose de l'enregistrer, puis l'ouvre dans
 * le navigateur, d'où elle s'imprime.
 * @param {number[]} matchIds
 * @param {typeof DEFAULT_SHEET_OPTIONS} options
 * @returns {Promise<string|null>} le chemin écrit, ou null.
 */
export async function printMatchSheet(matchIds, options) {
    if (!matchIds.length) return null;
    setStatusBarMessage(tMsg('matchSheet.building'));
    let html;
    try {
        html = await MatchSheetHTML(matchIds, { ...options, language: get(language) });
    } catch (err) {
        logger.error('could not build the match sheet:', err);
        setStatusBarMessage(tMsg('matchSheet.failed', { err }));
        return null;
    }
    try {
        const path = await SaveBoardImageDialog('html', sheetFilename(matchIds));
        if (!path) return null;
        await SaveBoardSVG(path, html);
        setStatusBarMessage(tMsg('matchSheet.saved', { path }));
        try {
            await OpenLocalPage(path);
        } catch (err) {
            logger.error('could not open the match sheet:', err);
        }
        return path;
    } catch (err) {
        logger.error('could not save the match sheet:', err);
        setStatusBarMessage(tMsg('matchSheet.failed', { err }));
        return null;
    }
}

/** @param {number[]} matchIds */
export function sheetFilename(matchIds) {
    return matchIds.length === 1 ? `blunderdb-match-${matchIds[0]}.html` : `blunderdb-matchs-${matchIds.length}.html`;
}
