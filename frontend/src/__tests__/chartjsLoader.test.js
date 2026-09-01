import { describe, test, expect, vi, beforeEach } from 'vitest';

// chart.js is loaded lazily by the stats charts; the loader must fetch it once,
// register the pieces the charts use, and recover from a failed load.

const register = vi.fn();
const REGISTRABLES = ['LineController', 'BarController', 'ScatterController', 'LineElement', 'BarElement', 'PointElement', 'LinearScale', 'CategoryScale', 'Tooltip', 'Legend', 'Filler'];
const fakeChartJs = () => ({ Chart: { register }, ...Object.fromEntries(REGISTRABLES.map((n) => [n, n])) });

vi.mock('../utils/logger.js', () => ({ logger: { error: vi.fn() } }));

describe('loadChart', () => {
    beforeEach(() => {
        vi.resetModules();
        vi.doUnmock('chart.js');
        register.mockClear();
    });

    test('resolves the Chart class, registers once, and memoises the promise', async () => {
        vi.doMock('chart.js', fakeChartJs);
        const { loadChart } = await import('../components/stats/charts/chartjs.js');
        const first = loadChart();
        const second = loadChart();
        expect(second).toBe(first);
        const Chart = await first;
        expect(Chart).toEqual({ register });
        expect(register).toHaveBeenCalledTimes(1);
        for (const name of REGISTRABLES) expect(register.mock.calls[0]).toContain(name);
        await loadChart();
        expect(register).toHaveBeenCalledTimes(1);
    });

    test('a failed load resolves to null, is logged, and the next call retries', async () => {
        vi.doMock('chart.js', () => {
            throw new Error('chunk unavailable');
        });
        const { loadChart } = await import('../components/stats/charts/chartjs.js');
        const { logger } = await import('../utils/logger.js');
        await expect(loadChart()).resolves.toBeNull();
        expect(logger.error).toHaveBeenCalledTimes(1);
        expect(register).not.toHaveBeenCalled();

        vi.doMock('chart.js', fakeChartJs);
        await expect(loadChart()).resolves.toEqual({ register });
        expect(register).toHaveBeenCalledTimes(1);
    });
});
