/**
 * Tests for logger.perf — instrumentation de performance.
 *
 * Vérifie que :
 * - perf() retourne la valeur de fn (sync et async)
 * - perf() ne logge pas en prod (DEV=false)
 * - perf() logge en dev avec seuil 0 (toute durée ≥ 0)
 */
import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { logger } from '../utils/logger.js';

describe('logger.perf', () => {
    beforeEach(() => {
        // Neutraliser l'API Performance pour des mesures prévisibles
        vi.spyOn(performance, 'mark').mockImplementation(() => {});
        vi.spyOn(performance, 'measure').mockReturnValue({ duration: 5 });
        vi.spyOn(performance, 'clearMarks').mockImplementation(() => {});
        vi.spyOn(performance, 'clearMeasures').mockImplementation(() => {});
    });

    afterEach(() => {
        vi.restoreAllMocks();
        vi.unstubAllEnvs();
    });

    test('Test 1 — retourne la valeur de fn synchrone', () => {
        const result = logger.perf('test-sync', () => 42);
        expect(result).toBe(42);
    });

    test('Test 2 — retourne une promesse résolue pour fn async', async () => {
        const result = logger.perf('test-async', async () => 42);
        await expect(result).resolves.toBe(42);
    });

    test('Test 3 — ne logge pas en prod (DEV=false)', () => {
        vi.stubEnv('DEV', false);
        const spy = vi.spyOn(console, 'log');
        logger.perf('test-prod', () => 42);
        expect(spy).not.toHaveBeenCalled();
    });

    test('Test 4 — logge en dev avec seuil 0 (durée > 0)', () => {
        vi.stubEnv('DEV', true);
        vi.stubEnv('VITE_PERF_THRESHOLD_MS', '0');
        const spy = vi.spyOn(console, 'log');
        // measure.duration = 5 (mocké), threshold = 0 → 5 >= 0 → log attendu
        logger.perf('test-threshold-zero', () => {});
        expect(spy).toHaveBeenCalledWith(expect.stringContaining('[perf] test-threshold-zero'));
    });
});
