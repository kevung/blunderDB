import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';

// #399: CTRL-S saves the Eval panel's board as it saves the Search panel's.
// The save is decided by saveScratchBoard() on the mode; what must not happen
// here is a key scope withholding the combo while focus is inside the Eval
// panel — on its own add button, for instance, right after a click.

const saveCurrentPosition = vi.fn();
const updatePosition = vi.fn();

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
    saveCurrentPosition,
    firstPosition: vi.fn(),
    previousPosition: vi.fn(),
    nextPosition: vi.fn(),
    lastPosition: vi.fn(),
    updatePosition,
    toggleAnalysisPanel: vi.fn(),
    toggleCommentPanel: vi.fn(),
    toggleMetadataPanel: vi.fn(),
    toggleAnkiPanel: vi.fn(),
    toggleCollectionPanelAction: vi.fn(),
    toggleMatchPanel: vi.fn(),
    toggleTournamentPanel: vi.fn(),
    toggleStatsPanel: vi.fn(),
    toggleSearchPanel: vi.fn(),
    toggleEvalMode: vi.fn(),
    togglePipcount: vi.fn(),
    reloadAllPositions: vi.fn(),
    loadRandomPosition: vi.fn(),
    showDatesAndMetadata: vi.fn()
}));

const { handleKeyDown } = await import('../services/keyboardService.js');
const { activeTabStore, statusBarModeStore } = await import('../stores/uiStore.js');

function ctrl(key) {
    const event = new KeyboardEvent('keydown', { key, code: `Key${key.toUpperCase()}`, ctrlKey: true, cancelable: true, bubbles: true });
    handleKeyDown(event);
    return event;
}

describe('CTRL-S on the Eval tab (#399)', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        activeTabStore.set('eval');
        statusBarModeStore.set('EVAL');
        document.body.innerHTML = '<section class="eval-panel" tabindex="-1"><div class="badges-strip"><button class="add-position">+</button></div></section>';
    });
    afterEach(() => {
        document.body.innerHTML = '';
        activeTabStore.set('matches');
        statusBarModeStore.set('NORMAL');
    });

    test('reaches the save with focus on the board', () => {
        ctrl('s');
        expect(saveCurrentPosition).toHaveBeenCalledTimes(1);
    });

    test('reaches the save with focus inside the Eval panel', () => {
        /** @type {HTMLElement} */ (document.querySelector('.add-position')).focus();
        ctrl('s');
        expect(saveCurrentPosition).toHaveBeenCalledTimes(1);
        expect(updatePosition).not.toHaveBeenCalled();
    });
});
