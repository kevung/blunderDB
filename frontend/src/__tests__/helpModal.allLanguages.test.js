/**
 * The help opens in every language: each compressed bundle is fetched, inflated and
 * exposes its four tabs through the `help` store.
 */
import { test, expect, vi } from 'vitest';
import { get } from 'svelte/store';
import { stubHelpFetch } from './helpers/helpBundles.js';
import { LOCALES } from '../i18n/index.js';

vi.mock('../../wailsjs/go/main/Config.js', () => ({ SaveLanguage: vi.fn() }));

test.each(LOCALES)('the %s help bundle loads and fills every tab', async (lang) => {
    vi.resetModules();
    stubHelpFetch(vi);
    const { loadHelpFor, help } = await import('../i18n/help/index.js');
    const { setLanguage } = await import('../i18n');
    await setLanguage(lang);
    await loadHelpFor(lang);
    const h = get(help);
    expect(h.failed).toBe(false);
    expect(h.ready).toBe(true);
    for (const tab of ['manual', 'shortcuts', 'commands', 'about']) {
        expect(h[tab].length, `${lang}/${tab}`).toBeGreaterThan(100);
    }
});
