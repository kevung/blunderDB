/**
 * TranscriptionPanel.mouse.test.js — T2.5 : les deux déclencheurs souris, du
 * clic jusqu'au geste envoyé au moteur.
 *
 * L'équivalence des gestes est mesurée dans la machine
 * (transcriptionKeys.cubeMouse.test.js) et la cible du plateau dans
 * boardInteractions.test.js. Ce qui manque entre les deux, et qui est ici, est
 * le CHAÎNAGE : que le videau cliqué sur le plateau arrive bien jusqu'à
 * `ApplyTranscriptionGesture`, que la demande soit jetée là où elle ne répond à
 * rien, et qu'une entrée du menu contextuel du Transcript envoie la même suite
 * de gestes que la relecture au clavier.
 *
 * Le panneau est un CLIENT du moteur Go (ADR-0045 règle 9) : ce qu'on regarde
 * est donc exactement ce qu'il envoie, et rien de ce qu'il en déduirait.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { get } from 'svelte/store';
import { tick } from 'svelte';

const engine = vi.hoisted(() => ({ sent: [], state: null }));

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListTranscriptions: vi.fn(() => Promise.resolve([])),
    CreateTranscription: vi.fn(() => Promise.resolve(engine.state)),
    OpenTranscription: vi.fn(() => Promise.resolve(engine.state)),
    ApplyTranscriptionGesture: vi.fn((id, gesture) => {
        engine.sent.push(gesture);
        return Promise.resolve(engine.state);
    }),
    TranscriptionMAT: vi.fn(() => Promise.resolve('')),
    SaveTranscriptionAsMatch: vi.fn(() => Promise.resolve(0)),
    SuggestTranscriptionMatFilename: vi.fn(() => Promise.resolve('')),
    ExportTranscriptionMAT: vi.fn(() => Promise.resolve()),
    CloseTranscription: vi.fn(() => Promise.resolve()),
    PendingTranscriptionAnalysis: vi.fn(() => Promise.resolve(null))
}));

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    LegalMoves: vi.fn(() => Promise.resolve([])),
    EvaluatePositionImmediate: vi.fn(() => Promise.resolve({ refused: true, moves: [] })),
    OpenExportMatDialog: vi.fn(() => Promise.resolve('')),
    StartGammonNetMatchBatch: vi.fn(() => Promise.resolve())
}));

vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetPruneK: vi.fn(() => Promise.resolve(0)),
    GetGammonNetAnalysisPly: vi.fn(() => Promise.resolve(0))
}));

import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { setTranscription, clearTranscription, transcriptionCubeRequestStore, transcriptionKeyStore } from '../stores/transcriptionStore.js';
import { PHASE } from '../services/transcriptionKeys.js';
import { TranscriptionMAT } from '../../wailsjs/go/database/Database.js';
import TranscriptionPanel from '../components/TranscriptionPanel.svelte';

const POSITION = (dice = [0, 0]) => ({
    board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [0, 0] },
    cube: { owner: -1, value: 0 },
    dice,
    score: [7, 7],
    player_on_roll: 0,
    decision_type: 0
});

/**
 * Un brouillon ouvert, tel que la liaison Wails le rend : deux coups écrits,
 * le Cursor au bout du document, et ce que le document attend ensuite.
 */
function draftState(expects = 'checker', cursor = 2) {
    const actions = [
        { index: 0, side: 0, kind: 'checker', before: POSITION([3, 1]), has_position: true, notation: '8/5 6/5', game_index: 0, game_number: 1, score: [0, 0], move_number: 0, inconsistencies: [] },
        { index: 1, side: 1, kind: 'checker', before: POSITION([5, 2]), has_position: true, notation: '13/8 13/11', game_index: 0, game_number: 1, score: [0, 0], move_number: 1, inconsistencies: [] }
    ];
    return {
        id: 1,
        can_undo: false,
        can_redo: false,
        annotated: {
            document: {
                header: { match_length: 7, player1: 'Kévin', player2: 'Alice' },
                actions: [
                    { side: 0, kind: 'checker', dice: [3, 1] },
                    { side: 1, kind: 'checker', dice: [5, 2] }
                ],
                cursor
            },
            actions,
            games: [{ number: 1, initial_score: [0, 0], winner: -1, points_won: 0, crawford: false, finished: false, first: 0, last: 1 }],
            next: { expects, side: 0, position: POSITION(), crawford: false },
            score: [0, 0],
            cursor,
            finished: false,
            winner: -1
        }
    };
}

/** Monte le panneau sur un brouillon ouvert et laisse les effets s'installer. */
async function mount(state) {
    engine.state = state;
    engine.sent = [];
    const view = render(TranscriptionPanel);
    setTranscription(state);
    await settle();
    return view;
}

async function settle() {
    await new Promise((resolve) => setTimeout(resolve, 0));
    for (let i = 0; i < 6; i++) await tick();
}

const kinds = () => engine.sent.map((g) => g.Kind);
const buttonNamed = (label) => [...document.querySelectorAll('button')].find((b) => b.textContent.trim() === label);
const menuItemNamed = (label) => [...document.querySelectorAll('.context-menu button')].find((b) => b.textContent.trim() === label);

beforeEach(() => {
    vi.clearAllMocks();
    engine.sent = [];
    clearTranscription();
    activeTabStore.set('match'); // la liste des brouillons ne nous concerne pas
    statusBarModeStore.set('TRANSCRIBE');
    databasePathStore.set('/tmp/test.db');
});

afterEach(() => {
    clearTranscription();
    cleanup();
});

describe('le videau cliqué sur le plateau', () => {
    test('devient un double envoyé au moteur, et la demande est servie une fois', async () => {
        await mount(draftState('checker'));

        transcriptionCubeRequestStore.set('double');
        await settle();

        expect(kinds()).toEqual(['double']);
        // Servie, donc effacée : elle ne doit pas repartir au geste suivant.
        expect(get(transcriptionCubeRequestStore)).toBeNull();
    });

    // La décision de T2.5 : devant une offre, le videau ne répond pas. Prendre
    // et passer sont deux réponses symétriques, elles vivent dans la rangée.
    test('devant une réponse attendue, la demande est jetée', async () => {
        await mount(draftState('take'));

        transcriptionCubeRequestStore.set('double');
        await settle();

        expect(kinds()).toEqual([]);
        expect(get(transcriptionCubeRequestStore)).toBeNull();
    });
});

describe('la rangée [D] [T] [P] [R] envoie ce que la touche envoie', () => {
    test('[D] envoie le double, comme la touche d', async () => {
        await mount(draftState('checker'));

        await fireEvent.click(buttonNamed('Double'));
        await settle();
        const byButton = [...engine.sent];

        engine.sent = [];
        document.dispatchEvent(new KeyboardEvent('keydown', { code: 'KeyD', key: 'd', bubbles: true }));
        await settle();

        expect(byButton).toEqual([{ Kind: 'double' }]);
        expect(engine.sent).toEqual(byButton);
    });

    test('[T] envoie la prise devant une offre', async () => {
        await mount(draftState('take'));
        await fireEvent.click(buttonNamed('Take'));
        await settle();
        expect(engine.sent).toEqual([{ Kind: 'take' }]);
    });

    // [R] n'écrit rien : il arme l'attente du niveau, et c'est le niveau qui
    // crée l'Action — les deux gestes d'ux.md §4.2.
    test('[R] arme le niveau, et le niveau cliqué crée la résignation', async () => {
        await mount(draftState('checker'));

        await fireEvent.click(buttonNamed('Resign'));
        await settle();
        expect(engine.sent).toEqual([]);
        expect(get(transcriptionKeyStore).phase).toBe(PHASE.RESIGN);

        await fireEvent.click(buttonNamed('Gammon'));
        await settle();
        expect(engine.sent).toEqual([{ Kind: 'resign', Level: 2 }]);
    });
});

describe('le menu contextuel du Transcript', () => {
    test('« supprimer » mène le Cursor à la cellule puis supprime', async () => {
        // Le Cursor est au bout du document (2) ; la cellule visée est la 0.
        const { container } = await mount(draftState('checker', 2));

        await fireEvent.contextMenu(container.querySelector('[data-index="0"]'), { clientX: 40, clientY: 60 });
        await tick();

        // Dans le MENU, et non dans la barre de correction, qui porte le même
        // mot : la barre agit sur le Cursor là où il est, le menu sur la
        // cellule visée.
        const entry = menuItemNamed('Delete');
        expect(entry).toBeTruthy();
        await fireEvent.click(entry);
        await settle();

        // Deux pas en arrière, puis la suppression : exactement `h` `h` `x`.
        expect(kinds()).toEqual(['cursor_back', 'cursor_back', 'delete']);
    });

    test('les quatre entrées sont les quatre corrections, et rien de plus', async () => {
        const { container } = await mount(draftState('checker', 2));

        await fireEvent.contextMenu(container.querySelector('[data-index="1"]'), { clientX: 40, clientY: 60 });
        await tick();

        const menu = document.querySelector('.context-menu');
        expect([...menu.querySelectorAll('button')].map((b) => b.textContent.trim())).toEqual(['Insert before', 'Insert after', 'Delete', 'Change side']);
    });
});

// ── la modale du texte .mat (ADR-0048 décision 6) ────────────────────────
//
// Le volet vivait sous le tableau des décisions, dans `TranscriptView`, avec un
// presse-papiers et un minuteur que le docstring de ce composant promet qu'il ne
// tient pas. Il est ici, en modale, et pour une raison mesurée : la ligne la plus
// longue d'un `.mat` fait 62 caractères, ~409 px en monospace 11 px, là où la
// colonne du Transcript en offrait 320 — l'alignement en colonnes, qui EST
// l'information, y était détruit.
describe('le texte .mat', () => {
    test('le bouton de la barre ouvre la modale, qui va chercher le texte', async () => {
        const MAT = '; [Player 1 "Kévin"]\n\n7 point match\n';
        TranscriptionMAT.mockResolvedValue(MAT);
        await mount(draftState());

        // Fermée, elle ne coûte aucun aller-retour : c'est ce qui permettait au
        // volet d'origine d'être là sans peser, et cela reste vrai.
        expect(document.querySelector('.mat-text')).toBeNull();
        expect(TranscriptionMAT).not.toHaveBeenCalled();

        await fireEvent.click(buttonNamed('.mat text'));
        await vi.waitFor(() => expect(document.querySelector('.mat-text')?.textContent).toBe(MAT));
    });
});
