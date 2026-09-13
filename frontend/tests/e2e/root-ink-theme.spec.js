/**
 * root-ink-theme.spec.js — #403
 *
 * L'encre héritée vient du thème. style.css peignait `body` d'un noir littéral :
 * tout ce qui ne fixait pas sa propre couleur restait noir quel que soit le
 * thème — en sombre, le titre de l'écran d'accueil et les titres de ses cartes
 * s'écrivaient en noir sur une surface quasi noire. #359 avait corrigé la même
 * cause localement, dans Modal.svelte ; ceci la tient à la racine.
 *
 * Les titres des cartes sont des <button> : un bouton n'hérite pas l'encre (le
 * navigateur le peint en `buttontext`), d'où leur assertion à part — c'est
 * celle qui resterait rouge si l'on ne corrigeait que `body`.
 *
 * Les valeurs attendues sont lues dans themes.js, pas recopiées.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { THEMES } from '../../src/utils/themes.js';

/** @param {string} hex `#rrggbb` → la forme que rend getComputedStyle. */
const rgb = (hex) => `rgb(${[1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16)).join(', ')})`;

for (const theme of ['light', 'dark']) {
    test(`le texte hérité et l'écran d'accueil prennent l'encre du thème ${theme}`, async ({ page }) => {
        await installWailsMock(page, { config: { GetTheme: theme } });
        await page.goto('/');
        await expect(page.locator('html')).toHaveAttribute('data-theme', theme);

        const ink = rgb(THEMES[theme].ui['--color-text']);
        await expect(page.locator('body')).toHaveCSS('color', ink);

        const home = page.locator('[data-tour="home"]');
        await expect(home).toBeVisible();
        await expect(home.locator('h1')).toHaveCSS('color', ink);
        const titles = home.locator('.choice-title');
        await expect(titles.first()).toBeVisible();
        for (const title of await titles.all()) await expect(title).toHaveCSS('color', ink);
    });
}
