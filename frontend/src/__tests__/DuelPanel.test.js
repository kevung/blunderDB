/**
 * L'onglet Duel monté : le formulaire sans Duel ouvert, les gestes de la décision attendue avec
 * un Duel, et rien du moteur. Chaque composant est monté, pour qu'une collision de nom entre une
 * prop et une rune se voie ici plutôt qu'au lancement.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';
import { must } from './helpers/must.js';

vi.mock('../services/duelService.js', () => ({
    loadDuelForm: vi.fn(() => ({ money: false, matchLength: 7, jacoby: true, start: 'opening', away: [7, 7], side: 0, level: 'normal', cadence: '', timeOut: 'continue', player: '', record: true })),
    saveDuelForm: vi.fn(),
    duelOffer: vi.fn(() =>
        Promise.resolve({
            cadences: [
                { name: 'standard', reservePerPoint: 120, delay: 12 },
                { name: 'speed', reservePerPoint: 24, delay: 10 }
            ],
            botLevels: ['instant', 'normal', 'thorough']
        })
    ),
    refreshDuels: vi.fn(() => Promise.resolve()),
    startDuel: vi.fn(),
    resumeDuel: vi.fn(),
    suspendDuel: vi.fn(),
    confirmForfeitDuel: vi.fn(),
    confirmCancelDuel: vi.fn()
}));

vi.mock('../services/modeMachine.js', () => ({ enterEvalOnDisplayed: vi.fn(), exitEvalMode: vi.fn() }));

import DuelPanel from '../components/DuelPanel.svelte';
import { get } from 'svelte/store';
import { statusBarModeStore } from '../stores/uiStore.js';
import { positionStore } from '../stores/positionStore.js';
import { enterEvalOnDisplayed, exitEvalMode } from '../services/modeMachine.js';
import { saveDuelForm, loadDuelForm } from '../services/duelService.js';
import DuelClocks from '../components/DuelClocks.svelte';
import { duelStore, duelListStore } from '../stores/duelStore.js';
import { databasePathStore } from '../stores/databaseStore.js';
import { startDuel, resumeDuel, suspendDuel, confirmForfeitDuel, confirmCancelDuel } from '../services/duelService.js';

const sheet = { document: { header: { match_length: 7 }, actions: [] }, actions: [], games: [], next: {}, cursor: 0 };

/** @param {any} awaiting */
function openDuel(awaiting) {
    duelStore.set({
        state: {
            id: 3,
            revision: 2,
            header: { match_length: 7, player1: 'Kévin', player2: 'gammonNet normal' },
            sides: [
                { kind: 'external', name: 'Kévin' },
                { kind: 'bot', level: 'normal' }
            ],
            score: [2, 1],
            fingerprint: 'ab12cd34',
            awaiting
        },
        sheet
    });
}

beforeEach(() => {
    databasePathStore.set('/tmp/lib.db');
    statusBarModeStore.set('NORMAL');
    duelStore.set(null);
    duelListStore.set([]);
});

afterEach(() => {
    cleanup();
    vi.clearAllMocks();
});

describe('DuelPanel', () => {
    test('without a Duel: the form plays with what it holds', async () => {
        const { getByText } = render(DuelPanel);
        await tick();
        await fireEvent.click(getByText('Play'));
        expect(startDuel).toHaveBeenCalledTimes(1);
        expect(/** @type {any} */ (startDuel).mock.calls[0][0].matchLength).toBe(7);
    });

    test('Play ends the band; nothing suspended, no section', async () => {
        const { container } = render(DuelPanel);
        await tick();
        expect(container.querySelector('[data-testid="panel-header"] .panel-actions [data-testid="duel-play"]')).not.toBeNull();
        expect(container.querySelector('[data-testid="duel-suspended"]')).toBeNull();
    });

    test('the suspended Duels resume', async () => {
        duelListStore.set([{ id: 9, label: 'Kévin vs gammonNet normal, 7 points', createdAt: '', updatedAt: '', open: false }]);
        const { getByText } = render(DuelPanel);
        await tick();
        await fireEvent.click(getByText('Resume'));
        expect(resumeDuel).toHaveBeenCalledWith(9, expect.anything());
    });

    test('with a Duel: the history, pause and stop; the gestures of the game are on the board', async () => {
        openDuel({ side: 0, kind: 'cube', position: { cube: { value: 1 } } });
        const { getByText, queryByText, container } = render(DuelPanel);
        await tick();
        expect(getByText('Your turn: click the dice to roll, or the cube to double.')).toBeTruthy();
        expect(queryByText('Double')).toBeNull();
        expect(queryByText('Validate')).toBeNull();
        expect(queryByText('Stop and keep')).toBeNull();
        await fireEvent.click(getByText('Pause match'));
        expect(suspendDuel).toHaveBeenCalled();
        await fireEvent.click(getByText('Cancel match'));
        expect(confirmCancelDuel).toHaveBeenCalled();
        await fireEvent.click(getByText('Forfeit match'));
        expect(confirmForfeitDuel).toHaveBeenCalled();
        expect(container.textContent).not.toMatch(/équité|equity|%/i);
    });

    test('a match is left only whole or not at all: forfeit, pause or cancel, never kept as it stands', async () => {
        openDuel({ side: 0, kind: 'cube', position: { cube: { value: 1 } } });
        const { container } = render(DuelPanel);
        await tick();
        const gestures = [...container.querySelectorAll('.gestures button')].map((b) => b.textContent.trim());
        expect(gestures).toEqual(['Forfeit match', 'Pause match', 'Cancel match']);
    });

    test('the Bot’s turn says so', async () => {
        openDuel({ side: 1, kind: 'move', position: {} });
        const { getByText } = render(DuelPanel);
        await tick();
        expect(getByText('The Bot is playing…')).toBeTruthy();
    });

    test('the seed’s fingerprint is a tooltip, not text, and the score is not repeated', async () => {
        openDuel({ side: 0, kind: 'cube', position: { cube: { value: 1 } } });
        const { container, queryByTestId } = render(DuelPanel);
        await tick();
        expect(container.textContent).not.toContain('ab12cd34');
        expect(queryByTestId('duel-hint')?.getAttribute('title')).toContain('ab12cd34');
        // The strip may cut the hint short: the tooltip gives it whole.
        expect(queryByTestId('duel-hint')?.getAttribute('title')).toContain(queryByTestId('duel-hint')?.textContent?.trim());
        // Without a Cadence there is no clock line, so no score either.
        expect(queryByTestId('duel-clocks')).toBeNull();
    });
});

describe('DuelClocks', () => {
    test('mounts with a clock and without one', async () => {
        const duel = {
            header: { match_length: 5 },
            sides: [{ kind: 'external' }, { kind: 'bot' }],
            score: [1, 3],
            awaiting: { side: 0, since: new Date().toISOString(), position: { cube: { value: 2 } } },
            clock: { cadence: { delay: 12 }, reserve: [60000, 90000], turn: 0, spent: 0 }
        };
        const { container } = render(DuelClocks, { props: { duel } });
        await tick();
        expect(container.textContent).toContain('1:00');
        expect(container.textContent).toContain('1:30');
        cleanup();
        const bare = render(DuelClocks, { props: { duel: { ...duel, clock: null }, compact: true } });
        await tick();
        expect(bare.container.textContent).toContain('3');
        cleanup();
        const only = render(DuelClocks, { props: { duel, clocksOnly: true } });
        await tick();
        expect(only.container.textContent).toContain('1:30');
        expect(only.container.querySelector('.points')).toBeNull();
    });
});

/** @param {any} over */
function fromBoard(over = {}) {
    vi.mocked(loadDuelForm).mockReturnValueOnce(/** @type {any} */ ({ matchLength: 7, start: 'board', away: [7, 7], ...over }));
}

/** @param {any} pos */
function scratch(pos) {
    statusBarModeStore.set('EVAL');
    positionStore.set({ board: { points: [], bearoff: [0, 0] }, cube: { owner: -1, value: 0 }, dice: [0, 0], score: [7, 7], player_on_roll: 0, decision_type: 0, ...pos });
}

/** @param {Element} el @param {string} value */
async function choose(el, value) {
    /** @type {HTMLSelectElement} */ (el).value = value;
    await fireEvent.change(el);
}

describe('DuelPanel: the start', () => {
    test('the band holds the start, the time control and Play, Play at its end', async () => {
        const { container } = render(DuelPanel);
        await tick();
        const band = must(container.querySelector('[data-testid="panel-header"]'));
        expect(band.querySelector('[data-testid="duel-start"]')).not.toBeNull();
        expect(band.querySelector('[data-testid="duel-cadence"]')).not.toBeNull();
        expect(band.querySelector('.panel-actions [data-testid="duel-play"]')).not.toBeNull();
    });

    test('from the board, the board is edited as in Eval; from the opening, it is left', async () => {
        fromBoard();
        render(DuelPanel);
        await tick();
        await tick();
        expect(enterEvalOnDisplayed).toHaveBeenCalled();
        cleanup();
        statusBarModeStore.set('EVAL');
        render(DuelPanel);
        await tick();
        await tick();
        expect(exitEvalMode).toHaveBeenCalled();
    });

    test('the board score fills the fields, Crawford included, and the fields write the board', async () => {
        fromBoard();
        scratch({ score: [1, 4] });
        const { container } = render(DuelPanel);
        await tick();
        await tick();
        const p1 = /** @type {HTMLSelectElement} */ (must(container.querySelector('[data-testid="duel-away-1"]')));
        const p2 = /** @type {HTMLSelectElement} */ (must(container.querySelector('[data-testid="duel-away-2"]')));
        expect(p1.selectedOptions[0].textContent).toBe('1 (Crawford)');
        expect(p2.value).toBe('4');
        await choose(p2, '3');
        expect(get(positionStore).score).toEqual([1, 3]);
        await choose(p1, '0');
        expect(get(positionStore).score).toEqual([0, 3]);
        positionStore.update((pos) => ({ ...pos, score: [5, 2] }));
        await tick();
        expect(p1.value).toBe('5');
        expect(p2.value).toBe('2');
    });

    test('the roll and the cube choices show in the band when the board calls for them', async () => {
        fromBoard();
        scratch({ score: [5, 5], dice: [6, 5] });
        const { container } = render(DuelPanel);
        await tick();
        await tick();
        const dice = /** @type {HTMLSelectElement} */ (must(container.querySelector('[data-testid="panel-header"] [data-testid="duel-dice"]')));
        expect(dice.selectedOptions[0].textContent).toBe('Board dice');
        expect(container.querySelector('[data-testid="duel-cube"]')).toBeNull();
        await choose(dice, 'true');
        const cube = /** @type {HTMLSelectElement} */ (must(container.querySelector('[data-testid="panel-header"] [data-testid="duel-cube"]')));
        expect(cube.selectedOptions[0].textContent).toBe('Before the cube');
        await choose(cube, 'true');
        await fireEvent.click(must(container.querySelector('[data-testid="duel-play"]')));
        expect(/** @type {any} */ (startDuel).mock.calls[0][0]).toMatchObject({ reroll: true, afterCube: true, start: 'board' });
    });

    test('a single game is played as ticked', async () => {
        const { container } = render(DuelPanel);
        await tick();
        await fireEvent.click(must(container.querySelector('[data-testid="duel-single-game"]')));
        await fireEvent.click(must(container.querySelector('[data-testid="duel-play"]')));
        expect(/** @type {any} */ (startDuel).mock.calls[0][0].singleGame).toBe(true);
    });

    test('a time control: a preset fills the two fields, one is saved under a name and deleted', async () => {
        const { container } = render(DuelPanel);
        await tick();
        await tick();
        await choose(must(container.querySelector('[data-testid="duel-cadence"]')), 'speed');
        expect(/** @type {HTMLInputElement} */ (must(container.querySelector('[data-testid="duel-minutes"]'))).value).toBe('0.4');
        expect(/** @type {HTMLInputElement} */ (must(container.querySelector('[data-testid="duel-delay"]'))).value).toBe('10');
        expect(container.querySelector('[data-testid="duel-cadence-delete"]')).toBeNull();
        await fireEvent.input(must(container.querySelector('[data-testid="duel-minutes"]')), { target: { value: '1' } });
        await fireEvent.input(must(container.querySelector('[data-testid="duel-cadence-name"]')), { target: { value: 'Club' } });
        await fireEvent.click(must(container.querySelector('[data-testid="duel-cadence-save"]')));
        const saved = /** @type {any} */ (saveDuelForm).mock.calls.at(-1)[0];
        expect(saved.cadences).toEqual([{ name: 'Club', minutesPerPoint: 1, delay: 10 }]);
        expect(saved.cadence).toBe('Club');
        await fireEvent.click(must(container.querySelector('[data-testid="duel-cadence-delete"]')));
        expect(/** @type {any} */ (saveDuelForm).mock.calls.at(-1)[0].cadences).toEqual([]);
    });
});
