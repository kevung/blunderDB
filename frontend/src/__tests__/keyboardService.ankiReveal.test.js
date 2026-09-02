/**
 * keyboardService.ankiReveal.test.js — Space reveals a review card's answer.
 *
 * Space opens the command line everywhere else in the app; the review guard
 * returns before that branch, so inside a review it was dead and is now the
 * reveal (ADR-0025 rule 3). It deliberately gets no second meaning once the
 * answer is shown — real Anki grades "Good" on the second press, and a
 * mistaken grade durably pollutes the FSRS schedule.
 */
import { describe, test, expect, vi, beforeEach } from 'vitest';

vi.mock('../services/clipboardService.js', () => ({ copyPosition: vi.fn(), copyBoardImage: vi.fn(), copyBoardWithAnalysisImage: vi.fn() }));
vi.mock('../services/importService.js', () => ({ pastePosition: vi.fn(), importDatabase: vi.fn(), importPosition: vi.fn(), importFolder: vi.fn() }));
vi.mock('../services/exportService.js', () => ({ exportDatabase: vi.fn() }));
vi.mock('../services/databaseService.js', () => ({ newDatabase: vi.fn(), openDatabase: vi.fn(), exitApp: vi.fn(), setStatusBarMessage: vi.fn() }));
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
    toggleEPCMode: vi.fn(),
    togglePipcount: vi.fn(),
    reloadAllPositions: vi.fn(),
    loadRandomPosition: vi.fn()
}));

const { handleKeyDown } = await import('../services/keyboardService.js');
const { showCommandInputStore, activeTabStore, activeModal } = await import('../stores/uiStore.js');
const { ankiViewModeStore, ankiAnswerShownStore, ankiReviewActionStore } = await import('../stores/ankiStore.js');
const { get } = await import('svelte/store');

function press(init) {
    const event = new KeyboardEvent('keydown', { cancelable: true, bubbles: true, ...init });
    handleKeyDown(event);
    return event;
}

beforeEach(() => {
    activeModal.set(null);
    showCommandInputStore.set(false);
    ankiAnswerShownStore.set(false);
    ankiReviewActionStore.set(null);
});

describe('Space during an Anki review', () => {
    beforeEach(() => {
        activeTabStore.set('anki');
        ankiViewModeStore.set('review');
    });

    test('reveals the answer instead of opening the command line', () => {
        const event = press({ code: 'Space', key: ' ' });
        expect(get(ankiAnswerShownStore)).toBe(true);
        expect(get(showCommandInputStore)).toBe(false);
        expect(event.defaultPrevented).toBe(true);
    });

    test('pressing it again does not grade the card', () => {
        press({ code: 'Space', key: ' ' });
        press({ code: 'Space', key: ' ' });
        expect(get(ankiReviewActionStore)).toBeNull();
        expect(get(ankiAnswerShownStore)).toBe(true);
    });

    test('the grading keys still work with the answer hidden', () => {
        press({ code: 'Digit1', key: '1' });
        expect(get(ankiReviewActionStore)).toBe(1);
        expect(get(ankiAnswerShownStore)).toBe(false);
    });
});

describe('Space outside a review', () => {
    test('still opens the command line on the Anki tab, deck list showing', () => {
        activeTabStore.set('anki');
        ankiViewModeStore.set('list');
        press({ code: 'Space', key: ' ' });
        expect(get(ankiAnswerShownStore)).toBe(false);
        expect(get(showCommandInputStore)).toBe(true);
    });

    test('a review left open on another tab does not capture Space', () => {
        activeTabStore.set('analysis');
        ankiViewModeStore.set('review');
        press({ code: 'Space', key: ' ' });
        expect(get(ankiAnswerShownStore)).toBe(false);
        expect(get(showCommandInputStore)).toBe(true);
    });
});
