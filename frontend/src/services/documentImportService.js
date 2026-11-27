import { get } from 'svelte/store';
import { CancelImport, GetAllCollections, AnalyzeDocument, SummarizeDocumentReleve, ReviewDocumentReleve, ConfirmDocumentReleve, ImportDocumentReleve } from '../../wailsjs/go/database/Database.js';
import { OpenDocumentDialog, ReadFileContent } from '../../wailsjs/go/gui/App.js';
import { EventsOn } from '../../wailsjs/runtime/runtime.js';
import { documentImportStore, initialDocumentImport } from '../stores/documentImportStore.js';
import { openModal, closeModal, MODAL } from '../stores/uiStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { readOfLibrary, importForegroundStore } from '../stores/groupActivityStore.js';
import { collectionsStore } from '../stores/collectionStore.js';
import { logger } from '../utils/logger.js';
import { tMsg } from '../i18n';

// L'import de document dans l'interface : les mêmes AnalyzeDocument,
// ImportDocumentReleve et règles de certitude que `document analyze|check|
// import` ; l'assistant ne décide rien que le Relevé ne dise déjà.

// Pendant la lecture et l'import, aucun changement de groupe : l'import écrit dans la
// bibliothèque à l'écran, la lecture n'y touche pas mais le résultat y retourne.
/** @param {unknown} err */
function messageOf(err) {
    return err instanceof Error ? err.message : String(err);
}

/** @param {string} phase */
function setBusy(phase) {
    importForegroundStore.set(phase === 'reading' || phase === 'importing');
}

/** @param {Partial<import('../stores/documentImportStore.js').DocumentImportState>} p */
function patch(p) {
    documentImportStore.update((s) => ({ ...s, ...p }));
    if (p.phase) setBusy(p.phase);
}

// Ce que l'import refuse, et ce que le repli offrirait, vient de Go
// (document.Refused) : l'assistant n'en refait pas le calcul.
// Jeton de la dernière demande de revue : une réponse arrivée après un choix plus récent
// (cases cochées, Relevé reconfirmé) ne correspond plus à l'écran et est ignorée.
let reviewToken = 0;

/**
 * @param {string} releve
 * @param {boolean} acceptComplete
 * @param {boolean} fallback
 * @returns {Promise<any|null>} null quand une demande plus récente a pris le relais
 */
async function requestReview(releve, acceptComplete, fallback) {
    const token = ++reviewToken;
    const review = await ReviewDocumentReleve(releve, acceptComplete, fallback);
    return token === reviewToken ? review : null;
}

/** @param {string} releve */
async function load(releve, choices = get(documentImportStore)) {
    const [summary, review] = await Promise.all([SummarizeDocumentReleve(releve), requestReview(releve, !!choices.acceptComplete, !!choices.fallback)]);
    if (!review) return;
    patch({ releve, summary, review, phase: 'review', error: '' });
}

// Annulation demandée par la personne : l'erreur qui revient du moteur n'en est pas une.
let cancelRequested = false;

/** @param {unknown} err */
function isCancellation(err) {
    return cancelRequested || /context canceled/i.test(messageOf(err));
}

/**
 * Lit un document (PDF, image, dossier d'images ou Relevé .json) et ouvre l'assistant.
 * Sans chemin, la personne le choisit dans la boîte de fichiers.
 * @param {string} [given]
 */
export async function startDocumentImport(given) {
    let path = '';
    let off = () => {};
    try {
        path = given || (await OpenDocumentDialog()) || '';
        if (!path) return;
        cancelRequested = false;
        documentImportStore.set({ ...initialDocumentImport(), phase: 'reading', path });
        setBusy('reading');
        openModal(MODAL.DOCUMENT_IMPORT);
        if (/\.json$/i.test(path)) {
            const file = await ReadFileContent(path);
            await load(file?.content ?? '');
        } else {
            const startedAt = Date.now();
            off = EventsOn('document-analyze:progress', (/** @type {{page: number, pages: number, diagrams: number}} */ p) => {
                patch({ progress: { page: p.page, pages: p.pages, diagrams: p.diagrams, startedAt } });
            });
            await load(await AnalyzeDocument(path, '', ''));
        }
    } catch (err) {
        if (path && isCancellation(err)) {
            closeDocumentImport();
            setStatusBarMessage(tMsg('documentImport.cancelled'));
            return;
        }
        logger.error('could not read the document:', err);
        if (path) {
            patch({ phase: 'review', error: messageOf(err) });
        } else {
            setStatusBarMessage(tMsg('documentImport.failed', { err }));
        }
    } finally {
        off();
        patch({ progress: null });
    }
}

/** Annule la lecture ou l'import en cours ; hors de là, ferme l'assistant. */
export function cancelDocumentImport() {
    const phase = get(documentImportStore).phase;
    if (phase !== 'reading' && phase !== 'importing') {
        closeDocumentImport();
        return;
    }
    cancelRequested = true;
    CancelImport().catch((err) => logger.error('Error calling CancelImport:', err));
}

/** Confirme d'un geste les défauts du document et son sens de jeu, puis relit le Relevé. */
export async function confirmDocument() {
    try {
        await load(await ConfirmDocumentReleve(get(documentImportStore).releve));
    } catch (err) {
        patch({ error: messageOf(err) });
    }
}

/** Un choix d'import (entrées complètes, repli) : la liste des refus est relue côté Go. */
/**
 * @param {'acceptComplete'|'fallback'|'collection'} name
 * @param {boolean|string} value
 */
export async function setChoice(name, value) {
    const s = get(documentImportStore);
    patch({ [name]: value });
    if (name === 'collection') return;
    try {
        const review = await requestReview(s.releve, name === 'acceptComplete' ? value === true : s.acceptComplete, name === 'fallback' ? value === true : s.fallback);
        if (review) patch({ review });
    } catch (err) {
        patch({ error: messageOf(err) });
    }
}

/** Importe le Relevé dans la collection nommée (vide : le titre du document). */
export async function runDocumentImport() {
    const s = get(documentImportStore);
    patch({ phase: 'importing', error: '' });
    cancelRequested = false;
    try {
        const result = await ImportDocumentReleve(s.releve, s.collection, s.acceptComplete, s.fallback, false);
        setStatusBarMessage(tMsg('documentImport.imported', { collection: result?.collection ?? '', added: result?.added ?? 0, repeated: result?.repeated ?? 0 }));
        try {
            collectionsStore.set((await readOfLibrary(() => GetAllCollections())) || []);
        } catch (err) {
            logger.warn('collections not refreshed after the document import:', err);
        }
        patch({ phase: 'done', result });
    } catch (err) {
        if (isCancellation(err)) {
            setStatusBarMessage(tMsg('documentImport.importCancelled'));
            patch({ phase: 'review', error: '' });
            return;
        }
        logger.error('could not import the document:', err);
        patch({ phase: 'review', error: messageOf(err) });
    }
}

export function closeDocumentImport() {
    setBusy('idle');
    closeModal();
    documentImportStore.set(initialDocumentImport());
}
