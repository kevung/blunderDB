import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';

// The analysis panel focuses itself when it opens, so its focus branch in the
// global dispatcher is what the user hits first. That branch withheld every
// non-Ctrl key it did not recognise as navigation — including Space, which
// opens the command line, and '?', which opens help. Both are the app-wide
// escape hatches panelKeyGuard() guarantees from every other panel; the
// analysis panel is the one place they went missing.

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
    toggleTournamentPanel: vi.fn(),
    toggleStatsPanel: vi.fn(),
    toggleSearchPanel: vi.fn(),
    toggleEPCMode: vi.fn(),
    togglePipcount: vi.fn(),
    reloadAllPositions: vi.fn(),
    loadRandomPosition: vi.fn()
}));

const { handleKeyDown } = await import('../services/keyboardService.js');
const { showCommandInputStore, activeModal, activeTabStore, MODAL } = await import('../stores/uiStore.js');
const { selectedMoveStore } = await import('../stores/analysisStore.js');
const { get } = await import('svelte/store');

function press(init) {
    const event = new KeyboardEvent('keydown', { cancelable: true, bubbles: true, ...init });
    handleKeyDown(event);
    return event;
}

describe('global shortcuts with the analysis panel focused', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        activeTabStore.set('matches');
        showCommandInputStore.set(false);
        activeModal.set(null);
        selectedMoveStore.set(null);
        document.body.innerHTML = '<section class="analysis-panel" id="analysisPanel" tabindex="-1"></section>';
        document.getElementById('analysisPanel').focus();
    });

    afterEach(() => {
        showCommandInputStore.set(false);
        activeModal.set(null);
        selectedMoveStore.set(null);
        document.body.innerHTML = '';
    });

    test('Space opens the command line', () => {
        press({ key: ' ', code: 'Space' });
        expect(get(showCommandInputStore)).toBe(true);
    });

    test('Space still opens the command line while a move is selected', () => {
        selectedMoveStore.set('24/23 13/11');
        press({ key: ' ', code: 'Space' });
        expect(get(showCommandInputStore)).toBe(true);
    });

    test("'?' opens the help modal", () => {
        press({ key: '?' });
        expect(get(activeModal)).toBe(MODAL.HELP);
    });
});
