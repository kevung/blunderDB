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

/** Drag the centre of one element to a point of another, in small steps (pointer events). */
async function drag(page, from, to, toX = 0.5, fromX = 0.5) {
    const a = await from.boundingBox();
    const b = await to.boundingBox();
    await page.mouse.move(a.x + a.width * fromX, a.y + a.height / 2);
    await page.mouse.down();
    await page.mouse.move(a.x + a.width * fromX + 10, a.y + a.height / 2, { steps: 3 });
    await page.mouse.move(b.x + b.width * toX, b.y + b.height / 2, { steps: 8 });
    await page.mouse.up();
}

const tabsOf = (page, group) => page.locator(`[role="tab"][data-group-id="${group}"]`);
const orderOf = async (page, group) => tabsOf(page, group).evaluateAll((els) => els.map((el) => el.dataset.viewId));
const groupsOrder = (page) => page.locator('[data-testid="library-group"]').evaluateAll((els) => els.map((el) => el.dataset.groupId));

test('tabs reorder inside their group, and never land in another group', async ({ page }) => {
    await openTwoGroups(page);
    // Group 2 (on screen) gets three views; group 1 keeps its single one.
    await page.locator('.add-btn').click();
    await page.locator('.add-btn').click();
    const before = await orderOf(page, 2);
    expect(before).toHaveLength(3);

    await drag(page, tabsOf(page, 2).nth(0), tabsOf(page, 2).nth(2), 0.9);
    expect(await orderOf(page, 2)).toEqual([before[1], before[2], before[0]]);

    // Onto the other group's tab or header: refused, nothing moves.
    const kept = await orderOf(page, 2);
    await drag(page, tabsOf(page, 2).nth(0), tabsOf(page, 1).nth(0));
    await drag(page, tabsOf(page, 2).nth(0), page.locator('[data-testid="library-group"]').nth(0));
    expect(await orderOf(page, 2)).toEqual(kept);
    expect(await orderOf(page, 1)).toHaveLength(1);
    await expect(tabsOf(page, 2)).toHaveCount(3);
});

test('a group moves with all its tabs, beside the other group', async ({ page }) => {
    await openTwoGroups(page);
    await page.locator('.add-btn').click();
    const tabsBefore = await orderOf(page, 2);
    expect(await groupsOrder(page)).toEqual(['1', '2']);

    await drag(page, page.locator('[data-testid="library-group"]').nth(1), page.locator('[data-testid="library-group"]').nth(0), 0.2, 0.15);
    expect(await groupsOrder(page)).toEqual(['2', '1']);
    expect(await orderOf(page, 2)).toEqual(tabsBefore);
    // The tabs of group 2 now come before the header of group 1 in the bar.
    const flat = await page.locator('[data-group-id]').evaluateAll((els) => els.map((el) => el.dataset.groupId));
    expect(flat).toEqual(['2', '2', '2', '1', '1']);
});
