/**
 * AnalysisPanel.selectedMoveCleanup.test.js
 *
 * TabbedPanel mounts and destroys each tab's panel component on every tab
 * switch (its {#if} pattern — see that file's header comment), never merely
 * hides it. AnalysisPanel used to decide its own "opened"/"closed" moments by
 * watching $openPanels.has(PANEL.ANALYSIS) — but the 'analysis' tab has never
 * driven that flag since the tabHandler.js refactor (applyTabPanels only
 * wires matches/stats/tournaments/collections to a PANEL), so openPanels was
 * always empty for this panel and the effect's "closed" branch — the one that
 * reset selectedMoveStore — never ran.
 *
 * Left stuck, a selected move silently froze keyboard navigation everywhere
 * in the app: keyboardService's global dispatch withholds j/k/ArrowLeft/
 * ArrowRight (position browsing) while selectedMoveStore is set, on the
 * assumption those keys are walking the candidate-move list instead. Clicking
 * a move row, then switching tabs without deselecting it, left every other
 * panel's navigation keys silently doing nothing.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database.js', () => ({ LoadAnalysis: vi.fn(() => Promise.resolve(null)) }));

const nextPosition = vi.fn();
vi.mock('../services/positionService.js', async () => {
    const actual = await vi.importActual('../services/positionService.js');
    return { ...actual, nextPosition };
});

const { handleKeyDown } = await import('../services/keyboardService.js');
const { selectedMoveStore, analysisStore } = await import('../stores/analysisStore.js');
const { matchContextStore } = await import('../stores/positionStore.js');
const { databasePathStore } = await import('../stores/databaseStore.js');
const AnalysisPanel = (await import('../components/AnalysisPanel.svelte')).default;

describe('AnalysisPanel: selectedMoveStore does not survive leaving the tab', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        selectedMoveStore.set(null);
        databasePathStore.set('/fake/db.sqlite');
        matchContextStore.set({ isMatchMode: false, movePositions: [], currentIndex: 0, player1Name: '', player2Name: '' });
        analysisStore.set({
            checkerAnalysis: {
                moves: [
                    { move: '8/5 6/5', equity: 0.1 },
                    { move: '24/21 13/11', equity: -0.05 }
                ]
            },
            analysisType: 'CheckerMove'
        });
        window.addEventListener('keydown', handleKeyDown);
    });

    afterEach(() => {
        window.removeEventListener('keydown', handleKeyDown);
        cleanup();
        document.body.innerHTML = '';
    });

    test('destroying the panel (tab switch) clears a stuck selectedMoveStore', async () => {
        const { unmount } = render(AnalysisPanel, { props: { onClose: vi.fn() } });
        await tick();

        // The user clicked a candidate move row.
        selectedMoveStore.set('8/5 6/5');

        // TabbedPanel switches tabs: the panel is destroyed, not hidden.
        unmount();
        await tick();

        let selected;
        selectedMoveStore.subscribe((v) => (selected = v))();
        expect(selected).toBeNull();
    });

    test('position navigation elsewhere in the app is not left frozen after leaving the tab', async () => {
        const { unmount } = render(AnalysisPanel, { props: { onClose: vi.fn() } });
        await tick();

        selectedMoveStore.set('8/5 6/5');
        unmount();
        await tick();

        document.body.innerHTML = '<div id="board" tabindex="-1"></div>';
        const board = document.getElementById('board');
        board.focus();
        board.dispatchEvent(new KeyboardEvent('keydown', { key: 'j', bubbles: true, cancelable: true }));
        await tick();

        expect(nextPosition).toHaveBeenCalledTimes(1);
    });
});
