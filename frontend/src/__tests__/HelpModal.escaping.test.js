/**
 * HelpModal.escaping.test.js
 *
 * D.15: the About tab splices `applicationVersion` (metaStore) and `databaseVersion`
 * (GetDatabaseVersion()) into the help corpus's raw HTML before rendering it via
 * {@html}. Both are trusted today, but the interpolation itself must not become an
 * XSS hole if either value is ever attacker-influenced (e.g. a corrupted .db file
 * feeding databaseVersion). This guards that the values are HTML-escaped: a version
 * string containing markup renders as inert text, never as a live element/attribute.
 */

import { test, expect, vi, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';

vi.mock('../../wailsjs/go/database/Database', () => ({
    GetDatabaseVersion: vi.fn(() => Promise.resolve('<img src=x onerror=alert(1)>'))
}));

import { metaStore } from '../stores/metaStore.js';
import HelpModal from '../components/HelpModal.svelte';

afterEach(() => {
    cleanup();
    metaStore.set({ applicationVersion: '0.35.0' });
});

test('malicious databaseVersion/applicationVersion render as inert text in the About tab', async () => {
    metaStore.set({ applicationVersion: '<script>window.__pwned = true</script>' });

    render(HelpModal, { visible: true, onClose: () => {} });
    await tick();
    // GetDatabaseVersion() resolves asynchronously in onMount.
    await tick();
    await tick();

    screen.getByRole('button', { name: /about/i }).click();
    await tick();

    // No live <img> or <script> reached the DOM — the corpus's own markup
    // (the enclosing <p> tags) still rendered as real HTML, but the interpolated
    // values did not.
    expect(document.querySelector('.tab-content img')).toBeNull();
    expect(document.querySelector('.tab-content script')).toBeNull();
    expect(window.__pwned).toBeUndefined();

    // The escaped source text is still visible to the user, just as text, not markup.
    expect(document.querySelector('.tab-content').textContent).toContain('<img src=x onerror=alert(1)>');
    expect(document.querySelector('.tab-content').textContent).toContain('<script>window.__pwned = true</script>');
});
