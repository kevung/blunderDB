/**
 * screenshot-panels.spec.js — galerie de captures pour le manuel et le guide
 * (fiche H.5, #247)
 *
 * Produit doc/source/img/panel_*.png : douze captures panneau par panneau,
 * sur le même jeu de données vitrine que screenshot.spec.js (helpers/showcase.js),
 * étendu aux panneaux Tournois, Collections, Anki, Commentaires et Stats
 * (showcaseGalleryMock). Anglais, 1280×960, thème par défaut — la doc source
 * est en français, mais la galerie reste dans la langue de la capture de
 * référence (README/index) plutôt que de multiplier les captures par langue.
 *
 * Une seule session de page pour les douze captures : les panneaux se
 * démontent/remontent au changement d'onglet (TabbedPanel), donc naviguer
 * d'un onglet à l'autre régénère leur contenu sans jamais rouvrir la base.
 *
 * Ce n'est pas un test : il ne tourne que sur demande, hors de la suite.
 *
 *     SCREENSHOT=1 npx playwright test screenshot
 *
 * (le nom du fichier contient « screenshot », comme screenshot.spec.js — la
 * commande ci-dessus lance les deux specs dans la même passe ; make screenshots
 * fait de même).
 */

import { test, expect } from '@playwright/test';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { installWailsMock, overrideDbMethodByArg } from './helpers/wailsMock.js';
import { showcaseGalleryMock, showcaseAnalyses, showcasePlayedMove } from './helpers/showcase.js';
import { captureAndOptimise } from './helpers/screenshotTools.js';

const IMG_DIR = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '../../../doc/source/img');
const MAX_BYTES = 500 * 1024;
const VIEWPORT = { width: 1280, height: 960 };

test.skip(!process.env.SCREENSHOT, 'Capture documentaire : lancer avec SCREENSHOT=1');

function outputFor(name) {
    return path.join(IMG_DIR, `panel_${name}.png`);
}

/** Clique un onglet du panneau inférieur et attend qu'il devienne actif. */
async function clickTab(page, tabId) {
    await page.click(`[data-testid="tab-${tabId}"]`);
    await expect(page.locator(`[data-testid="tab-${tabId}"]`)).toHaveClass(/active/);
}

test('galerie de captures panneau par panneau', async ({ page }) => {
    test.setTimeout(60000);
    await page.setViewportSize(VIEWPORT);
    await installWailsMock(page, showcaseGalleryMock());
    await page.goto('/');

    const statusBar = page.getByTestId('status-bar');
    await expect(statusBar).toContainText('30 / 30');
    await overrideDbMethodByArg(page, 'LoadAnalysis', showcaseAnalyses);

    const sizes = {};
    async function capture(name) {
        // Ni recouvrement modal, ni repli i18n visible — comme screenshot.spec.js.
        await expect(page.locator('.modal, [role="dialog"]')).toHaveCount(0);
        const body = await page.locator('body').innerText();
        expect(body, `${name}: repli i18n visible`).not.toMatch(/\b[a-z]+\.[a-zA-Z]+\.[a-zA-Z]+\b/);
        sizes[name] = await captureAndOptimise(page, outputFor(name));
    }

    // 1. Panneau des matchs — onglet ouvert au chargement, avant toute revue.
    await capture('matches');

    // 2. Panneau Analyse — revue du match, coup de pions (position vitrine).
    // Clic sur le nom du joueur, pas sur la ligne entière : son centre tombe
    // sur la cellule tournoi, qui ouvre son propre éditeur et vole Entrée.
    const matchPanel = page.getByRole('region', { name: 'Match navigator' });
    await matchPanel.getByRole('row', { name: /Alice/ }).getByText('Alice').click();
    await page.keyboard.press('Enter');
    await expect(page.getByTestId('match-info-bar')).toContainText('Alice');
    await expect(statusBar).toContainText('move 32/35');
    await expect(page.getByTestId('tab-analysis')).toHaveClass(/active/);
    await expect(page.locator('.checker-table tbody tr')).toHaveCount(9);
    await expect(page.locator('.checker-table tr.played')).toContainText(showcasePlayedMove);
    await expect(page.locator('#backgammon-board svg path').first()).toBeVisible();
    await page.mouse.move(0, 0);
    await page.evaluate(() => window.dispatchEvent(new Event('resize')));
    await page.evaluate(() => new Promise((resolve) => requestAnimationFrame(() => requestAnimationFrame(resolve))));
    await capture('analysis');

    // 3. Panneau Eval — même position (dés posés) : évaluation gammonNet.
    // Cliquer l'onglet Eval directement ouvre le panneau sur un plateau vierge
    // (defaultEPCPosition) : c'est le clic droit sur le plateau, « Evaluate
    // this position » (Board.svelte → sendPositionToEval), qui y envoie la
    // position étudiée — le même geste que documenté (guide_utilisateur.rst,
    // section Calculer l'EPC : « le calculateur fonctionne pour les deux
    // joueurs », depuis n'importe quel mode).
    await page.locator('#backgammon-board').click({ button: 'right' });
    await page.getByRole('menuitem', { name: 'Evaluate this position' }).click();
    await expect(page.getByTestId('tab-epc')).toHaveClass(/active/);
    await expect(page.locator('.epc-panel .checker-table tbody tr').first()).toBeVisible();
    await capture('eval');

    // 4. Panneau Commentaires — deux commentaires sur la position vitrine.
    await clickTab(page, 'comments');
    await expect(page.locator('.comment-panel')).toBeVisible();
    await expect(page.locator('.comment-panel .msg-text').first()).toBeVisible();
    await capture('comments');

    // 5. Panneau Analyse — décision de videau (trois coups plus tôt dans le
    // même match) : les flèches de match reculent de trois positions, la
    // dernière étant la décision de videau elle-même.
    await clickTab(page, 'analysis');
    await page.keyboard.press('ArrowLeft');
    await page.keyboard.press('ArrowLeft');
    await page.keyboard.press('ArrowLeft');
    await expect(page.locator('.cube-table')).toBeVisible();
    await capture('cube');

    // 6-8. Panneau Stats — Dashboard, Erreurs, Joueurs. Chart.js anime
    // l'entrée des barres (~1s) : un court délai après l'affichage du
    // conteneur évite de capturer une barre encore en train de grandir.
    await clickTab(page, 'stats');
    await expect(page.locator('.cards-grid')).toBeVisible();
    await page.waitForTimeout(800);
    await capture('stats_dashboard');

    await page.getByRole('tab', { name: 'Errors' }).click();
    await expect(page.locator('.chart-section').first()).toBeVisible();
    await page.waitForTimeout(800);
    await capture('stats_errors');

    await page.getByRole('tab', { name: 'Players' }).click();
    await expect(page.locator('.players-tab')).toBeVisible();
    await capture('stats_players');

    // 9. Panneau Anki — liste des paquets.
    await clickTab(page, 'anki');
    await expect(page.locator('.anki-panel')).toBeVisible();
    await expect(page.locator('.anki-panel .deck-name').first()).toBeVisible();
    await capture('anki');

    // 10. Panneau Collections.
    await clickTab(page, 'collections');
    await expect(page.locator('.collection-panel')).toBeVisible();
    await capture('collections');

    // 11. Panneau Tournois.
    await clickTab(page, 'tournaments');
    await expect(page.locator('.tournament-panel')).toBeVisible();
    await capture('tournaments');

    // 12. Panneau Recherche.
    await clickTab(page, 'search');
    await expect(page.locator('.search-panel')).toBeVisible();
    await capture('search');

    for (const [name, size] of Object.entries(sizes)) {
        expect(size, `panel_${name}.png`).toBeLessThanOrEqual(MAX_BYTES);
    }
});
