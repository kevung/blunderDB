/**
 * transcriptionSave.test.js — T1.9 : enregistrer, exporter, fermer.
 *
 * Trois avertissements et une confirmation, qui sont tout ce que
 * l'utilisateur voit de fonctionnel.md §4 et §5 : les incohérences sont
 * ANNONCÉES avant l'enregistrement et jamais opposées (ADR-0044), le coup
 * illégal est annoncé avant l'export parce que gnubg et XG s'en plaindront,
 * et la fermeture demande confirmation en disant ce qui se perd.
 *
 * Ce qui est vérifié en plus : le lot d'analyse ciblé part sur le match qui
 * vient d'être écrit, et seulement s'il reste quelque chose à analyser.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// vi.hoisted, parce que les fabriques de vi.mock remontent en tête de fichier :
// un `const` ordinaire ne serait pas encore initialisé quand elles s'exécutent.
const {
    SaveTranscriptionAsMatch,
    SuggestTranscriptionMatFilename,
    ExportTranscriptionMAT,
    CloseTranscription,
    PendingTranscriptionAnalysis,
    OpenExportMatDialog,
    StartGammonNetMatchBatch,
    confirmAction
} = vi.hoisted(() => ({
    SaveTranscriptionAsMatch: vi.fn(),
    PendingTranscriptionAnalysis: vi.fn(),
    SuggestTranscriptionMatFilename: vi.fn(),
    ExportTranscriptionMAT: vi.fn(),
    CloseTranscription: vi.fn(),
    OpenExportMatDialog: vi.fn(),
    StartGammonNetMatchBatch: vi.fn(),
    confirmAction: vi.fn()
}));

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    SaveTranscriptionAsMatch,
    SuggestTranscriptionMatFilename,
    ExportTranscriptionMAT,
    CloseTranscription,
    PendingTranscriptionAnalysis
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    OpenExportMatDialog,
    StartGammonNetMatchBatch
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetAnalysisPly: vi.fn().mockResolvedValue(2),
    GetGammonNetPruneK: vi.fn().mockResolvedValue(12)
}));
vi.mock('../services/confirmService.js', () => ({ confirmAction }));

import {
    saveDraft,
    exportDraftMat,
    closeDraft,
    draftSaveState,
    documentSignature,
    hasIllegalMove,
    hasInconsistency,
    transcriptionSaveStore,
    resetTranscriptionSave,
    transcriptionResumeStore,
    refreshTranscriptionResume,
    resumeTranscriptionAnalysis,
    dismissTranscriptionResume
} from '../services/transcriptionSave.js';

/** Un brouillon tel que le panneau le tient : l'id de la ligne et le document annoté. */
function draft({ id = 1, matchId = 0, actions = [], header = {} } = {}) {
    return {
        id,
        annotated: {
            document: {
                header: { match_length: 7, match_id: matchId || undefined, ...header },
                actions: actions.map((a) => ({ kind: a.kind ?? 'checker' }))
            },
            actions
        }
    };
}

const clean = [{ kind: 'checker', inconsistencies: [] }];
const inconsistent = [{ kind: 'checker', inconsistencies: [{ kind: 'double_turn' }] }];
const illegal = [{ kind: 'checker', inconsistencies: [{ kind: 'illegal_move' }] }];

beforeEach(() => {
    vi.clearAllMocks();
    resetTranscriptionSave();
    SaveTranscriptionAsMatch.mockResolvedValue({ match_id: 7, replaced: false, to_analyze: 3 });
    SuggestTranscriptionMatFilename.mockResolvedValue('A_B_2026-09-07_7p.mat');
    OpenExportMatDialog.mockResolvedValue('/tmp/A_B.mat');
    ExportTranscriptionMAT.mockResolvedValue(undefined);
    CloseTranscription.mockResolvedValue(undefined);
    PendingTranscriptionAnalysis.mockResolvedValue(null);
    dismissTranscriptionResume();
    confirmAction.mockResolvedValue(true);
});

describe("l'enregistrement", () => {
    test('un brouillon sans incohérence est écrit sans rien demander', async () => {
        const result = await saveDraft(draft({ actions: clean }));
        expect(confirmAction).not.toHaveBeenCalled();
        expect(SaveTranscriptionAsMatch).toHaveBeenCalledWith(1);
        expect(result.match_id).toBe(7);
    });

    test('une incohérence est annoncée, et enregistrer quand même écrit', async () => {
        await saveDraft(draft({ actions: inconsistent }));
        expect(confirmAction).toHaveBeenCalledTimes(1);
        expect(SaveTranscriptionAsMatch).toHaveBeenCalledTimes(1);
    });

    test("le refus de l'avertissement n'écrit rien", async () => {
        confirmAction.mockResolvedValue(false);
        const result = await saveDraft(draft({ actions: inconsistent }));
        expect(result).toBeNull();
        expect(SaveTranscriptionAsMatch).not.toHaveBeenCalled();
    });

    test("le lot d'analyse ciblé part sur le match écrit", async () => {
        await saveDraft(draft({ actions: clean }));
        expect(StartGammonNetMatchBatch).toHaveBeenCalledWith(7, 2, 12, 0);
    });

    test('rien à analyser, pas de lot', async () => {
        SaveTranscriptionAsMatch.mockResolvedValue({ match_id: 7, replaced: true, to_analyze: 0 });
        await saveDraft(draft({ actions: clean }));
        expect(StartGammonNetMatchBatch).not.toHaveBeenCalled();
    });

    test("l'échec de l'écriture ne laisse pas d'état d'enregistrement", async () => {
        SaveTranscriptionAsMatch.mockRejectedValue(new Error('disk full'));
        expect(await saveDraft(draft({ actions: clean }))).toBeNull();
        expect(get(transcriptionSaveStore)).toBeNull();
    });
});

describe("l'export .mat", () => {
    test('un document propre part directement au dialogue', async () => {
        expect(await exportDraftMat(draft({ actions: clean }))).toBe(true);
        expect(confirmAction).not.toHaveBeenCalled();
        expect(OpenExportMatDialog).toHaveBeenCalledWith('A_B_2026-09-07_7p.mat');
        expect(ExportTranscriptionMAT).toHaveBeenCalledWith(1, '/tmp/A_B.mat');
    });

    test('un coup illégal est annoncé avant le dialogue', async () => {
        await exportDraftMat(draft({ actions: illegal }));
        expect(confirmAction).toHaveBeenCalledTimes(1);
        expect(ExportTranscriptionMAT).toHaveBeenCalledTimes(1);
    });

    test('un jet incohérent avec le coup vaut aussi avertissement', () => {
        expect(hasIllegalMove({ actions: [{ inconsistencies: [{ kind: 'inconsistent_dice' }] }] })).toBe(true);
        expect(hasIllegalMove({ actions: illegal })).toBe(true);
        expect(hasIllegalMove({ actions: inconsistent })).toBe(false);
        expect(hasInconsistency({ actions: inconsistent })).toBe(true);
    });

    test("le dialogue annulé n'écrit aucun fichier", async () => {
        OpenExportMatDialog.mockResolvedValue('');
        expect(await exportDraftMat(draft({ actions: clean }))).toBe(false);
        expect(ExportTranscriptionMAT).not.toHaveBeenCalled();
    });
});

describe('la fermeture', () => {
    test('un brouillon jamais enregistré est confirmé, et sa ligne supprimée', async () => {
        expect(await closeDraft(draft({ actions: clean }))).toBe(true);
        expect(confirmAction).toHaveBeenCalledTimes(1);
        expect(confirmAction.mock.calls[0][0]).toMatch(/never saved/i);
        expect(CloseTranscription).toHaveBeenCalledWith(1);
    });

    test('un brouillon déjà enregistré dit que son match reste', async () => {
        await closeDraft(draft({ matchId: 7, actions: clean }));
        expect(confirmAction.mock.calls[0][0]).toMatch(/match #7/i);
    });

    test('le refus ne supprime rien', async () => {
        confirmAction.mockResolvedValue(false);
        expect(await closeDraft(draft({ actions: clean }))).toBe(false);
        expect(CloseTranscription).not.toHaveBeenCalled();
    });
});

describe("l'état du brouillon", () => {
    test('jamais enregistré tant que le document ne porte pas de match', () => {
        expect(draftSaveState(draft({ actions: clean }), null).key).toBe('transcription.stateNeverSaved');
    });

    test('enregistré il y a tant, puis modifié depuis', () => {
        const d = draft({ matchId: 7, actions: clean });
        const saved = { id: 1, matchId: 7, at: 0, signature: documentSignature(d.annotated) };

        expect(draftSaveState(d, saved, 30_000).key).toBe('transcription.stateSavedJustNow');
        expect(draftSaveState(d, saved, 5 * 60_000)).toEqual({
            key: 'transcription.stateSavedMinutes',
            params: { n: 5 }
        });
        expect(draftSaveState(d, saved, 3 * 3_600_000)).toEqual({
            key: 'transcription.stateSavedHours',
            params: { n: 3 }
        });

        const corrected = draft({ matchId: 7, actions: [...clean, { kind: 'checker', inconsistencies: [] }] });
        expect(draftSaveState(corrected, saved, 30_000).key).toBe('transcription.stateModifiedSince');
    });

    test("un brouillon enregistré lors d'une session précédente le dit sans mentir sur l'heure", () => {
        const d = draft({ matchId: 7, actions: clean });
        expect(draftSaveState(d, null)).toEqual({ key: 'transcription.stateSavedAs', params: { id: 7 } });
    });
});

describe("la reprise de l'analyse", () => {
    // T3.3 : rien n'est stocké (ADR-0045 §8), donc tout se joue sur le comptage
    // refait à l'ouverture et sur le lot CIBLÉ qu'il relance.
    test('un match transcrit à trous propose de terminer', async () => {
        PendingTranscriptionAnalysis.mockResolvedValue({ transcription_id: 1, match_id: 7, label: 'A vs B', to_analyze: 12 });
        await refreshTranscriptionResume();
        expect(get(transcriptionResumeStore)).toEqual({ transcription_id: 1, match_id: 7, label: 'A vs B', to_analyze: 12 });
    });

    test('un match complet ne propose rien', async () => {
        await refreshTranscriptionResume();
        expect(get(transcriptionResumeStore)).toBeNull();
    });

    test('accepter relance le lot du seul match, jamais celui de la bibliothèque', async () => {
        PendingTranscriptionAnalysis.mockResolvedValue({ transcription_id: 1, match_id: 7, label: '', to_analyze: 12 });
        await refreshTranscriptionResume();
        await resumeTranscriptionAnalysis();
        expect(StartGammonNetMatchBatch).toHaveBeenCalledWith(7, 2, 12, 0);
        expect(get(transcriptionResumeStore)).toBeNull();
    });

    test("écarter la proposition n'écrit rien : la question reposée la ramène", async () => {
        PendingTranscriptionAnalysis.mockResolvedValue({ transcription_id: 1, match_id: 7, label: '', to_analyze: 12 });
        await refreshTranscriptionResume();
        dismissTranscriptionResume();
        expect(get(transcriptionResumeStore)).toBeNull();
        expect(StartGammonNetMatchBatch).not.toHaveBeenCalled();

        await refreshTranscriptionResume();
        expect(get(transcriptionResumeStore)?.to_analyze).toBe(12);
    });
});
