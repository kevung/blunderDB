import { describe, test, expect, vi, beforeEach } from 'vitest';

const IsPositionStudied = vi.fn();
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ImportStudyQueue: vi.fn(),
    StudyBacklog: vi.fn(),
    StudyPlanQueue: vi.fn(),
    SetPositionStudied: vi.fn().mockResolvedValue(undefined),
    IsPositionStudied
}));

describe('positionContext studiedMark', () => {
    beforeEach(() => {
        vi.resetModules();
        IsPositionStudied.mockReset();
    });

    test('is read from the library once a position is on the board, and follows a toggle', async () => {
        IsPositionStudied.mockResolvedValue(true);
        const { positionStore } = await import('../stores/positionStore');
        const { databasePathStore } = await import('../stores/databaseStore.js');
        databasePathStore.set('/lib.db');
        const { positionContext } = await import('../services/positionActions.js');
        const { setPositionStudied } = await import('../services/studyQueueService.js');
        expect(positionContext({ positionId: 7 }).studiedMark).toBeUndefined();
        positionStore.set(/** @type {any} */ ({ id: 7 }));
        await vi.waitFor(() => expect(positionContext({ positionId: 7 }).studiedMark).toBe(true));
        await setPositionStudied(7, false);
        expect(positionContext({ positionId: 7 }).studiedMark).toBe(false);
        expect(IsPositionStudied).toHaveBeenCalledTimes(1);
        // Another library: the marks are forgotten.
        databasePathStore.set('/other.db');
        expect(positionContext({ positionId: 7 }).studiedMark).toBeUndefined();
    }, 30000);
});

describe('refreshStudied', () => {
    beforeEach(() => {
        vi.resetModules();
        IsPositionStudied.mockReset();
    });

    test('reads nothing without a library and does not log a failing read as an error', async () => {
        const { databasePathStore } = await import('../stores/databaseStore.js');
        const { refreshStudied } = await import('../services/studyQueueService.js');
        const { logger } = await import('../utils/logger.js');
        const error = vi.spyOn(logger, 'error').mockImplementation(() => {});
        databasePathStore.set('');
        expect(await refreshStudied(3)).toBeUndefined();
        expect(IsPositionStudied).not.toHaveBeenCalled();
        databasePathStore.set('/lib.db');
        IsPositionStudied.mockRejectedValue(new Error('unsupported'));
        expect(await refreshStudied(3)).toBeUndefined();
        expect(error).not.toHaveBeenCalled();
    }, 30000);
});
