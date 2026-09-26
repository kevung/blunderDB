/**
 * root-ink-theme.spec.js
 *
 * L'encre héritée vient du thème, tenue à la racine (`body`) plutôt que noire en dur : tout ce
 * qui ne fixe pas sa propre couleur doit rester lisible quel que soit le thème.
 *
 * Les titres des cartes sont des <button> : un bouton n'hérite pas l'encre (le
 * navigateur le peint en `buttontext`), d'où leur assertion à part.
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
