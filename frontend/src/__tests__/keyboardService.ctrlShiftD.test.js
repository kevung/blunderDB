import { describe, test, expect, vi, beforeEach } from 'vitest';

// #443 a retiré CTRL-MAJ-D (doublon de CTRL-Y) ; la branche CTRL-D, qui n'exclut pas MAJ,
// l'a alors reçu et ouvrait Stats. CTRL-MAJ-D ne fait plus rien.

const toggleMatchPanel = vi.fn();
const toggleSearchPanel = vi.fn();
const toggleStatsPanel = vi.fn();
const toggleTournamentPanel = vi.fn();

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

function ctrl(key, extra = {}) {
    const event = new KeyboardEvent('keydown', { key, code: `Key${key.toUpperCase()}`, ctrlKey: true, cancelable: true, bubbles: true, ...extra });
    handleKeyDown(event);
    return event;
}

describe('CTRL-D et CTRL-MAJ-D', () => {
    beforeEach(() => {
        vi.clearAllMocks();
        document.body.innerHTML = '';
    });

    test('CTRL-D ouvre Stats', () => {
        ctrl('d');
        expect(toggleStatsPanel).toHaveBeenCalledTimes(1);
    });

    test('CTRL-MAJ-D n’ouvre pas Stats', () => {
        ctrl('D', { shiftKey: true });
        expect(toggleStatsPanel).not.toHaveBeenCalled();
        expect(toggleTournamentPanel).not.toHaveBeenCalled();
    });
});
