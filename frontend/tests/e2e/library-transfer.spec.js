/**
 * library-transfer.spec.js
 *
 * Copier d'une bibliothèque à l'autre. Deux groupes ouverts, B affiché : la position affichée,
 * saisie par son compteur, glissée sur l'en-tête de A y est copiée en fond (pastille sur A) ;
 * une base protégée des copies refuse et le dit (le lot : elsewhere.spec.js).
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, getWailsCalls } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';

const pathB = '/libs/club.db';
const summary = { group: 1, positions: 1, matches: 0, alreadyHeld: 0, collections: 0, lessons: 0, errors: [], cancelled: false };

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
                SetLibraryGroupLocked: null,
                TransferToGroup: summary
            }
        })
    );
    await page.goto('/');
    await expect(page.locator('[data-testid="status-bar"]')).toBeVisible({ timeout: 8000 });
    await page.waitForFunction(() => performance.getEntriesByName('bdb:mounted').length > 0);
    await page.locator('[data-testid="open-library-group"]').click();
    await expect(page.locator('[data-testid="library-group"]').nth(1)).toHaveClass(/active/);
});

/** Drags from one element to another with the pointer, in steps, as a hand would. */
async function drag(page, from, to) {
    const a = await from.boundingBox();
    const b = await to.boundingBox();
    await page.mouse.move(a.x + a.width / 2, a.y + a.height / 2);
    await page.mouse.down();
    await page.mouse.move(a.x + a.width / 2 + 20, a.y + a.height / 2 - 20, { steps: 4 });
    await page.mouse.move(b.x + b.width / 2, b.y + b.height / 2, { steps: 8 });
    await page.mouse.up();
}

const headerA = (page) => page.locator('[data-testid="library-group"][data-group-id="1"]');

test('the displayed position dragged onto the other group is copied there, in the background', async ({ page }) => {
    await drag(page, page.getByTestId('position-counter'), headerA(page));

    await expect(headerA(page).locator('[data-testid="library-group-badge"]')).toHaveAttribute('data-state', 'done');
    const calls = await getWailsCalls(page, 'TransferToGroup');
    expect(calls).toHaveLength(1);
    const [dst, batches] = calls[0].args;
    expect(dst).toBe(1);
    expect(batches).toHaveLength(1);
    expect(batches[0].group).toBe(2);
    expect(batches[0].positionIDs).toHaveLength(1);
});

test('a locked group refuses the copy and says so', async ({ page }) => {
    await headerA(page).locator('[data-testid="library-group-lock"]').click();
    await expect(headerA(page)).toHaveClass(/locked/);
    expect((await getWailsCalls(page, 'SetLibraryGroupLocked')).map((c) => c.args)).toEqual([[1, true]]);

    await drag(page, page.getByTestId('position-counter'), headerA(page));
    await expect(page.getByTestId('status-bar')).toContainText('protected');
    expect(await getWailsCalls(page, 'TransferToGroup')).toEqual([]);
});
