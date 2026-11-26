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
 */

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
        error: ''
    };
}

export const documentImportStore = writable(initialDocumentImport());
