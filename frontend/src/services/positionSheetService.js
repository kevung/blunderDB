import { get, writable } from 'svelte/store';
import { PositionSheetHTML, CountCollectionPositions } from '../../wailsjs/go/database/Database.js';
import { setStatusBarMessage } from './databaseService.js';
import { saveAndOpenPage } from './printService.js';
import { browsingLibrary, listedIds } from '../stores/positionStore.js';
import { activeCollectionStore } from '../stores/collectionStore.js';
import { lastSearchStore } from '../stores/searchHistoryStore.js';
import { logger } from '../utils/logger.js';
import { language, tMsg } from '../i18n';

// La liste de positions imprimable : une ligne par position en ArkID, dans un
// fichier HTML autonome construit par le moteur (printsheet.BuildPositions),
// le même que pour `sheet` et positions.sheet. L'écran ne fournit que la
// liste — toujours entière, jamais les seules lignes chargées — et les choix
// de la boîte d'impression.

/** Les choix par défaut de la boîte, ceux de printsheet.DefaultPositionsOptions. */
export const DEFAULT_POSITION_SHEET_OPTIONS = Object.freeze({
    analysis: true,
    verso: false,
    comments: true,
    expanded: false
});

/** Le plus de positions qu'un document imprime : printsheet.MaxPositions, que le moteur fait respecter. */
export const MAX_SHEET_POSITIONS = 3000;

/**
 * La demande d'impression en cours, null quand la boîte est fermée : la liste
 * (`{ ids }` ou `{ collection }`), son titre proposé et sa longueur.
 * @type {import('svelte/store').Writable<{ source: { ids?: number[], collection?: number }, title: string, count: number } | null>}
 */
export const positionPrintRequest = writable(null);

/**
 * Ouvre la boîte sur la liste parcourue : le résultat de la recherche, la
 * collection ouverte ou la bibliothèque, lue entière.
 */
export async function openBrowsedListPrint() {
    const ids = await listedIds();
    const collection = get(activeCollectionStore);
    const search = /** @type {{ command?: string } | null} */ (get(lastSearchStore));
    const title = browsingLibrary() ? tMsg('positionSheet.library') : collection?.name || search?.command || '';
    positionPrintRequest.set({ source: { ids }, title, count: ids.length });
}

/**
 * Ouvre la boîte sur une collection, résolue par le moteur (une collection
 * vivante par sa requête).
 * @param {{ id: number, name: string }} collection
 */
export async function openCollectionPrint(collection) {
    let count = 0;
    try {
        count = (await CountCollectionPositions(collection.id)) || 0;
    } catch (err) {
        logger.error('could not count the collection:', err);
    }
    positionPrintRequest.set({ source: { collection: collection.id }, title: collection.name, count });
}

/**
 * Ouvre la boîte sur des positions désignées, dans leur ordre (le panier).
 * @param {number[]} ids
 * @param {string} title
 */
export function openIdsPrint(ids, title) {
    positionPrintRequest.set({ source: { ids }, title, count: ids.length });
}

/**
 * Construit la liste, propose de l'enregistrer, puis l'ouvre dans le
 * navigateur, d'où elle s'imprime.
 * @param {{ ids?: number[], collection?: number }} source
 * @param {typeof DEFAULT_POSITION_SHEET_OPTIONS & { title?: string }} options
 * @returns {Promise<string|null>} le chemin écrit, ou null.
 */
export async function printPositionSheet(source, options) {
    if (source.ids && !source.ids.length) return null;
    if (source.ids && source.ids.length > MAX_SHEET_POSITIONS) {
        setStatusBarMessage(tMsg('positionSheet.tooMany', { n: source.ids.length, max: MAX_SHEET_POSITIONS }));
        return null;
    }
    setStatusBarMessage(tMsg('positionSheet.building'));
    let html;
    try {
        html = await PositionSheetHTML(/** @type {any} */ (source), /** @type {any} */ ({ ...options, language: get(language) }));
    } catch (err) {
        logger.error('could not build the position list:', err);
        setStatusBarMessage(tMsg('positionSheet.failed', { err }));
        return null;
    }
    return saveAndOpenPage(html, 'blunderdb-positions.html', { saved: 'positionSheet.saved', failed: 'positionSheet.failed' });
}
