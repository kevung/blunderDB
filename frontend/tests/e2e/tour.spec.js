/**
 * tour.spec.js — visual verification of the guided-tour feature (driver.js).
 *
 * Opens the tour catalog, checks the 4 tours are listed, starts the General
 * tour, and screenshots the spotlight on a few steps so the cut-out framing and
 * popover text can be inspected.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';

const SHOT = 'test-results/tour';

async function waitForApp(page) {
    await page.waitForSelector('[data-testid="status-bar"]', { timeout: 8000 });
    await page.waitForTimeout(300);
}

test.beforeEach(async ({ page }) => {
    await installWailsMock(page);
    await page.goto('http://localhost:5173/');
    await waitForApp(page);
    // Dismiss the first-run catalog if it auto-opened, so each test starts clean.
    await page.keyboard.press('Escape');
    await page.waitForTimeout(100);
});

test('catalog lists the four tours and a demo button', async ({ page }) => {
    await page.click('[data-tour="tour"]');
    await page.waitForSelector('.tour-list', { timeout: 4000 });
    const items = page.locator('.tour-list li');
    await expect(items).toHaveCount(4);
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
    await page.waitForSelector('.driver-popover', { timeout: 4000 });
    await page.waitForTimeout(300);
    await page.screenshot({ path: `${SHOT}-step1-welcome.png` });

    // Step 2 — toolbar highlighted.
    await page.click('.driver-popover-next-btn');
    await page.waitForTimeout(400);
    await expect(page.locator('[data-tour="toolbar"]')).toHaveClass(/driver-active-element/);
    await page.screenshot({ path: `${SHOT}-step2-toolbar.png` });

    // Step 3 — board highlighted.
    await page.click('.driver-popover-next-btn');
    await page.waitForTimeout(400);
    await expect(page.locator('[data-tour="board"]')).toHaveClass(/driver-active-element/);
    await page.screenshot({ path: `${SHOT}-step3-board.png` });

    // Step 5 — command line step must mention Space, never "EDIT mode".
    await page.click('.driver-popover-next-btn'); // panels
    await page.waitForTimeout(200);
    await page.click('.driver-popover-next-btn'); // command line
    await page.waitForTimeout(300);
    const desc = await page.locator('.driver-popover-description').innerText();
    expect(desc).toMatch(/Space/i);
    expect(desc).not.toMatch(/EDIT mode|NORMAL mode/);
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
    await page.waitForTimeout(500);
    await expect(page.locator('[data-testid="tab-search"]')).toHaveClass(/active/);
    await expect(page.locator('[data-tour="panels"]')).toHaveClass(/driver-active-element/);
    await page.screenshot({ path: `${SHOT}-search-panel.png` });
});
