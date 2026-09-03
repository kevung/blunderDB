/**
 * Board.contextMenu.test.js
 *
 * Right-clicking the board offers "Evaluate this position" (which opens the
 * Eval panel on the position the board is showing instead of the panel's
 * default bearoff), plus the small ergonomics batch added for #215: evaluate
 * the mirror, copy the board image with its analysis, open a new view, and —
 * only once the position has a database id — add it to an Anki deck. The
 * menu is deliberately absent in EDIT and EPC, where the right button
 * already places the other colour's checkers.
 *
 * two.js is mocked exactly as in Board.redraw.test.js — the drawing backend is
 * irrelevant here, only the contextmenu handler and the menu it renders are.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, screen } from '@testing-library/svelte';
import { tick } from 'svelte';
import { get } from 'svelte/store';

vi.mock('two.js', () => {
    function makeFakeShape() {
        return { translation: { set: () => {} } };
    }
    class FakeTwo {
        constructor(params) {
            this.width = params?.width ?? 0;
            this.height = params?.height ?? 0;
            this.renderer = { setSize: vi.fn() };
            this.clear = vi.fn();
            this.update = vi.fn();
        }
        appendTo() {
            return this;
        }
        makeGroup() {
            return { children: [], add: () => {}, remove: () => {} };
        }
        makeText() {
            return makeFakeShape();
        }
        makeCircle() {
            return makeFakeShape();
        }
        makeRectangle() {
            return makeFakeShape();
        }
        makeLine() {
            return makeFakeShape();
        }
        makePath() {
            return makeFakeShape();
        }
    }
    return { default: FakeTwo };
});

vi.mock('../services/positionService.js', () => ({ sendPositionToEval: vi.fn() }));
vi.mock('../services/clipboardService.js', () => ({ copyBoardWithAnalysisImage: vi.fn() }));
vi.mock('../services/ankiService.js', () => ({
    loadDecks: vi.fn(() => Promise.resolve()),
    addPositionToDeck: vi.fn(() => Promise.resolve())
}));

import { sendPositionToEval } from '../services/positionService.js';
import { copyBoardWithAnalysisImage } from '../services/clipboardService.js';
import * as anki from '../services/ankiService.js';
import Board from '../components/Board.svelte';
import { positionStore, emptyPosition } from '../stores/positionStore.js';
import { statusBarModeStore, statusBarTextStore } from '../stores/uiStore.js';
import { ankiDecksStore } from '../stores/ankiStore.js';
import { viewStore } from '../stores/viewStore.js';

/** Mount the board and right-click its canvas. */
async function rightClickBoard() {
    render(Board);
    await tick();
    await tick();
    const canvas = document.getElementById('backgammon-board');
    const event = new MouseEvent('contextmenu', { bubbles: true, cancelable: true, clientX: 120, clientY: 80 });
    canvas.dispatchEvent(event);
    await tick();
    return event;
}

describe('Board context menu', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        const pos = emptyPosition();
        pos.id = 42;
        positionStore.set(pos);
        statusBarModeStore.set('NORMAL');
        ankiDecksStore.set([]);
    });

    afterEach(() => {
        cleanup();
        statusBarModeStore.set('NORMAL');
        ankiDecksStore.set([]);
    });

    test('offers "evaluate this position" in NORMAL mode and hands the board to the Eval panel', async () => {
        const event = await rightClickBoard();

        expect(event.defaultPrevented, 'the native menu never shows').toBe(true);
        const item = screen.getByRole('menuitem', { name: 'Evaluate this position' });
        item.click();

        expect(sendPositionToEval).toHaveBeenCalledTimes(1);
        expect(sendPositionToEval.mock.calls[0][0].id).toBe(42);
    });

    test('offers "evaluate the mirror of this position", sending the mirrored board', async () => {
        await rightClickBoard();

        screen.getByRole('menuitem', { name: 'Evaluate the mirror of this position' }).click();

        expect(sendPositionToEval).toHaveBeenCalledTimes(1);
        const sent = sendPositionToEval.mock.calls[0][0];
        // player_on_roll flips under mirroring; the position stays the one on
        // screen (id preserved) but the checkers are swapped.
        expect(sent.player_on_roll).toBe(1);
    });

    test('offers "copy the board image with the analysis"', async () => {
        await rightClickBoard();

        screen.getByRole('menuitem', { name: 'Copy the board image with the analysis' }).click();

        expect(copyBoardWithAnalysisImage).toHaveBeenCalledTimes(1);
    });

    test('offers "new view", which opens another view', async () => {
        const addView = vi.spyOn(viewStore, 'addView');

        await rightClickBoard();
        screen.getByRole('menuitem', { name: 'New view' }).click();

        expect(addView).toHaveBeenCalledTimes(1);
    });

    test('does not offer an Anki deck when the position has no database id (scratch board)', async () => {
        const pos = emptyPosition();
        pos.id = 0;
        positionStore.set(pos);
        ankiDecksStore.set([{ id: 7, name: 'Openings' }]);

        await rightClickBoard();

        expect(screen.queryByRole('menuitem', { name: /Add to Anki deck/ })).toBeNull();
        expect(anki.loadDecks).not.toHaveBeenCalled();
    });

    test('offers each already-loaded Anki deck, and adds the position to it on click', async () => {
        ankiDecksStore.set([{ id: 7, name: 'Openings' }]);

        await rightClickBoard();
        const deckItem = screen.getByRole('menuitem', { name: 'Add to Anki deck: Openings' });
        deckItem.click();
        await tick();

        expect(anki.addPositionToDeck).toHaveBeenCalledWith(7, 42);
        // The status bar holds a deferred i18n descriptor (tMsg), not a
        // resolved string — see i18n/index.js — so check its key/params
        // rather than the rendered text.
        expect(get(statusBarTextStore)).toMatchObject({
            i18nKey: 'status.positionAddedToDeck',
            i18nParams: { deck: 'Openings' }
        });
    });

    test.each([['EDIT'], ['EPC']])('stays out of the way in %s mode, where the right button places checkers', async (mode) => {
        statusBarModeStore.set(mode);

        const event = await rightClickBoard();

        expect(event.defaultPrevented, 'the native menu is still suppressed').toBe(true);
        expect(screen.queryByRole('menuitem')).toBeNull();
        expect(sendPositionToEval).not.toHaveBeenCalled();
    });
});
