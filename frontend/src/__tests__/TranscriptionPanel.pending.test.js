/**
 * TranscriptionPanel.pending.test.js — la file des gestes et ce qui la suit.
 *
 * Enregistrer ou exporter part du brouillon tel que les gestes déjà tapés
 * l'ont fait, jamais d'un brouillon en retard d'une frappe. Et la réponse d'un
 * geste ne vaut que pour le brouillon qui l'a envoyé : si l'utilisateur est
 * passé à un autre (ou a changé de base), elle est ignorée.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListTranscriptions: vi.fn().mockResolvedValue([]),
    CreateTranscription: vi.fn(),
    OpenTranscription: vi.fn(),
    ApplyTranscriptionGesture: vi.fn()
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    LegalMoves: vi.fn(),
    EvaluatePositionImmediate: vi.fn()
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetPruneK: vi.fn().mockResolvedValue(0)
}));
vi.mock('../services/transcriptionSave.js', async (importOriginal) => ({
    .../** @type {any} */ (await importOriginal()),
    finishDraft: vi.fn().mockResolvedValue(null),
    exportDraftMat: vi.fn().mockResolvedValue(false)
}));

import { ApplyTranscriptionGesture, ListTranscriptions } from '../../wailsjs/go/database/Database.js';
import { LegalMoves, EvaluatePositionImmediate } from '../../wailsjs/go/gui/App.js';
import { finishDraft, exportDraftMat } from '../services/transcriptionSave.js';

import TranscriptionPanel from '../components/TranscriptionPanel.svelte';
import { transcriptionListStore, transcriptionStore, clearTranscription, bumpTranscriptionLibrary } from '../stores/transcriptionStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';

const POSITION = {
    board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [0, 0] },
    cube: { owner: -1, value: 0 },
    dice: [0, 0],
    score: [7, 7],
    player_on_roll: 0,
    decision_type: 0
};

function annotated(actions = 0) {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice' }, actions: Array.from({ length: actions }, () => ({ kind: 'resign' })), cursor: actions },
        actions: [],
        games: [],
        next: { expects: 'checker', side: 0, position: POSITION, crawford: false, game_number: 1 },
        score: [0, 0],
        finished: false,
        winner: -1,
        cursor: actions
    };
}

/** Une réponse du moteur que le test rend quand il le décide. */
function deferred() {
    /** @type {(value: any) => void} */
    let resolve = () => {};
    const promise = new Promise((r) => (resolve = r));
    return { promise, resolve };
}

function press(/** @type {string} */ code) {
    const digit = /^Digit([1-9])$/.exec(code);
    const letter = /^Key([A-Z])$/.exec(code);
    return fireEvent.keyDown(document, { code, key: digit ? digit[1] : letter ? letter[1].toLowerCase() : code });
}

async function openedPanel() {
    transcriptionStore.set({ id: 1, annotated: annotated() });
    render(TranscriptionPanel);
    await tick();
    document.getElementById('transcriptionPanel')?.focus();
}

const buttonNamed = (/** @type {string} */ label) => [...document.querySelectorAll('button')].find((b) => b.textContent.trim() === label);

beforeEach(() => {
    vi.clearAllMocks();
    /** @type {any} */ (ListTranscriptions).mockResolvedValue([]);
    /** @type {any} */ (LegalMoves).mockResolvedValue([]);
    /** @type {any} */ (EvaluatePositionImmediate).mockResolvedValue({ moves: [] });
    transcriptionListStore.set([]);
    clearTranscription();
    selectedMoveStore.set(null);
    databasePathStore.set('/tmp/library.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
});

afterEach(cleanup);

describe('la file des gestes', () => {
    test("Terminer attend le geste en vol et part du brouillon qu'il a fait", async () => {
        const reply = deferred();
        /** @type {any} */ (ApplyTranscriptionGesture).mockReturnValue(reply.promise);
        await openedPanel();
        await press('KeyR');
        await press('Digit2');
        await vi.waitFor(() => expect(ApplyTranscriptionGesture).toHaveBeenCalledTimes(1));

        await fireEvent.click(/** @type {HTMLElement} */ (document.querySelector('.primary-btn')));
        await tick();
        expect(finishDraft).not.toHaveBeenCalled();

        reply.resolve({ id: 1, annotated: annotated(1) });
        await vi.waitFor(() => expect(finishDraft).toHaveBeenCalledTimes(1));
        expect(/** @type {any} */ (finishDraft).mock.calls[0][0].annotated.document.actions).toHaveLength(1);
    });

    test("une frappe pendant Terminer n'est pas envoyée au brouillon qu'il libère", async () => {
        const finishing = deferred();
        /** @type {any} */ (finishDraft).mockReturnValueOnce(finishing.promise);
        await openedPanel();

        await fireEvent.click(/** @type {HTMLElement} */ (document.querySelector('.primary-btn')));
        await vi.waitFor(() => expect(finishDraft).toHaveBeenCalledTimes(1));
        await press('Digit3');
        await tick();
        expect(ApplyTranscriptionGesture).not.toHaveBeenCalled();

        finishing.resolve(null);
    });

    test("l'export attend le geste en vol", async () => {
        const reply = deferred();
        /** @type {any} */ (ApplyTranscriptionGesture).mockReturnValue(reply.promise);
        await openedPanel();
        await press('KeyR');
        await press('Digit2');
        await vi.waitFor(() => expect(ApplyTranscriptionGesture).toHaveBeenCalledTimes(1));

        await fireEvent.click(/** @type {HTMLElement} */ (buttonNamed('Export .mat')));
        await tick();
        expect(exportDraftMat).not.toHaveBeenCalled();

        reply.resolve({ id: 1, annotated: annotated(1) });
        await vi.waitFor(() => expect(exportDraftMat).toHaveBeenCalledTimes(1));
        expect(/** @type {any} */ (exportDraftMat).mock.calls[0][0].annotated.document.actions).toHaveLength(1);
    });

    test("la réponse d'un brouillon quitté entre-temps est ignorée", async () => {
        const reply = deferred();
        /** @type {any} */ (ApplyTranscriptionGesture).mockReturnValue(reply.promise);
        await openedPanel();
        await press('KeyR');
        await press('Digit2');
        await vi.waitFor(() => expect(ApplyTranscriptionGesture).toHaveBeenCalledTimes(1));

        transcriptionStore.set({ id: 2, annotated: annotated() });
        reply.resolve({ id: 1, annotated: annotated(1) });
        await reply.promise;
        await tick();
        await tick();
        expect(get(transcriptionStore)?.id).toBe(2);
    });

    test("la réponse en vol ne sert pas le brouillon de même id d'une autre base", async () => {
        const reply = deferred();
        /** @type {any} */ (ApplyTranscriptionGesture).mockReturnValue(reply.promise);
        await openedPanel();
        await press('KeyR');
        await press('Digit2');
        await vi.waitFor(() => expect(ApplyTranscriptionGesture).toHaveBeenCalledTimes(1));

        // Changement de base, puis ouverture du brouillon n° 1 de la nouvelle.
        bumpTranscriptionLibrary();
        const other = { id: 1, annotated: annotated() };
        transcriptionStore.set(other);
        reply.resolve({ id: 1, annotated: annotated(1) });
        await reply.promise;
        await tick();
        await tick();
        expect(get(transcriptionStore)).toBe(other);
    });
});
