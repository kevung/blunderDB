/**
 * A WebView without DecompressionStream (WebKitGTK 2.36, macOS 11) gets the help through the
 * Go binding HelpBundle, which hands over the inflated JSON.
 */
import { test, expect, vi } from 'vitest';
import { get } from 'svelte/store';
import { readHelpBundle } from './helpers/helpBundles.js';

vi.mock('../../wailsjs/go/main/Config.js', () => ({ SaveLanguage: vi.fn() }));
const HelpBundle = vi.fn(async (lang) => JSON.stringify(readHelpBundle(lang)));
vi.mock('../../wailsjs/go/gui/App.js', () => ({ HelpBundle: (/** @type {string} */ lang) => HelpBundle(lang) }));

test('without DecompressionStream the help comes from the Go binding', async () => {
    vi.stubGlobal('DecompressionStream', undefined);
    vi.stubGlobal('fetch', vi.fn());
    const { loadHelpFor, help } = await import('../i18n/help/index.js');
    const { setLanguage } = await import('../i18n');
    await setLanguage('de');
    await loadHelpFor('de');
    const h = get(help);
    expect(h.ready).toBe(true);
    expect(h.failed).toBe(false);
    expect(h.manual.length).toBeGreaterThan(100);
    expect(HelpBundle).toHaveBeenCalledWith('de');
    expect(fetch).not.toHaveBeenCalled();
    vi.unstubAllGlobals();
});
