/**
 * A help bundle that fails to load falls back on English; when English fails too the modal says so
 * instead of spinning forever. Neither rejects.
 */
import { test, expect, vi } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/main/Config.js', () => ({ SaveLanguage: vi.fn() }));

test('loadHelpFor never rejects and falls back', async () => {
    vi.resetModules();
    vi.doMock('../i18n/help/de.js', () => {
        throw new Error('chunk failed');
    });
    const { loadHelpFor, help } = await import('../i18n/help/index.js');
    const { setLanguage } = await import('../i18n');
    await setLanguage('de');
    await expect(loadHelpFor('de')).resolves.toBeUndefined();
    const h = get(help);
    expect(h.ready).toBe(true);
    expect(h.failed).toBe(false);
    expect(h.manual.length).toBeGreaterThan(100);
});

test('when English cannot load either, help.failed is set', async () => {
    vi.resetModules();
    vi.doMock('../i18n/help/en.js', () => {
        throw new Error('chunk failed');
    });
    const { loadHelpFor, help } = await import('../i18n/help/index.js');
    await expect(loadHelpFor('en')).resolves.toBeUndefined();
    expect(get(help).failed).toBe(true);
    expect(get(help).ready).toBe(false);
});
