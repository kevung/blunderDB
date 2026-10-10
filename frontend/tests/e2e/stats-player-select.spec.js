/**
 * stats-player-select.spec.js — le menu déroulant du joueur (barre de filtres Stats)
 *
 * Le nom doit tenir en hauteur quelle que soit la taille de police : la hauteur suit le
 * contenu, elle n'est pas figée. En largeur il reste tronqué (ellipsis) avec un title.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock, statsResult } from './helpers/fixtures.js';

const LONG = 'Alexandre-Maximilien de la Tour-Dupont-Castelnau';

test('le nom du joueur n’est pas coupé en hauteur et reste tronqué en largeur', async ({ page }) => {
    await installWailsMock(
        page,
        openLibraryMock({ database: { ComputeStats: statsResult, GetAllPlayerNames: [{ Name: LONG, Count: 12 }], GetAllTournaments: [], GetStatsDateRange: null, GetStatsAnalysisEngines: [] } })
    );
    await page.goto('/');
    await expect(page.getByTestId('status-bar')).toContainText('3 / 3');
    await page.getByTestId('tab-stats').click();
    const select = page.locator('#fb-player');
    await expect(select).toBeVisible();
    await select.selectOption(LONG);
    await page.addStyleTag({ content: ':root { --font-size-small: 24px !important; }' });

    const m = await select.evaluate((el) => {
        const cs = getComputedStyle(el);
        return {
            height: el.getBoundingClientRect().height,
            font: parseFloat(cs.fontSize),
            chrome: parseFloat(cs.paddingTop) + parseFloat(cs.paddingBottom) + parseFloat(cs.borderTopWidth) + parseFloat(cs.borderBottomWidth),
            title: el.title
        };
    });
    expect(m.font).toBe(24);
    expect(m.height).toBeGreaterThanOrEqual(m.font * 1.15 + m.chrome);
    expect(m.title).toBe(LONG);
    const heights = await page
        .locator('.filter-bar')
        .evaluate((bar) => [...bar.querySelectorAll('.fb-select, .fb-date, .fb-engine, .fb-depth, .fb-tour-btn, .fb-reset')].map((el) => Math.round(el.getBoundingClientRect().height)));
    expect(heights.length).toBeGreaterThan(3);
    expect(new Set(heights).size, `hauteurs ${heights}`).toBe(1);
    expect(await select.evaluate((el) => getComputedStyle(el).textOverflow)).toBe('ellipsis');
});
