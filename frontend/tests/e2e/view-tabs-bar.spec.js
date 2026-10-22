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

test('the landing mark shows on a dragged-over tab and on the active group header', async ({ page }) => {
    await openTwoGroups(page);
    await page.locator('.add-btn').click();
    const shadow = (loc) => loc.evaluate((el) => getComputedStyle(el).boxShadow);
    const tab = tabsOf(page, 2).nth(1);
    const a = await tabsOf(page, 2).nth(0).boundingBox();
    const t = await tab.boundingBox();
    await page.mouse.move(a.x + a.width / 2, a.y + a.height / 2);
    await page.mouse.down();
    await page.mouse.move(a.x + a.width / 2 + 10, a.y + a.height / 2, { steps: 3 });
    await page.mouse.move(t.x + t.width * 0.9, t.y + t.height / 2, { steps: 5 });
    await expect(tab).toHaveClass(/tab-drop-after/);
    expect(await shadow(tab)).not.toBe('none');
    await page.mouse.up();

    // The click right after a drag is swallowed for a moment.
    await page.waitForTimeout(300);
    // The active header: group 1 on screen, group 2 dragged to the left onto it.
    await page
        .locator('[data-testid="library-group"]')
        .nth(0)
        .click({ position: { x: 8, y: 8 } });
    const target = page.locator('[data-testid="library-group"]').nth(0);
    await expect(target).toHaveClass(/active/);
    const h1 = await target.boundingBox();
    const h2 = await page.locator('[data-testid="library-group"]').nth(1).boundingBox();
    await page.mouse.move(h2.x + h2.width * 0.15, h2.y + h2.height / 2);
    await page.mouse.down();
    await page.mouse.move(h2.x + h2.width * 0.15 - 10, h2.y + h2.height / 2, { steps: 3 });
    await page.mouse.move(h1.x + h1.width * 0.2, h1.y + h1.height / 2, { steps: 5 });
    await expect(target).toHaveClass(/active/);
    await expect(target).toHaveClass(/tab-drop-(before|after)/);
    expect(await shadow(target)).not.toBe('none');
    await page.mouse.up();
});

test('the + button follows the active group, and the tools sit apart on the right', async ({ page }) => {
    await openTwoGroups(page);
    await page.locator('[role="tab"][data-group-id="1"]').click();
    await expect(page.getByTestId('library-group').nth(0)).toHaveClass(/active/);

    const add = page.getByTestId('add-view');
    await expect(add).toHaveCount(1);
    // The "+" is not a child of any tablist, which holds only tabs.
    expect(await page.locator('[role="tablist"] button.add-btn').count()).toBe(0);
    expect(await page.locator('[role="tablist"] > :not([role="tab"])').count()).toBe(0);
    const last = await tabsOf(page, 1).last().boundingBox();
    const plus = await add.boundingBox();
    const nextHeader = await page.getByTestId('library-group').nth(1).boundingBox();
    // Right after the last tab of group 1, before group 2's header.
    expect(plus.x).toBeGreaterThanOrEqual(last.x + last.width - 1);
    expect(plus.x + plus.width).toBeLessThanOrEqual(nextHeader.x + 1);

    await add.click();
    await expect(tabsOf(page, 1)).toHaveCount(2);
    await expect(tabsOf(page, 2)).toHaveCount(1);

    // The tools are aligned to the right edge of the bar, apart from the tabs.
    const bar = await page.locator('.view-tabs').boundingBox();
    const split = await page.getByTestId('toggle-split').boundingBox();
    const basket = await page.getByTestId('transfer-basket').boundingBox();
    expect(bar.x + bar.width - (basket.x + basket.width)).toBeLessThan(40);
    expect(split.x).toBeGreaterThan(nextHeader.x + nextHeader.width);
});

test('a scrolling row keeps the + in sight, and the active tab wins when both cannot fit', async ({ page }) => {
    await page.setViewportSize({ width: 1024, height: 768 });
    await openTwoGroups(page);
    const add = page.locator('.view-tabs .add-btn');
    for (let i = 0; i < 30; i++) await add.click();
    const row = page.locator('.view-tab-list');
    expect(await row.evaluate((el) => el.scrollWidth > el.clientWidth)).toBe(true);
    await expect(add).toBeInViewport();
    await expect(page.locator('.view-tab.active')).toBeInViewport();
    // A tab at the far left: the active tab stays in sight even though the "+" is far right.
    const first = tabsOf(page, 2).first();
    await first.click();
    await expect(first).toHaveClass(/active/);
    await expect(page.locator('.view-tab.active')).toBeInViewport();
});

test('an overflowing row shows arrows and fades, shrinks tabs to a minimum, and lists every view in a menu', async ({ page }) => {
    await page.setViewportSize({ width: 1024, height: 768 });
    await openTwoGroups(page);
    // Few tabs: nothing hidden, so no arrow and no fade.
    await expect(page.getByTestId('tabs-scroll-left')).toHaveCount(0);
    await expect(page.getByTestId('tabs-scroll-right')).toHaveCount(0);

    const add = page.locator('.view-tabs .add-btn');
    for (let i = 0; i < 30; i++) await add.click();
    const row = page.locator('.view-tab-list');
    await expect(page.getByTestId('tabs-scroll-right')).toBeVisible();
    await expect(page.getByTestId('tabs-scroll-left')).toBeVisible();

    // Tabs shrink to a minimum width, not to a sliver, before the row scrolls.
    const widths = await page.locator('.view-tab').evaluateAll((els) => els.map((e) => e.getBoundingClientRect().width));
    expect(Math.min(...widths)).toBeGreaterThanOrEqual(80);

    // Jump to the start: left arrow disabled and no left fade; fade only on the hidden side.
    await row.evaluate((el) => (el.scrollLeft = 0));
    await expect(page.getByTestId('tabs-scroll-left')).toBeDisabled();
    await expect(page.getByTestId('tabs-fade-left')).toHaveCount(0);
    await expect(page.getByTestId('tabs-fade-right')).toHaveCount(1);

    // A click on the right arrow scrolls one step.
    await page.getByTestId('tabs-scroll-right').click();
    await expect.poll(() => row.evaluate((el) => el.scrollLeft)).toBeGreaterThan(50);
    await expect(page.getByTestId('tabs-fade-left')).toHaveCount(1);

    // Held, it keeps scrolling until the end, where it is disabled.
    const right = await page.getByTestId('tabs-scroll-right').boundingBox();
    await page.mouse.move(right.x + right.width / 2, right.y + right.height / 2);
    await page.mouse.down();
    await expect(page.getByTestId('tabs-scroll-right')).toBeDisabled({ timeout: 8000 });
    await page.mouse.up();

    // The menu lists every view of both libraries, the current one marked; a click activates it.
    await page.getByTestId('all-views').click();
    const items = page.getByTestId('all-views-item');
    await expect(items).toHaveCount(await page.locator('.view-tab').count());
    await expect(page.locator('[data-testid="all-views-item"][aria-current="true"]')).toHaveCount(1);
    await page.locator('[data-testid="all-views-item"][data-group-id="1"]').first().click();
    await expect(page.getByTestId('all-views-menu')).toHaveCount(0);
    await expect(page.getByTestId('library-group').nth(0)).toHaveClass(/active/);
    await expect(page.locator('.view-tab.active')).toBeInViewport();
});

for (const width of [1024, 1366, 1920]) {
    test(`with 30 tabs on two groups the tab row never overlaps the tools (${width}px)`, async ({ page }) => {
        await page.setViewportSize({ width, height: 700 });
        await openTwoGroups(page);
        for (let i = 0; i < 29; i++) await page.locator('.add-btn').click();

        const rects = await page.evaluate(() => {
            const r = (el) => {
                const b = el.getBoundingClientRect();
                return { left: b.left, right: b.right, width: b.width };
            };
            const tools = r(document.querySelector('.view-tab-tools'));
            const list = r(document.querySelector('.view-tab-list'));
            const add = r(document.querySelector('.add-btn'));
            const tabs = [...document.querySelectorAll('.view-tab-list [role="tab"]')].map(r);
            const clip = r(document.querySelector('.tab-clip'));
            return { tools, list, add, tabs, clip, bar: document.documentElement.clientWidth };
        });
        expect(rects.tools.width).toBeGreaterThan(0);
        // The scrolling row, the "+" and every visible tab stay left of the tools.
        expect(rects.list.right).toBeLessThanOrEqual(rects.tools.left + 0.5);
        expect(rects.clip.right).toBeLessThanOrEqual(rects.tools.left + 0.5);
        expect(rects.add.right <= rects.tools.left + 0.5 || rects.add.left >= rects.list.right - 0.5 || rects.add.right <= rects.list.right + 0.5).toBe(true);
        for (const t of rects.tabs) {
            if (t.right > rects.list.left && t.left < rects.list.right) expect(Math.min(t.right, rects.list.right)).toBeLessThanOrEqual(rects.tools.left + 0.5);
        }
        // The overflow arrow stays visible and clickable; the "+" is reachable.
        const arrow = page.locator('.scroll-btn').last();
        await expect(arrow).toBeVisible();
        const ab = await arrow.boundingBox();
        expect(ab.x + ab.width).toBeLessThanOrEqual(rects.tools.left + 0.5);
        await page.locator('.add-btn').scrollIntoViewIfNeeded();
        const ad = await page.locator('.add-btn').boundingBox();
        expect(ad.x + ad.width).toBeLessThanOrEqual(rects.tools.left + 0.5);
    });
}
