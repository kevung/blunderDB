/**
 * help-search.spec.js — la recherche dans l'aide
 *
 * « ? » ouvre l'aide, « / » met le focus dans le champ de recherche sans la fermer, la saisie
 * compte les occurrences et Entrée passe à la suivante.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';

test('« / » cherche dans l’aide, Entrée passe à l’occurrence suivante', async ({ page }) => {
    await installWailsMock(page, openLibraryMock());
    await page.goto('/');
    await expect(page.getByTestId('status-bar')).toContainText('3 / 3');

    await page.keyboard.press('?');
    const search = page.getByTestId('help-search');
    await expect(search).toBeVisible();
    await expect(page.getByTestId('help-loading')).toHaveCount(0);

    await page.keyboard.press('/');
    await expect(search, `focus is on ${await page.evaluate(() => document.activeElement?.outerHTML.slice(0, 160))}`).toBeFocused();
    await search.pressSequentially('position');
    await expect(search).toHaveValue('position');
    await expect(page.getByTestId('help-search-count')).toHaveText(/^0 \/ \d+$/);
    await search.press('Enter');
    await expect(page.getByTestId('help-search-count')).toHaveText(/^1 \/ \d+$/);
    await search.press('Enter');
    await expect(page.getByTestId('help-search-count')).toHaveText(/^2 \/ \d+$/);
    await expect(search).toBeVisible();
});

test('un clic dans le champ de l’aide y met le focus et la frappe le garde', async ({ page }) => {
    await installWailsMock(page, openLibraryMock());
    await page.goto('/');
    await expect(page.getByTestId('status-bar')).toContainText('3 / 3');

    await page.keyboard.press('?');
    const search = page.getByTestId('help-search');
    await expect(search).toBeVisible();
    await expect(page.getByTestId('help-loading')).toHaveCount(0);

    await search.click();
    await expect(search).toBeFocused();
    await page.keyboard.type('position');
    await expect(search).toHaveValue('position');
    await expect(search).toBeFocused();
    await page.waitForTimeout(400);
    await expect(search).toBeFocused();
});

// The manual is rendered lazily (content-visibility): a hit scrolled to must still be on
// screen once the content around it has been laid out, a couple of frames later.
test('l’occurrence atteinte reste dans la vue une fois le manuel rendu autour', async ({ page }) => {
    // The French manual is the source text, and the one whose layout exposed the drift.
    const mock = openLibraryMock();
    await installWailsMock(page, { ...mock, config: { ...(mock.config || {}), GetLanguage: 'fr' } });
    await page.goto('/');
    await expect(page.getByTestId('status-bar')).toContainText('3 / 3');

    await page.keyboard.press('?');
    const search = page.getByTestId('help-search');
    await expect(page.getByTestId('help-loading')).toHaveCount(0);

    // Where the hit lies relative to the help's scroll box two frames after the jump, or null
    // when it is inside. Typing paints the hits (CSS highlights), Enter selects one.
    /** @param {'highlight' | 'selection'} source */
    const offScreen = (source) =>
        page.evaluate(
            (source) =>
                new Promise((resolve) => {
                    requestAnimationFrame(() =>
                        requestAnimationFrame(() => {
                            const sel = window.getSelection();
                            const range = source === 'selection' ? (sel && sel.rangeCount ? sel.getRangeAt(0) : null) : CSS.highlights.get('help-search')?.values().next().value;
                            const hit = range?.getBoundingClientRect();
                            const box = range?.startContainer.parentElement?.closest('.tab-content')?.getBoundingClientRect();
                            if (!hit || !box) resolve('none');
                            else resolve(hit.height === 0 || hit.top < box.top || hit.bottom > box.bottom ? Math.round(hit.top - box.top) : null);
                        })
                    );
                }),
            source
        );

    const misses = [];
    for (const word of ['cube', 'filtre', 'match', 'raccourci', 'tableau', 'position', 'partie']) {
        await search.fill(word);
        await expect(page.getByTestId('help-search-count')).toHaveText(/^0 \/ \d+$/);
        const first = await offScreen('highlight');
        if (first !== null) misses.push(`${word} first: ${first}`);
        // Shift+Enter from the start lands on the last occurrence, the farthest jump.
        await search.press('Shift+Enter');
        const last = await offScreen('selection');
        if (last !== null) misses.push(`${word} last: ${last}`);
        for (let i = 0; i < 4; i++) {
            await search.press('Enter');
            const step = await offScreen('selection');
            if (step !== null) misses.push(`${word} +${i + 1}: ${step}`);
        }
    }
    expect(misses).toEqual([]);
});
