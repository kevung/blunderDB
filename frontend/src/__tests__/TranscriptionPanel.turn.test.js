/**
 * TranscriptionPanel.turn.test.js — T1.3 : le tour de pions dans le panneau.
 *
 * La machine à touches est testée à part (transcriptionKeys.turn.test.js) ; ce
 * qui est vérifié ici, c'est la JOINTURE que seul le panneau fait : le moteur
 * CLASSE (EvaluatePositionImmediate, 0-ply, tous les coups), le générateur
 * NUMÉROTE (LegalMoves), et `select_candidate` s'adresse au second par le
 * rang du premier. Une jointure fausse choisit un autre coup que celui affiché,
 * ce qu'aucun test de la machine seule ne verrait.
 *
 * Et la danse : un jet sans aucun coup légal crée l'Action sans une touche de
 * plus.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, screen, fireEvent } from '@testing-library/svelte';
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

import { ApplyTranscriptionGesture, ListTranscriptions } from '../../wailsjs/go/database/Database.js';
import { LegalMoves, EvaluatePositionImmediate } from '../../wailsjs/go/gui/App.js';

import TranscriptionPanel from '../components/TranscriptionPanel.svelte';
import { transcriptionListStore, transcriptionStore, transcriptionKeyStore, clearTranscription } from '../stores/transcriptionStore.js';
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

function annotated({ expects = 'checker', side = 0 } = {}) {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice' }, actions: [], cursor: 0 },
        actions: [],
        games: [],
        next: { expects, side, position: POSITION, crawford: false },
        score: [0, 0],
        cursor: 0
    };
}

const state = (ann) => ({ id: 1, annotated: ann });

// Le générateur rend les coups dans SON ordre ; l'évaluation les rend classés.
// Les deux ordres diffèrent ici exprès : c'est tout l'objet du test.
const PLAYS = [{ notation: '24/23 13/11' }, { notation: '13/11 6/5' }, { notation: '8/5' }];
const RANKED = [
    { index: 0, move: '8/5', equity: 0.12 },
    { index: 1, move: '13/11 6/5', equity: 0.04, equityError: 0.08 },
    { index: 2, move: '24/23 13/11', equity: -0.1, equityError: 0.22 }
];

function press(code) {
    const digit = /^Digit([1-9])$/.exec(code);
    const letter = /^Key([A-Z])$/.exec(code);
    return fireEvent.keyDown(document, { code, key: digit ? digit[1] : letter ? letter[1].toLowerCase() : code });
}

async function openedPanel() {
    transcriptionStore.set(state(annotated()));
    render(TranscriptionPanel);
    await tick();
    document.getElementById('transcriptionPanel')?.focus();
}

/** Les gestes envoyés au moteur, dans l'ordre. */
const gestures = () => ApplyTranscriptionGesture.mock.calls.map((call) => call[1]);

beforeEach(() => {
    vi.clearAllMocks();
    ListTranscriptions.mockResolvedValue([]);
    ApplyTranscriptionGesture.mockImplementation(() => Promise.resolve(state(annotated())));
    LegalMoves.mockResolvedValue(PLAYS);
    EvaluatePositionImmediate.mockResolvedValue({ moves: RANKED });
    transcriptionListStore.set([]);
    clearTranscription();
    selectedMoveStore.set(null);
    databasePathStore.set('/tmp/library.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
});

afterEach(cleanup);

describe('le tour de pions', () => {
    test('deux dés listent TOUS les coups légaux, classés en 0-ply', async () => {
        await openedPanel();
        await press('Digit3');
        await press('Digit1');

        await vi.waitFor(() => expect(EvaluatePositionImmediate).toHaveBeenCalled());

        // Le jet et le camp sont posés sur la position du Cursor.
        const [pos] = LegalMoves.mock.calls[0];
        expect(pos.dice).toEqual([3, 1]);
        expect(pos.player_on_roll).toBe(0);

        // candidates = 0 : tous les coups, pas les dix d'une analyse stockée.
        expect(EvaluatePositionImmediate.mock.calls[0][2]).toBe(0);

        // La table montre l'ordre de l'évaluation, meilleur d'abord.
        await vi.waitFor(() => expect(screen.getByText('8/5')).toBeTruthy());
        const rows = [...document.querySelectorAll('.candidates tbody tr')].map((tr) => tr.textContent);
        expect(rows[0]).toContain('8/5');
        expect(rows[2]).toContain('24/23 13/11');
    });

    // La jointure : le meilleur coup au classement est le TROISIÈME du
    // générateur, et c'est ce rang-là que le geste doit porter.
    test('le candidat présélectionné est adressé par son rang chez le générateur', async () => {
        await openedPanel();
        await press('Digit3');
        await press('Digit1');

        await vi.waitFor(() => expect(gestures()).toContainEqual({ Kind: 'select_candidate', Candidate: 2 }));
        expect(get(transcriptionKeyStore).selected).toBe(0);
    });

    test('les flèches du candidat sélectionné partent sur le plateau', async () => {
        await openedPanel();
        await press('Digit3');
        await press('Digit1');
        await vi.waitFor(() => expect(get(selectedMoveStore)).toBe('8/5'));

        await press('KeyJ');
        await vi.waitFor(() => expect(get(selectedMoveStore)).toBe('13/11 6/5'));
        // Rang 1 au classement = rang 1 chez le générateur, ici par coïncidence :
        // ce qui compte est que le geste suive la ligne affichée.
        expect(gestures()).toContainEqual({ Kind: 'select_candidate', Candidate: 1 });
    });

    test('un chiffre depuis « candidat choisi » valide puis ouvre le jet suivant', async () => {
        await openedPanel();
        await press('Digit3');
        await press('Digit1');
        await vi.waitFor(() => expect(get(selectedMoveStore)).toBe('8/5'));
        await press('KeyJ');
        await press('Digit6');

        await vi.waitFor(() => expect(gestures().slice(-2)).toEqual([{ Kind: 'validate' }, { Kind: 'enter_die', Die: 6 }]));
    });

    test('Entrée valide seule', async () => {
        await openedPanel();
        await press('Digit3');
        await press('Digit1');
        await vi.waitFor(() => expect(get(selectedMoveStore)).toBe('8/5'));
        await press('Enter');
        await vi.waitFor(() => expect(gestures().at(-1)).toEqual({ Kind: 'validate' }));
    });

    test('la danse est créée sans une touche de plus', async () => {
        LegalMoves.mockResolvedValue([]);
        await openedPanel();
        await press('Digit6');
        await press('Digit6');

        await vi.waitFor(() => expect(gestures().at(-1)).toEqual({ Kind: 'dance' }));
        // Aucun classement n'a été demandé : il n'y a rien à classer.
        expect(EvaluatePositionImmediate).not.toHaveBeenCalled();
        expect(await screen.findByText('No legal play: the dance is recorded.')).toBeTruthy();
    });

    // Un moteur qui refuse (un score hors de portée de la MET, une compilation
    // sans poids) ne doit pas empêcher de transcrire.
    test('sans classement, les coups légaux restent saisissables', async () => {
        EvaluatePositionImmediate.mockResolvedValue({ refused: true });
        await openedPanel();
        await press('Digit3');
        await press('Digit1');

        await vi.waitFor(() => expect(screen.getByText("Evaluation unavailable: the legal plays are in the generator's order.")).toBeTruthy());
        expect(gestures()).toContainEqual({ Kind: 'select_candidate', Candidate: 0 });
    });
});
