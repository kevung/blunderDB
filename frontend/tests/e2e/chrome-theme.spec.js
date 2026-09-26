/**
 * La chrome de l'application suit le thème : la barre d'outils, la barre d'onglets du panneau
 * inférieur, la barre d'état et l'en-tête de partie (« Alice vs Bob · tournoi · … ») ne doivent
 * pas garder de fond clair littéral en thème sombre, où un jeton clair (les noms des joueurs)
 * deviendrait illisible. Chaque zone est tenue par son fond ET par l'encre de son texte : un
 * fond passé au jeton sans son texte (ou l'inverse) reproduit le même défaut.
 *
 * Les valeurs attendues sont lues dans themes.js, pas recopiées.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { showcaseGalleryMock } from './helpers/showcase.js';
import { THEMES } from '../../src/utils/themes.js';

/** @param {string} hex `#rrggbb` → la forme que rend getComputedStyle. */
const rgb = (hex) => `rgb(${[1, 3, 5].map((i) => parseInt(hex.slice(i, i + 2), 16)).join(', ')})`;

for (const theme of ['light', 'dark']) {
    test(`barres, en-tête de partie et libellés de recherche prennent les jetons du thème ${theme}`, async ({ page }) => {
        test.setTimeout(30000);
        const mock = showcaseGalleryMock();
        await installWailsMock(page, { ...mock, config: { ...(mock.config || {}), GetTheme: theme } });
        await page.goto('/');
        await expect(page.locator('html')).toHaveAttribute('data-theme', theme);

        const ui = THEMES[theme].ui;
        const surface = rgb(ui['--color-surface']);
        const surfaceAlt = rgb(ui['--color-surface-alt']);
        const text = rgb(ui['--color-text']);
        const muted = rgb(ui['--color-text-muted']);

        // Barre d'état.
        const statusBar = page.getByTestId('status-bar');
        await expect(statusBar).toContainText('30 / 30');
        await expect(statusBar).toHaveCSS('background-color', surfaceAlt);
        await expect(page.getByTestId('status-bar-message')).toHaveCSS('color', muted);

        // Barre d'outils : ses boutons sont des <button>, qui n'héritent pas l'encre.
        const toolbar = page.locator('[data-tour="toolbar"]');
        await expect(toolbar).toHaveCSS('background-color', surfaceAlt);
        await expect(toolbar.locator('button').first()).toHaveCSS('color', text);

        // En-tête de partie.
        const infoBar = page.getByTestId('match-info-bar');
        await expect(infoBar).toBeVisible();
        await expect(infoBar).toHaveCSS('background-color', surfaceAlt);
        await expect(infoBar).toHaveCSS('color', muted);
        const names = infoBar.locator('.name');
        await expect(names.first()).toBeVisible();
        for (const name of await names.all()) await expect(name).toHaveCSS('color', text);

        // Barre d'onglets du panneau inférieur.
        const tabBar = page.locator('.tab-bar');
        await expect(tabBar).toHaveCSS('background-color', surfaceAlt);
        const searchTab = page.getByTestId('tab-search');
        await expect(searchTab).not.toHaveClass(/active/);
        await expect(searchTab).toHaveCSS('color', muted);

        // Panneau Recherche et ses libellés.
        await searchTab.click();
        await expect(searchTab).toHaveClass(/active/);
        await expect(searchTab).toHaveCSS('background-color', surface);
        const searchPanel = page.locator('.search-panel');
        await expect(searchPanel).toBeVisible();
        await expect(searchPanel).toHaveCSS('background-color', surface);
        const labels = searchPanel.locator('.filter-label');
        await expect(labels.first()).toBeVisible();
        await expect(labels.first()).toHaveCSS('color', text);
        await expect(searchPanel.locator('.group-header').first()).toHaveCSS('color', muted);
    });
}
