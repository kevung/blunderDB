/**
 * TranscriptionPanel.correction.test.js — T1.7 : la correction dans le panneau.
 *
 * Les touches et leurs budgets sont tenus à part
 * (transcriptionKeys.correction.test.js), et `Ctrl+Z` par le répartiteur
 * (keyboardService.transcriptionUndo.test.js). Ce qui est vérifié ici est ce
 * que seul le panneau fait : traduire chaque geste de correction en appel Go
 * sans lui poser de camp — celui d'une insertion est PROPOSÉ par le moteur et
 * celui d'une Action lui appartient (ADR-0045 §4) —, demander les candidats de
 * la POSITION VISÉE plutôt que de celle atteinte par le match, et afficher ce
 * que le moteur dit de la saisie en cours.
 *
 * Le panneau ne juge rien et ne dérive rien : il est un client du moteur
 * (règle 9). D'où des documents annotés tout faits, et un regard sur l'écran.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, screen, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListTranscriptions: vi.fn().mockResolvedValue([]),
    CreateTranscription: vi.fn(),
    OpenTranscription: vi.fn(),
    ApplyTranscriptionGesture: vi.fn(),
    TranscriptionMAT: vi.fn().mockResolvedValue('')
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    LegalMoves: vi.fn(),
    EvaluatePositionImmediate: vi.fn()
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetPruneK: vi.fn().mockResolvedValue(0)
}));

import { ApplyTranscriptionGesture, ListTranscriptions } from '../../wailsjs/go/database/Database.js';
import { LegalMoves, EvaluatePositionImmediate } from '../../wailsjs/go/gui/App.js';

import TranscriptionPanel from '../components/TranscriptionPanel.svelte';
import { get } from 'svelte/store';
import { transcriptionListStore, transcriptionStore, transcriptionHistoryStore, transcriptionNoticeStore, transcriptionPromptStore, clearTranscription } from '../stores/transcriptionStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';

const board = () => ({ points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [0, 0] });

const position = (dice = [0, 0], mark = 0) => ({ board: board(), cube: { owner: -1, value: 0 }, dice, score: [7, 7], player_on_roll: mark, decision_type: 0 });

/**
 * Un document de trois coups, le Cursor sur celui du milieu — l'état exact dans
 * lequel une correction se fait. `entry` est ce que le moteur dit de la saisie
 * en cours (transcript.EntryInfo) ; nul quand rien n'est tapé.
 */
function annotated({ cursor = 1, entry = null, flags = [] } = {}) {
    const action = (i) => ({
        index: i,
        side: i % 2,
        kind: 'checker',
        before: position([6, 3], i % 2),
        has_position: true,
        notation: `24/18 13/1${i}`,
        game_index: 0,
        game_number: 1,
        score: [0, 0],
        move_number: i,
        inconsistencies: i === cursor ? flags.map((kind) => ({ kind, detail: 'engine prose' })) : []
    });
    return {
        document: {
            header: { match_length: 7, player1: 'Kévin', player2: 'Alice' },
            actions: [0, 1, 2].map((i) => ({ side: i % 2, kind: 'checker', dice: [6, 3], steps: [] })),
            cursor
        },
        actions: [0, 1, 2].map(action),
        games: [{ number: 1, initial_score: [0, 0], winner: -1, points_won: 0, crawford: false, finished: false, first: 0, last: 2 }],
        next: { expects: 'checker', side: 1, position: position(), crawford: false, game_number: 1 },
        entry,
        score: [0, 0],
        finished: false,
        winner: -1,
        cursor
    };
}

const state = (ann, history = {}) => ({ id: 1, annotated: ann, can_undo: false, can_redo: false, ...history });

function press(code, init = {}) {
    const digit = /^Digit([1-9])$/.exec(code);
    const letter = /^Key([A-Z])$/.exec(code);
    return fireEvent.keyDown(document, { code, key: digit ? digit[1] : letter ? letter[1].toLowerCase() : code, ...init });
}

async function openedPanel(ann = annotated(), history = {}) {
    transcriptionStore.set(state(ann, history));
    transcriptionHistoryStore.set({ canUndo: history.can_undo === true, canRedo: history.can_redo === true });
    render(TranscriptionPanel);
    await tick();
    document.getElementById('transcriptionPanel')?.focus();
}

const gestures = () => ApplyTranscriptionGesture.mock.calls.map((call) => call[1]);

beforeEach(() => {
    vi.clearAllMocks();
    ListTranscriptions.mockResolvedValue([]);
    ApplyTranscriptionGesture.mockImplementation(() => Promise.resolve(state(annotated())));
    LegalMoves.mockResolvedValue([]);
    EvaluatePositionImmediate.mockResolvedValue({ moves: [] });
    transcriptionListStore.set([]);
    clearTranscription();
    selectedMoveStore.set(null);
    databasePathStore.set('/tmp/library.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
});

afterEach(cleanup);

describe('les touches de correction deviennent des gestes, sans camp', () => {
    test.each([
        ['KeyI', 'insert_before'],
        ['KeyA', 'insert_after'],
        ['KeyX', 'delete'],
        ['Delete', 'delete'],
        ['KeyS', 'flip_side']
    ])('%s → %s', async (code, kind) => {
        await openedPanel();
        await press(code);
        await vi.waitFor(() => expect(gestures().at(-1)).toEqual({ Kind: kind }));
    });
});

describe('les mêmes gestes à la souris', () => {
    // ADR-0048 décision 3 : la barre de correction n'existe plus. Ses six
    // boutons agissaient sur « l'Action au Cursor », c'est-à-dire sur une cible
    // que l'utilisateur ne voit pas nécessairement, quand le clic droit sur une
    // cellule du Transcript offre les quatre mêmes gestes EN NOMMANT sa cible.
    // Les quatre passent donc par le menu — tenu par TranscriptionPanel.mouse —
    // et l'annulation garde ses deux flèches, qui sont le chemin souris exigé
    // par R3 (décision 11).
    test('annuler et rétablir ont chacun leur bouton dans la barre du brouillon', async () => {
        // La pile reste garnie des deux côtés : ce test regarde le CHEMIN (R3),
        // pas l'épuisement de la pile, qui est le test suivant.
        ApplyTranscriptionGesture.mockImplementation(() => Promise.resolve(state(annotated(), { can_undo: true, can_redo: true })));
        await openedPanel(annotated(), { can_undo: true, can_redo: true });
        for (const [title, kind] of [
            [/Undo the last gesture/i, 'undo'],
            [/Redo the undone gesture/i, 'redo']
        ]) {
            ApplyTranscriptionGesture.mockClear();
            await fireEvent.click(screen.getByTitle(title));
            await vi.waitFor(() => expect(gestures().at(-1)).toEqual({ Kind: kind }));
        }
    });

    // ADR-0048 décision 9 : le bouton grisé était le seul de la famille des
    // gestes sans effet à dire quelque chose, et c'est celui qui disparaît. La
    // réponse est désormais transitoire, dans la barre d'état — ce qui vaut
    // aussi pour le clavier, où aucun bouton grisé n'a jamais rien dit.
    test('annuler sur une pile vide répond au lieu de se taire', async () => {
        await openedPanel();
        await fireEvent.click(screen.getByTitle(/Undo the last gesture/i));
        expect(get(transcriptionNoticeStore)?.key).toBe('transcription.notice.nothingToUndo');
        expect(gestures()).toEqual([]);
    });

    test('x et s en bout de document répondent, et n’envoient rien', async () => {
        await openedPanel(annotated({ cursor: 3 }));
        await press('KeyX');
        expect(get(transcriptionNoticeStore)?.key).toBe('transcription.notice.noAction');
        await press('KeyS');
        await tick();
        expect(gestures()).toEqual([]);
    });
});

describe('une correction se joue depuis la position visée', () => {
    test('les candidats demandés sont ceux de l’Action corrigée, pas ceux de la fin du document', async () => {
        const entry = { at: 1, replacing: true, side: 1, dice: [5, 2], selected: true, review: false };
        const corrected = annotated({ cursor: 1, entry });
        ApplyTranscriptionGesture.mockImplementation(() => Promise.resolve(state(corrected)));
        await openedPanel(corrected);

        // Deux dés retapés sur l'Action du Cursor.
        await press('Digit5');
        await press('Digit2');

        await vi.waitFor(() => expect(LegalMoves).toHaveBeenCalled());
        // `LegalMoves` est appelée aussi pour armer le coup joué au plateau —
        // une fois par jet, T2.3 — donc l'appel visé ici est nommé par son jet
        // et non par son rang dans la liste des appels.
        const asked = LegalMoves.mock.calls.findLast(([p]) => p.dice[0] === 5 && p.dice[1] === 2)[0];
        // La position de l'Action 1, jouée par le camp que le moteur nomme —
        // et non `next.position`, qui est celle de la fin du match.
        expect(asked.player_on_roll).toBe(1);
        expect(asked.dice).toEqual([5, 2]);
    });

    // Les deux phrases sont des états de l'Entry, donc de la nature « ce que le
    // document attend » : elles habitent la barre d'état (ADR-0048 décision 8),
    // que ce test lit par son magasin puisqu'il monte le panneau seul.
    test('« à revoir » est dit par le moteur, jamais deviné par le panneau', async () => {
        await openedPanel(annotated({ cursor: 1, entry: { at: 1, replacing: true, side: 1, dice: [2, 1], selected: true, review: true } }));
        expect(get(transcriptionPromptStore)?.key).toBe('transcription.reviewHint');
    });

    test('une correction en place se dit à l’écran, et une saisie neuve ne le dit pas', async () => {
        await openedPanel(annotated({ cursor: 1, entry: { at: 1, replacing: true, side: 1, dice: [6, 3], selected: true, review: false } }));
        expect(get(transcriptionPromptStore)?.key).toBe('transcription.correcting');

        cleanup();
        await openedPanel(annotated({ cursor: 3, entry: { at: 3, replacing: false, side: 1, dice: [0, 0], selected: false, review: false } }));
        expect(get(transcriptionPromptStore)?.key).not.toBe('transcription.correcting');
    });
});

describe('rien n’est refusé : l’Incohérence est montrée là où elle est', () => {
    test('la cellule du Transcript porte la marque, et le Cursor est dessus', async () => {
        await openedPanel(annotated({ cursor: 1, flags: ['double_turn'] }));
        const cell = document.querySelector('[data-inconsistency~="double_turn"]');
        expect(cell).not.toBeNull();
        expect(cell.getAttribute('data-index')).toBe('1');
        expect(cell.getAttribute('aria-current')).toBe('true');
    });
});
