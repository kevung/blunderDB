/**
 * Les gestes du plateau d'un Duel jusqu'à l'Arbitre : le lancer, le double confirmé au plateau,
 * la prise, l'inversion et l'annulation du coup. Et la confirmation posée sur le plateau, montée.
 */
import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest';
import { get } from 'svelte/store';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    CreateDuel: vi.fn(),
    OpenDuel: vi.fn(),
    SuspendDuel: vi.fn(),
    PlayDuel: vi.fn(() => Promise.resolve(null)),
    FlagDuel: vi.fn(),
    StopDuel: vi.fn(),
    ListDuels: vi.fn(() => Promise.resolve([])),
    DuelOffer: vi.fn()
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({ LegalMoves: vi.fn(), StartGammonNetMatchBatch: vi.fn() }));
vi.mock('../../wailsjs/go/main/Config.js', () => ({ GetGammonNetAnalysisPly: vi.fn(), GetGammonNetPruneK: vi.fn(), GetDuelForm: vi.fn(), SaveDuelForm: vi.fn() }));

import { PlayDuel } from '../../wailsjs/go/database/Database.js';
import { duelStore, duelBoardStore, duelAnimatingStore } from '../stores/duelStore.js';
import { quizPlayStore } from '../stores/quizPlayStore.js';
import { duelBoardPress, duelBoardContextMenu, duelBoardDrop, duelKeyGuard, confirmDouble, cancelDouble } from '../services/duelService.js';
import { newPlay } from '../services/quizPlay.js';
import { scoreStart } from '../services/duel.js';
import DuelBoardPrompt from '../components/DuelBoardPrompt.svelte';

const opening = { ...scoreStart(7, [7, 7]), dice: [3, 1], decision_type: 0 };
/** @type {any[]} */
const plays = [
    {
        steps: [
            { from: 8, to: 5 },
            { from: 6, to: 5 }
        ]
    },
    {
        steps: [
            { from: 13, to: 10 },
            { from: 10, to: 9 }
        ]
    }
];

/** @param {any} awaiting */
function open(awaiting) {
    duelStore.set({
        state: { id: 4, revision: 7, sides: [{ kind: 'external' }, { kind: 'bot' }], awaiting },
        sheet: { actions: [] }
    });
}

beforeEach(() => {
    duelBoardStore.set({ swapped: false, prompt: null });
    duelAnimatingStore.set(false);
    quizPlayStore.set(null);
});

afterEach(() => {
    cleanup();
    vi.clearAllMocks();
    duelStore.set(null);
});

describe('the board’s gestures reach the Arbiter', () => {
    test('a click on the dice rolls', async () => {
        open({ side: 0, kind: 'cube', position: opening });
        expect(duelBoardPress({ kind: 'die', index: 0 })).toBe(true);
        await tick();
        expect(PlayDuel).toHaveBeenCalledWith(4, 7, { side: 0, kind: 'roll' });
    });

    test('a click anywhere on the board rolls, the cube still proposes the double', async () => {
        open({ side: 0, kind: 'cube', position: opening });
        duelBoardPress({ kind: 'none' });
        await tick();
        expect(PlayDuel).toHaveBeenCalledWith(4, 7, { side: 0, kind: 'roll' });
        vi.clearAllMocks();
        duelBoardPress({ kind: 'cube' });
        expect(PlayDuel).not.toHaveBeenCalled();
        expect(get(duelBoardStore).prompt).toBe('double');
    });

    test('the cube asks first; the double leaves only once confirmed, and cancel sends nothing', async () => {
        open({ side: 0, kind: 'cube', position: opening });
        duelBoardPress({ kind: 'cube' });
        expect(get(duelBoardStore).prompt).toBe('double');
        expect(PlayDuel).not.toHaveBeenCalled();
        cancelDouble();
        expect(get(duelBoardStore).prompt).toBeNull();
        duelBoardPress({ kind: 'cube' });
        confirmDouble();
        await tick();
        expect(PlayDuel).toHaveBeenCalledWith(4, 7, { side: 0, kind: 'double' });
    });

    test('checkers move on the board, a right click takes them back, then swaps the dice', () => {
        open({ side: 0, kind: 'move', position: opening });
        quizPlayStore.set(newPlay(opening, plays));
        duelBoardPress({ kind: 'point', point: 8 });
        expect(/** @type {any} */ (get(quizPlayStore)).steps).toHaveLength(1);
        expect(duelBoardContextMenu({ kind: 'point', point: 13 })).toBe(false);
        expect(/** @type {any} */ (get(quizPlayStore)).steps).toHaveLength(0);
        expect(duelBoardContextMenu({ kind: 'point', point: 13 })).toBe(false);
        expect(get(duelBoardStore).swapped).toBe(true);
        // Outside the frame: the menu.
        expect(duelBoardContextMenu({ kind: 'outside' })).toBe(true);
    });

    test('a drag plays one legal step, and nothing else', () => {
        open({ side: 0, kind: 'move', position: opening });
        quizPlayStore.set(newPlay(opening, plays));
        duelBoardDrop(8, 2);
        expect(/** @type {any} */ (get(quizPlayStore)).steps).toHaveLength(0);
        duelBoardDrop(13, 10);
        expect(/** @type {any} */ (get(quizPlayStore)).steps).toEqual([{ from: 13, to: 10 }]);
    });
});

describe('Space during a Duel', () => {
    const space = (/** @type {any} */ target) => {
        const event = new KeyboardEvent('keydown', { code: 'Space', key: ' ', cancelable: true });
        Object.defineProperty(event, 'target', { value: target });
        return event;
    };
    const finished = () => {
        open({ side: 0, kind: 'move', position: opening });
        quizPlayStore.set(newPlay(opening, plays));
        quizPlayStore.set(/** @type {any} */ ({ ...get(quizPlayStore), steps: plays[0].steps }));
    };

    test('validates a finished move and stops the page from scrolling', async () => {
        finished();
        const event = space(document.body);
        expect(duelKeyGuard(event)).toBe(true);
        expect(event.defaultPrevented).toBe(true);
        await tick();
        expect(PlayDuel).toHaveBeenCalledWith(4, 7, expect.objectContaining({ side: 0, kind: 'move' }));
    });

    test('does nothing on a partial move, but still does not press a focused button', async () => {
        open({ side: 0, kind: 'move', position: opening });
        quizPlayStore.set(newPlay(opening, plays));
        const button = document.createElement('button');
        const event = space(button);
        expect(duelKeyGuard(event)).toBe(true);
        expect(event.defaultPrevented).toBe(true);
        await tick();
        expect(PlayDuel).not.toHaveBeenCalled();
    });

    test('keeps its meaning in a field', () => {
        finished();
        const event = space(document.createElement('input'));
        expect(duelKeyGuard(event)).toBe(true);
        expect(event.defaultPrevented).toBe(false);
        expect(PlayDuel).not.toHaveBeenCalled();
    });
});

describe('DuelBoardPrompt', () => {
    test('take or pass on the board when the Bot doubles', async () => {
        open({ side: 0, kind: 'answer', position: opening });
        const { getByTestId } = render(DuelBoardPrompt);
        await tick();
        await fireEvent.click(getByTestId('duel-take'));
        expect(PlayDuel).toHaveBeenCalledWith(4, 7, { side: 0, kind: 'take' });
    });

    test('a cube decision is set at the anchor, the validation keeps its corner', async () => {
        open({ side: 0, kind: 'answer', position: opening });
        const { getByTestId, component } = render(DuelBoardPrompt, { props: { anchor: { dx: 120, dy: -30 } } });
        await tick();
        const box = getByTestId('duel-board-prompt');
        expect(box.classList.contains('centred')).toBe(true);
        expect(box.getAttribute('style')).toContain('calc(50% + 120px)');
        expect(box.getAttribute('style')).toContain('30px');
        expect(component).toBeDefined();
        cleanup();
        open({ side: 0, kind: 'move', position: opening });
        quizPlayStore.set(newPlay(opening, plays));
        const second = render(DuelBoardPrompt, { props: { anchor: { dx: 120, dy: -30 } } });
        duelBoardPress({ kind: 'point', point: 8 });
        duelBoardPress({ kind: 'point', point: 6 });
        await tick();
        expect(second.getByTestId('duel-board-prompt').classList.contains('centred')).toBe(false);
    });

    test('double or cancel once the cube is clicked; nothing before', async () => {
        open({ side: 0, kind: 'cube', position: opening });
        const { queryByTestId, getByTestId } = render(DuelBoardPrompt);
        await tick();
        expect(queryByTestId('duel-board-prompt')).toBeNull();
        duelBoardStore.set({ swapped: false, prompt: 'double' });
        await tick();
        await fireEvent.click(getByTestId('duel-cancel-double'));
        expect(get(duelBoardStore).prompt).toBeNull();
        expect(PlayDuel).not.toHaveBeenCalled();
    });

    test('a finished move offers its validation', async () => {
        open({ side: 0, kind: 'move', position: opening });
        quizPlayStore.set(newPlay(opening, plays));
        const { queryByTestId } = render(DuelBoardPrompt);
        await tick();
        expect(queryByTestId('duel-validate')).toBeNull();
        duelBoardPress({ kind: 'point', point: 8 });
        duelBoardPress({ kind: 'point', point: 6 });
        await tick();
        expect(queryByTestId('duel-validate')).not.toBeNull();
    });
});
