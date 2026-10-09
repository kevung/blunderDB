import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';

// j/k browse positions from anywhere but a panel that walks its own rows. The challenge hides
// the Analysis panel's rows, so a move selected there must not keep them for an invisible list.

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
    toggleEvalMode: vi.fn(),
    togglePipcount: vi.fn(),
    reloadAllPositions: vi.fn(),
    loadRandomPosition: vi.fn()
}));

const { handleKeyDown } = await import('../services/keyboardService.js');
const { nextPosition, previousPosition } = await import('../services/positionService.js');
const { activeTabStore, activeModal } = await import('../stores/uiStore.js');
const { selectedMoveStore } = await import('../stores/analysisStore.js');
const { analysisChallengeStore, toggleAnalysisChallenge } = await import('../stores/analysisChallengeStore.js');
const { get } = await import('svelte/store');

/** @param {string} key */
function press(key) {
    const event = new KeyboardEvent('keydown', { key, cancelable: true, bubbles: true });
    handleKeyDown(event);
    return event;
}

describe.each([
    ['challenge off', false],
    ['challenge on', true]
])('j/k browse positions, %s', (_label, challenge) => {
    beforeEach(() => {
        vi.clearAllMocks();
        activeTabStore.set('analysis');
        activeModal.set(null);
        selectedMoveStore.set(null);
        analysisChallengeStore.set(false);
        document.body.innerHTML = '<section class="analysis-panel" id="analysisPanel" tabindex="-1"></section><div id="elsewhere" tabindex="-1"></div>';
    });

    afterEach(() => {
        selectedMoveStore.set(null);
        analysisChallengeStore.set(false);
        document.body.innerHTML = '';
    });

    test.each([
        ['focused', 'analysisPanel'],
        ['not focused', 'elsewhere']
    ])('Analysis panel %s', (_f, focusId) => {
        if (challenge) toggleAnalysisChallenge();
        document.getElementById(focusId)?.focus();
        press('j');
        press('k');
        expect(nextPosition).toHaveBeenCalledTimes(1);
        expect(previousPosition).toHaveBeenCalledTimes(1);
    });

    test.each([
        ['focused', 'analysisPanel'],
        ['not focused', 'elsewhere']
    ])('a move selected before, Analysis panel %s', (_f, focusId) => {
        selectedMoveStore.set('24/23 13/11');
        if (challenge) toggleAnalysisChallenge();
        document.getElementById(focusId)?.focus();
        press('j');
        press('k');
        // Unmasked, the selection walks the visible list; masked, there is no list to walk.
        expect(nextPosition).toHaveBeenCalledTimes(challenge ? 1 : 0);
        expect(previousPosition).toHaveBeenCalledTimes(challenge ? 1 : 0);
    });
});

describe('m with the Analysis panel focused', () => {
    afterEach(() => {
        analysisChallengeStore.set(false);
        document.body.innerHTML = '';
    });

    test('toggles the challenge', () => {
        analysisChallengeStore.set(false);
        activeModal.set(null);
        document.body.innerHTML = '<section class="analysis-panel" id="analysisPanel" tabindex="-1"></section>';
        document.getElementById('analysisPanel')?.focus();
        press('m');
        expect(get(analysisChallengeStore)).toBe(true);
        press('m');
        expect(get(analysisChallengeStore)).toBe(false);
    });
});
