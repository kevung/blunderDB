import { writable } from 'svelte/store';

/**
 * L'assistant d'import de document : le Relevé en cours (texte JSON, tel que
 * le rendent AnalyzeDocument et ConfirmDocumentReleve), ce que le backend en
 * dit (résumé, entrées à vérifier) et les choix de la personne.
 *
 * @typedef {Object} DocumentImportState
 * @property {'idle'|'reading'|'review'|'importing'|'done'} phase
 * @property {string} path
 * @property {string} releve
 * @property {any} summary
 * @property {{items: any[], refused: number[], fallbackRanks: number[]}} review
 * @property {boolean} acceptComplete
 * @property {boolean} fallback
 * @property {string} collection
 * @property {any} result
 * @property {string} error
 * @property {{page: number, pages: number, diagrams: number, startedAt: number}|null} progress la lecture en cours, rendue par `document-analyze:progress`
 * @property {SheetEntry[]} sheet chaque entrée du Relevé, telle que la rend DocumentSheet
 * @property {SheetFilter} filter le compteur cliqué sur la planche de vérification
 * @property {number} rank l'entrée montrée sur la planche (0 : aucune)
 */

/**
 * @typedef {'sure'|'to-verify'|'uncertain'|'verified'} SheetFilter
 * @typedef {{rank: number, page: number, reference: string, position: string, state: 'sure'|'to-verify'|'uncertain', verified: boolean, fields: string[], rect?: {x: number, y: number, w: number, h: number}}} SheetEntry
 */

/**
 * Le compteur d'une entrée : vérifiée l'emporte sur l'état lu.
 * @param {SheetEntry} e
 * @returns {SheetFilter}
 */
export function sheetCategory(e) {
    return e.verified ? 'verified' : e.state;
}

/** @returns {DocumentImportState} */
export function initialDocumentImport() {
    return {
        phase: 'idle',
        path: '',
        releve: '',
        summary: null,
        review: { items: [], refused: [], fallbackRanks: [] },
        acceptComplete: false,
        fallback: false,
        collection: '',
        result: null,
        error: '',
        progress: null,
        sheet: [],
        filter: 'to-verify',
        rank: 0
    };
}

export const documentImportStore = writable(initialDocumentImport());
