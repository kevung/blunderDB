/**
 * tour.spec.js — visual verification of the guided-tour feature (driver.js).
 *
 * Opens the tour catalog, checks the 4 tours are listed, starts the General
 * tour, and screenshots the spotlight on a few steps so the cut-out framing and
 * popover text can be inspected.
 */

import { test, expect } from '@playwright/test';
import { TOURS } from '../../src/tours.js';
import en from '../../src/i18n/locales/en.json' with { type: 'json' };
import { installWailsMock } from './helpers/wailsMock.js';

const SHOT = 'test-results/tour';

async function waitForApp(page) {
    await expect(page.locator('[data-testid="status-bar"]')).toBeVisible({ timeout: 8000 });
}

test.beforeEach(async ({ page }) => {
    await installWailsMock(page);
    await page.goto('/');
    await waitForApp(page);
    // Dismiss the first-run catalog if it auto-opened, so each test starts clean.
    await page.keyboard.press('Escape');
    await expect(page.locator('.tour-list')).toHaveCount(0);
});

test('catalog lists every tour and a demo button', async ({ page }) => {
    await page.click('[data-tour="tour"]');
    await page.waitForSelector('.tour-list', { timeout: 4000 });
    const items = page.locator('.tour-list li');
    // One entry per TOURS entry — the catalogue is generated from that array,
    // so this count is what catches a tour added to the data and forgotten in
    // the locales (the entry would render, with a raw key for a label).
    await expect(items).toHaveCount(TOURS.length);
    for (const tour of TOURS) {
        await expect(page.locator('.tour-list')).toContainText(en.tour[tour.id].title);
    }
    // Demo-data button is present and clickable (no-op under the Wails mock).
    const demo = page.locator('.demo-button');
    await expect(demo).toBeVisible();
    await page.screenshot({ path: `${SHOT}-catalog.png` });
    await demo.click();
    await expect(page.locator('.tour-list')).toHaveCount(0); // catalog closed
});

test('general tour spotlight + text', async ({ page }) => {
    await page.click('[data-tour="tour"]');
    await page.waitForSelector('.tour-list');

    // Start the first tour (General).
    await page.locator('.tour-list li').first().locator('.start-button').click();

    // Step 1 — centered welcome popover.
    await expect(page.locator('.driver-popover')).toBeVisible({ timeout: 4000 });
    await page.screenshot({ path: `${SHOT}-step1-welcome.png` });

    // Step 2 — toolbar highlighted.
    await page.click('.driver-popover-next-btn');
    await expect(page.locator('[data-tour="toolbar"]')).toHaveClass(/driver-active-element/);
    await page.screenshot({ path: `${SHOT}-step2-toolbar.png` });

    // Step 3 — board highlighted.
    await page.click('.driver-popover-next-btn');
    await expect(page.locator('[data-tour="board"]')).toHaveClass(/driver-active-element/);
    await page.screenshot({ path: `${SHOT}-step3-board.png` });

    // Step 5 — command line step must mention Space, never "EDIT mode".
    await page.click('.driver-popover-next-btn'); // panels
    await page.click('.driver-popover-next-btn'); // command line
    const desc = page.locator('.driver-popover-description');
    await expect(desc).toContainText(/Space/i);
    expect(await desc.innerText()).not.toMatch(/EDIT mode|NORMAL mode/);
    await page.screenshot({ path: `${SHOT}-step5-commandline.png` });
});

test('search tour activates the Search tab under the spotlight', async ({ page }) => {
    await page.click('[data-tour="tour"]');
    await page.waitForSelector('.tour-list');

    // Start the Search tour (2nd in the catalog).
    await page.locator('.tour-list li').nth(1).locator('.start-button').click();
    await page.waitForSelector('.driver-popover', { timeout: 4000 });

    // Step 2 highlights the panels and must switch to the Search tab.
    await page.click('.driver-popover-next-btn');
    await expect(page.locator('[data-testid="tab-search"]')).toHaveClass(/active/);
    await expect(page.locator('[data-tour="panels"]')).toHaveClass(/driver-active-element/);
    await page.screenshot({ path: `${SHOT}-search-panel.png` });
});

// The three tours added by H.12 (#254) each exist only to bring the reader to
// a panel they had no tour for. What can silently go wrong is not the text but
// the wiring: an `activateTab` naming a tab id that does not exist leaves the
// tour running on whatever panel was already open, and no unit test sees it —
// tours.js only knows strings. So each is started for real and the tab it
// claims to open is checked.
for (const [id, tab] of [
    ['eval', 'epc'],
    ['anki', 'anki'],
    ['stats', 'stats']
]) {
    test(`${id} tour opens the ${tab} tab`, async ({ page }) => {
        await page.click('[data-tour="tour"]');
        await page.waitForSelector('.tour-list');

        const index = TOURS.findIndex((t) => t.id === id);
        await page.locator('.tour-list li').nth(index).locator('.start-button').click();
        await page.waitForSelector('.driver-popover', { timeout: 4000 });

        // Step 2 of each of these tours is the one carrying activateTab.
        await page.click('.driver-popover-next-btn');
        await expect(page.locator(`[data-testid="tab-${tab}"]`)).toHaveClass(/active/);
        await expect(page.locator('[data-tour="panels"]')).toHaveClass(/driver-active-element/);
    });
}
