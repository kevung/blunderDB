/**
 * TranscriptionPanel.directPlay.test.js — le coup joué directement (ADR-0052).
 *
 * Le réducteur est éprouvé à part (transcriptionPlay.test.js) et le glissé qui
 * le fait avancer aussi (boardInteractions.test.js). Ce qui ne se voit qu'ici
 * est la couture avec le panneau :
 *
 *  - le jet saisi arme le plateau des coups de CE jet, en bout de document
 *    comme sur une Action relue ;
 *  - chaque pas joué réduit la liste des candidats, et le premier restant est
 *    présélectionné, par son rang chez le générateur ;
 *  - un coup légal achevé part seul — et sur une Action relue, il la remplace ;
 *  - un glissé hors des règles vide la liste, le dit en une ligne, et c'est
 *    Entrée qui l'enregistre avec le plateau obtenu ;
 *  - le double-clic sur une cellule du Transcript y ouvre un champ : Entrée
 *    écrit le coup tapé, Échap n'écrit rien.
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
import { dragStep } from '../services/transcriptionPlay.js';

const BLACK = 0;
const WHITE = 1;

function positionWith(/** @type {any} */ stacks, mover = BLACK, dice = [0, 0]) {
    const points = Array.from({ length: 26 }, () => ({ checkers: 0, color: -1 }));
    for (const [point, [checkers, color]] of Object.entries(stacks)) points[Number(point)] = { checkers, color };
    return {
        id: 0,
        board: { points, bearoff: [0, 0] },
        cube: { owner: -1, value: 0 },
        dice,
        score: [7, 7],
        player_on_roll: mover,
        decision_type: 0
    };
}

const POSITION = positionWith({ 13: [5, BLACK], 8: [3, BLACK], 6: [5, BLACK], 12: [2, WHITE], 1: [2, WHITE] });

const step = (/** @type {number} */ from, /** @type {number} */ to) => ({ from, to, hit: false });

// Les coups de 6-1, dans l'ordre du GÉNÉRATEUR ; le classement 0-ply les rend
// dans un autre ordre, exprès : le geste envoyé au moteur porte le rang du
// générateur, l'écran celui du classement.
const PLAYS_61 = [
    { notation: '13/7 8/7', steps: [step(13, 7), step(8, 7)] },
    { notation: '13/7 13/12', steps: [step(13, 7), step(13, 12)] },
    { notation: '8/2 6/5', steps: [step(8, 2), step(6, 5)] }
];
const RANKED_61 = [
    { index: 0, move: '8/2 6/5', equity: 0.12 },
    { index: 1, move: '13/7 13/12', equity: 0.04, equityError: 0.08 },
    { index: 2, move: '13/7 8/7', equity: -0.1, equityError: 0.22 }
];

/** Un brouillon en bout de document, rien de relu. */
function atEnd() {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice' }, actions: [], cursor: 0 },
        actions: [],
        games: [],
        next: { expects: 'checker', side: BLACK, position: POSITION, crawford: false },
        score: [0, 0],
        cursor: 0
    };
}

// Un document d'une Action : 61: 13/7 8/7 par le joueur 1. Le Cursor en bout
// (`cursor: 1`, rien sous lui) ou sur elle (`cursor: 0`, l'Entry chargée de
// ses dés et de son coup, comme `loadEntry` la pose).
const RECORDED = { side: BLACK, kind: 'checker', dice: [6, 1] };
function withOneAction(/** @type {number} */ cursor, /** @type {any} */ entry = undefined) {
    return {
        document: { header: { match_length: 7, player1: 'Kévin', player2: 'Alice' }, actions: [RECORDED], cursor },
        actions: [
            {
                index: 0,
                side: BLACK,
                kind: 'checker',
                before: positionWith({ 13: [5, BLACK], 8: [3, BLACK], 6: [5, BLACK], 12: [2, WHITE], 1: [2, WHITE] }, BLACK, [6, 1]),
                has_position: true,
                notation: '13/7 8/7',
                game_index: 0,
                game_number: 1,
                score: [0, 0],
                move_number: 1,
                inconsistencies: []
            }
        ],
        games: [{ number: 1, initial_score: [0, 0], crawford: false, finished: false, first: 0, last: 0 }],
        next: { expects: 'checker', side: WHITE, position: POSITION, crawford: false },
        entry: entry !== undefined ? entry : cursor === 0 ? { at: 0, replacing: true, side: BLACK, dice: [6, 1], selected: true, review: false, kind: 'checker', notation: '13/7 8/7' } : null,
        score: [0, 0],
        cursor
    };
}

const state = (/** @type {any} */ ann) => ({ id: 1, annotated: ann });
const gestures = () => /** @type {any} */ (ApplyTranscriptionGesture).mock.calls.map((/** @type {any} */ call) => call[1]);
const selects = () => gestures().filter((/** @type {any} */ g) => g.Kind === 'select_candidate');
const rows = () => [...document.querySelectorAll('.candidates tbody tr')].map((tr) => tr.textContent);

function press(/** @type {string} */ code, target = /** @type {any} */ (document)) {
    const digit = /^Digit([1-9])$/.exec(code);
    const letter = /^Key([A-Z])$/.exec(code);
    return fireEvent.keyDown(target, { code, key: digit ? digit[1] : letter ? letter[1].toLowerCase() : code });
}

async function settle(times = 10) {
    for (let i = 0; i < times; i++) {
        await tick();
        await Promise.resolve();
    }
}

/** Le coup joué au plateau, un glissé par pas. */
function drag(/** @type {number} */ from, /** @type {number} */ to) {
    quizPlayStore.update((/** @type {any} */ s) => dragStep(s, from, to));
}

/** Le brouillon en bout de document, le jet 6-1 tapé (dans l'ordre donné). */
async function rolled(first = 'Digit6', second = 'Digit1') {
    transcriptionStore.set(state(atEnd()));
    render(TranscriptionPanel);
    await tick();
    document.getElementById('transcriptionPanel')?.focus();
    await press(first);
    await press(second);
    await vi.waitFor(() => expect(rows()).toHaveLength(3));
    await vi.waitFor(() => expect(/** @type {any} */ (get(quizPlayStore))?.rolled).toBeTruthy());
}

/** Les quatre gestes d'une Action posée par ses pas, ceux qui précèdent exclus. */
function recorded() {
    const sent = gestures();
    const at = sent.findIndex((/** @type {any} */ g) => g.Kind === 'enter_play');
    expect(at, 'aucun enter_play envoyé').toBeGreaterThanOrEqual(2);
    return sent.slice(at - 2, at + 2);
}

beforeEach(() => {
    vi.clearAllMocks();
    /** @type {any} */ (ListTranscriptions).mockResolvedValue([]);
    /** @type {any} */ (ApplyTranscriptionGesture).mockImplementation(() => Promise.resolve(state(atEnd())));
    /** @type {any} */ (LegalMoves).mockImplementation((/** @type {any} */ pos) => {
        const [a, b] = pos.dice;
        return Promise.resolve((a === 6 && b === 1) || (a === 1 && b === 6) ? PLAYS_61 : []);
    });
    /** @type {any} */ (EvaluatePositionImmediate).mockResolvedValue({ moves: RANKED_61 });
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

describe('le jet saisi, le plateau joue ses coups', () => {
    test('le plateau est armé des coups de CE jet, et de lui seul', async () => {
        await rolled();
        const play = /** @type {any} */ (get(quizPlayStore));
        expect(play.rolled).toEqual([6, 1]);
        expect(play.plays).toHaveLength(3);
        expect(play.free).toBe(false);
    });

    test('chaque pas réduit la liste aux coups qui le contiennent, dans l’ordre du classement', async () => {
        await rolled();
        drag(13, 7);
        await tick();
        const shown = rows();
        expect(shown).toHaveLength(2);
        expect(shown[0]).toContain('13/7 13/12');
        expect(shown[1]).toContain('13/7 8/7');
    });

    // La jointure : le premier de la liste réduite est le SECOND du générateur,
    // et c'est ce rang-là que le geste doit porter.
    test('le premier candidat restant est présélectionné, par son rang chez le générateur', async () => {
        await rolled();
        drag(13, 7);
        await vi.waitFor(() => expect(selects().at(-1)).toEqual({ Kind: 'select_candidate', Candidate: 1 }));
        expect(get(selectedMoveStore)).toBe('13/7 13/12');
    });

    test('j se déplace DANS la liste réduite', async () => {
        await rolled();
        drag(13, 7);
        await vi.waitFor(() => expect(selects().at(-1)).toEqual({ Kind: 'select_candidate', Candidate: 1 }));
        await press('KeyJ');
        await vi.waitFor(() => expect(selects().at(-1)).toEqual({ Kind: 'select_candidate', Candidate: 0 }));
    });

    test('un pas défait rend la liste entière', async () => {
        await rolled();
        drag(13, 7);
        await tick();
        expect(rows()).toHaveLength(2);
        await press('Backspace');
        await vi.waitFor(() => expect(rows()).toHaveLength(3));
    });

    test('un coup légal achevé part seul, avec les dés dans l’ordre tapé', async () => {
        await rolled('Digit1', 'Digit6');
        drag(13, 7);
        drag(8, 7);
        await vi.waitFor(() => expect(gestures().some((/** @type {any} */ g) => g.Kind === 'validate')).toBe(true));
        const [die1, die2, play, validate] = recorded();
        expect(die1).toEqual({ Kind: 'enter_die', Die: 1 });
        expect(die2).toEqual({ Kind: 'enter_die', Die: 6 });
        expect(play.Steps).toEqual([step(13, 7), step(8, 7)]);
        // Un coup LÉGAL ne porte pas de plateau : il n'a rien d'illégal à dire.
        expect(play.BoardAfter).toBeNull();
        expect(validate).toEqual({ Kind: 'validate' });
    });

    test('un coup à moitié joué n’enregistre rien', async () => {
        await rolled();
        drag(13, 7);
        await settle();
        expect(gestures().some((/** @type {any} */ g) => g.Kind === 'enter_play')).toBe(false);
    });
});

describe('le glissé hors des règles, puis Entrée', () => {
    test('la liste cède la place à une ligne, et rien ne part tout seul', async () => {
        await rolled();
        drag(13, 3);
        await tick();
        expect(/** @type {any} */ (get(quizPlayStore)).free).toBe(true);
        expect(rows()).toHaveLength(0);
        expect(document.querySelector('[data-testid="transcription-free-hint"]')?.textContent).toContain('Enter');
        await settle();
        expect(gestures().some((/** @type {any} */ g) => g.Kind === 'enter_play')).toBe(false);
    });

    test('Entrée enregistre les dés saisis, les pas et le plateau obtenu', async () => {
        await rolled();
        drag(13, 3);
        drag(6, 5);
        await tick();
        await press('Enter');
        await vi.waitFor(() => expect(gestures().some((/** @type {any} */ g) => g.Kind === 'validate')).toBe(true));
        const [die1, die2, play, validate] = recorded();
        expect(die1).toEqual({ Kind: 'enter_die', Die: 6 });
        expect(die2).toEqual({ Kind: 'enter_die', Die: 1 });
        expect(play.Steps).toEqual([step(13, 3), step(6, 5)]);
        // Le plateau part avec les pas : c'est lui qui dit ce qui s'est passé.
        expect(play.BoardAfter.points[3]).toEqual({ checkers: 1, color: BLACK });
        expect(play.BoardAfter.points[13].checkers).toBe(4);
        expect(validate).toEqual({ Kind: 'validate' });
    });

    // Le chiffre du jet suivant porte la validation du tour : il enregistre le
    // coup libre d'abord, puis commence le jet — jamais le candidat que le
    // moteur tenait sélectionné.
    test('le chiffre du jet suivant enregistre le coup libre, puis commence le jet', async () => {
        await rolled();
        drag(13, 3);
        await tick();
        await press('Digit4');
        await vi.waitFor(() => expect(gestures().filter((/** @type {any} */ g) => g.Kind === 'enter_die' && g.Die === 4)).toHaveLength(1));
        const kinds = gestures().map((/** @type {any} */ g) => g.Kind);
        const play = gestures().find((/** @type {any} */ g) => g.Kind === 'enter_play');
        expect(play.Steps).toEqual([step(13, 3)]);
        expect(play.BoardAfter).toBeTruthy();
        // Dans l'ordre : le coup libre validé, puis le dé du jet d'après.
        expect(kinds.lastIndexOf('validate')).toBeLessThan(kinds.lastIndexOf('enter_die'));
    });

    test('Retour arrière défait le pas hors des règles et rend la liste', async () => {
        await rolled();
        drag(13, 3);
        await tick();
        expect(rows()).toHaveLength(0);
        await press('Backspace');
        await vi.waitFor(() => expect(rows()).toHaveLength(3));
        expect(/** @type {any} */ (get(quizPlayStore)).free).toBe(false);
        expect(document.querySelector('[data-testid="transcription-free-hint"]')).toBeNull();
    });

    test('sans jet saisi, le glissé reste contraint', async () => {
        transcriptionStore.set(state(atEnd()));
        render(TranscriptionPanel);
        await tick();
        await vi.waitFor(() => expect(get(quizPlayStore)).not.toBeNull());
        drag(13, 3);
        await tick();
        const play = /** @type {any} */ (get(quizPlayStore));
        expect(play.free).toBe(false);
        expect(play.steps).toEqual([]);
    });
});

describe('sur une Action relue', () => {
    /** Le brouillon, le Cursor mené sur la cellule par un clic. */
    async function onRecorded() {
        /** @type {any} */ (ApplyTranscriptionGesture).mockImplementation((/** @type {any} */ _id, /** @type {any} */ gesture) =>
            Promise.resolve(state(gesture.Kind === 'cursor_back' ? withOneAction(0) : withOneAction(1)))
        );
        transcriptionStore.set(state(withOneAction(1)));
        render(TranscriptionPanel);
        await tick();
        document.getElementById('transcriptionPanel')?.focus();
        const cell = /** @type {HTMLElement} */ (document.querySelector('.transcript-view .cell[data-index="0"]'));
        await fireEvent.click(cell);
        await vi.waitFor(() => expect(/** @type {any} */ (get(quizPlayStore))?.rolled).toEqual([6, 1]));
        return cell;
    }

    test('ses dés arment le plateau, et un coup légal joué la REMPLACE', async () => {
        await onRecorded();
        drag(13, 7);
        drag(13, 12);
        await vi.waitFor(() => expect(gestures().some((/** @type {any} */ g) => g.Kind === 'validate')).toBe(true));
        const sent = gestures();
        // Le Cursor sur la cellule, puis l'Action retapée à sa place.
        expect(sent[0]).toEqual({ Kind: 'cursor_back' });
        const [die1, die2, play, validate] = recorded();
        expect(die1).toEqual({ Kind: 'enter_die', Die: 6 });
        expect(die2).toEqual({ Kind: 'enter_die', Die: 1 });
        expect(play.Steps).toEqual([step(13, 7), step(13, 12)]);
        expect(validate).toEqual({ Kind: 'validate' });
    });
});

describe('le coup tapé dans sa cellule (double-clic)', () => {
    const field = () => /** @type {HTMLInputElement | null} */ (document.querySelector('.transcript-view input.move-field'));

    async function openedOn(/** @type {any} */ ann) {
        transcriptionStore.set(state(ann));
        render(TranscriptionPanel);
        await tick();
        document.getElementById('transcriptionPanel')?.focus();
    }

    /** Le double-clic tel que le navigateur le livre : deux clics, puis dblclick. */
    async function doubleClick(/** @type {Element} */ cell) {
        await fireEvent.click(cell);
        await fireEvent.click(cell);
        await fireEvent.dblClick(cell);
        await tick();
    }

    beforeEach(() => {
        /** @type {any} */ (ApplyTranscriptionGesture).mockImplementation((/** @type {any} */ _id, /** @type {any} */ gesture) =>
            Promise.resolve(state(gesture.Kind === 'cursor_back' ? withOneAction(0) : withOneAction(1)))
        );
    });

    test('la cellule devient un champ, pré-rempli de sa notation', async () => {
        await openedOn(withOneAction(1));
        await doubleClick(/** @type {Element} */ (document.querySelector('.transcript-view .cell[data-index="0"]')));
        await vi.waitFor(() => expect(field()).not.toBeNull());
        expect(field()?.value).toBe('13/7 8/7');
        expect(document.activeElement).toBe(field());
    });

    test('Entrée écrit le coup tapé sur cette Action, dés de la cellule, même illégal', async () => {
        await openedOn(withOneAction(1));
        await doubleClick(/** @type {Element} */ (document.querySelector('.transcript-view .cell[data-index="0"]')));
        await vi.waitFor(() => expect(field()).not.toBeNull());
        await settle();
        const input = /** @type {HTMLInputElement} */ (field());
        await fireEvent.input(input, { target: { value: '13/3' } });
        await press('Enter', input);

        await vi.waitFor(() => expect(gestures().some((/** @type {any} */ g) => g.Kind === 'validate')).toBe(true));
        const sent = gestures();
        // Le Cursor y est allé une fois, pas deux : le second clic ne relance
        // pas le chemin.
        expect(sent.filter((/** @type {any} */ g) => g.Kind === 'cursor_back')).toHaveLength(1);
        const [die1, die2, play, validate] = recorded();
        expect(die1).toEqual({ Kind: 'enter_die', Die: 6 });
        expect(die2).toEqual({ Kind: 'enter_die', Die: 1 });
        expect(play.Steps).toEqual([step(13, 3)]);
        expect(play.BoardAfter.points[3]).toEqual({ checkers: 1, color: BLACK });
        expect(validate).toEqual({ Kind: 'validate' });
        expect(field()).toBeNull();
    });

    test('Échap ferme le champ et n’écrit rien', async () => {
        await openedOn(withOneAction(1));
        await doubleClick(/** @type {Element} */ (document.querySelector('.transcript-view .cell[data-index="0"]')));
        await vi.waitFor(() => expect(field()).not.toBeNull());
        await settle();
        const before = gestures().length;
        await press('Escape', /** @type {HTMLInputElement} */ (field()));
        await tick();
        expect(field()).toBeNull();
        await settle();
        expect(gestures()).toHaveLength(before);
    });

    test('un texte qui ne dit aucun coup laisse le champ ouvert', async () => {
        await openedOn(withOneAction(1));
        await doubleClick(/** @type {Element} */ (document.querySelector('.transcript-view .cell[data-index="0"]')));
        await vi.waitFor(() => expect(field()).not.toBeNull());
        await settle();
        const input = /** @type {HTMLInputElement} */ (field());
        await fireEvent.input(input, { target: { value: 'n’importe quoi' } });
        await press('Enter', input);
        await settle();
        expect(field()).not.toBeNull();
        expect(gestures().some((/** @type {any} */ g) => g.Kind === 'enter_play')).toBe(false);
    });

    test('les lettres tapées dans le champ n’atteignent pas la machine à touches', async () => {
        await openedOn(withOneAction(1));
        await doubleClick(/** @type {Element} */ (document.querySelector('.transcript-view .cell[data-index="0"]')));
        await vi.waitFor(() => expect(field()).not.toBeNull());
        await settle();
        const before = gestures().length;
        // `x` supprimerait l'Action, un chiffre recommencerait le jet.
        await press('KeyX', /** @type {HTMLInputElement} */ (field()));
        await press('Digit3', /** @type {HTMLInputElement} */ (field()));
        await settle();
        expect(gestures()).toHaveLength(before);
    });

    // La saisie en cours, dés tapés : la cellule en pointillés se tape de même.
    // C'est ce qui remplace le champ de notation du volet ✎.
    test('la cellule en pointillés d’une Action neuve se tape aussi, dés saisis', async () => {
        const pending = withOneAction(1, { at: 1, replacing: false, side: WHITE, dice: [5, 2], selected: false, review: false, kind: 'checker', notation: '' });
        /** @type {any} */ (ApplyTranscriptionGesture).mockImplementation(() => Promise.resolve(state(pending)));
        await openedOn(pending);
        const cell = /** @type {Element} */ (document.querySelector('.transcript-view .cell[data-pending="true"]'));
        expect(cell).not.toBeNull();
        await fireEvent.dblClick(cell);
        await vi.waitFor(() => expect(field()).not.toBeNull());
        expect(field()?.value).toBe('');
        const input = /** @type {HTMLInputElement} */ (field());
        await fireEvent.input(input, { target: { value: '13/8 13/11' } });
        await press('Enter', input);

        await vi.waitFor(() => expect(gestures().some((/** @type {any} */ g) => g.Kind === 'enter_play')).toBe(true));
        const sent = gestures();
        expect(sent.some((/** @type {any} */ g) => g.Kind === 'cursor_back' || g.Kind === 'cursor_forward')).toBe(false);
        const [die1, die2, play] = recorded();
        expect(die1).toEqual({ Kind: 'enter_die', Die: 5 });
        expect(die2).toEqual({ Kind: 'enter_die', Die: 2 });
        // Le joueur 2 : sa notation se lit dans son repère, 25 − p.
        expect(play.Steps).toEqual([step(12, 17), step(12, 14)]);
    });

    test('sans ses deux dés, la cellule en pointillés ne s’ouvre pas', async () => {
        const pending = withOneAction(1, { at: 1, replacing: false, side: WHITE, dice: [5, 0], selected: false, review: false, kind: 'checker', notation: '' });
        await openedOn(pending);
        const cell = /** @type {Element} */ (document.querySelector('.transcript-view .cell[data-pending="true"]'));
        await fireEvent.dblClick(cell);
        await tick();
        expect(field()).toBeNull();
    });
});
