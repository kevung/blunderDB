/**
 * The Eval panel walks its candidate list with the keyboard exactly as the
 * analysis panel does (doc/source/raccourcis.rst): a click selects, then
 * j/BAS and k/HAUT move the selection one rank at a time and Escape drops it.
 *
 * The regression this guards is silent: keyboardService withholds
 * j/k/arrows app-wide while selectedMoveStore is set, so a panel that shows
 * candidates without handling those keys itself leaves them doing nothing at
 * all — which is what the Eval panel did until #eval-move-nav.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, fireEvent } from '@testing-library/svelte';
import { get } from 'svelte/store';

const MOVES = vi.hoisted(() => [
    { move: '8/5 6/5', equity: 0.1, equityError: 0 },
    { move: '24/21 13/11', equity: 0.05, equityError: -0.05 },
    { move: '13/10 13/11', equity: 0.0, equityError: -0.1 }
]);

vi.mock('../../wailsjs/go/gui/App.js', () => ({
    EvaluatePositionImmediate: vi.fn().mockResolvedValue({ moves: MOVES, preRoll: null }),
    StartEvaluationAtRest: vi.fn().mockResolvedValue(undefined),
    CancelEvaluationAtRest: vi.fn().mockResolvedValue(undefined)
}));

vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetEpcChallenge: vi.fn().mockResolvedValue(false),
    SaveEpcChallenge: vi.fn().mockResolvedValue(undefined),
    GetGammonNetDisplayPly: vi.fn().mockResolvedValue(2),
    GetGammonNetPruneK: vi.fn().mockResolvedValue(12),
    GetGammonNetCandidates: vi.fn().mockResolvedValue(10)
}));

vi.mock('../../wailsjs/runtime/runtime.js', () => ({
    EventsOn: vi.fn(() => () => {}),
    BrowserOpenURL: vi.fn()
}));

import { statusBarModeStore } from '../stores/uiStore.js';
import { positionStore, emptyPosition } from '../stores/positionStore.js';
import { selectedMoveStore } from '../stores/analysisStore.js';
import EPCPanel from '../components/EPCPanel.svelte';

async function mountWithMoves() {
    const { container } = render(EPCPanel);
    // Let the 0-ply round trip land; the 2-ply escalation stays pending.
    await vi.advanceTimersByTimeAsync(0);
    const panel = container.querySelector('.epc-panel');
    const rows = [...container.querySelectorAll('tbody tr')];
    return { panel, rows };
}

describe('EPCPanel candidate-list keyboard navigation', () => {
    beforeEach(() => {
        vi.useFakeTimers();
        statusBarModeStore.set('EPC');
        positionStore.set(emptyPosition()); // dice [3, 1]: the moves list is shown
        selectedMoveStore.set(null);
    });

    afterEach(() => {
        vi.runOnlyPendingTimers();
        vi.useRealTimers();
        statusBarModeStore.set('NORMAL');
        selectedMoveStore.set(null);
    });

    test('j/ArrowDown and k/ArrowUp walk the ranking once a move is selected', async () => {
        const { panel, rows } = await mountWithMoves();
        expect(rows).toHaveLength(MOVES.length);

        await fireEvent.click(rows[0]);
        expect(get(selectedMoveStore)).toBe(MOVES[0].move);

        await fireEvent.keyDown(panel, { key: 'j' });
        expect(get(selectedMoveStore)).toBe(MOVES[1].move);

        await fireEvent.keyDown(panel, { key: 'ArrowDown' });
        expect(get(selectedMoveStore)).toBe(MOVES[2].move);

        // The ends of the list hold.
        await fireEvent.keyDown(panel, { key: 'ArrowDown' });
        expect(get(selectedMoveStore)).toBe(MOVES[2].move);

        await fireEvent.keyDown(panel, { key: 'k' });
        expect(get(selectedMoveStore)).toBe(MOVES[1].move);

        await fireEvent.keyDown(panel, { key: 'ArrowUp' });
        expect(get(selectedMoveStore)).toBe(MOVES[0].move);

        await fireEvent.keyDown(panel, { key: 'ArrowUp' });
        expect(get(selectedMoveStore)).toBe(MOVES[0].move);
    });

    test('Escape drops the selection, and nothing moves without one', async () => {
        const { panel, rows } = await mountWithMoves();

        await fireEvent.keyDown(panel, { key: 'j' });
        expect(get(selectedMoveStore)).toBeNull();

        await fireEvent.click(rows[1]);
        await fireEvent.keyDown(panel, { key: 'Escape' });
        expect(get(selectedMoveStore)).toBeNull();
    });

    test('clicking a row gives the panel the keyboard', async () => {
        const { panel, rows } = await mountWithMoves();

        await fireEvent.click(rows[0]);
        expect(document.activeElement).toBe(panel);
    });
});
