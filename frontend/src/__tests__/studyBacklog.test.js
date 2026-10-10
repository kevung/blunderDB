import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

const StudyBacklog = vi.fn();
const SetPositionStudied = vi.fn();
const ImportStudyQueue = vi.fn();

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    StudyBacklog: (/** @type {any[]} */ ...a) => StudyBacklog(...a),
    SetPositionStudied: (/** @type {any[]} */ ...a) => SetPositionStudied(...a),
    ImportStudyQueue: (/** @type {any[]} */ ...a) => ImportStudyQueue(...a)
}));
vi.mock('../services/importService.js', () => ({ showImportedPosition: vi.fn().mockResolvedValue(undefined) }));
vi.mock('../services/databaseService.js', () => ({ setStatusBarMessage: vi.fn() }));

import { startStudyBacklog, startStudyQueue, markCurrentStudied, unmarkLastStudied } from '../services/studyQueueService.js';
import { studyQueueStore, studyQueueIndexStore, studyQueueActiveStore, studyQueueBacklogStore, studyQueueLastMarkedStore } from '../stores/studyQueueStore.js';

const entries = [
    { positionId: 11, matchId: 1, reason: 'backlog', label: 'A - B', errorMp: 300, isCube: false },
    { positionId: 12, matchId: 2, reason: 'backlog', label: 'C - D', errorMp: 120, isCube: true }
];

beforeEach(() => {
    vi.clearAllMocks();
    studyQueueStore.set([]);
    studyQueueIndexStore.set(0);
    studyQueueActiveStore.set(false);
    studyQueueBacklogStore.set(false);
    studyQueueLastMarkedStore.set(0);
});

describe('file transversale', () => {
    test('démarre sur la position la plus coûteuse et se déclare transversale', async () => {
        StudyBacklog.mockResolvedValue(entries);
        expect(await startStudyBacklog()).toBe(true);
        expect(get(studyQueueBacklogStore)).toBe(true);
        expect(get(studyQueueActiveStore)).toBe(true);
        expect(get(studyQueueStore)).toHaveLength(2);
    });

    test("une file vide n'ouvre rien", async () => {
        StudyBacklog.mockResolvedValue([]);
        expect(await startStudyBacklog()).toBe(false);
        expect(get(studyQueueActiveStore)).toBe(false);
    });

    test('marquer vu écrit la marque, retient la position et avance', async () => {
        StudyBacklog.mockResolvedValue(entries);
        SetPositionStudied.mockResolvedValue(undefined);
        await startStudyBacklog();
        await markCurrentStudied();
        expect(SetPositionStudied).toHaveBeenCalledWith(11, true);
        expect(get(studyQueueLastMarkedStore)).toBe(11);
        expect(get(studyQueueIndexStore)).toBe(1);
    });

    test('démarquer retire la dernière marque', async () => {
        StudyBacklog.mockResolvedValue(entries);
        SetPositionStudied.mockResolvedValue(undefined);
        await startStudyBacklog();
        await markCurrentStudied();
        await unmarkLastStudied();
        expect(SetPositionStudied).toHaveBeenLastCalledWith(11, false);
        expect(get(studyQueueLastMarkedStore)).toBe(0);
    });

    test("la file d'un lot n'offre pas la marque", async () => {
        ImportStudyQueue.mockResolvedValue([{ ...entries[0], reason: 'blunder' }]);
        studyQueueBacklogStore.set(true);
        await startStudyQueue(5);
        expect(get(studyQueueBacklogStore)).toBe(false);
    });
});
