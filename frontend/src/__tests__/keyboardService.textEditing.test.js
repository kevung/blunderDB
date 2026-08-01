import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';

// While focus is in an editable field, the clipboard/selection/undo combos
// belong to the field, not to the board. The trap this locks: the comment panel
// binds nothing itself, so Ctrl-C in the comment textarea reached the global
// dispatcher, copied the *position* instead of the selected text, and the panel
// guard preventDefault()-ed every Ctrl combo on top — leaving the user unable to
// copy, cut, select all or undo anything they had just typed. Worse, Ctrl-Delete
// (delete word) fell through to deletePosition().

const copyPosition = vi.fn();
const pastePosition = vi.fn();
const copyBoardImage = vi.fn();
const deletePosition = vi.fn();
const saveCurrentPosition = vi.fn();

vi.mock('../services/clipboardService.js', () => ({
    copyPosition,
    copyBoardImage,
    copyBoardWithAnalysisImage: vi.fn()
}));
vi.mock('../services/importService.js', () => ({
    pastePosition,
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
    deletePosition,
    saveCurrentPosition,
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
    toggleTournamentPanel: vi.fn(),
    toggleStatsPanel: vi.fn(),
    toggleEPCMode: vi.fn(),
    togglePipcount: vi.fn(),
    reloadAllPositions: vi.fn(),
    loadRandomPosition: vi.fn()
}));

const { handleKeyDown } = await import('../services/keyboardService.js');
const { openPanels, PANEL } = await import('../stores/uiStore.js');

/** Dispatch a Ctrl-combo and report whether the WebView default survived. */
function ctrl(key, extra = {}) {
    const event = new KeyboardEvent('keydown', { key, ctrlKey: true, cancelable: true, bubbles: true, ...extra });
    handleKeyDown(event);
    return { defaultPrevented: event.defaultPrevented };
}

describe('keyboard shortcuts while editing text', () => {
    let textarea;

    beforeEach(() => {
        vi.clearAllMocks();
        // The comment panel is open, which is what used to arm the blanket
        // preventDefault() on every Ctrl combo.
        openPanels.set(new Set([PANEL.COMMENT]));
        document.body.innerHTML = '<div class="comment-panel"><textarea id="commentTextArea"></textarea></div>';
        textarea = document.getElementById('commentTextArea');
        textarea.focus();
    });

    afterEach(() => {
        openPanels.set(new Set());
        document.body.innerHTML = '';
    });

    test('Ctrl-C copies the selected text, not the position', () => {
        const { defaultPrevented } = ctrl('c');
        expect(copyPosition).not.toHaveBeenCalled();
        expect(defaultPrevented).toBe(false);
    });

    test('Ctrl-V pastes into the field, not a position into the database', () => {
        const { defaultPrevented } = ctrl('v');
        expect(pastePosition).not.toHaveBeenCalled();
        expect(defaultPrevented).toBe(false);
    });

    test('Ctrl-X cuts the selection instead of copying the board image', () => {
        const { defaultPrevented } = ctrl('x');
        expect(copyBoardImage).not.toHaveBeenCalled();
        expect(defaultPrevented).toBe(false);
    });

    test('Ctrl-A and Ctrl-Z reach the field', () => {
        expect(ctrl('a').defaultPrevented).toBe(false);
        expect(ctrl('z').defaultPrevented).toBe(false);
    });

    test('Ctrl-Delete deletes a word, never the position', () => {
        ctrl('Delete', { code: 'Delete' });
        expect(deletePosition).not.toHaveBeenCalled();
    });

    test('AZERTY and QWERTZ get the same rule — the character decides', () => {
        // On AZERTY the physical KeyW slot produces "z"; matching on event.code
        // would have missed the undo combo entirely.
        expect(ctrl('z', { code: 'KeyW' }).defaultPrevented).toBe(false);
    });

    test('non-editing Ctrl combos still act on the position', () => {
        ctrl('s');
        expect(saveCurrentPosition).toHaveBeenCalledTimes(1);
    });
});

describe('keyboard shortcuts with the board focused', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        openPanels.set(new Set());
        document.body.innerHTML = '<div id="board" tabindex="-1"></div>';
        document.getElementById('board').focus();
    });

    afterEach(() => {
        document.body.innerHTML = '';
    });

    test('Ctrl-C still copies the position', () => {
        ctrl('c');
        expect(copyPosition).toHaveBeenCalledTimes(1);
    });

    test('Ctrl-V still pastes a position', () => {
        ctrl('v');
        expect(pastePosition).toHaveBeenCalledTimes(1);
    });
});
