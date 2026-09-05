/**
 * match-panel-detail-on-demand.spec.js
 *
 * Panneau Match : le volet de détail n'existe que pour un match sélectionné.
 *
 * #201 (D.1) avait réservé sa largeur — le volet toujours monté, une invite
 * l'occupant tant que rien n'était choisi — pour que la ligne cliquée ne se
 * dérobe pas sous le curseur avant le second clic d'un double-clic. Ce remède
 * coûtait 55 % du panneau pour afficher une phrase, et laissait la liste des
 * matchs trop étroite pour être lue. La réservation a donc disparu : la liste
 * occupe toute la largeur jusqu'à ce qu'une sélection donne au volet quelque
 * chose à montrer.
 *
 * Ce que ce test épingle est la paire : pas de volet et liste pleine largeur
 * avant sélection, volet et liste rétrécie après, retour en arrière à la
 * désélection.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock, matchSample, matchGames, matchMovePositions } from './helpers/fixtures.js';

const statusBar = (page) => page.getByTestId('status-bar');

test.beforeEach(async ({ page }) => {
    await installWailsMock(
        page,
        openLibraryMock({
            database: { GetAllMatches: [matchSample], GetMatchMovePositions: matchMovePositions, GetGamesByMatch: matchGames },
            // Une hauteur de panneau fixe, comme en production : sans réponse le
            // panneau prend la hauteur de son contenu et monterait quand la
            // transcription remplit le volet — ici seule la liste compte.
            config: { GetPanelHeight: 250, GetPanelWidth: 420 }
        })
    );
    await page.goto('/');
    await expect(statusBar(page)).toContainText('3 / 3');
});

test('le volet de détail n’apparaît qu’avec un match sélectionné', async ({ page }) => {
    const panel = page.getByRole('region', { name: 'Match navigator' });
    const row = panel.getByRole('row', { name: /Alice/ });
    const list = panel.locator('.match-list-pane');
    const detail = panel.locator('.detail-pane');

    // Avant tout clic : aucun volet, et la liste occupe toute la largeur.
    await expect(detail).toHaveCount(0);
    const listFull = await list.boundingBox();

    await row.click();
    await expect(panel.getByRole('button', { name: /Review/ })).toBeVisible();
    await expect(detail).toHaveCount(1);

    // La liste a fait de la place au volet.
    const listNarrow = await list.boundingBox();
    expect(listNarrow.width).toBeLessThan(listFull.width);

    // Le second clic sur la même ligne la désélectionne : le volet disparaît
    // et la liste retrouve toute la largeur.
    await row.click();
    await expect(detail).toHaveCount(0);
    expect((await list.boundingBox()).width).toBeCloseTo(listFull.width, 0);
});
