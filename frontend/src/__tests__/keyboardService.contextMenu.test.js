import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';

// Un menu contextuel ouvert garde les flèches : elles passent d'une entrée à l'autre, jamais
// à la position du plateau cachée derrière.

const { nextPosition } = vi.hoisted(() => ({ nextPosition: vi.fn() }));

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
    nextPosition,
    lastPosition: vi.fn(),
    updatePosition: vi.fn(),
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
const { activeTabStore } = await import('../stores/uiStore.js');

describe('flèches et menu contextuel', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        activeTabStore.set('matches');
        document.body.innerHTML = '';
    });
    afterEach(() => {
        document.body.innerHTML = '';
    });

    const press = (key) => handleKeyDown(new KeyboardEvent('keydown', { key, code: key, cancelable: true, bubbles: true }));

    test('sans menu, la flèche droite parcourt le plateau', () => {
        press('ArrowRight');
        expect(nextPosition).toHaveBeenCalled();
    });

    test('avec un menu ouvert, elle ne le parcourt pas', () => {
        document.body.innerHTML = '<div class="context-menu" role="menu"><button>x</button></div>';
        press('ArrowRight');
        expect(nextPosition).not.toHaveBeenCalled();
    });
});
