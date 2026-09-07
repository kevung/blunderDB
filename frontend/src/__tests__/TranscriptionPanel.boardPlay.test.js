/**
 * TranscriptionPanel.boardPlay.test.js — T2.3 dans le panneau.
 *
 * Le réducteur est éprouvé à part (transcriptionPlay.test.js) et le clic qui le
 * fait avancer aussi (boardInteractions.test.js). Ce qui ne se voit qu'ici est
 * la couture : que le plateau soit ARMÉ de l'union des vingt et un jets tant
 * qu'aucun dé n'est tapé, que le coup achevé parte au moteur avec ses deux dés
 * déduits et pas un chiffre tapé, et que l'ambiguïté n'enregistre rien et
 * n'offre que les jets possibles.
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
import { transcriptionListStore, transcriptionStore, clearTranscription } from '../stores/transcriptionStore.js';
import { quizPlayStore } from '../stores/quizPlayStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';
import { selectSource, playHop } from '../services/quizPlay.js';

const BLACK = 0;
const WHITE = 1;

function positionWith(stacks, mover = BLACK) {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    for (const [point, [checkers, color]] of Object.entries(stacks)) points[Number(point)] = { checkers, color };
    return {
        id: 0,
        board: { points, bearoff: [0, 0] },
        cube: { owner: -1, value: 0 },
        dice: [0, 0],
        score: [7, 7],
        player_on_roll: mover,
        decision_type: 0
    };
}

const POSITION = positionWith({ 13: [5, BLACK], 8: [3, BLACK], 6: [5, BLACK], 12: [2, WHITE] });

const step = (from, to) => ({ from, to, hit: false });
const play = (...steps) => ({ steps, notation: steps.map((s) => `${s.from}/${s.to}`).join(' '), result: {} });

// Ce que `LegalMoves` rendrait, jet par jet. Tout ce qui n'est pas nommé ici est
// un jet sans coup, et il s'écarte de lui-même de l'union.
const PLAYS_BY_ROLL = {
    61: [play(step(13, 7), step(8, 7))],
    62: [play(step(13, 7), step(13, 11))],
    66: [play(step(13, 7), step(13, 7), step(8, 2), step(8, 2))],
    21: [play(step(13, 11), step(13, 12))]
};

function annotated(extra = {}) {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice' }, actions: [], cursor: 0 },
        actions: [],
        games: [],
        next: { expects: 'checker', side: BLACK, position: POSITION, crawford: false },
        score: [0, 0],
        cursor: 0,
        ...extra
    };
}

const state = (ann) => ({ id: 1, annotated: ann });
const gestures = () => ApplyTranscriptionGesture.mock.calls.map((call) => call[1]);

function press(code) {
    const digit = /^Digit([1-9])$/.exec(code);
    return fireEvent.keyDown(document, { code, key: digit ? digit[1] : code });
}

/** Le panneau ouvert sur un brouillon, et le plateau armé du coup à jouer. */
async function armed() {
    transcriptionStore.set(state(annotated()));
    render(TranscriptionPanel);
    await tick();
    document.getElementById('transcriptionPanel')?.focus();
    await vi.waitFor(() => expect(get(quizPlayStore)).not.toBeNull());
    return get(quizPlayStore);
}

/** Un pas joué au plateau, tel que le clic le joue. */
function hop(from, to) {
    quizPlayStore.update((s) => playHop(selectSource(s, from), from, to));
}

const diceCells = () => [...document.querySelectorAll('.dice-triangle button')];
const cell = (label) => diceCells().find((b) => b.textContent.trim() === label);

beforeEach(() => {
    vi.clearAllMocks();
    ListTranscriptions.mockResolvedValue([]);
    ApplyTranscriptionGesture.mockImplementation(() => Promise.resolve(state(annotated())));
    LegalMoves.mockImplementation((pos) => {
        const [a, b] = pos.dice;
        return Promise.resolve(PLAYS_BY_ROLL[a >= b ? `${a}${b}` : `${b}${a}`] ?? []);
    });
    EvaluatePositionImmediate.mockResolvedValue({ moves: [] });
    transcriptionListStore.set([]);
    clearTranscription();
    quizPlayStore.set(null);
    selectedMoveStore.set(null);
    databasePathStore.set('/tmp/library.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
});

afterEach(() => {
    cleanup();
    quizPlayStore.set(null);
});

describe('le plateau joue le coup et déduit les dés (T2.3)', () => {
    test('tant qu’aucun dé n’est tapé, le plateau porte l’union des jets', async () => {
        const play = await armed();
        // Un coup par jet jouable, quatre jets.
        expect(play.plays).toHaveLength(4);
        expect(new Set(play.plays.map((p) => p.roll.join('')))).toEqual(new Set(['61', '62', '66', '21']));
    });

    test('un dé tapé rend la main à la saisie par les dés', async () => {
        await armed();
        await press('Digit3');
        await vi.waitFor(() => expect(get(quizPlayStore)).toBeNull());
    });

    // La recette de la fiche : deux clics sur un double, l'Action créée avec
    // les deux dés à 6 et les quatre pas joués, sans une touche.
    test('les quatre pas d’un double enregistrent l’Action, dés déduits', async () => {
        await armed();
        hop(13, 7);
        hop(13, 7);
        hop(8, 2);
        hop(8, 2);

        await vi.waitFor(() => expect(gestures().some((g) => g.Kind === 'validate')).toBe(true));
        const sent = gestures();
        expect(sent[0]).toEqual({ Kind: 'enter_die', Die: 6 });
        expect(sent[1]).toEqual({ Kind: 'enter_die', Die: 6 });
        expect(sent[2].Kind).toBe('enter_play');
        expect(sent[2].Steps).toEqual([
            { from: 13, to: 7, hit: false },
            { from: 13, to: 7, hit: false },
            { from: 8, to: 2, hit: false },
            { from: 8, to: 2, hit: false }
        ]);
        expect(sent[3]).toEqual({ Kind: 'validate' });
    });

    test('un coup à moitié joué n’enregistre rien', async () => {
        await armed();
        hop(13, 7);
        await tick();
        expect(gestures()).toEqual([]);
    });
});

describe('l’ambiguïté n’est jamais tranchée par le logiciel', () => {
    // Deux jets dont un seul dé est jouable : le même pas les achève tous deux.
    const AMBIGUOUS = { 61: [play(step(13, 7))], 62: [play(step(13, 7))] };

    async function ambiguous() {
        LegalMoves.mockImplementation((pos) => {
            const [a, b] = pos.dice;
            return Promise.resolve(AMBIGUOUS[a >= b ? `${a}${b}` : `${b}${a}`] ?? []);
        });
        await armed();
        hop(13, 7);
        await tick();
    }

    test('rien n’est enregistré, et le triangle n’offre que les jets possibles', async () => {
        await ambiguous();
        expect(gestures()).toEqual([]);
        await vi.waitFor(() => expect(cell('61')?.disabled).toBe(false));
        expect(cell('62').disabled).toBe(false);
        expect(cell('31').disabled).toBe(true);
        expect(cell('66').disabled).toBe(true);
    });

    test('la case cliquée dit le jet, et l’Action part avec les pas joués', async () => {
        await ambiguous();
        await vi.waitFor(() => expect(cell('62')?.disabled).toBe(false));
        await fireEvent.click(cell('62'));

        await vi.waitFor(() => expect(gestures().some((g) => g.Kind === 'validate')).toBe(true));
        const sent = gestures();
        expect(sent[0]).toEqual({ Kind: 'enter_die', Die: 6 });
        expect(sent[1]).toEqual({ Kind: 'enter_die', Die: 2 });
        expect(sent[2].Steps).toEqual([{ from: 13, to: 7, hit: false }]);
    });
});
