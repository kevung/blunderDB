import { describe, test, expect, vi } from 'vitest';

const compute = vi.hoisted(() => vi.fn());
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ComputeTrainingStats: compute,
    ComputeStats: vi.fn(),
    ComputeRecurringErrors: vi.fn(),
    GetPlayerStatsTable: vi.fn()
}));

import { refreshTrainingStats, invalidateTrainingStats } from '../stores/statsStore.js';

describe("la série d'entraînement", () => {
    test('une lecture identique est servie du cache, une invalidation la relit', async () => {
        compute.mockResolvedValue({ Periods: [] });
        await refreshTrainingStats({}, 'k', 'week');
        await refreshTrainingStats({}, 'k', 'week');
        expect(compute).toHaveBeenCalledTimes(1);
        invalidateTrainingStats();
        await refreshTrainingStats({}, 'k', 'week');
        expect(compute).toHaveBeenCalledTimes(2);
    });
});
