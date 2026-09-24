/**
 * color-scheme-theme.spec.js — #402
 *
 * La racine déclare le schéma de couleur du thème actif. Les jetons ne
 * peignent que ce que les composants stylent ; les contrôles NATIFS — boutons,
 * listes, champs, cases, barres de défilement — sont peints par le moteur, qui
 * ne sait qu'une page est sombre que si `color-scheme` le lui dit. Sans lui, le
 * thème sombre gardait des contrôles clairs sur ses surfaces sombres.
 *
 * Tenu sur la propriété CALCULÉE de <html>, pour chaque thème nommé et pour
 * `system` dans les deux préférences du bureau, y compris quand celle-ci change
 * en cours de session. Le schéma attendu est lu dans themes.js, pas recopié.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { THEMES } from '../../src/utils/themes.js';

const colorScheme = (page) => page.evaluate(() => getComputedStyle(document.documentElement).colorScheme);

for (const [theme, { scheme }] of Object.entries(THEMES)) {
    test(`le thème ${theme} déclare le schéma ${scheme} sur la racine`, async ({ page }) => {
        await installWailsMock(page, { config: { GetTheme: theme } });
        await page.goto('/');
        await expect(page.locator('html')).toHaveAttribute('data-theme', theme);
        await expect.poll(() => colorScheme(page)).toBe(scheme);
    });
}

for (const preference of ['light', 'dark']) {
    test(`le thème système suit un bureau ${preference}, et son changement en cours de session`, async ({ page }) => {
        await page.emulateMedia({ colorScheme: preference });
        await installWailsMock(page, { config: { GetTheme: 'system' } });
        await page.goto('/');
        await expect(page.locator('html')).toHaveAttribute('data-theme', preference);
        await expect.poll(() => colorScheme(page)).toBe(preference);

        const other = preference === 'light' ? 'dark' : 'light';
        await page.emulateMedia({ colorScheme: other });
        await expect(page.locator('html')).toHaveAttribute('data-theme', other);
        await expect.poll(() => colorScheme(page)).toBe(other);
    });
}

// Ce que le schéma change réellement : un bouton que rien ne style prend la
// couleur de bouton du moteur, claire ou sombre selon le schéma déclaré. Les
// boutons « 5 pts / 7 pts / 9 pts » de la matrice du videau en sont.
test('en sombre, un bouton natif sans style est peint sombre', async ({ page }) => {
    await installWailsMock(page, { config: { GetTheme: 'dark' } });
    await page.goto('/');
    await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark');
    const luminance = await page.evaluate(() => {
        const button = document.createElement('button');
        button.textContent = 'probe';
        document.body.appendChild(button);
        const [r, g, b] = getComputedStyle(button)
            .backgroundColor.match(/\d+(\.\d+)?/g)
            .map(Number);
        button.remove();
        return (0.2126 * r + 0.7152 * g + 0.0722 * b) / 255;
    });
    expect(luminance).toBeLessThan(0.5);
});
