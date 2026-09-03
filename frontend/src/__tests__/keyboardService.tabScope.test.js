import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';

// #204: bare Tab used to be hijacked into "open the search panel"
// unconditionally, everywhere in the app — standard keyboard focus
// navigation (moving between buttons, links, form fields) did not exist.
// It must now only do that while focus sits on the board itself (nothing
// inside `.scrollable-content` is a real focus target, so in practice this
// means focus is on <body> — the default state — or, if ever given one, on
// something inside the board container). Once focus has moved to any other
// real element, Tab must be left alone so the browser's native focus
// advancement can run.
//
// Ctrl-Tab is a separate case: not a focus-navigation combo, so it stays a
// global toggle regardless of focus — and #202's "show/hide" fix (toggling
// back when the tab is already open) had missed it, since it isn't dispatched
// through the letter() helper the other seven Ctrl+letter toggles share.

const toggleMatchPanel = vi.fn();
const toggleSearchPanel = vi.fn();

vi.mock('../services/clipboardService.js', () => ({
    copyPosition: vi.fn(),
    copyBoardImage: vi.fn(),
    copyBoardWithAnalysisImage: vi.fn()
}));
vi.mock('../services/importService.js', () => ({
    pastePosition: vi.fn(),
    importDatabase: vi.fn(),
    importPosition: vi.fn(),
    importFolder: vi.fn()
}));
vi.mock('../services/exportService.js', () => ({ exportDatabase: vi.fn() }));
vi.mock('../services/databaseService.js', () => ({
    newDatabase: vi.fn(),
    openDatabase: vi.fn(),
    exitApp: vi.fn(),
    setStatusBarMessage: vi.fn()
}));
vi.mock('../services/positionService.js', () => ({
    deletePosition: vi.fn(),
    saveCurrentPosition: vi.fn(),
    firstPosition: vi.fn(),
    previousPosition: vi.fn(),
    nextPosition: vi.fn(),
    lastPosition: vi.fn(),
    updatePosition: vi.fn(),
    toggleAnalysisPanel: vi.fn(),
    toggleCommentPanel: vi.fn(),
    toggleMetadataPanel: vi.fn(),
    toggleAnkiPanel: vi.fn(),
    toggleCollectionPanelAction: vi.fn(),
    toggleMatchPanel,
    toggleTournamentPanel: vi.fn(),
    toggleStatsPanel: vi.fn(),
    toggleSearchPanel,
    toggleEPCMode: vi.fn(),
    togglePipcount: vi.fn(),
    reloadAllPositions: vi.fn(),
    loadRandomPosition: vi.fn(),
    showDatesAndMetadata: vi.fn()
}));

const { handleKeyDown } = await import('../services/keyboardService.js');
const { activeTabStore } = await import('../stores/uiStore.js');
const { get } = await import('svelte/store');

function tab(extra = {}) {
    const event = new KeyboardEvent('keydown', { key: 'Tab', code: 'Tab', cancelable: true, bubbles: true, ...extra });
    handleKeyDown(event);
    return event;
}

describe('bare Tab only opens the search panel while focus is on the board (#204)', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        activeTabStore.set('matches');
        document.body.innerHTML = '';
    });
    afterEach(() => {
        document.body.innerHTML = '';
    });

    test('nothing focused (document.body, the default state) — Tab opens the search panel', () => {
        expect(document.activeElement).toBe(document.body);
        const event = tab();
        expect(get(activeTabStore)).toBe('search');
        expect(event.defaultPrevented).toBe(true);
    });

    test('focus inside .scrollable-content — Tab still opens the search panel', () => {
        document.body.innerHTML = '<div class="scrollable-content"><div id="in-board" tabindex="0"></div></div>';
        document.getElementById('in-board').focus();
        const event = tab();
        expect(get(activeTabStore)).toBe('search');
        expect(event.defaultPrevented).toBe(true);
    });

    test('focus on an unrelated button — Tab is left alone for native focus navigation', () => {
        document.body.innerHTML = '<button id="toolbar-btn">Toolbar</button>';
        document.getElementById('toolbar-btn').focus();
        const event = tab();
        expect(get(activeTabStore)).toBe('matches');
        expect(event.defaultPrevented).toBe(false);
    });

    test('focus inside the comment panel textarea — Tab moves focus normally, not to the search tab', () => {
        activeTabStore.set('comments');
        document.body.innerHTML = '<div class="comment-panel"><textarea id="commentTextArea"></textarea><button id="next-field"></button></div>';
        document.getElementById('commentTextArea').focus();
        const event = tab();
        expect(get(activeTabStore)).toBe('comments');
        expect(event.defaultPrevented).toBe(false);
    });
});

describe('Ctrl-Tab toggles the matches panel like the other seven Ctrl+letter toggles', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        activeTabStore.set('matches');
        document.body.innerHTML = '';
    });

    test('calls toggleMatchPanel() — not the pre-#202 plain activeTabStore.set()', () => {
        const event = tab({ ctrlKey: true });
        expect(toggleMatchPanel).toHaveBeenCalledTimes(1);
        expect(event.defaultPrevented).toBe(true);
    });

    test('fires regardless of where focus is (unlike bare Tab)', () => {
        document.body.innerHTML = '<button id="toolbar-btn">Toolbar</button>';
        document.getElementById('toolbar-btn').focus();
        const event = tab({ ctrlKey: true });
        expect(toggleMatchPanel).toHaveBeenCalledTimes(1);
        expect(event.defaultPrevented).toBe(true);
    });
});
