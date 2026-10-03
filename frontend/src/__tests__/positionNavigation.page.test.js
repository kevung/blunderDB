import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// PageUp / PageDown move by pageStepStore positions, clamp at the ends, and keep
// their previous/next-game meaning inside a match.

vi.mock('../../wailsjs/go/database/Database.js', () => ({ SaveLastVisitedPosition: vi.fn(async () => {}) }));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));
vi.mock('../services/positionService.js', () => ({ showPosition: vi.fn(async () => {}) }));

const { pagePosition } = await import('../services/positionNavigation.js');
const { positionsStore, matchContextStore } = await import('../stores/positionStore.js');
const { currentPositionIndexStore, pageStepStore, pageStepSize, statusBarModeStore } = await import('../stores/uiStore.js');
const { databasePathStore } = await import('../stores/databaseStore.js');

beforeEach(() => {
    databasePathStore.set('/tmp/a.db');
    statusBarModeStore.set('NORMAL');
    matchContextStore.set({ isMatchMode: false, matchID: null, movePositions: [], currentIndex: 0 });
    pageStepStore.set(100);
    positionsStore.set(Array.from({ length: 250 }, (_, i) => ({ id: i + 1 })));
    currentPositionIndexStore.set(120);
});

describe('pageStepSize', () => {
    test('a count, or a percentage of the list, never below one', () => {
        expect(pageStepSize('100', 5000)).toBe(100);
        expect(pageStepSize(50, 5000)).toBe(50);
        expect(pageStepSize('10%', 1_000_000)).toBe(100_000);
        expect(pageStepSize('10%', 4)).toBe(1);
        expect(pageStepSize('', 10)).toBe(1);
        expect(pageStepSize('abc%', 10)).toBe(1);
    });
});

describe('pagePosition', () => {
    test('jumps a page and stops at the ends', async () => {
        await pagePosition(1);
        expect(get(currentPositionIndexStore)).toBe(220);
        await pagePosition(1);
        expect(get(currentPositionIndexStore)).toBe(249);
        await pagePosition(-1);
        expect(get(currentPositionIndexStore)).toBe(149);
        currentPositionIndexStore.set(30);
        await pagePosition(-1);
        expect(get(currentPositionIndexStore)).toBe(0);
    });

    test('the page size follows pageStepStore', async () => {
        pageStepStore.set(10);
        await pagePosition(1);
        expect(get(currentPositionIndexStore)).toBe(130);
    });

    test('a percentage step jumps that share of the list', async () => {
        pageStepStore.set('10%');
        await pagePosition(1);
        expect(get(currentPositionIndexStore)).toBe(145);
    });

    test('does nothing while editing', async () => {
        statusBarModeStore.set('EDIT');
        await pagePosition(1);
        expect(get(currentPositionIndexStore)).toBe(120);
    });
});
