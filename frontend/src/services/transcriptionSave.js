/**
 * transcriptionSave.js — les sorties d'un brouillon (Terminer, Abandonner),
 * son export `.mat`, et l'entrée depuis un Match existant (Éditer la
 * transcription). Hors du panneau, client du moteur qui ne décide de rien
 * (ADR-0045 règle 9). Tout l'état (incohérences, coup illégal, match id) est
 * lu dans le document annoté renvoyé par le Go.
 *
 * Terminer suit ADR-0045 §2 : incohérences ANNONCÉES, jamais opposées
 * (ADR-0044) ; Match créé, ou remplacé au même `id` pour un brouillon ouvert
 * depuis un Match ; brouillon libéré ; lot d'analyse ciblé sur les positions
 * nouvelles (ADR-0013, ADR-0045 §8), suivi par la barre d'état via
 * `gammonnet-batch:*`.
 */

import { get, writable } from 'svelte/store';

import { translate, tMsg } from '../i18n';
import { logger } from '../utils/logger.js';
import { statusBarTextStore, activeTabStore } from '../stores/uiStore.js';
import { setTranscription, resetTranscriptionKeys } from '../stores/transcriptionStore.js';
import { confirmAction } from './confirmService.js';
import {
    FinishTranscription,
    AbandonTranscription,
    OpenTranscription,
    EditMatchTranscription,
    MatchTranscriptionLosses,
    SuggestTranscriptionMatFilename,
    ExportTranscriptionMAT,
    PendingTranscriptionAnalysis
} from '../../wailsjs/go/database/Database.js';
import { OpenExportMatDialog, StartGammonNetMatchBatch } from '../../wailsjs/go/gui/App.js';
import { GetGammonNetAnalysisPly, GetGammonNetPruneK } from '../../wailsjs/go/main/Config.js';

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
 * Le Match dont le brouillon a été ouvert, 0 pour un brouillon qui n'en a
 * pas encore produit.
 *
 * @param {any} annotated
 * @returns {number}
 */
export function savedMatchID(annotated) {
    return annotated?.document?.header?.match_id ?? 0;
}

/**
 * Ce que la barre du brouillon dit de sa SORTIE, jamais du salut du brouillon
 * (ADR-0048 décision 12), le brouillon étant écrit après chaque geste : un
 * brouillon ouvert depuis un Match le remplacera, les autres en créeront un.
 * Rendu en clé i18n et paramètres, testable sans la langue.
 *
 * @param {{annotated: any} | null | undefined} draft
 */
export function draftState(draft) {
    const matchId = savedMatchID(draft?.annotated);
    if (!matchId) return { key: 'transcription.stateNoMatch', params: {} };
    return { key: 'transcription.stateEditsMatch', params: { id: matchId } };
}

/**
 * Terminer : écrit le Match (création, ou remplacement de celui dont le
 * brouillon a été ouvert), libère le brouillon, puis lance le lot d'analyse
 * ciblé. Les incohérences sont annoncées ; seul l'utilisateur peut refuser.
 *
 * @param {any} draft
 * @returns le résultat du moteur, ou null si rien n'a été écrit.
 */
export async function finishDraft(draft) {
    const id = draft?.id;
    const annotated = draft?.annotated;
    if (id == null || !annotated) return null;

    if (hasInconsistency(annotated)) {
        const go = await confirmAction(/** @type {string} */ (translate('transcription.finishInconsistentWarning')), {
            confirmLabel: /** @type {string} */ (translate('transcription.finishAnyway'))
        });
        if (!go) return null;
    }

    let result;
    try {
        result = await FinishTranscription(id);
    } catch (error) {
        logger.error('Failed to finish a transcription draft:', error);
        statusBarTextStore.set(tMsg('transcription.finishFailed', { error: String(error) }));
        return null;
    }

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
 * La reprise de l'analyse (fonctionnel.md §4, ADR-0045 §8) : le dernier match
 * transcrit s'il a des positions sans analyse, sinon null.
 * Rien n'est stocké : recompté à chaque ouverture de base, la proposition
 * revient tant qu'il en manque.
 *
 * @type {import('svelte/store').Writable<{match_id: number, label: string, to_analyze: number} | null>}
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
 * Abandonner : la ligne est supprimée, sans corbeille, et sans Match. La
 * confirmation ne vaut que pour un brouillon qui n'a pas de Match : il
 * emporte tout ce qui a été tapé. Celui ouvert depuis un Match laisse ce
 * Match tel quel, et ne perd que les corrections non terminées.
 *
 * @param {any} draft
 * @returns true si le brouillon a été abandonné.
 */
export async function abandonDraft(draft) {
    const id = draft?.id;
    if (id == null) return false;

    if (!(await currentMatchID(draft))) {
        const go = await confirmAction(/** @type {string} */ (translate('transcription.abandonConfirm')), {
            confirmLabel: /** @type {string} */ (translate('transcription.abandon'))
        });
        if (!go) return false;
    }

    try {
        await AbandonTranscription(id);
    } catch (error) {
        logger.error('Failed to abandon a transcription draft:', error);
        statusBarTextStore.set(tMsg('transcription.abandonFailed', { error: String(error) }));
        return false;
    }
    statusBarTextStore.set(tMsg('transcription.abandoned'));
    return true;
}

/**
 * Le Match d'origine tel que le moteur le voit maintenant : le document
 * affiché peut encore nommer un Match supprimé depuis, que la lecture Go
 * oublie ; le brouillon redevient alors sans match, et tout ce qui y est
 * écrit se perd à l'abandon.
 *
 * @param {any} draft
 * @returns {Promise<number>}
 */
async function currentMatchID(draft) {
    if (!savedMatchID(draft.annotated)) return 0;
    try {
        return savedMatchID((await OpenTranscription(draft.id))?.annotated);
    } catch (error) {
        logger.error('Failed to reread a transcription draft before abandoning it:', error);
        return 0;
    }
}

/**
 * Éditer la transcription d'un Match : ouvre le brouillon déjà ouvert sur lui,
 * sinon un brouillon neuf rejoué depuis son `.mat`, puis amène l'onglet
 * Transcription. Un match importé porte ce qu'un `.mat` ne porte pas : le
 * dialogue chiffre ces pertes avant d'ouvrir (ADR-0045 §2).
 *
 * @param {number} matchId
 * @returns l'état du brouillon ouvert, ou null.
 */
export async function editMatchTranscription(matchId) {
    if (!matchId) return null;
    try {
        const losses = await MatchTranscriptionLosses(matchId);
        const lossy = losses && !losses.draft_id && losses.imported && losses.analyses + losses.comments > 0;
        if (lossy) {
            const go = await confirmAction(/** @type {string} */ (translate('transcription.editLossWarning', { analyses: losses.analyses, comments: losses.comments })), {
                confirmLabel: /** @type {string} */ (translate('transcription.editAnyway'))
            });
            if (!go) return null;
        }
        const state = await EditMatchTranscription(matchId);
        setTranscription(state);
        resetTranscriptionKeys();
        activeTabStore.set('transcription');
        return state;
    } catch (error) {
        logger.error('Failed to open a transcription draft on a match:', error);
        statusBarTextStore.set(tMsg('transcription.editFailed', { error: String(error) }));
        return null;
    }
}
