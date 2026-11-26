import { get } from 'svelte/store';
import { GetAllCollections, AnalyzeDocument, SummarizeDocumentReleve, ReviewDocumentReleve, ConfirmDocumentReleve, ImportDocumentReleve } from '../../wailsjs/go/database/Database.js';
import { OpenDocumentDialog, ReadFileContent } from '../../wailsjs/go/gui/App.js';
import { documentImportStore, initialDocumentImport } from '../stores/documentImportStore.js';
import { openModal, closeModal, MODAL } from '../stores/uiStore.js';
import { setStatusBarMessage } from './databaseService.js';
import { collectionsStore } from '../stores/collectionStore.js';
import { logger } from '../utils/logger.js';
import { tMsg } from '../i18n';

// L'import de document dans l'interface : les mêmes AnalyzeDocument,
// ImportDocumentReleve et règles de certitude que `document analyze|check|
// import` ; l'assistant ne décide rien que le Relevé ne dise déjà.

/**
 * Ce que l'import refuserait avec les choix courants : les rangs à vérifier,
 * moins ceux que le repli prend sous le contexte de leur position.
 * @param {{unverified?: number[], unverifiedComplete?: number[], fallback?: number[], fallbackComplete?: number[]}|null} summary
 * @param {{acceptComplete: boolean, fallback: boolean}} choices
 * @returns {number[]}
 */
export function refusedRanks(summary, choices) {
    if (!summary) return [];
    const refused = (choices.acceptComplete ? summary.unverifiedComplete : summary.unverified) ?? [];
    if (!choices.fallback) return refused;
    const taken = new Set((choices.acceptComplete ? summary.fallbackComplete : summary.fallback) ?? []);
    return refused.filter((r) => !taken.has(r));
}

/**
 * Les défauts du document et le sens de jeu qui attendent une confirmation.
 * @param {{defaults?: {confirmed: boolean, direction?: {confirmed: boolean}}}|null} summary
 */
export function pendingConfirmations(summary) {
    const d = summary?.defaults;
    return { defaults: !!d && !d.confirmed, direction: !!d?.direction && !d.direction.confirmed };
}

function patch(p) {
    documentImportStore.update((s) => ({ ...s, ...p }));
}

async function load(releve) {
    const [summary, review] = await Promise.all([SummarizeDocumentReleve(releve), ReviewDocumentReleve(releve)]);
    patch({ releve, summary, review: review ?? [], phase: 'review', error: '' });
}

/** Choisit un document (PDF, image ou Relevé .json), le lit et ouvre l'assistant. */
export async function startDocumentImport() {
    let path = '';
    try {
        path = (await OpenDocumentDialog()) || '';
        if (!path) return;
        documentImportStore.set({ ...initialDocumentImport(), phase: 'reading', path });
        openModal(MODAL.DOCUMENT_IMPORT);
        if (/\.json$/i.test(path)) {
            const file = await ReadFileContent(path);
            await load(file?.content ?? '');
        } else {
            await load(await AnalyzeDocument(path, '', ''));
        }
    } catch (err) {
        logger.error('could not read the document:', err);
        if (path) {
            patch({ phase: 'review', error: String(err?.message ?? err) });
        } else {
            setStatusBarMessage(tMsg('documentImport.failed', { err }));
        }
    }
}

/** Confirme les défauts du document et/ou le sens de jeu, puis relit le Relevé. */
export async function confirmDocument({ defaults, direction }) {
    try {
        await load(await ConfirmDocumentReleve(get(documentImportStore).releve, !!defaults, !!direction));
    } catch (err) {
        patch({ error: String(err?.message ?? err) });
    }
}

export function setChoice(name, value) {
    patch({ [name]: value });
}

/** Importe le Relevé dans la collection nommée (vide : le titre du document). */
export async function runDocumentImport() {
    const s = get(documentImportStore);
    patch({ phase: 'importing', error: '' });
    try {
        const result = await ImportDocumentReleve(s.releve, s.collection, s.acceptComplete, s.fallback, false);
        patch({ phase: 'done', result });
        setStatusBarMessage(tMsg('documentImport.imported', { collection: result?.collection ?? '', added: result?.added ?? 0, repeated: result?.repeated ?? 0 }));
        collectionsStore.set((await GetAllCollections()) || []);
    } catch (err) {
        logger.error('could not import the document:', err);
        patch({ phase: 'review', error: String(err?.message ?? err) });
    }
}

export function closeDocumentImport() {
    closeModal();
    documentImportStore.set(initialDocumentImport());
}
