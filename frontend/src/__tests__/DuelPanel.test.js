/**
 * L'onglet Duel monté : le formulaire sans Duel ouvert, les gestes de la décision attendue avec
 * un Duel, et rien du moteur. Chaque composant est monté, pour qu'une collision de nom entre une
 * prop et une rune se voie ici plutôt qu'au lancement.
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, fireEvent } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../services/duelService.js', () => ({
    loadDuelForm: vi.fn(() => ({ money: false, matchLength: 7, jacoby: true, start: 'opening', away: [7, 7], side: 0, level: 'normal', cadence: '', timeOut: 'continue', player: '', record: true })),
    duelOffer: vi.fn(() => Promise.resolve({ cadences: [{ name: 'tournament', reservePerPoint: 120, delay: 12 }], botLevels: ['instant', 'normal', 'thorough'] })),
    refreshDuels: vi.fn(() => Promise.resolve()),
    startDuel: vi.fn(),
    resumeDuel: vi.fn(),
    suspendDuel: vi.fn(),
    confirmForfeitDuel: vi.fn(),
    confirmCancelDuel: vi.fn()
}));

import DuelPanel from '../components/DuelPanel.svelte';
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

    test('the suspended Duels resume', async () => {
        duelListStore.set([{ id: 9, label: 'Kévin vs gammonNet normal, 7 points', createdAt: '', updatedAt: '', open: false }]);
        const { getByText } = render(DuelPanel);
        await tick();
        await fireEvent.click(getByText('Resume'));
        expect(resumeDuel).toHaveBeenCalledWith(9);
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
        expect(queryByTestId('duel-hint').getAttribute('title')).toContain('ab12cd34');
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
