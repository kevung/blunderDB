/**
 * view-tabs-bar.spec.js
 *
 * La barre d'onglets avec deux bibliothèques ouvertes : le nom de chaque base reste lisible
 * quel que soit le nombre d'onglets, et les libellés ne se sélectionnent pas à la souris.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';

const pathB = '/libs/club.db';

/** @param {import('@playwright/test').Page} page */
async function openTwoGroups(page) {
    await page.goto('/');
    await expect(page.locator('[data-testid="status-bar"]')).toBeVisible({ timeout: 8000 });
    await page.locator('[data-testid="open-library-group"]').click();
    await expect(page.locator('[data-testid="library-group"]')).toHaveCount(2);
}

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
});

test('the base name stays readable with many tabs, and the labels are not selectable', async ({ page }) => {
    await openTwoGroups(page);
    for (let i = 0; i < 12; i++) await page.locator('.add-btn').click();

    const names = page.locator('[data-testid="library-group-name"]');
    await expect(names).toHaveCount(2);
    for (const i of [0, 1]) {
        const box = await names.nth(i).evaluate((el) => ({ client: el.clientWidth, scroll: el.scrollWidth }));
        expect(box.client).toBeGreaterThan(0);
        // Not truncated: the short name of a base is shown whole.
        expect(box.scroll).toBeLessThanOrEqual(box.client);
    }
    await expect(names.nth(1)).toHaveText('club');
    await expect(page.locator('[data-testid="library-group"]').nth(1)).toHaveAttribute('title', pathB);

    const selectable = await page.evaluate(() => {
        const sel = (q) => getComputedStyle(document.querySelector(q)).userSelect;
        return [sel('[data-testid="library-group-name"]'), sel('.tab-name'), sel('[role="tab"]'), sel('[data-testid="library-group"]')];
    });
    expect(selectable).toEqual(['none', 'none', 'none', 'none']);
});
