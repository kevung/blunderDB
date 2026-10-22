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
        const { positionContext } = await import('../services/positionActions.js');
        const { setPositionStudied } = await import('../services/studyQueueService.js');
        expect(positionContext({ positionId: 7 }).studiedMark).toBeUndefined();
        positionStore.set(/** @type {any} */ ({ id: 7 }));
        await vi.waitFor(() => expect(positionContext({ positionId: 7 }).studiedMark).toBe(true));
        await setPositionStudied(7, false);
        expect(positionContext({ positionId: 7 }).studiedMark).toBe(false);
    }, 30000);
});
