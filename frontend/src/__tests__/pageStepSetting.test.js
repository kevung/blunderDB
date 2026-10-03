import { describe, test, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';

// The PageUp / PageDown step chosen in the Settings is applied at once, persisted, and read
// back at start-up; an unknown stored value falls back to the default.

const config = vi.hoisted(() => ({ GetPageStep: vi.fn(), SavePageStep: vi.fn(async () => {}) }));
vi.mock('../../wailsjs/go/main/Config.js', () => config);

const { initPageStep, setPageStep } = await import('../services/pageStepSetting.js');
const { pageStepStore, PAGE_STEP_DEFAULT } = await import('../stores/uiStore.js');

beforeEach(() => {
    pageStepStore.set(PAGE_STEP_DEFAULT);
    config.SavePageStep.mockClear();
});

describe('page step setting', () => {
    test('a chosen step is applied and persisted', () => {
        setPageStep('10%');
        expect(get(pageStepStore)).toBe('10%');
        expect(config.SavePageStep).toHaveBeenCalledWith('10%');
    });

    test('start-up reads the persisted step', async () => {
        config.GetPageStep.mockResolvedValueOnce('500');
        await initPageStep();
        expect(get(pageStepStore)).toBe('500');
    });

    test('an unknown value falls back to the default', async () => {
        config.GetPageStep.mockResolvedValueOnce('7');
        await initPageStep();
        expect(get(pageStepStore)).toBe(PAGE_STEP_DEFAULT);
        setPageStep('abc');
        expect(config.SavePageStep).toHaveBeenCalledWith(PAGE_STEP_DEFAULT);
    });
});
