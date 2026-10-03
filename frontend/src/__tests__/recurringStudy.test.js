import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    CreateCollection: vi.fn(() => Promise.resolve(5)),
    AddPositionsToCollection: vi.fn(() => Promise.resolve()),
    CreateStudyDeck: vi.fn(() => Promise.resolve(1)),
    StudyPositionIDs: vi.fn(() => Promise.resolve([7, 8]))
}));
vi.mock('../services/positionLoader.js', () => ({ loadPositionsFromSelection: vi.fn(() => Promise.resolve()) }));
vi.mock('../services/trainingTabService.js', () => ({ startTrainingSession: vi.fn(() => Promise.resolve(true)) }));
vi.mock('../services/ankiService.js', () => ({ loadDecks: vi.fn(() => Promise.resolve()) }));

import { quizOnIds, quizOnWorstGroups, deckFromIds, collectionFromIds, WORST_QUIZ_SIZE } from '../services/recurringStudy.js';
import { CreateCollection, AddPositionsToCollection, CreateStudyDeck, StudyPositionIDs } from '../../wailsjs/go/database/Database.js';
import { loadPositionsFromSelection } from '../services/positionLoader.js';
import { startTrainingSession } from '../services/trainingTabService.js';
import { loadDecks } from '../services/ankiService.js';
import { activeTabStore } from '../stores/uiStore.js';

beforeEach(() => vi.clearAllMocks());

describe('recurringStudy', () => {
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

    test('the worst-groups quiz asks the engine for the draw and starts it', async () => {
        await quizOnWorstGroups();
        expect(StudyPositionIDs).toHaveBeenCalledWith(expect.anything(), 0, WORST_QUIZ_SIZE);
        expect(loadPositionsFromSelection).toHaveBeenCalledWith([7, 8]);
    });

    test('a deck is made by the backend, then the deck list is reloaded', async () => {
        await deckFromIds('D', [1, 2]);
        expect(CreateStudyDeck).toHaveBeenCalledWith('D', [1, 2]);
        expect(loadDecks).toHaveBeenCalled();
    });

    test('a collection is created, then filled', async () => {
        await collectionFromIds('C', [1, 2]);
        expect(CreateCollection).toHaveBeenCalledWith('C', '');
        expect(AddPositionsToCollection).toHaveBeenCalledWith(5, [1, 2]);
    });
});
