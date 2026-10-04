import { describe, test, expect, vi, beforeEach } from 'vitest';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    LoadTrainingMissed: vi.fn(() => Promise.resolve([12, 7]))
}));
vi.mock('../i18n', () => ({ tMsg: (/** @type {string} */ key, /** @type {any} */ values) => `${key} ${JSON.stringify(values ?? {})}` }));
vi.mock('../services/recurringStudy.js', () => ({
    quizOnIds: vi.fn(() => Promise.resolve(true)),
    deckFromIds: vi.fn(() => Promise.resolve(true)),
    collectionFromIds: vi.fn(() => Promise.resolve(true))
}));

import { retakeMissed, missedToDeck, missedToCollection, missedStudyName } from '../services/trainingMissed.js';
import { LoadTrainingMissed } from '../../wailsjs/go/database/Database.js';
import { quizOnIds, deckFromIds, collectionFromIds } from '../services/recurringStudy.js';

beforeEach(() => vi.clearAllMocks());

describe('trainingMissed', () => {
    test('reprendre mes ratés relance Décision sur les positions ratées, toutes sessions', async () => {
        await retakeMissed();
        expect(LoadTrainingMissed).toHaveBeenCalledWith({ exercise: 'decision', sessionId: 0, limit: 0 });
        expect(quizOnIds).toHaveBeenCalledWith([12, 7]);
    });

    test('un paquet et une collection des ratés portent un nom daté', async () => {
        await missedToDeck();
        await missedToCollection();
        expect(deckFromIds).toHaveBeenCalledWith(missedStudyName(), [12, 7]);
        expect(collectionFromIds).toHaveBeenCalledWith(missedStudyName(), [12, 7]);
        expect(missedStudyName(new Date('2026-10-04T12:00:00Z'))).toContain('2026-10-04');
    });

    test('sans raté, ni paquet ni collection', async () => {
        vi.mocked(LoadTrainingMissed).mockResolvedValueOnce([]).mockResolvedValueOnce([]);
        expect(await missedToDeck()).toBe(false);
        expect(await missedToCollection()).toBe(false);
        expect(deckFromIds).not.toHaveBeenCalled();
        expect(collectionFromIds).not.toHaveBeenCalled();
    });
});
