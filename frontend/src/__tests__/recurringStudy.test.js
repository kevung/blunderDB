import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    CreateCollection: vi.fn(() => Promise.resolve(5)),
    AddPositionsToCollection: vi.fn(() => Promise.resolve())
}));
vi.mock('../services/positionLoader.js', () => ({ loadPositionsFromSelection: vi.fn(() => Promise.resolve()) }));
vi.mock('../services/trainingTabService.js', () => ({ startTrainingSession: vi.fn(() => Promise.resolve(true)) }));
vi.mock('../services/ankiService.js', () => ({ createDeck: vi.fn(() => Promise.resolve(1)) }));

import { idsOfGroups, drawSample, quizOnIds, quizOnWorstGroups, deckFromIds, collectionFromIds, WORST_QUIZ_SIZE } from '../services/recurringStudy.js';
import { CreateCollection, AddPositionsToCollection } from '../../wailsjs/go/database/Database.js';
import { loadPositionsFromSelection } from '../services/positionLoader.js';
import { startTrainingSession } from '../services/trainingTabService.js';
import { createDeck } from '../services/ankiService.js';
import { activeTabStore } from '../stores/uiStore.js';

beforeEach(() => vi.clearAllMocks());

describe('recurringStudy', () => {
    test('the ids of several groups are distinct and keep the ranking order', () => {
        expect(idsOfGroups([{ PositionIDs: [3, 1] }, { PositionIDs: [1, 2] }, {}])).toEqual([3, 1, 2]);
    });

    test('a sample is bounded, distinct, and leaves its input alone', () => {
        const items = Array.from({ length: 50 }, (_, i) => i);
        const sample = drawSample(items, 20);
        expect(sample).toHaveLength(20);
        expect(new Set(sample).size).toBe(20);
        expect(items[0]).toBe(0);
        expect(drawSample([1, 2], 20)).toHaveLength(2);
    });

    test('a quiz loads the ids as the browsed list, then starts the Decision exercise', async () => {
        await quizOnIds([4, 5]);
        expect(loadPositionsFromSelection).toHaveBeenCalledWith([4, 5]);
        expect(get(activeTabStore)).toBe('training');
        expect(startTrainingSession).toHaveBeenCalledWith({ exercise: 'decision', seedSource: 'library' });
    });

    test('an empty group starts nothing', async () => {
        expect(await quizOnIds([])).toBe(false);
        expect(startTrainingSession).not.toHaveBeenCalled();
    });

    test('the worst-groups quiz draws twenty positions from the first three groups only', async () => {
        const group = (from) => ({ PositionIDs: Array.from({ length: 15 }, (_, i) => from + i) });
        await quizOnWorstGroups([group(0), group(100), group(200), group(300)]);
        const ids = vi.mocked(loadPositionsFromSelection).mock.calls[0][0];
        expect(ids).toHaveLength(WORST_QUIZ_SIZE);
        expect(ids.every((id) => id < 300)).toBe(true);
    });

    test('a deck is a search deck holding exactly the ids', async () => {
        await deckFromIds('D', [1, 2]);
        expect(createDeck).toHaveBeenCalledWith({ name: 'D', sourceType: 'search', sourceId: 0, positionIds: [1, 2] });
    });

    test('a collection is created, then filled', async () => {
        await collectionFromIds('C', [1, 2]);
        expect(CreateCollection).toHaveBeenCalledWith('C', '');
        expect(AddPositionsToCollection).toHaveBeenCalledWith(5, [1, 2]);
    });
});
