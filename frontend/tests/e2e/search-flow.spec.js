/**
 * search-flow.spec.js — flux Recherche
 *
 * Ouvre le panneau Recherche au raccourci (Ctrl+F), pose deux filtres numériques
 * et une structure (XGID collé sur le plateau d'édition), lance la recherche,
 * vérifie les résultats et leur parcours (j/k, flèches), puis réinitialise.
 *
 * Le backend est mocké : LoadPositionIDsByFilters renvoie deux ids quelle que
 * soit la requête (depuis D.8/#208 une recherche ne rapporte que des ids, le
 * plateau charge ensuite la fenêtre affichée) ; c'est la requête elle-même que
 * la spec vérifie, via le journal d'appels du mock.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, getWailsCalls } from './helpers/wailsMock.js';
import { openLibraryMock, positionB, positionC, xgidSample, parsedPositionResult } from './helpers/fixtures.js';

const statusBar = (page) => page.getByTestId('status-bar');
const tab = (page, id) => page.getByTestId(`tab-${id}`);
const filterItem = (page, label) => page.locator('.filter-item', { hasText: label });

test.beforeEach(async ({ page }) => {
    await installWailsMock(
        page,
        openLibraryMock({
            database: { LoadPositionIDsByFilters: [positionB.id, positionC.id], ParsePositionText: parsedPositionResult },
            runtime: { ClipboardGetText: xgidSample }
        })
    );
    await page.goto('/');
    await expect(statusBar(page)).toContainText('3 / 3');
});

test('recherche : deux filtres numériques + structure, résultats, navigation, réinitialisation', async ({ page }) => {
    // Panneau Recherche au raccourci Ctrl+F (l'onglet Matchs, ouvert au
    // chargement, garde les touches nues pour sa propre liste)
    await page.keyboard.press('Control+f');
    await expect(tab(page, 'search')).toHaveClass(/active/);

    // Filtre 1 — différence de pips, borne minimale
    const pipDiff = filterItem(page, 'Pipcount Difference');
    await pipDiff.getByRole('checkbox').check();
    await pipDiff.getByRole('spinbutton').first().fill('10');

    // Filtre 2 — pips absolus du joueur, borne maximale
    const absPips = filterItem(page, 'Player Absolute Pipcount');
    await absPips.getByRole('checkbox').check();
    await absPips.getByRole('radio', { name: 'Max' }).check();
    await absPips.getByRole('spinbutton').nth(1).fill('100');
    await expect(page.locator('.active-count')).toHaveText('2 active');

    // Structure — XGID collé sur le plateau d'édition. Le panneau garde toutes
    // les touches tant qu'un champ a le focus : sortir du champ numérique en
    // cliquant le bouton (déjà actif) « At least » avant Ctrl+V.
    await page.getByRole('button', { name: 'At least' }).click();
    await page.keyboard.press('Control+v');
    await expect(page.getByTestId('status-bar-message')).toHaveText('Position pasted to board from clipboard');

    // Lancer
    await page.locator('.top-action-bar').getByRole('button', { name: 'Search', exact: true }).click();
    await expect(statusBar(page)).toContainText('1 / 2');
    await expect(tab(page, 'analysis')).toHaveClass(/active/);

    const searches = await getWailsCalls(page, 'LoadPositionIDsByFilters');
    expect(searches).toHaveLength(1);
    const query = searches[0].args[0];
    expect(query.pipCountFilter).toBe('p>10');
    expect(query.player1AbsolutePipCountFilter).toBe('P<100');
    const checkers = query.filter.board.points.reduce((n, p) => n + p.checkers, 0);
    expect(checkers).toBeGreaterThan(0);

    // Parcours des résultats
    await page.keyboard.press('ArrowRight');
    await expect(statusBar(page)).toContainText('2 / 2');
    await page.keyboard.press('ArrowLeft');
    await expect(statusBar(page)).toContainText('1 / 2');
    await page.keyboard.press('j');
    await expect(statusBar(page)).toContainText('2 / 2');
    await page.keyboard.press('k');
    await expect(statusBar(page)).toContainText('1 / 2');

    // Réinitialisation des filtres
    await page.keyboard.press('Control+f');
    await expect(tab(page, 'search')).toHaveClass(/active/);
    await page.locator('.top-action-bar').getByRole('button', { name: 'Clear' }).click();
    await expect(pipDiff.getByRole('checkbox')).not.toBeChecked();
    await expect(absPips.getByRole('checkbox')).not.toBeChecked();
    await expect(page.locator('.active-count')).toHaveText('0 active');

    // Retour à la bibliothèque entière par la commande `e`
    await page.keyboard.press('Space');
    const commandLine = page.getByPlaceholder('Type command...');
    await commandLine.fill('e');
    await commandLine.press('Enter');
    await expect(statusBar(page)).toContainText('3 / 3');
});

test('recherche sans résultat : message et plateau d’édition conservé', async ({ page }) => {
    await installWailsMock(page, openLibraryMock({ database: { LoadPositionIDsByFilters: [] } }));
    await page.goto('/');
    await expect(statusBar(page)).toContainText('3 / 3');

    await page.keyboard.press('Control+f');
    await filterItem(page, 'Pipcount Difference').getByRole('checkbox').check();
    await page.locator('.top-action-bar').getByRole('button', { name: 'Search', exact: true }).click();

    await expect(page.getByTestId('status-bar-message')).toHaveText('No matching positions found');
    await expect(tab(page, 'search')).toHaveClass(/active/);
    await expect(statusBar(page)).toContainText('3 / 3');
});
