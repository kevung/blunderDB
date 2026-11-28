import { get } from 'svelte/store';
import { StartMatchSheet } from '../../wailsjs/go/gui/App.js';
import { printSheet } from './printService.js';
import { setStatusBarMessage } from './databaseService.js';
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

/** Le plus de matchs qu'un document imprime : printsheet.MaxMatches, que le moteur fait respecter. */
export const MAX_SHEET_MATCHES = 100;

/**
 * Construit la feuille des matchs, propose de l'enregistrer, puis l'ouvre dans
 * le navigateur, d'où elle s'imprime ; la construction tourne comme tâche du
 * panneau Calculs, annulable.
 * @param {number[]} matchIds
 * @param {{ [K in keyof typeof DEFAULT_SHEET_OPTIONS]: boolean }} options
 * @param {{ onLaunch?: () => void }} [hooks] appelé une fois la tâche lancée (la boîte se ferme)
 * @returns {Promise<string|null>} le chemin écrit, ou null.
 */
export async function printMatchSheet(matchIds, options, hooks = {}) {
    if (!matchIds.length) return null;
    if (matchIds.length > MAX_SHEET_MATCHES) {
        setStatusBarMessage(tMsg('matchSheet.tooMany', { n: matchIds.length, max: MAX_SHEET_MATCHES }));
        return null;
    }
    const opts = { ...options, language: get(language) };
    return printSheet(
        sheetFilename(matchIds),
        { saved: 'matchSheet.saved', failed: 'matchSheet.failed', cancelled: 'matchSheet.cancelled' },
        (path) => {
            setStatusBarMessage(tMsg('matchSheet.building'));
            return StartMatchSheet(matchIds, /** @type {any} */ (opts), path);
        },
        hooks
    );
}

/** @param {number[]} matchIds */
export function sheetFilename(matchIds) {
    return matchIds.length === 1 ? `blunderdb-match-${matchIds[0]}.html` : `blunderdb-matchs-${matchIds.length}.html`;
}
