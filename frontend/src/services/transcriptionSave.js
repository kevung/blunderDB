/**
 * transcriptionSave.js — enregistrer un brouillon en Match, l'exporter en
 * `.mat`, le fermer. Hors du panneau, client du moteur qui ne décide de rien
 * (ADR-0045 règle 9). Tout l'état (incohérences, coup illégal, match id) est
 * lu dans le document annoté renvoyé par le Go.
 *
 * L'enregistrement suit fonctionnel.md §4 : incohérences ANNONCÉES, jamais
 * opposées (ADR-0044) ; Match créé puis remplacé (ADR-0045 §2) ; lot
 * d'analyse ciblé sur les positions nouvelles (ADR-0013, ADR-0045 §8), suivi
 * par la barre d'état via `gammonnet-batch:*`.
 */

import { get, writable } from 'svelte/store';

import { translate, tMsg } from '../i18n';
import { logger } from '../utils/logger.js';
import { statusBarTextStore } from '../stores/uiStore.js';
import { confirmAction } from './confirmService.js';
import { SaveTranscriptionAsMatch, SuggestTranscriptionMatFilename, ExportTranscriptionMAT, CloseTranscription, PendingTranscriptionAnalysis } from '../../wailsjs/go/database/Database.js';
import { OpenExportMatDialog, StartGammonNetMatchBatch } from '../../wailsjs/go/gui/App.js';
import { GetGammonNetAnalysisPly, GetGammonNetPruneK } from '../../wailsjs/go/main/Config.js';

/**
 * Le dernier enregistrement de cette session : `{ id, matchId, at,
 * signature }`, ou null. En mémoire : la base ne garde que `match_id` ; ce
 * store ajoute l'heure et la signature, d'où « modifié depuis ».
 *
 * @type {import('svelte/store').Writable<{id: number, matchId: number, at: number, signature: string} | null>}
 */
export const transcriptionSaveStore = writable(null);

/** Repart de zéro : à la fermeture d'un brouillon, et au changement de base. */
export function resetTranscriptionSave() {
    transcriptionSaveStore.set(null);
}

/**
 * La signature du document (en-tête et Actions, rien de dérivé) : même
 * signature, même Match.
 *
 * @param {any} annotated
 */
export function documentSignature(annotated) {
    const doc = annotated?.document;
    if (!doc) return '';
    return JSON.stringify([doc.header ?? null, doc.actions ?? []]);
}

/**
 * @param {any} annotated
 * @returns {Set<string>}
 */
export function inconsistencyKinds(annotated) {
    const kinds = new Set();
    for (const info of annotated?.actions ?? []) {
        for (const flag of info?.inconsistencies ?? []) {
            if (flag?.kind) kinds.add(flag.kind);
        }
    }
    return kinds;
}

/** @param {any} annotated */
export function hasInconsistency(annotated) {
    return inconsistencyKinds(annotated).size > 0;
}

/**
 * Un coup que les règles n'atteignent pas (« Invalid move » dans gnubg et XG),
 * ou un jet incohérent qui le requalifie en illégal : avertissement à l'export.
 *
 * @param {any} annotated
 */
export function hasIllegalMove(annotated) {
    const kinds = inconsistencyKinds(annotated);
    return kinds.has('illegal_move') || kinds.has('inconsistent_dice');
}

/**
 * Le match que le brouillon possède déjà, 0 s'il n'a jamais été enregistré.
 *
 * @param {any} annotated
 * @returns {number}
 */
export function savedMatchID(annotated) {
    return annotated?.document?.header?.match_id ?? 0;
}

/**
 * Ce que la barre du brouillon dit de son MATCH, jamais du salut du brouillon
 * (ADR-0048 décision 12), le brouillon étant écrit après chaque Action. Rendu
 * en clé i18n et paramètres, testable sans la langue. Prend le brouillon
 * entier : l'enregistrement de la session ne vaut que pour celui qui l'a fait.
 *
 * @param {{id: number, annotated: any} | null | undefined} draft
 * @param {{id: number, matchId: number, at: number, signature: string} | null | undefined} saved
 * @param {number} [now]
 */
export function draftSaveState(draft, saved, now = Date.now()) {
    const annotated = draft?.annotated ?? null;
    const own = saved && draft && saved.id === draft.id ? saved : null;
    const matchId = savedMatchID(annotated) || own?.matchId || 0;
    if (!matchId) return { key: 'transcription.stateNoMatch', params: {} };

    // Un brouillon enregistré lors d'une session précédente porte son match id
    // et rien d'autre : l'heure de l'enregistrement n'a jamais été écrite nulle
    // part (ADR-0045 §8 — aucun état n'est stocké pour cela).
    if (!own || own.matchId !== matchId) {
        return { key: 'transcription.stateMatchUpToDate', params: { id: matchId } };
    }
    if (own.signature !== documentSignature(annotated)) {
        return { key: 'transcription.stateMatchBehind', params: {} };
    }
    const minutes = Math.floor(Math.max(0, now - own.at) / 60000);
    if (minutes < 1) return { key: 'transcription.stateMatchJustUpdated', params: {} };
    if (minutes < 60) return { key: 'transcription.stateMatchMinutes', params: { n: minutes } };
    return { key: 'transcription.stateMatchHours', params: { n: Math.floor(minutes / 60) } };
}

/**
 * Enregistre le brouillon en Match (création, puis remplacement au même `id`),
 * puis lance le lot d'analyse ciblé. Les incohérences sont annoncées ; seul
 * l'utilisateur peut refuser.
 *
 * @param {any} draft
 * @returns le résultat du moteur, ou null si rien n'a été écrit.
 */
export async function saveDraft(draft) {
    const id = draft?.id;
    const annotated = draft?.annotated;
    if (id == null || !annotated) return null;

    if (hasInconsistency(annotated)) {
        const go = await confirmAction(/** @type {string} */ (translate('transcription.saveInconsistentWarning')), {
            confirmLabel: /** @type {string} */ (translate('transcription.saveAnyway'))
        });
        if (!go) return null;
    }

    let result;
    try {
        result = await SaveTranscriptionAsMatch(id);
    } catch (error) {
        logger.error('Failed to save a transcription draft as a match:', error);
        statusBarTextStore.set(tMsg('transcription.saveFailed', { error: String(error) }));
        return null;
    }

    transcriptionSaveStore.set({
        id,
        matchId: result?.match_id ?? 0,
        at: Date.now(),
        signature: documentSignature(annotated)
    });
    statusBarTextStore.set(tMsg(result?.replaced ? 'transcription.replacedMatch' : 'transcription.savedMatch', { id: result?.match_id ?? 0 }));

    await startTargetedAnalysis(result);
    return result;
}

/**
 * Le lot gammonNet limité aux positions sans analyse de ce match, à la
 * profondeur de la bibliothèque.
 *
 * @param {any} result
 */
async function startTargetedAnalysis(result) {
    if (!result?.match_id || !result?.to_analyze) return;
    try {
        const [ply, pruneK] = await Promise.all([GetGammonNetAnalysisPly(), GetGammonNetPruneK()]);
        await StartGammonNetMatchBatch(result.match_id, ply, pruneK, 0);
    } catch (error) {
        logger.error('The targeted gammonNet batch of a saved transcription failed to start:', error);
    }
}

/**
 * La reprise de l'analyse (fonctionnel.md §4, ADR-0045 §8) : le match du
 * dernier brouillon enregistré s'il a des positions sans analyse, sinon null.
 * Rien n'est stocké : recompté à chaque ouverture de base, la proposition
 * revient tant qu'il en manque.
 *
 * @type {import('svelte/store').Writable<{transcription_id: number, match_id: number, label: string, to_analyze: number} | null>}
 */
export const transcriptionResumeStore = writable(null);

/**
 * Recompte, à l'ouverture d'une base seulement.
 */
export async function refreshTranscriptionResume() {
    try {
        transcriptionResumeStore.set((await PendingTranscriptionAnalysis()) ?? null);
    } catch (error) {
        logger.error('Failed to look for a transcription analysis to finish:', error);
        transcriptionResumeStore.set(null);
    }
}

/**
 * Termine le lot sur le seul match du brouillon, jamais le rattrapage de toute
 * la bibliothèque.
 */
export async function resumeTranscriptionAnalysis() {
    const pending = get(transcriptionResumeStore);
    if (!pending) return;

    transcriptionResumeStore.set(null);
    await startTargetedAnalysis(pending);
}

/** Écarte la proposition pour cette ouverture-ci. Rien n'est retenu. */
export function dismissTranscriptionResume() {
    transcriptionResumeStore.set(null);
}

/**
 * Exporte le brouillon en `.mat` tel qu'écrit, jamais refusé (fonctionnel.md
 * §5) : un coup illégal sort comme joué, avec l'avertissement que gnubg et XG
 * divergeront.
 *
 * @param {any} draft
 * @returns true si un fichier a été écrit.
 */
export async function exportDraftMat(draft) {
    const id = draft?.id;
    const annotated = draft?.annotated;
    if (id == null || !annotated) return false;

    if (hasIllegalMove(annotated)) {
        const go = await confirmAction(/** @type {string} */ (translate('transcription.illegalExportWarning')), {
            confirmLabel: /** @type {string} */ (translate('transcription.exportAnyway'))
        });
        if (!go) return false;
    }

    try {
        const suggested = await SuggestTranscriptionMatFilename(id);
        const path = await OpenExportMatDialog(suggested);
        if (!path) return false;
        await ExportTranscriptionMAT(id, path);
        statusBarTextStore.set(tMsg('transcription.exported'));
        return true;
    } catch (error) {
        logger.error('Failed to export a transcription draft to .mat:', error);
        statusBarTextStore.set(tMsg('transcription.exportFailed', { error: String(error) }));
        return false;
    }
}

/**
 * Ferme le brouillon : la ligne est supprimée, sans corbeille. Confirmation
 * toujours, la phrase nommant la perte : sans enregistrement, tout le
 * brouillon ; sinon, toute correction future du Match (rien n'édite ses coups).
 *
 * @param {any} draft
 * @returns true si le brouillon a été fermé.
 */
export async function closeDraft(draft) {
    const id = draft?.id;
    if (id == null) return false;

    const matchId = savedMatchID(draft.annotated);
    const message = /** @type {string} */ (matchId ? translate('transcription.closeSavedConfirm', { id: matchId }) : translate('transcription.closeUnsavedConfirm'));
    const go = await confirmAction(message, { confirmLabel: /** @type {string} */ (translate('transcription.closeDraft')) });
    if (!go) return false;

    try {
        await CloseTranscription(id);
    } catch (error) {
        logger.error('Failed to close a transcription draft:', error);
        statusBarTextStore.set(tMsg('transcription.closeFailed', { error: String(error) }));
        return false;
    }
    resetTranscriptionSave();
    statusBarTextStore.set(tMsg('transcription.closed'));
    return true;
}
