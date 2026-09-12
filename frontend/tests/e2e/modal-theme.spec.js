/**
 * modal-theme.spec.js — #359
 *
 * Une fenêtre modale se peint dans la surface et l'encre du thème. Modal.svelte
 * peignait sa boîte d'un blanc littéral : sous le thème sombre, le texte
 * alentour passait au clair et la boîte restait blanche — toute fenêtre de
 * l'application devenait illisible, et rien ne le voyait, puisque les tests
 * unitaires ne calculent pas de style.
 *
 * Les valeurs attendues sont lues dans themes.js, pas recopiées : un thème qui
 * change de surface ne rend pas ce test faux.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { THEMES } from '../../src/utils/themes.js';

/** @param {string} hex `#rrggbb` → la forme que rend getComputedStyle. */
const rgb = (hex) => `rgb(${[1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16)).join(', ')})`;

for (const theme of ['light', 'dark']) {
    test(`la fenêtre modale prend la surface et l'encre du thème ${theme}`, async ({ page }) => {
        await installWailsMock(page, { config: { GetTheme: theme } });
        await page.goto('/');
        await expect(page.locator('html')).toHaveAttribute('data-theme', theme);

        await page.keyboard.press('?');
        const box = page.locator('.modal-box');
        await expect(box).toBeVisible();

        await expect(box).toHaveCSS('background-color', rgb(THEMES[theme].ui['--color-surface']));
        await expect(box).toHaveCSS('color', rgb(THEMES[theme].ui['--color-text']));
    });
}
