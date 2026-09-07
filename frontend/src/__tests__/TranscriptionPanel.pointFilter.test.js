/**
 * TranscriptionPanel.pointFilter.test.js — T2.2 : la liste réduite dans le panneau.
 *
 * Le filtre lui-même est pur et testé à part (transcriptionFilter.test.js), le
 * clic qui le pose l'est dans boardInteractions.test.js. Ce qui ne se voit
 * qu'ici : que la liste MONTRÉE soit la liste réduite, que tout rang s'adresse
 * à celle-là — `j` comme le geste `select_candidate` envoyé au moteur, dont
 * l'index est celui du générateur et non du classement — et que le filtre
 * meure avec le jet auquel il appartenait, sans jamais rien écrire dans le
 * document.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

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
import { transcriptionListStore, transcriptionStore, transcriptionPointFilterStore, transcriptionCandidateStepsStore, clearTranscription } from '../stores/transcriptionStore.js';
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

function annotated() {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice' }, actions: [], cursor: 0 },
        actions: [],
        games: [],
        next: { expects: 'checker', side: 0, position: POSITION, crawford: false },
        score: [0, 0],
        cursor: 0
    };
}

const state = (ann) => ({ id: 1, annotated: ann });

const step = (from) => ({ from, to: from - 1, hit: false });
// L'ordre du générateur et celui du classement diffèrent exprès : le geste
// envoyé au moteur porte le rang du GÉNÉRATEUR, celui de l'écran est le rang du
// classement, et le filtre ne doit pas confondre les deux.
const PLAYS = [
    { notation: '24/23 13/11', steps: [step(24), step(13)] },
    { notation: '13/11 6/5', steps: [step(13), step(6)] },
    { notation: '8/5', steps: [step(8)] }
];
const RANKED = [
    { index: 0, move: '8/5', equity: 0.12 },
    { index: 1, move: '13/11 6/5', equity: 0.04, equityError: 0.08 },
    { index: 2, move: '24/23 13/11', equity: -0.1, equityError: 0.22 }
];

const gestures = () => ApplyTranscriptionGesture.mock.calls.map((call) => call[1]);
const selects = () => gestures().filter((g) => g.Kind === 'select_candidate');
const rows = () => [...document.querySelectorAll('.candidates tbody tr')].map((tr) => tr.textContent);

function press(code) {
    const digit = /^Digit([1-9])$/.exec(code);
    const letter = /^Key([A-Z])$/.exec(code);
    return fireEvent.keyDown(document, { code, key: digit ? digit[1] : letter ? letter[1].toLowerCase() : code });
}

/** Un jet 3-1 saisi, ses trois candidats classés et affichés. */
async function rolled() {
    transcriptionStore.set(state(annotated()));
    render(TranscriptionPanel);
    await tick();
    document.getElementById('transcriptionPanel')?.focus();
    await press('Digit3');
    await press('Digit1');
    await vi.waitFor(() => expect(rows()).toHaveLength(3));
}

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

describe('la liste réduite par le point de départ', () => {
    // Ce que le plateau lit pour décider si un clic le concerne.
    test('les pas des candidats sont publiés pour le plateau', async () => {
        await rolled();
        await vi.waitFor(() => expect(get(transcriptionCandidateStepsStore)).toHaveLength(3));
        expect(get(transcriptionCandidateStepsStore)[0].steps).toEqual([step(8)]);
    });

    test('un point ne laisse que les coups qui en partent, dans l’ordre du classement', async () => {
        await rolled();
        transcriptionPointFilterStore.set([13]);
        await tick();

        const shown = rows();
        expect(shown).toHaveLength(2);
        expect(shown[0]).toContain('13/11 6/5');
        expect(shown[1]).toContain('24/23 13/11');
    });

    // La jointure, sous filtre : le premier de la liste réduite est le SECOND
    // du générateur, et c'est ce rang-là que le geste doit porter.
    test('le premier de la liste réduite est présélectionné, par son rang chez le générateur', async () => {
        await rolled();
        transcriptionPointFilterStore.set([13]);
        await vi.waitFor(() => expect(selects().at(-1)).toEqual({ Kind: 'select_candidate', Candidate: 1 }));
    });

    test('j se déplace DANS la liste réduite', async () => {
        await rolled();
        transcriptionPointFilterStore.set([13]);
        await vi.waitFor(() => expect(selects().at(-1)).toEqual({ Kind: 'select_candidate', Candidate: 1 }));

        await press('KeyJ');
        await vi.waitFor(() => expect(selects().at(-1)).toEqual({ Kind: 'select_candidate', Candidate: 0 }));

        // Et il ne sort pas de la liste réduite : un second `j` ne va nulle part.
        await press('KeyJ');
        expect(selects().at(-1)).toEqual({ Kind: 'select_candidate', Candidate: 0 });
    });

    test('les flèches du plateau suivent la liste réduite', async () => {
        await rolled();
        expect(get(selectedMoveStore)).toBe('8/5');
        transcriptionPointFilterStore.set([13]);
        await tick();
        expect(get(selectedMoveStore)).toBe('13/11 6/5');
    });

    test('le filtre est dit à l’écran', async () => {
        await rolled();
        transcriptionPointFilterStore.set([13]);
        await tick();
        expect(document.body.textContent).toContain('13');
    });

    // Le filtre appartient au jet : un nouveau jet le lève, sinon la liste du
    // suivant serait réduite par un point du précédent.
    test('un nouveau jet lève le filtre', async () => {
        await rolled();
        transcriptionPointFilterStore.set([13]);
        await tick();

        await press('Digit4');
        await press('Digit2');
        await vi.waitFor(() => expect(get(transcriptionPointFilterStore)).toEqual([]));
        await vi.waitFor(() => expect(rows()).toHaveLength(3));
    });

    // Un état d'AFFICHAGE : aucun geste ne part au moteur pour le filtre
    // lui-même, seule la présélection bouge.
    test('poser un filtre ne crée aucune Action', async () => {
        await rolled();
        const before = gestures().filter((g) => g.Kind !== 'select_candidate').length;
        transcriptionPointFilterStore.set([13]);
        await tick();
        await vi.waitFor(() => expect(selects().length).toBeGreaterThan(1));
        expect(gestures().filter((g) => g.Kind !== 'select_candidate')).toHaveLength(before);
    });
});
