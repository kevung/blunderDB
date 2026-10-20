/**
 * library-groups-jobs.spec.js
 *
 * Les tâches de fond restent attachées à leur bibliothèque. A est quittée pour B : ce que le
 * dossier surveillé de A apporte va dans A, en fond, et l'en-tête de A porte une pastille ; une
 * analyse par lots de A ne montre pas sa progression dans B. Au retour sur A, la pastille s'en
 * va et la barre d'état dit ce qui s'est passé.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, getWailsCalls } from './helpers/wailsMock.js';
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
                CloseLibraryGroup: null,
                ImportFilesIn: { succeeded: 2, skipped: 0, failed: 0, errors: [], hadMatches: true, lastPositionID: 0, cancelled: false, group: 1 }
            }
        })
    );
    await page.addInitScript(() => {
        /** @type {Record<string, Function>} */
        window.__events = {};
        window.runtime.EventsOnMultiple = (name, cb) => {
            window.__events[name] = cb;
            return () => {};
        };
    });
    await page.goto('/');
    await expect(page.locator('[data-testid="status-bar"]')).toBeVisible({ timeout: 8000 });
    await page.waitForFunction(() => performance.getEntriesByName('bdb:mounted').length > 0);
    await page.locator('[data-testid="open-library-group"]').click();
    await expect(page.locator('[data-testid="library-group"]').nth(1)).toHaveClass(/active/);
});

test('a watched import of a library left goes into it, badges its group, and is told on return', async ({ page }) => {
    await page.evaluate(() => window.__events['folder-watch:files']({ group: 1, files: ['/tmp/watch/a.xg', '/tmp/watch/b.xg'] }));

    const badge = page.locator('[data-testid="library-group"][data-group-id="1"] [data-testid="library-group-badge"]');
    await expect(badge).toHaveAttribute('data-state', 'done');
    const calls = await getWailsCalls(page, 'ImportFilesIn');
    expect(calls.map((c) => c.args.slice(0, 2))).toEqual([[1, ['/tmp/watch/a.xg', '/tmp/watch/b.xg']]]);
    // Nothing went through the bound handle, which carries B.
    expect(await getWailsCalls(page, 'ImportFiles')).toEqual([]);

    await page.locator('[role="tab"][data-group-id="1"]').click();
    await expect(page.locator('[data-testid="library-group"][data-group-id="1"]')).toHaveClass(/active/);
    await expect(badge).toHaveCount(0);
    await expect(page.getByTestId('status-bar')).toContainText('Watched folder: 2 imported, 0 skipped, 0 failed');
});

test("a batch of the library left shows its progress on its group's header, not in the status bar", async ({ page }) => {
    await page.evaluate(() => window.__events['gammonnet-batch:progress']({ done: 3, total: 10, ply: 2, group: 1 }));
    const badge = page.locator('[data-testid="library-group"][data-group-id="1"] [data-testid="library-group-badge"]');
    await expect(badge).toHaveAttribute('data-state', 'running');
    await expect(page.locator('.gammonnet-batch-chip')).toHaveCount(0);

    await page.evaluate(() => window.__events['gammonnet-batch:done']({ evaluated: 10, refused: 0, failed: 0, group: 1 }));
    await expect(badge).toHaveAttribute('data-state', 'done');
});
