/**
 * transcriptionSave.js — enregistrer un brouillon, l'exporter, le fermer (T1.9).
 *
 * Les trois gestes qui font sortir une Transcription d'elle-même : elle devient
 * un Match de la bibliothèque, un fichier `.mat`, ou plus rien du tout. Ils
 * vivent ici et non dans le panneau pour deux raisons — le panneau est un
 * client du moteur et ne décide de rien (ADR-0045 règle 9), et ces trois-là
 * s'écrivent et se testent sans monter un composant.
 *
 * Ce que le service NE fait pas : dériver l'état du match. Les incohérences,
 * le coup illégal, le match id, tout est lu dans le document annoté que le Go
 * a renvoyé. Ce fichier ne fait qu'y poser des questions.
 *
 * L'enregistrement suit fonctionnel.md §4 dans l'ordre : le document est
 * rejoué (il l'est déjà, c'est ce que le panneau tient en main), ses
 * incohérences sont ANNONCÉES et jamais opposées (ADR-0044), le Match est créé
 * puis remplacé aux fois suivantes (ADR-0045 §2), et le lot d'analyse ciblé
 * démarre aussitôt sur les seules positions nouvelles (ADR-0013, ADR-0045 §8).
 * Sa progression et son annulation sont celles de la barre d'état, qui écoute
 * déjà les événements `gammonnet-batch:*` : il n'y a rien à rebrancher ici.
 */

import { writable } from 'svelte/store';

import { translate, tMsg } from '../i18n';
import { logger } from '../utils/logger.js';
import { statusBarTextStore } from '../stores/uiStore.js';
import { confirmAction } from './confirmService.js';
import { SaveTranscriptionAsMatch, SuggestTranscriptionMatFilename, ExportTranscriptionMAT, CloseTranscription } from '../../wailsjs/go/database/Database.js';
import { OpenExportMatDialog, StartGammonNetMatchBatch } from '../../wailsjs/go/gui/App.js';
import { GetGammonNetAnalysisPly, GetGammonNetPruneK } from '../../wailsjs/go/main/Config.js';

/**
 * Ce que le dernier enregistrement de CETTE session a laissé :
 * `{ id, matchId, at, signature }`, ou null.
 *
 * Il est en mémoire, comme la pile d'annulation, et pour la même raison : la
 * seule chose durable est le `match_id` posé sur le document, que le moteur
 * écrit dans la ligne. Ce store ajoute ce que la base ne dit pas — l'heure de
 * l'enregistrement et l'état du document à ce moment-là, dont se déduit
 * « modifié depuis ».
 *
 * @type {import('svelte/store').Writable<{id: number, matchId: number, at: number, signature: string} | null>}
 */
export const transcriptionSaveStore = writable(null);

/** Repart de zéro : à la fermeture d'un brouillon, et au changement de base. */
export function resetTranscriptionSave() {
    transcriptionSaveStore.set(null);
}

/**
 * La signature du document : son en-tête et ses Actions, rien de dérivé. Deux
 * documents de même signature produiraient le même Match, ce qui est
 * exactement la question que pose « modifié depuis l'enregistrement ».
 */
export function documentSignature(annotated) {
    const doc = annotated?.document;
    if (!doc) return '';
    return JSON.stringify([doc.header ?? null, doc.actions ?? []]);
}

/** Les sortes d'Incohérence que porte le document, sans doublon. */
export function inconsistencyKinds(annotated) {
    const kinds = new Set();
    for (const info of annotated?.actions ?? []) {
        for (const flag of info?.inconsistencies ?? []) {
            if (flag?.kind) kinds.add(flag.kind);
        }
    }
    return kinds;
}

export function hasInconsistency(annotated) {
    return inconsistencyKinds(annotated).size > 0;
}

/**
 * Un coup que les règles ne peuvent pas atteindre — celui qui fera dire
 * « Invalid move » à gnubg et à XG, et divergera ensuite. Un jet incohérent
 * avec le coup requalifie le coup en illégal (transcript.InconsistentDice) :
 * les deux valent avertissement à l'export.
 */
export function hasIllegalMove(annotated) {
    const kinds = inconsistencyKinds(annotated);
    return kinds.has('illegal_move') || kinds.has('inconsistent_dice');
}

/** Le match que le brouillon possède déjà, 0 s'il n'a jamais été enregistré. */
export function savedMatchID(annotated) {
    return annotated?.document?.header?.match_id ?? 0;
}

/**
 * Ce que la barre du brouillon dit de son état : jamais enregistré, enregistré
 * il y a tant, ou modifié depuis. Rendue comme une clé i18n et ses paramètres
 * plutôt que comme une phrase, pour que le calcul se teste sans la langue.
 *
 * Le brouillon entier, et pas seulement son document : l'enregistrement de
 * cette session ne vaut que pour LE brouillon qui l'a fait — revenir à la liste
 * et en ouvrir un autre ne doit pas lui prêter l'heure du premier. C'est aussi
 * ce qui rend la barre juste dans la seconde qui suit le premier
 * enregistrement, avant que le geste suivant ne rapporte un document portant
 * enfin son `match_id`.
 */
export function draftSaveState(draft, saved, now = Date.now()) {
    const annotated = draft?.annotated ?? null;
    const own = saved && draft && saved.id === draft.id ? saved : null;
    const matchId = savedMatchID(annotated) || own?.matchId || 0;
    if (!matchId) return { key: 'transcription.stateNeverSaved', params: {} };

    // Un brouillon enregistré lors d'une session précédente porte son match id
    // et rien d'autre : l'heure de l'enregistrement n'a jamais été écrite nulle
    // part (ADR-0045 §8 — aucun état n'est stocké pour cela).
    if (!own || own.matchId !== matchId) {
        return { key: 'transcription.stateSavedAs', params: { id: matchId } };
    }
    if (own.signature !== documentSignature(annotated)) {
        return { key: 'transcription.stateModifiedSince', params: {} };
    }
    const minutes = Math.floor(Math.max(0, now - own.at) / 60000);
    if (minutes < 1) return { key: 'transcription.stateSavedJustNow', params: {} };
    if (minutes < 60) return { key: 'transcription.stateSavedMinutes', params: { n: minutes } };
    return { key: 'transcription.stateSavedHours', params: { n: Math.floor(minutes / 60) } };
}

/**
 * Enregistre le brouillon en Match : création la première fois, remplacement
 * ensuite (même `id`), puis le lot d'analyse ciblé.
 *
 * Les incohérences sont annoncées avant, jamais opposées : la question posée
 * est « enregistrer quand même ? », et son seul refus possible est celui de
 * l'utilisateur.
 *
 * @returns le résultat du moteur, ou null si rien n'a été écrit.
 */
export async function saveDraft(draft) {
    const id = draft?.id;
    const annotated = draft?.annotated;
    if (id == null || !annotated) return null;

    if (hasInconsistency(annotated)) {
        const go = await confirmAction(translate('transcription.saveInconsistentWarning'), {
            confirmLabel: translate('transcription.saveAnyway')
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
 * Le lot gammonNet restreint aux positions de CE match qui n'ont pas
 * d'analyse. La profondeur est celle que la bibliothèque s'est donnée pour ses
 * analyses — la même que le rattrapage d'après import ; le panneau n'en a pas
 * une à lui.
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
 * Exporte le brouillon en `.mat`, tel qu'il est écrit. Un coup illégal sort
 * comme il a été joué, avec l'avertissement que gnubg et XG le signaleront et
 * divergeront ensuite ; l'export n'est jamais refusé (fonctionnel.md §5).
 *
 * @returns true si un fichier a été écrit.
 */
export async function exportDraftMat(draft) {
    const id = draft?.id;
    const annotated = draft?.annotated;
    if (id == null || !annotated) return false;

    if (hasIllegalMove(annotated)) {
        const go = await confirmAction(translate('transcription.illegalExportWarning'), {
            confirmLabel: translate('transcription.exportAnyway')
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
 * Ferme le brouillon : la ligne est supprimée, rien n'est mis à la corbeille.
 *
 * La confirmation est demandée dans les deux cas, et ce n'est pas une
 * prudence de principe : les deux pertes sont réelles et différentes. Un
 * brouillon jamais enregistré emporte tout ce qui y est écrit ; un brouillon
 * enregistré laisse son Match, définitif — plus rien ne pourra en corriger un
 * coup, puisque rien dans blunderDB n'édite les coups d'un Match. La phrase
 * dit laquelle des deux s'applique.
 *
 * @returns true si le brouillon a été fermé.
 */
export async function closeDraft(draft) {
    const id = draft?.id;
    if (id == null) return false;

    const matchId = savedMatchID(draft.annotated);
    const message = matchId ? translate('transcription.closeSavedConfirm', { id: matchId }) : translate('transcription.closeUnsavedConfirm');
    const go = await confirmAction(message, { confirmLabel: translate('transcription.closeDraft') });
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
