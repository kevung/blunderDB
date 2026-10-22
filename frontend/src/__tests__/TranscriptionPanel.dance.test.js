/**
 * TranscriptionPanel.dance.test.js — `f` : le joueur au trait fait gala.
 *
 * Le jet dansant est cherché par le générateur de coups (LegalMoves), jamais
 * par une règle du panneau : le générateur simulé ici ferme les points 5 et 6,
 * et le jet attendu est donc le premier de DANCE_ROLLS dont aucun dé n'entre.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    TranscriptionCandidates: vi.fn(),
    ListTranscriptions: vi.fn().mockResolvedValue([]),
    CreateTranscription: vi.fn(),
    OpenTranscription: vi.fn(),
    ApplyTranscriptionGesture: vi.fn()
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({
    LegalMoves: vi.fn()
}));

import { ApplyTranscriptionGesture, ListTranscriptions, TranscriptionCandidates } from '../../wailsjs/go/database/Database.js';
import { LegalMoves } from '../../wailsjs/go/gui/App.js';

import { candidatesAnswer } from './helpers/transcriptionCandidates.js';
import TranscriptionPanel from '../components/TranscriptionPanel.svelte';
import { transcriptionListStore, transcriptionStore, transcriptionKeyStore, transcriptionNoticeStore, clearTranscription } from '../stores/transcriptionStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { activeTabStore, statusBarModeStore } from '../stores/uiStore.js';
import { DANCE_ROLLS, onBar, pressKey, initialKeyState, COMMAND } from '../services/transcriptionKeys.js';

/** @param {boolean} barred - Noir, au trait, a un pion sur sa barre (point 25) */
function position(barred) {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    if (barred) points[25] = { checkers: 1, color: 0 };
    return { board: { points, bearoff: [0, 0] }, cube: { owner: -1, value: 0 }, dice: [0, 0], score: [7, 7], player_on_roll: 0, decision_type: 0 };
}

let barred = true;

function annotated() {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice' }, actions: [], cursor: 0 },
        actions: [],
        games: [],
        next: { expects: 'checker', side: 0, position: position(barred), crawford: false },
        score: [0, 0],
        cursor: 0
    };
}

const state = (/** @type {any} */ ann) => ({ id: 1, annotated: ann });

const PLAYS = [{ notation: '24/23 13/11' }];
/** Les dés qui ne rentrent pas : le jan adverse tient ses points 5 et 6. */
let closed = new Set([5, 6]);
/** @param {any} pos */
const legalMoves = (pos) => Promise.resolve(onBar(pos) && closed.has(pos.dice[0]) && closed.has(pos.dice[1]) ? [] : PLAYS);

/** @param {string} code @param {Document | Element} [target] */
function press(code, target = document) {
    const digit = /^Digit([1-9])$/.exec(code);
    const letter = /^Key([A-Z])$/.exec(code);
    return fireEvent.keyDown(target, { code, key: digit ? digit[1] : letter ? letter[1].toLowerCase() : code });
}

async function openedPanel() {
    transcriptionStore.set(state(annotated()));
    render(TranscriptionPanel);
    await tick();
    document.getElementById('transcriptionPanel')?.focus();
}

const gestures = () => /** @type {any} */ (ApplyTranscriptionGesture).mock.calls.map((/** @type {any} */ call) => call[1]);

beforeEach(() => {
    vi.clearAllMocks();
    barred = true;
    closed = new Set([5, 6]);
    /** @type {any} */ (ListTranscriptions).mockResolvedValue([]);
    /** @type {any} */ (ApplyTranscriptionGesture).mockImplementation(() => Promise.resolve(state(annotated())));
    /** @type {any} */ (LegalMoves).mockImplementation(legalMoves);
    // La liaison répond pour le jet que le panneau a saisi : un jet dont les dés sont fermés danse.
    /** @type {any} */ (TranscriptionCandidates).mockImplementation(() => {
        const dice = get(transcriptionKeyStore).dice;
        if (closed.has(dice[0]) && closed.has(dice[1])) return Promise.resolve({ side: 0, dice, unranked: false, candidates: [] });
        return Promise.resolve(candidatesAnswer(PLAYS, [{ index: 0, move: '24/23 13/11', equity: 0 }], { dice }));
    });
    transcriptionListStore.set([]);
    transcriptionNoticeStore.set(null);
    clearTranscription();
    selectedMoveStore.set(null);
    databasePathStore.set('/tmp/library.db');
    activeTabStore.set('transcription');
    statusBarModeStore.set('TRANSCRIBE');
});

afterEach(cleanup);

describe('la machine', () => {
    test('les non-doubles passent avant les doubles, chaque jet une seule fois', () => {
        expect(DANCE_ROLLS).toHaveLength(21);
        expect(DANCE_ROLLS[0]).toEqual([1, 2]);
        expect(DANCE_ROLLS[14]).toEqual([5, 6]);
        expect(DANCE_ROLLS[15]).toEqual([1, 1]);
        expect(new Set(DANCE_ROLLS.map((d) => d.join('-'))).size).toBe(21);
    });

    test('la barre est celle du camp au trait', () => {
        expect(onBar(position(true))).toBe(true);
        expect(onBar(position(false))).toBe(false);
        expect(onBar({ ...position(true), player_on_roll: 1 })).toBe(false);
    });

    test('f ne vaut que là où des dés sont attendus', () => {
        const f = /** @type {any} */ ({ key: 'f', code: 'KeyF' });
        expect(pressKey(initialKeyState(), f).commands).toEqual([{ kind: COMMAND.DANCE_ROLL }]);
        expect(pressKey(initialKeyState(), f, { expects: 'take' }).handled).toBe(false);
    });
});

describe('f dans le panneau', () => {
    test('sur la barre devant un jan qui laisse danser : le jet est saisi, la danse créée', async () => {
        await openedPanel();
        await press('KeyF');

        await vi.waitFor(() => expect(gestures()).toEqual([{ Kind: 'enter_die', Die: 6 }, { Kind: 'enter_die', Die: 5 }, { Kind: 'dance' }]));
        // C'est la liaison qui dit que le jet danse : elle ne rend aucun candidat.
        expect(TranscriptionCandidates).toHaveBeenCalledWith(1);
    });

    test('un double est pris quand un seul point est fermé', async () => {
        closed = new Set([4]);
        await openedPanel();
        await press('KeyF');

        await vi.waitFor(() => expect(gestures()).toEqual([{ Kind: 'enter_die', Die: 4 }, { Kind: 'enter_die', Die: 4 }, { Kind: 'dance' }]));
    });

    test('un jet en attente est validé, comme par un chiffre, avant la danse', async () => {
        await openedPanel();
        await press('Digit3');
        await press('Digit1');
        await vi.waitFor(() => expect(get(selectedMoveStore)).toBe('24/23 13/11'));
        /** @type {any} */ (ApplyTranscriptionGesture).mockClear();

        await press('KeyF');
        await vi.waitFor(() => expect(gestures()).toEqual([{ Kind: 'validate' }, { Kind: 'enter_die', Die: 6 }, { Kind: 'enter_die', Die: 5 }, { Kind: 'dance' }]));
    });

    test('hors de la barre : rien saisi, un message', async () => {
        barred = false;
        await openedPanel();
        await press('KeyF');

        await vi.waitFor(() => expect(get(transcriptionNoticeStore)?.key).toBe('transcription.notice.notOnBar'));
        expect(gestures()).toEqual([]);
    });

    test('tous les jets rentrent : rien saisi, un message', async () => {
        closed = new Set();
        await openedPanel();
        await press('KeyF');

        await vi.waitFor(() => expect(get(transcriptionNoticeStore)?.key).toBe('transcription.notice.noDanceRoll'));
        expect(gestures()).toEqual([]);
        expect(LegalMoves).toHaveBeenCalledTimes(21);
    });

    test('dans un champ texte, f est une lettre', async () => {
        await openedPanel();
        const input = document.createElement('input');
        document.getElementById('transcriptionPanel')?.appendChild(input);
        input.focus();
        await press('KeyF', input);
        await tick();

        expect(LegalMoves).not.toHaveBeenCalled();
        expect(gestures()).toEqual([]);
        expect(get(transcriptionNoticeStore)).toBeNull();
    });
});
