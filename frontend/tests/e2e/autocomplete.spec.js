/**
 * autocomplete.spec.js — verifies command-line autocompletion in the real
 * command line (the status bar), opened with the Space bar.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';

async function openCommandLine(page) {
    await page.waitForSelector('[data-testid="status-bar"]', { timeout: 8000 });
    await page.waitForTimeout(200);
    await page.keyboard.press('Escape'); // dismiss the first-run tour catalog
    await page.waitForTimeout(100);
    await page.keyboard.press('Space'); // Space opens the command line
    await page.waitForSelector('.command-input', { timeout: 4000 });
}

test.beforeEach(async ({ page }) => {
    await installWailsMock(page);
    await page.goto('http://localhost:5173/');
    await openCommandLine(page);
});

test('typing a prefix shows command suggestions', async ({ page }) => {
    await page.locator('.command-input').fill('wr');
    await page.waitForSelector('.command-suggestions', { timeout: 2000 });
    const names = await page.locator('.command-suggestions .cmd-name').allInnerTexts();
    // 'wr' is an alias of both write and write!
    expect(names).toContain('write');
    expect(names).toContain('write!');
    await page.screenshot({ path: 'test-results/autocomplete.png' });
});

test('Tab completes the highlighted suggestion', async ({ page }) => {
    await page.locator('.command-input').fill('ep');
    await page.waitForSelector('.command-suggestions');
    await page.keyboard.press('Tab');
    await expect(page.locator('.command-input')).toHaveValue('epc');
});

test('no suggestions for a position number', async ({ page }) => {
    await page.locator('.command-input').fill('12');
    await expect(page.locator('.command-suggestions')).toHaveCount(0);
});
