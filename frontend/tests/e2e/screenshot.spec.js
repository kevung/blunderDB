/**
 * screenshot.spec.js — capture documentaire
 *
 * Produit doc/source/_static/screenshot.png (l'image du README et de la doc)
 * à partir de l'interface réelle servie par Vite, sur le jeu de données
 * vitrine de helpers/showcase.js : bibliothèque de 30 positions, match en
 * revue sur une position de milieu de partie, onglet Analyse avec la table
 * des coups. Anglais, 1280×960, thème par défaut.
 *
 * Pourquoi 960 et non 800 de haut : le plateau two.js écrit ses étiquettes
 * (numéros de points, pips, score, videau) en taille fixe. Sous ~600 px de
 * large elles se chevauchent et débordent du cadre ; avec le panneau du bas à
 * sa hauteur utile (neuf coups, 280 px) et la barre d'outils, 800 px ne
 * laissent au plateau que ~420 px de haut, soit 580 px de large. À 960 il
 * dispose de 580 px de haut et s'étale sur 800 px : tout est lisible.
 *
 * Ce n'est pas un test : il ne tourne que sur demande, hors de la suite.
 *
 *     SCREENSHOT=1 npx playwright test screenshot
 *
 * Le PNG est ensuite passé à pngquant, oxipng ou optipng — le premier trouvé
 * sur le PATH ; sans aucun, la capture reste celle de Playwright (échelle
 * CSS, un pixel par pixel).
 */

import { test, expect } from '@playwright/test';
import { spawnSync } from 'node:child_process';
import { statSync } from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { installWailsMock, overrideDbMethodByArg } from './helpers/wailsMock.js';
import { showcaseMock, showcaseAnalyses, showcasePlayedMove } from './helpers/showcase.js';

const OUTPUT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../../doc/source/_static/screenshot.png');
const MAX_BYTES = 400 * 1024;
const VIEWPORT = { width: 1280, height: 960 };

test.skip(!process.env.SCREENSHOT, 'Capture documentaire : lancer avec SCREENSHOT=1');

/** Lance le premier optimiseur PNG disponible ; sans-bruit s'il n'y en a pas. */
function optimise(file) {
    const candidates = [
        ['pngquant', ['--force', '--skip-if-larger', '--quality', '70-95', '--output', file, file]],
        ['oxipng', ['-o', '4', '--strip', 'safe', file]],
        ['optipng', ['-quiet', '-o2', file]]
    ];
    for (const [bin, args] of candidates) {
        if (spawnSync('which', [bin]).status !== 0) continue;
        const run = spawnSync(bin, args, { stdio: 'inherit' });
        return { bin, ok: run.status === 0 };
    }
    return null;
}

test('capture de l’interface pour la documentation', async ({ page }) => {
    test.setTimeout(30000);
    await page.setViewportSize(VIEWPORT);
    await installWailsMock(page, showcaseMock());
    await page.goto('/');

    const statusBar = page.getByTestId('status-bar');
    await expect(statusBar).toContainText('30 / 30');
    await overrideDbMethodByArg(page, 'LoadAnalysis', showcaseAnalyses);

    // Le panneau Match est l'onglet ouvert au chargement : Entrée sur la ligne
    // du match entre en revue sur son dernier coup visité, onglet Analyse actif.
    const matchPanel = page.getByRole('region', { name: 'Match navigator' });
    await matchPanel.getByRole('row', { name: /Alice/ }).click();
    await page.keyboard.press('Enter');
    await expect(page.getByTestId('match-info-bar')).toContainText('Alice');
    await expect(statusBar).toContainText('move 32/35');
    await expect(page.getByTestId('tab-analysis')).toHaveClass(/active/);

    // La table des coups est là, le coup joué surligné.
    const rows = page.locator('.checker-table tbody tr');
    await expect(rows).toHaveCount(9);
    await expect(page.locator('.checker-table tr.played')).toContainText(showcasePlayedMove);

    // Le plateau two.js a rendu ses pions dans le SVG. Il s'est ajusté au
    // chargement, avant que la barre d'information du match ne prenne sa
    // ligne : le même 'resize' que dispatch App.svelte quand la disposition
    // change le fait remesurer sa zone, sinon la rangée 12–1 passe sous le
    // panneau.
    await expect(page.locator('#backgammon-board svg path').first()).toBeVisible();
    await page.mouse.move(0, 0);
    await page.evaluate(() => window.dispatchEvent(new Event('resize')));
    await page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));

    // Ni texte de repli i18n, ni recouvrement.
    const body = await page.locator('body').innerText();
    expect(body).not.toMatch(/\b[a-z]+\.[a-zA-Z]+\.[a-zA-Z]+\b/);
    await expect(page.locator('.modal, [role="dialog"]')).toHaveCount(0);

    await page.screenshot({ path: OUTPUT, scale: 'css', animations: 'disabled' });
    const optimised = optimise(OUTPUT);
    const size = statSync(OUTPUT).size;
    // eslint-disable-next-line no-console -- le poids du fichier produit est la sortie utile de ce script
    console.log(`${OUTPUT}: ${(size / 1024).toFixed(0)} KiB${optimised ? ` (${optimised.bin})` : ''}`);
    expect(size).toBeLessThanOrEqual(MAX_BYTES);
});
