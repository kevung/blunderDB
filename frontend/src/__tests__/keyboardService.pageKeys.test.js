import { describe, test, expect, vi, beforeEach } from 'vitest';

// Home / End go to the first / last position; PageUp / PageDown jump a page
// (positionNavigation.pagePosition) instead of going to the ends.

const firstPosition = vi.fn();
const lastPosition = vi.fn();
const pagePosition = vi.fn();
const toggleMatchPanel = vi.fn();
const toggleSearchPanel = vi.fn();
const toggleStatsPanel = vi.fn();
const toggleTournamentPanel = vi.fn();

vi.mock('../services/positionNavigation.js', () => ({ pagePosition }));
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
    firstPosition,
    previousPosition: vi.fn(),
    nextPosition: vi.fn(),
    lastPosition,
    updatePosition: vi.fn(),
    toggleAnalysisPanel: vi.fn(),
    toggleCommentPanel: vi.fn(),
    toggleMetadataPanel: vi.fn(),
    toggleAnkiPanel: vi.fn(),
    toggleCollectionPanelAction: vi.fn(),
    toggleMatchPanel,
    toggleTournamentPanel,
    toggleStatsPanel,
    toggleSearchPanel,
    toggleEvalMode: vi.fn(),
    togglePipcount: vi.fn(),
    reloadAllPositions: vi.fn(),
    loadRandomPosition: vi.fn(),
    showDatesAndMetadata: vi.fn()
}));

const { handleKeyDown } = await import('../services/keyboardService.js');

function press(key) {
    const event = new KeyboardEvent('keydown', { key, code: key, cancelable: true, bubbles: true });
    handleKeyDown(event);
    return event;
}

describe('position-list keys', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        document.body.innerHTML = '';
    });

    test('Home and End reach the ends', () => {
        press('Home');
        expect(firstPosition).toHaveBeenCalledTimes(1);
        press('End');
        expect(lastPosition).toHaveBeenCalledTimes(1);
        expect(pagePosition).not.toHaveBeenCalled();
    });

    test('PageUp and PageDown page through the list', () => {
        press('PageUp');
        press('PageDown');
        expect(pagePosition.mock.calls).toEqual([[-1], [1]]);
        expect(firstPosition).not.toHaveBeenCalled();
        expect(lastPosition).not.toHaveBeenCalled();
    });

    test('in a text field Home/End stay the field’s', () => {
        const input = document.createElement('input');
        document.body.appendChild(input);
        input.focus();
        press('Home');
        expect(firstPosition).not.toHaveBeenCalled();
    });
});
