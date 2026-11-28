import { get } from 'svelte/store';
import {
    CancelImport,
    GetAllCollections,
    AnalyzeDocument,
    SummarizeDocumentReleve,
    ReviewDocumentReleve,
    ConfirmDocumentReleve,
    ImportDocumentReleve,
    DocumentSheet,
    LoadDocumentReleve,
    SaveDocumentReleve,
    VerifyDocumentEntry,
    DocumentDiagramImage,
    ParsePositionText
} from '../../wailsjs/go/database/Database.js';
import { OpenDocumentDialog, ReadFileContent } from '../../wailsjs/go/gui/App.js';
import { EventsOn } from '../../wailsjs/runtime/runtime.js';
import { documentImportStore, initialDocumentImport, sheetCategory } from '../stores/documentImportStore.js';
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
    const [summary, review, sheet] = await Promise.all([SummarizeDocumentReleve(releve), requestReview(releve, !!choices.acceptComplete, !!choices.fallback), DocumentSheet(releve)]);
    if (!review) return;
    const entries = /** @type {import('../stores/documentImportStore.js').SheetEntry[]} */ (sheet || []);
    const prev = get(documentImportStore);
    // Au premier chargement, la planche s'ouvre sur ce qui reste à vérifier.
    const filter = prev.sheet.length > 0 ? prev.filter : firstFilter(entries);
    patch({ releve, summary, review, sheet: entries, filter, rank: keepRank(entries, filter, prev.rank), phase: 'review', error: '' });
}

/** @param {import('../stores/documentImportStore.js').SheetEntry[]} entries */
function firstFilter(entries) {
    for (const f of /** @type {const} */ (['to-verify', 'uncertain', 'verified', 'sure'])) {
        if (entries.some((e) => sheetCategory(e) === f)) return f;
    }
    return 'to-verify';
}

/**
 * L'entrée montrée reste la même tant qu'elle est sous le filtre ; sortie du filtre (on vient
 * de la valider), c'est celle qui prend sa place dans la liste, donc la suivante.
 * @param {import('../stores/documentImportStore.js').SheetEntry[]} entries
 * @param {import('../stores/documentImportStore.js').SheetFilter} filter
 * @param {number} rank
 */
function keepRank(entries, filter, rank) {
    const shown = entries.filter((e) => sheetCategory(e) === filter);
    if (shown.length === 0) return 0;
    if (shown.some((e) => e.rank === rank)) return rank;
    return (shown.find((e) => e.rank > rank) ?? shown[shown.length - 1]).rank;
}

/**
 * Garde le Relevé à côté du document : la vérification d'un livre se fait en plusieurs séances.
 * Un dossier en lecture seule n'empêche pas de vérifier ; le verdict vit jusqu'à l'import.
 * @param {string} path
 * @param {string} releve
 */
async function keep(path, releve) {
    try {
        await SaveDocumentReleve(path, releve);
    } catch (err) {
        logger.warn('the relevé could not be kept beside the document:', err);
    }
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
        // A sheet or an archive blunderDB printed imports by its codes, not its diagrams.
        const archive = await import('./paperArchiveService.js');
        if (await archive.isBlunderDBArchive(path)) {
            await archive.importPaperArchive([path]);
            return;
        }
        cancelRequested = false;
        documentImportStore.set({ ...initialDocumentImport(), phase: 'reading', path });
        setBusy('reading');
        openModal(MODAL.DOCUMENT_IMPORT);
        if (/\.json$/i.test(path)) {
            const file = await ReadFileContent(path);
            await load(file?.content ?? '');
        } else {
            const kept = await LoadDocumentReleve(path);
            if (kept) {
                await load(kept);
                return;
            }
            const startedAt = Date.now();
            off = EventsOn('document-analyze:progress', (/** @type {{page: number, pages: number, diagrams: number}} */ p) => {
                patch({ progress: { page: p.page, pages: p.pages, diagrams: p.diagrams, startedAt } });
            });
            const releve = await AnalyzeDocument(path, '', '');
            await keep(path, releve);
            await load(releve);
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
        const s = get(documentImportStore);
        const releve = await ConfirmDocumentReleve(s.releve);
        if (!/\.json$/i.test(s.path)) await keep(s.path, releve);
        await load(releve);
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
    thumbnails.clear();
    setBusy('idle');
    closeModal();
    documentImportStore.set(initialDocumentImport());
}

/**
 * Le verdict d'une personne sur une entrée : vérifiée ou non, avec la position corrigée quand
 * elle en donne une. Go l'écrit dans le Relevé, et le Relevé à côté du document.
 * @param {number} rank
 * @param {boolean} verified
 * @param {string} [position] XGID de la correction
 * @returns {Promise<boolean>} faux quand Go refuse (position illégale)
 */
export async function verifyEntry(rank, verified, position = '') {
    const s = get(documentImportStore);
    try {
        await load(await VerifyDocumentEntry(s.path, s.releve, rank, verified, position));
        return true;
    } catch (err) {
        patch({ error: messageOf(err) });
        return false;
    }
}

/** @param {import('../stores/documentImportStore.js').SheetFilter} filter */
export function setSheetFilter(filter) {
    const s = get(documentImportStore);
    const first = s.sheet.find((e) => sheetCategory(e) === filter);
    patch({ filter, rank: first?.rank ?? 0 });
}

/** @param {number} delta +1 l'entrée suivante sous le filtre, -1 la précédente */
export function moveSheet(delta) {
    const s = get(documentImportStore);
    const shown = s.sheet.filter((e) => sheetCategory(e) === s.filter);
    if (shown.length === 0) return;
    const at = shown.findIndex((e) => e.rank === s.rank);
    const next = Math.min(shown.length - 1, Math.max(0, (at < 0 ? 0 : at) + delta));
    patch({ rank: shown[next].rank });
}

// Vignettes recadrées par Go depuis le fichier d'origine : gardées le temps de l'assistant
// seulement, jamais écrites nulle part.
/** @type {Map<string, Promise<string>>} */
const thumbnails = new Map();

/**
 * La vignette du diagramme d'origine (data URL PNG), '' quand la page n'a pas d'image
 * (plateau dessiné par une police : lecture exacte) ou que le document n'est plus là.
 * @param {import('../stores/documentImportStore.js').SheetEntry} e
 */
export function diagramImage(e) {
    const path = get(documentImportStore).path;
    if (!e.rect || !path) return Promise.resolve('');
    const key = `${path}|${e.page}|${e.rect.x}|${e.rect.y}|${e.rect.w}|${e.rect.h}`;
    let p = thumbnails.get(key);
    if (!p) {
        p = DocumentDiagramImage(path, e.page, e.rect.x, e.rect.y, e.rect.w, e.rect.h).catch((err) => {
            logger.warn('no thumbnail for the diagram:', err);
            return '';
        });
        thumbnails.set(key, p);
    }
    return p;
}

/**
 * La position d'un XGID au format du store, pour la redessiner ; null quand Go ne la lit pas.
 * @param {string} xgid
 */
export async function positionOfXGID(xgid) {
    if (!xgid) return null;
    try {
        const r = await ParsePositionText(xgid.startsWith('XGID=') ? xgid : `XGID=${xgid}`);
        return r?.position ?? null;
    } catch {
        return null;
    }
}
