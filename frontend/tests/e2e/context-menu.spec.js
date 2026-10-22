/**
 * context-menu.spec.js — the context menu built from the shared action catalog.
 *
 * On the board, the frequent actions sit at the first level and the others in one submenu per
 * group: the keyboard walks into a submenu with →, back with ←, and Escape closes the submenu
 * before the menu. A match row offers its own menu from the same catalog, by right click or
 * by Shift+F10 on the focused row.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, getWailsCalls } from './helpers/wailsMock.js';
import { openLibraryMock, matchSample, matchMovePositions } from './helpers/fixtures.js';
import en from '../../src/i18n/locales/en.json' with { type: 'json' };

const menu = (page) => page.getByRole('menu', { name: en.common.contextMenu }).first();
const focused = (page) => page.evaluate(() => document.activeElement?.textContent?.trim());

test.beforeEach(async ({ page }) => {
    await installWailsMock(page, openLibraryMock({ runtime: { ClipboardSetText: true }, database: { GetAllMatches: [matchSample], GetMatchMovePositions: matchMovePositions } }));
    await page.goto('/');
    await expect(page.getByTestId('status-bar')).toContainText('3 / 3');
});

test('the board menu: → opens a group on its first action, ← comes back, Enter runs the action', async ({ page }) => {
    await page.locator('#backgammon-board').click({ button: 'right' });
    await expect(menu(page)).toBeVisible();
    await expect(page.getByRole('menuitem', { name: en.board.menu.evaluate })).toBeVisible();
    await expect(page.getByRole('menuitem', { name: en.actions.copyImage })).toBeVisible();
    await expect(page.getByRole('menuitem', { name: en.actions.copyXGID })).toHaveCount(0);

    expect(await focused(page)).toBe(en.board.menu.evaluate);
    await page.keyboard.press('ArrowDown');
    await page.keyboard.press('ArrowDown');
    expect(await focused(page)).toContain(en.actions.group.copy);
    await page.keyboard.press('ArrowRight');
    await expect(page.getByRole('menuitem', { name: en.actions.group.copy })).toHaveAttribute('aria-expanded', 'true');
    expect(await focused(page)).toContain(en.actions.copyImage);
    await page.keyboard.press('ArrowLeft');
    await expect(page.getByRole('menuitem', { name: en.actions.copyXGID })).toHaveCount(0);
    expect(await focused(page)).toContain(en.actions.group.copy);

    await page.keyboard.press('ArrowRight');
    await page.keyboard.press('ArrowDown');
    expect(await focused(page)).toBe(en.actions.copyXGID);
    await page.keyboard.press('Enter');
    await expect(menu(page)).toHaveCount(0);
    await expect(page.getByTestId('status-bar-message')).toHaveText(en.status.xgidCopied);
    const [copied] = await getWailsCalls(page, 'ClipboardSetText');
    expect(copied.args[0]).toMatch(/^XGID=/);
});

test('Escape closes the open submenu first, then the menu', async ({ page }) => {
    await page.locator('#backgammon-board').click({ button: 'right' });
    await page.getByRole('menuitem', { name: en.actions.group.analyze }).hover();
    await expect(page.getByRole('menuitem', { name: en.actions.rolloutFast })).toBeVisible();

    await page.keyboard.press('Escape');
    await expect(page.getByRole('menuitem', { name: en.actions.rolloutFast })).toHaveCount(0);
    await expect(menu(page)).toBeVisible();
    await page.keyboard.press('Escape');
    await expect(menu(page)).toHaveCount(0);
});

test('a match row: its menu by right click and by Shift+F10, from the same catalog', async ({ page }) => {
    const panel = page.getByRole('region', { name: 'Match navigator' });
    const row = panel.getByRole('row', { name: /Alice/ });

    await row.click({ button: 'right' });
    const rowMenu = page.getByTestId('row-menu');
    await expect(rowMenu.getByRole('menuitem', { name: en.actions.openMatch })).toBeVisible();
    await rowMenu.getByRole('menuitem', { name: en.actions.group.manage }).click();
    await expect(rowMenu.getByRole('menuitem', { name: en.match.editTranscription })).toBeFocused();
    await expect(rowMenu.getByRole('menuitem', { name: en.actions.trash })).toBeVisible();
    await page.keyboard.press('Escape');
    await page.keyboard.press('Escape');
    await expect(rowMenu).toHaveCount(0);

    await row.focus();
    await page.keyboard.press('Shift+F10');
    await expect(rowMenu.getByRole('menuitem', { name: en.actions.openMatch })).toBeFocused();
});
