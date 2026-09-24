/**
 * TranscriptionPanel.declaredScore.test.js — le score annoncé d'une partie
 * (ADR-0053).
 *
 * Le moteur (pkg/blunderdb/transcript) pose le score, le rejoue et marque ce
 * qui diffère ; ce qui se tient ici est la couture avec le panneau : le
 * double-clic sur le score de l'en-tête d'une partie ouvre un champ pré-rempli,
 * Entrée envoie `set_score` avec le score tapé, un champ vidé l'efface, Échap
 * n'écrit rien, et une session d'argent n'offre pas de champ.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ListTranscriptions: vi.fn().mockResolvedValue([]),
    CreateTranscription: vi.fn(),
    OpenTranscription: vi.fn(),
    ApplyTranscriptionGesture: vi.fn(),
    TranscriptionMAT: vi.fn().mockResolvedValue('')
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    LegalMoves: vi.fn().mockResolvedValue([]),
    EvaluatePositionImmediate: vi.fn().mockResolvedValue({ moves: [] })
}));
vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetGammonNetPruneK: vi.fn().mockResolvedValue(0)
}));

import { ApplyTranscriptionGesture, ListTranscriptions } from '../../wailsjs/go/database/Database.js';

import TranscriptionPanel from '../components/TranscriptionPanel.svelte';
import { parseScore } from '../components/TranscriptView.svelte';
import { transcriptionListStore, transcriptionStore, clearTranscription } from '../stores/transcriptionStore.js';
import { quizPlayStore } from '../stores/quizPlayStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';

const POSITION = {
    id: 0,
    board: { points: Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 })), bearoff: [0, 0] },
    cube: { owner: -1, value: 0 },
    dice: [0, 0],
    score: [7, 7],
    player_on_roll: 0,
    decision_type: 0
};

/**
 * Deux parties gagnées sur un refus, la seconde jouée à 3–0 quand la première
 * donnait 1–0 : le score annoncé diffère du score dérivé.
 *
 * @param {number} length
 */
function twoGames(length = 7) {
    const game = [
        { side: 0, kind: 'opening', dice: [3, 1] },
        { side: 0, kind: 'double' },
        { side: 1, kind: 'pass' }
    ];
    const actions = [...game, { ...game[0], score: [3, 0] }, game[1], game[2]];
    const declared = length > 0;
    return {
        document: { header: { match_length: length, player1: 'Kévin', player2: 'Alice' }, actions, cursor: actions.length },
        actions: actions.map((a, index) => ({
            index,
            side: a.side,
            kind: a.kind,
            before: POSITION,
            has_position: a.kind !== 'opening',
            game_index: index < 3 ? 0 : 1,
            game_number: index < 3 ? 1 : 2,
            score: index < 3 ? [0, 0] : [3, 0],
            move_number: a.kind === 'opening' ? -1 : 0,
            inconsistencies: index === 3 && declared ? [{ kind: 'score_mismatch', detail: '' }] : []
        })),
        games: [
            { number: 1, initial_score: [0, 0], declared: false, derived_score: [0, 0], winner: 0, points_won: 1, crawford: false, finished: true, first: 0, last: 2 },
            {
                number: 2,
                initial_score: declared ? [3, 0] : [1, 0],
                declared,
                derived_score: [1, 0],
                winner: 0,
                points_won: 1,
                crawford: false,
                finished: true,
                first: 3,
                last: 5
            }
        ],
        next: { expects: 'opening', side: 0, position: POSITION, crawford: false },
        score: [4, 0],
        cursor: actions.length
    };
}

const state = (/** @type {any} */ ann) => ({ id: 1, annotated: ann });
const gestures = () => /** @type {any} */ (ApplyTranscriptionGesture).mock.calls.map((/** @type {any} */ call) => call[1]);
const field = () => /** @type {HTMLInputElement | null} */ (document.querySelector('.transcript-view input.score-field'));
const scoreOf = (/** @type {number} */ n) => /** @type {HTMLElement} */ (document.querySelectorAll('.transcript-view .game-score')[n - 1]);

function press(/** @type {string} */ key, /** @type {Element} */ target) {
    return fireEvent.keyDown(target, { key, code: key });
}

async function settle(times = 10) {
    for (let i = 0; i < times; i++) {
        await tick();
        await Promise.resolve();
    }
}

async function openedOn(/** @type {any} */ ann) {
    transcriptionStore.set(state(ann));
    render(TranscriptionPanel);
    await tick();
    document.getElementById('transcriptionPanel')?.focus();
}

/** Le double-clic tel que le navigateur le livre : deux clics, puis dblclick. */
async function doubleClick(/** @type {Element} */ el) {
    await fireEvent.click(el);
    await fireEvent.click(el);
    await fireEvent.dblClick(el);
    await tick();
}

beforeEach(() => {
    vi.clearAllMocks();
    /** @type {any} */ (ListTranscriptions).mockResolvedValue([]);
    /** @type {any} */ (ApplyTranscriptionGesture).mockImplementation(() => Promise.resolve(state(twoGames())));
    transcriptionListStore.set([]);
    clearTranscription();
    quizPlayStore.set(null);
    databasePathStore.set('/tmp/library.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
});

afterEach(() => {
    cleanup();
    quizPlayStore.set(null);
});

describe('parseScore', () => {
    test('lit « 3-2 », « 3–2 » et « 3 2 »', () => {
        expect(parseScore('3-2')).toEqual([3, 2]);
        expect(parseScore(' 3–2 ')).toEqual([3, 2]);
        expect(parseScore('3 2')).toEqual([3, 2]);
        expect(parseScore('10 - 0')).toEqual([10, 0]);
    });
    test('un champ vide efface, un texte sans score ne dit rien', () => {
        expect(parseScore('')).toBeNull();
        expect(parseScore('   ')).toBeNull();
        expect(parseScore('3')).toBeUndefined();
        expect(parseScore('a-b')).toBeUndefined();
        expect(parseScore('-1-2')).toBeUndefined();
    });
});

describe('le score annoncé d’une partie (double-clic sur son score)', () => {
    test('le score qui diffère du score dérivé est marqué', async () => {
        await openedOn(twoGames());
        const second = scoreOf(2);
        expect(second.textContent).toContain('3–0');
        expect(second.classList.contains('flawed')).toBe(true);
        expect(second.title).toContain('1–0');
        expect(scoreOf(1).classList.contains('flawed')).toBe(false);
    });

    test('le double-clic ouvre un champ pré-rempli du score affiché', async () => {
        await openedOn(twoGames());
        const details = /** @type {HTMLDetailsElement} */ (scoreOf(2).closest('details'));
        const open = details.open;
        await doubleClick(scoreOf(2));
        await vi.waitFor(() => expect(field()).not.toBeNull());
        expect(field()?.value).toBe('3-0');
        expect(document.activeElement).toBe(field());
        // Le score qui se tape ne plie ni ne déplie la partie.
        expect(details.open).toBe(open);
    });

    test('un clic sur le score qui se tape ne plie pas la partie', async () => {
        await openedOn(twoGames());
        const details = /** @type {HTMLDetailsElement} */ (scoreOf(1).closest('details'));
        const open = details.open;
        await fireEvent.click(scoreOf(1));
        await tick();
        expect(details.open).toBe(open);
    });

    test('Entrée envoie set_score sur l’ouverture de la partie', async () => {
        await openedOn(twoGames());
        await doubleClick(scoreOf(2));
        await vi.waitFor(() => expect(field()).not.toBeNull());
        const input = /** @type {HTMLInputElement} */ (field());
        await fireEvent.input(input, { target: { value: '2–1' } });
        await press('Enter', input);
        await vi.waitFor(() => expect(gestures().some((/** @type {any} */ g) => g.Kind === 'set_score')).toBe(true));
        expect(gestures().filter((/** @type {any} */ g) => g.Kind === 'set_score')).toEqual([{ Kind: 'set_score', At: 3, Score: [2, 1] }]);
        expect(field()).toBeNull();
    });

    test('un champ vidé efface le score annoncé', async () => {
        await openedOn(twoGames());
        await doubleClick(scoreOf(2));
        await vi.waitFor(() => expect(field()).not.toBeNull());
        const input = /** @type {HTMLInputElement} */ (field());
        await fireEvent.input(input, { target: { value: '' } });
        await press('Enter', input);
        await vi.waitFor(() => expect(gestures().some((/** @type {any} */ g) => g.Kind === 'set_score')).toBe(true));
        expect(gestures().find((/** @type {any} */ g) => g.Kind === 'set_score')).toEqual({ Kind: 'set_score', At: 3, Score: null });
    });

    test('Échap ferme le champ et n’écrit rien', async () => {
        await openedOn(twoGames());
        await doubleClick(scoreOf(2));
        await vi.waitFor(() => expect(field()).not.toBeNull());
        await settle();
        const before = gestures().length;
        await press('Escape', /** @type {HTMLInputElement} */ (field()));
        await tick();
        expect(field()).toBeNull();
        await settle();
        expect(gestures()).toHaveLength(before);
    });

    test('un texte qui ne dit aucun score laisse le champ ouvert', async () => {
        await openedOn(twoGames());
        await doubleClick(scoreOf(2));
        await vi.waitFor(() => expect(field()).not.toBeNull());
        const input = /** @type {HTMLInputElement} */ (field());
        await fireEvent.input(input, { target: { value: 'trois' } });
        await press('Enter', input);
        await settle();
        expect(field()).not.toBeNull();
        expect(gestures().some((/** @type {any} */ g) => g.Kind === 'set_score')).toBe(false);
    });

    test('les touches tapées dans le champ n’atteignent pas la machine à touches', async () => {
        await openedOn(twoGames());
        await doubleClick(scoreOf(2));
        await vi.waitFor(() => expect(field()).not.toBeNull());
        await settle();
        const before = gestures().length;
        const input = /** @type {HTMLInputElement} */ (field());
        await fireEvent.keyDown(input, { key: 'x', code: 'KeyX' });
        await fireEvent.keyDown(input, { key: '3', code: 'Digit3' });
        await settle();
        expect(gestures()).toHaveLength(before);
        expect(field()).not.toBeNull();
    });

    test('la perte du focus ferme le champ sans rien écrire', async () => {
        await openedOn(twoGames());
        await doubleClick(scoreOf(2));
        await vi.waitFor(() => expect(field()).not.toBeNull());
        await settle();
        const before = gestures().length;
        await fireEvent.blur(/** @type {HTMLInputElement} */ (field()));
        await tick();
        expect(field()).toBeNull();
        await settle();
        expect(gestures()).toHaveLength(before);
    });

    test('une session d’argent n’a pas de score à taper', async () => {
        await openedOn(twoGames(0));
        await doubleClick(scoreOf(2));
        await settle();
        expect(field()).toBeNull();
        expect(scoreOf(2).classList.contains('editable')).toBe(false);
    });
});
