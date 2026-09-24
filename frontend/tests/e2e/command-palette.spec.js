/**
 * command-palette.spec.js — #287
 *
 * La palette de commandes s'ouvre sur CTRL-MAJ-P, se peint dans la surface et
 * l'encre du thème (clair comme sombre), se parcourt au clavier et se referme
 * sur Échap. Les couleurs attendues sont lues dans themes.js, pas recopiées.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { THEMES } from '../../src/utils/themes.js';

/** @param {string} hex `#rrggbb` → la forme que rend getComputedStyle. */
const rgb = (hex) => `rgb(${[1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16)).join(', ')})`;

for (const theme of ['light', 'dark']) {
    test(`la palette prend la surface et l'encre du thème ${theme}`, async ({ page }) => {
        await installWailsMock(page, { config: { GetTheme: theme } });
        await page.goto('/');
        await expect(page.locator('html')).toHaveAttribute('data-theme', theme);

        await page.keyboard.press('Control+Shift+P');
        const palette = page.locator('.palette');
        await expect(palette).toBeVisible();
        await expect(palette).toHaveCSS('background-color', rgb(THEMES[theme].ui['--color-surface']));
        await expect(palette).toHaveCSS('color', rgb(THEMES[theme].ui['--color-text']));
        await expect(page.locator('.palette-input')).toBeFocused();

        await page.keyboard.type('a');
        if (process.env.PALETTE_SHOT) await page.screenshot({ path: `${process.env.PALETTE_SHOT}/palette-all-${theme}.png` });
        await page.keyboard.type('nki');
        await expect(page.locator('[role="option"]').first()).toContainText('Anki');
        await expect(page.locator('[role="option"]').first()).toHaveAttribute('aria-selected', 'true');
        if (process.env.PALETTE_SHOT) await page.screenshot({ path: `${process.env.PALETTE_SHOT}/palette-${theme}.png` });

        await page.keyboard.press('Escape');
        await expect(palette).toBeHidden();
    });
}
