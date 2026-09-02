/**
 * Board.contextMenu.test.js
 *
 * Right-clicking the board offers "Evaluate this position", which opens the
 * Eval panel on the position the board is showing instead of the panel's
 * default bearoff. The menu is deliberately absent in EDIT and EPC, where the
 * right button already places the other colour's checkers.
 *
 * two.js is mocked exactly as in Board.redraw.test.js — the drawing backend is
 * irrelevant here, only the contextmenu handler and the menu it renders are.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup, screen } from '@testing-library/svelte';
import { tick } from 'svelte';

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

import { sendPositionToEval } from '../services/positionService.js';
import Board from '../components/Board.svelte';
import { positionStore, emptyPosition } from '../stores/positionStore.js';
import { statusBarModeStore } from '../stores/uiStore.js';

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
    });

    afterEach(() => {
        cleanup();
        statusBarModeStore.set('NORMAL');
    });

    test('offers "evaluate this position" in NORMAL mode and hands the board to the Eval panel', async () => {
        const event = await rightClickBoard();

        expect(event.defaultPrevented, 'the native menu never shows').toBe(true);
        const item = screen.getByRole('menuitem');
        item.click();

        expect(sendPositionToEval).toHaveBeenCalledTimes(1);
        expect(sendPositionToEval.mock.calls[0][0].id).toBe(42);
    });

    test.each([['EDIT'], ['EPC']])('stays out of the way in %s mode, where the right button places checkers', async (mode) => {
        statusBarModeStore.set(mode);

        const event = await rightClickBoard();

        expect(event.defaultPrevented, 'the native menu is still suppressed').toBe(true);
        expect(screen.queryByRole('menuitem')).toBeNull();
        expect(sendPositionToEval).not.toHaveBeenCalled();
    });
});
