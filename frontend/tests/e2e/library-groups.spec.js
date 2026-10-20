/**
 * library-groups.spec.js
 *
 * Deux bibliothèques ouvertes, un groupe d'onglets chacune. Ouvrir B dans un nouveau groupe
 * laisse A ouverte ; cliquer un onglet de A ramène sa bibliothèque et son onglet d'un geste, et
 * la bascule ne demande au backend que l'échange en mémoire avant l'affichage (mesuré).
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';

const pathB = '/libs/club.db';

test.beforeEach(async ({ page }) => {
    await installWailsMock(
        page,
        openLibraryMock({
            app: {
                OpenDatabaseDialog: pathB,
                LibraryGroupHolding: 0,
                OpenLibraryGroup: { id: 2, path: pathB, active: true, readOnly: false },
                ActivateLibraryGroup: null,
                CloseLibraryGroup: null
            }
        })
    );
    await page.goto('/');
    await expect(page.locator('[data-testid="status-bar"]')).toBeVisible({ timeout: 8000 });
    await page.waitForFunction(() => performance.getEntriesByName('bdb:mounted').length > 0);
});

test('a second library opens in its own group, and a tab of the first brings it back at once', async ({ page }) => {
    await page.locator('[data-testid="open-library-group"]').click();
    const headers = page.locator('[data-testid="library-group"]');
    await expect(headers).toHaveCount(2);
    await expect(headers.nth(1)).toHaveClass(/active/);
    await expect(headers.nth(1)).toContainText('club');

    // Click A's tab and time the switch up to the moment its header shows active, noting every
    // backend call made before that moment.
    const measure = await page.evaluate(
        () =>
            new Promise((resolve) => {
                const header = /** @type {HTMLElement} */ (document.querySelector('[data-testid="library-group"][data-group-id="1"]'));
                const tab = /** @type {HTMLElement} */ (document.querySelector('[role="tab"][data-group-id="1"]'));
                const callsBefore = window.__wailsCalls.length;
                const started = performance.now();
                const observer = new MutationObserver(() => {
                    if (!header.classList.contains('active')) return;
                    observer.disconnect();
                    resolve({ ms: performance.now() - started, calls: window.__wailsCalls.slice(callsBefore).map((c) => c.method) });
                });
                observer.observe(header, { attributes: true, attributeFilter: ['class'] });
                tab.click();
            })
    );
    // eslint-disable-next-line no-console -- the measure is the point of this test
    console.info(`library group switch: ${measure.ms.toFixed(1)} ms, calls before display: ${measure.calls.join(', ')}`);
    // The switch awaits the exchange only; the board's panels may ask about the position they now
    // show, but nothing re-reads the list or its positions before the group is on screen.
    expect(measure.calls[0]).toBe('ActivateLibraryGroup');
    const listReads = ['LoadPositionsByIDs', 'ListPositionIDs', 'CountPositions', 'SearchPositionIDs', 'IndexOfPosition', 'LoadSessionState'];
    expect(measure.calls.filter((m) => listReads.includes(m))).toEqual([]);
    expect(measure.ms).toBeLessThan(250);
    await expect(page.locator('[role="tab"][data-group-id="1"]')).toHaveAttribute('aria-selected', 'true');
});

test('the tab keys walk from one library to the other', async ({ page }) => {
    await page.locator('[data-testid="open-library-group"]').click();
    const headers = page.locator('[data-testid="library-group"]');
    await expect(headers.nth(1)).toHaveClass(/active/);
    await page.keyboard.press('Control+PageDown');
    await expect(headers.nth(0)).toHaveClass(/active/);
    await page.keyboard.press('Control+Shift+PageDown');
    await expect(headers.nth(1)).toHaveClass(/active/);
});

test('closing a group closes its library and leaves one group', async ({ page }) => {
    await page.locator('[data-testid="open-library-group"]').click();
    const headers = page.locator('[data-testid="library-group"]');
    await expect(headers).toHaveCount(2);
    await headers.nth(1).locator('.close-btn').click();
    await expect(headers).toHaveCount(0);
    const closes = await page.evaluate(() => window.__wailsCalls.filter((c) => c.method === 'CloseLibraryGroup').map((c) => c.args[0]));
    expect(closes).toEqual([2]);
});
