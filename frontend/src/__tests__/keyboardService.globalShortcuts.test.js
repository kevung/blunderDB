/**
 * keyboardService.globalShortcuts.test.js
 *
 * SHIFT-J / SHIFT-K switch views, and did so "depending on which panel was
 * open". Two layers filter a keystroke and they had each grown their own copy
 * of "what always gets through": panelKeyGuard, for the docked panels' own
 * listeners, and a handful of hand-written branches inside the global
 * dispatcher, one per focused panel. Adding the pair to the first left it dead
 * behind the second.
 *
 * These tests drive the dispatcher directly, with the focus inside each panel
 * that owns a branch, and check the pair reaches viewStore — alongside the
 * shortcuts that were already global, so the shared predicate cannot be
 * narrowed later without a failure.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('../services/importService.js', () => ({
    importDatabase: vi.fn(),
    importPosition: vi.fn(),
    importFolder: vi.fn(),
    pastePosition: vi.fn()
}));

const selectPreviousView = vi.fn();
const selectNextView = vi.fn();
vi.mock('../stores/viewStore.js', async (importOriginal) => {
    const actual = await importOriginal();
    return {
        ...actual,
        viewStore: {
            ...actual.viewStore,
            selectPreviousView: (...a) => selectPreviousView(...a),
            selectNextView: (...a) => selectNextView(...a)
        }
    };
});

import { handleKeyDown, isAlwaysGlobal, panelKeyGuard } from '../services/keyboardService.js';
import { activeTabStore } from '../stores/uiStore.js';

/** Give the document a focused element inside a panel of class `className`. */
function focusInside(className) {
    const panel = document.createElement('div');
    panel.className = className;
    const inner = document.createElement('div');
    inner.tabIndex = -1;
    panel.appendChild(inner);
    document.body.appendChild(panel);
    inner.focus();
    return panel;
}

function press(key, extra = {}) {
    const event = new KeyboardEvent('keydown', { key, bubbles: true, cancelable: true, ...extra });
    handleKeyDown(event);
    return event;
}

let mounted = [];

beforeEach(() => {
    vi.clearAllMocks();
    activeTabStore.set('analysis');
});

afterEach(() => {
    mounted.forEach((el) => el.remove());
    mounted = [];
    activeTabStore.set('analysis');
});

describe('the always-global predicate', () => {
    test('holds the pair that switches views, and the ones that already worked', () => {
        expect(isAlwaysGlobal(new KeyboardEvent('keydown', { key: 'J', shiftKey: true }))).toBe(true);
        expect(isAlwaysGlobal(new KeyboardEvent('keydown', { key: 'K', shiftKey: true }))).toBe(true);
        expect(isAlwaysGlobal(new KeyboardEvent('keydown', { key: 'x', ctrlKey: true }))).toBe(true);
        expect(isAlwaysGlobal(new KeyboardEvent('keydown', { key: ' ', code: 'Space' }))).toBe(true);
        expect(isAlwaysGlobal(new KeyboardEvent('keydown', { key: '?' }))).toBe(true);
    });

    test('an ordinary key is not global — panels keep their own letters', () => {
        expect(isAlwaysGlobal(new KeyboardEvent('keydown', { key: 'j' }))).toBe(false);
        expect(isAlwaysGlobal(new KeyboardEvent('keydown', { key: 'p' }))).toBe(false);
    });

    test('panelKeyGuard is built on it, so the two layers cannot drift apart', () => {
        expect(panelKeyGuard(new KeyboardEvent('keydown', { key: 'J', shiftKey: true }))).toBe(true);
        expect(panelKeyGuard(new KeyboardEvent('keydown', { key: 'j' }))).toBe(false);
    });
});

describe('SHIFT-J / SHIFT-K reach the view switch from every panel', () => {
    // One case per branch the dispatcher carries. '.analysis-panel' and
    // '.comment-panel' are the two my first pass missed: they are filtered by
    // the dispatcher itself, not by panelKeyGuard.
    for (const className of ['match-panel', 'collection-panel', 'tournament-panel', 'analysis-panel', 'comment-panel']) {
        test(`with focus inside .${className}`, () => {
            if (className === 'comment-panel') activeTabStore.set('comments');
            mounted.push(focusInside(className));

            press('J', { shiftKey: true });
            expect(selectPreviousView, `SHIFT-J from .${className}`).toHaveBeenCalledTimes(1);

            press('K', { shiftKey: true });
            expect(selectNextView, `SHIFT-K from .${className}`).toHaveBeenCalledTimes(1);
        });
    }

    test('with focus nowhere in particular', () => {
        document.body.focus();
        press('J', { shiftKey: true });
        expect(selectPreviousView).toHaveBeenCalledTimes(1);
    });

    test('but not while typing in a field', () => {
        const input = document.createElement('input');
        document.body.appendChild(input);
        mounted.push(input);
        input.focus();

        press('J', { shiftKey: true });
        expect(selectPreviousView, 'a capital J belongs to the text being typed').not.toHaveBeenCalled();
    });
});
