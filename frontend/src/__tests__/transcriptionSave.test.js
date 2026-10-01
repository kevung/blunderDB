/**
 * transcriptionSave.test.js — terminer, exporter, abandonner, éditer.
 *
 * Quatre avertissements et une confirmation, qui sont tout ce que
 * l'utilisateur voit de fonctionnel.md §4 et §5 : les incohérences sont
 * ANNONCÉES avant Terminer et jamais opposées (ADR-0044), le coup illégal
 * est annoncé avant l'export parce que gnubg et XG s'en plaindront,
 * l'abandon d'un brouillon sans match demande confirmation, et l'édition
 * d'un match importé chiffre ce qu'elle peut perdre (ADR-0045 §2).
 *
 * Ce qui est vérifié en plus : le lot d'analyse ciblé part sur le match qui
 * vient d'être écrit, et seulement s'il reste quelque chose à analyser.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// vi.hoisted, parce que les fabriques de vi.mock remontent en tête de fichier :
// un `const` ordinaire ne serait pas encore initialisé quand elles s'exécutent.
const {
    FinishTranscription,
    AbandonTranscription,
    OpenTranscription,
    EditMatchTranscription,
    MatchTranscriptionLosses,
    SuggestTranscriptionMatFilename,
    ExportTranscriptionMAT,
    PendingTranscriptionAnalysis,
    OpenExportMatDialog,
    StartGammonNetMatchBatch,
    confirmAction
} = vi.hoisted(() => ({
    FinishTranscription: vi.fn(),
    AbandonTranscription: vi.fn(),
    OpenTranscription: vi.fn(),
    EditMatchTranscription: vi.fn(),
    MatchTranscriptionLosses: vi.fn(),
    PendingTranscriptionAnalysis: vi.fn(),
    SuggestTranscriptionMatFilename: vi.fn(),
    ExportTranscriptionMAT: vi.fn(),
    OpenExportMatDialog: vi.fn(),
    StartGammonNetMatchBatch: vi.fn(),
    confirmAction: vi.fn()
}));

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    FinishTranscription,
    AbandonTranscription,
    OpenTranscription,
    EditMatchTranscription,
    MatchTranscriptionLosses,
    SuggestTranscriptionMatFilename,
    ExportTranscriptionMAT,
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

import { transcriptionStore } from '../stores/transcriptionStore.js';
import { activeTabStore } from '../stores/uiStore.js';
import {
    finishDraft,
    exportDraftMat,
    abandonDraft,
    editMatchTranscription,
    draftState,
    hasIllegalMove,
    hasInconsistency,
    transcriptionResumeStore,
    refreshTranscriptionResume,
    resumeTranscriptionAnalysis,
    dismissTranscriptionResume
} from '../services/transcriptionSave.js';

/**
 * Un brouillon tel que le panneau le tient : l'id de la ligne et le document annoté.
 *
 * @param {{id?: number, matchId?: number, actions?: any[], header?: object}} [options]
 */
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
    transcriptionStore.set(null);
    FinishTranscription.mockResolvedValue({ match_id: 7, replaced: false, to_analyze: 3 });
    MatchTranscriptionLosses.mockResolvedValue({ match_id: 7, imported: false, analyses: 0, comments: 0, draft_id: 0 });
    EditMatchTranscription.mockResolvedValue({ id: 4, annotated: { document: { header: { match_id: 7 }, actions: [] } } });
    SuggestTranscriptionMatFilename.mockResolvedValue('A_B_2026-09-07_7p.mat');
    OpenExportMatDialog.mockResolvedValue('/tmp/A_B.mat');
    ExportTranscriptionMAT.mockResolvedValue(undefined);
    AbandonTranscription.mockResolvedValue(undefined);
    OpenTranscription.mockImplementation(async (id) => ({ id, annotated: { document: { header: { match_id: 7 }, actions: [] } } }));
    PendingTranscriptionAnalysis.mockResolvedValue(null);
    dismissTranscriptionResume();
    confirmAction.mockResolvedValue(true);
});

describe('Terminer', () => {
    test('un brouillon sans incohérence est écrit sans rien demander', async () => {
        const result = /** @type {any} */ (await finishDraft(draft({ actions: clean })));
        expect(confirmAction).not.toHaveBeenCalled();
        expect(FinishTranscription).toHaveBeenCalledWith(1);
        expect(result.match_id).toBe(7);
    });

    test('une incohérence est annoncée, et terminer quand même écrit', async () => {
        await finishDraft(draft({ actions: inconsistent }));
        expect(confirmAction).toHaveBeenCalledTimes(1);
        expect(FinishTranscription).toHaveBeenCalledTimes(1);
    });

    test("le refus de l'avertissement n'écrit rien", async () => {
        confirmAction.mockResolvedValue(false);
        const result = await finishDraft(draft({ actions: inconsistent }));
        expect(result).toBeNull();
        expect(FinishTranscription).not.toHaveBeenCalled();
    });

    test("le lot d'analyse ciblé part sur le match écrit", async () => {
        await finishDraft(draft({ actions: clean }));
        expect(StartGammonNetMatchBatch).toHaveBeenCalledWith(7, 2, 12, 0);
    });

    test('rien à analyser, pas de lot', async () => {
        FinishTranscription.mockResolvedValue({ match_id: 7, replaced: true, to_analyze: 0 });
        await finishDraft(draft({ actions: clean }));
        expect(StartGammonNetMatchBatch).not.toHaveBeenCalled();
    });

    test("l'échec de l'écriture ne rend rien et ne lance aucun lot", async () => {
        FinishTranscription.mockRejectedValue(new Error('disk full'));
        expect(await finishDraft(draft({ actions: clean }))).toBeNull();
        expect(StartGammonNetMatchBatch).not.toHaveBeenCalled();
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

describe('Abandonner', () => {
    test('un brouillon jamais terminé est confirmé, et sa ligne supprimée', async () => {
        expect(await abandonDraft(draft({ actions: clean }))).toBe(true);
        expect(confirmAction).toHaveBeenCalledTimes(1);
        expect(confirmAction.mock.calls[0][0]).toMatch(/never been finished/i);
        expect(AbandonTranscription).toHaveBeenCalledWith(1);
    });

    test('un brouillon ouvert depuis un match est abandonné sans question : le match reste', async () => {
        expect(await abandonDraft(draft({ matchId: 7, actions: clean }))).toBe(true);
        expect(confirmAction).not.toHaveBeenCalled();
        expect(AbandonTranscription).toHaveBeenCalledWith(1);
    });

    test('un brouillon dont le match d’origine a été supprimé redevient sans match : confirmé', async () => {
        OpenTranscription.mockResolvedValue({ id: 1, annotated: { document: { header: {}, actions: [] } } });
        expect(await abandonDraft(draft({ matchId: 7, actions: clean }))).toBe(true);
        expect(confirmAction).toHaveBeenCalledTimes(1);
        expect(confirmAction.mock.calls[0][0]).toMatch(/never been finished/i);
    });

    test('le refus ne supprime rien', async () => {
        confirmAction.mockResolvedValue(false);
        expect(await abandonDraft(draft({ actions: clean }))).toBe(false);
        expect(AbandonTranscription).not.toHaveBeenCalled();
    });
});

describe('Éditer la transcription', () => {
    test("un match transcrit s'ouvre sans avertissement, dans l'onglet Transcription", async () => {
        activeTabStore.set('matches');
        const state = await editMatchTranscription(7);
        expect(confirmAction).not.toHaveBeenCalled();
        expect(EditMatchTranscription).toHaveBeenCalledWith(7);
        expect(state?.id).toBe(4);
        expect(get(transcriptionStore)?.id).toBe(4);
        expect(get(activeTabStore)).toBe('transcription');
    });

    test('un match importé chiffre ses pertes avant d’ouvrir', async () => {
        MatchTranscriptionLosses.mockResolvedValue({ match_id: 7, imported: true, analyses: 12, comments: 3, draft_id: 0 });
        await editMatchTranscription(7);
        expect(confirmAction).toHaveBeenCalledTimes(1);
        expect(confirmAction.mock.calls[0][0]).toMatch(/12 analysis/);
        expect(confirmAction.mock.calls[0][0]).toMatch(/3 comment/);
        expect(EditMatchTranscription).toHaveBeenCalledWith(7);
    });

    test("le refus de l'avertissement n'ouvre rien", async () => {
        MatchTranscriptionLosses.mockResolvedValue({ match_id: 7, imported: true, analyses: 12, comments: 0, draft_id: 0 });
        confirmAction.mockResolvedValue(false);
        expect(await editMatchTranscription(7)).toBeNull();
        expect(EditMatchTranscription).not.toHaveBeenCalled();
        expect(get(transcriptionStore)).toBeNull();
    });

    test('le brouillon déjà ouvert sur le match se rouvre sans avertissement', async () => {
        MatchTranscriptionLosses.mockResolvedValue({ match_id: 7, imported: true, analyses: 12, comments: 3, draft_id: 4 });
        await editMatchTranscription(7);
        expect(confirmAction).not.toHaveBeenCalled();
        expect(EditMatchTranscription).toHaveBeenCalledWith(7);
    });
});

describe('ce que Terminer fera', () => {
    test('un brouillon sans match en créera un', () => {
        expect(draftState(draft({ actions: clean })).key).toBe('transcription.stateNoMatch');
    });

    test('un brouillon ouvert depuis un match le remplacera', () => {
        expect(draftState(draft({ matchId: 7, actions: clean }))).toEqual({ key: 'transcription.stateEditsMatch', params: { id: 7 } });
    });
});

describe("la reprise de l'analyse", () => {
    // T3.3 : rien n'est stocké (ADR-0045 §8), donc tout se joue sur le comptage
    // refait à l'ouverture et sur le lot CIBLÉ qu'il relance.
    test('un match transcrit à trous propose de terminer', async () => {
        PendingTranscriptionAnalysis.mockResolvedValue({ match_id: 7, label: 'A vs B', to_analyze: 12 });
        await refreshTranscriptionResume();
        expect(get(transcriptionResumeStore)).toEqual({ match_id: 7, label: 'A vs B', to_analyze: 12 });
    });

    test('un match complet ne propose rien', async () => {
        await refreshTranscriptionResume();
        expect(get(transcriptionResumeStore)).toBeNull();
    });

    test('accepter relance le lot du seul match, jamais celui de la bibliothèque', async () => {
        PendingTranscriptionAnalysis.mockResolvedValue({ match_id: 7, label: '', to_analyze: 12 });
        await refreshTranscriptionResume();
        await resumeTranscriptionAnalysis();
        expect(StartGammonNetMatchBatch).toHaveBeenCalledWith(7, 2, 12, 0);
        expect(get(transcriptionResumeStore)).toBeNull();
    });

    test("écarter la proposition n'écrit rien : la question reposée la ramène", async () => {
        PendingTranscriptionAnalysis.mockResolvedValue({ match_id: 7, label: '', to_analyze: 12 });
        await refreshTranscriptionResume();
        dismissTranscriptionResume();
        expect(get(transcriptionResumeStore)).toBeNull();
        expect(StartGammonNetMatchBatch).not.toHaveBeenCalled();

        await refreshTranscriptionResume();
        expect(get(transcriptionResumeStore)?.to_analyze).toBe(12);
    });
});
