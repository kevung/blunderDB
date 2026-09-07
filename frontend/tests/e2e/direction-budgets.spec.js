/**
 * direction-budgets.spec.js — les budgets de gestes d'ux.md §4, comptés sur l'application
 * réelle (D3.5, #390).
 *
 * ## Pourquoi ces specs existent
 *
 * Les budgets de ce chantier sont une contrainte de premier rang posée par l'auteur : la souris
 * est première, la charge mentale d'un directeur occasionnel prime, le coût d'entrée est bas.
 * Ce qui n'est pas mesuré dérive — il suffit d'un clic d'armement de plus, d'une confirmation
 * ajoutée « par prudence », pour qu'un budget se perde sans que rien ne rougisse.
 *
 * ## Ce qui est faux ici, et assumé
 *
 * Le moteur Nicomaque, remplacé par `helpers/directionEngine.js`. Il ne calcule aucun
 * appariement : il tient exactement ce qu'un geste doit faire bouger à l'écran, sans quoi un
 * enchaînement de clics ne voudrait rien dire. Ce que le moteur décide est tenu en Go, décision
 * par décision ; ce qui est tenu ici est COMBIEN de gestes le directeur fait.
 *
 * ## Un dépassement doit se lire sans ouvrir le code
 *
 * D'où `budget()` : le message d'échec nomme le flux, le budget et ce qui a été compté.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { countGestures } from './helpers/gestureCount.js';
import { installDirectionEngine, ENTRANTS } from './helpers/directionEngine.js';

/**
 * Vérifie un budget en le nommant. Playwright rapporte le message, pas la ligne : un échec doit
 * dire « apparier à la main : 9 gestes pour un budget de 7 » et rien de plus à chercher.
 */
function budget(flow, counted, max) {
    expect(counted.total, `${flow} : ${counted.total} gestes (${counted.clicks} clics, ${counted.keys} touches) pour un budget de ${max}`).toBeLessThanOrEqual(max);
}

/** Ouvre l'onglet Tournois sur une base chargée. Le décor n'est jamais compté. */
async function openTournaments(page, opts = {}) {
    await installWailsMock(page, openLibraryMock());
    await installDirectionEngine(page, opts);
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await expect(page.locator('#tournamentPanel')).toBeVisible();
}

/** Le décor des flux de direction : un tournoi dirigé, ouvert, la page Direction devant soi. */
async function openDirection(page) {
    await openTournaments(page);
    await page.locator('#tournamentPanel tbody tr').first().click();
    await page.locator('#tournamentPanel .direction-btn').click();
    await expect(page.locator('.direction-view')).toBeVisible();
    await page.locator('[data-testid="direction-tab-direction"]').click();
    await expect(page.locator('.proposals .queue li').first()).toBeVisible();
}

const proposal = '.proposals .queue li';

test.describe('ux.md §4 — les budgets de gestes du directeur', () => {
    // « confirmer une proposition | 1 clic ». Le bouton est sur la ligne : pas de sélection
    // préalable, pas de confirmation après.
    test('confirmer une proposition tient en un clic', async ({ page }) => {
        await openDirection(page);
        const before = await page.locator(proposal).count();

        const counted = await countGestures(page, async (g) => {
            await g.click(page.locator(`${proposal} .go`).first());
            await expect(page.locator(proposal)).toHaveCount(before - 1);
        });
        budget('confirmer une proposition', counted, 1);
    });

    // « tout lancer, quel que soit le nombre | 2 clics ». Le second clic est la confirmation,
    // et elle est là exprès : c'est le seul geste de la page qui agit sur plusieurs matchs.
    test('tout lancer tient en deux clics quel que soit le nombre', async ({ page }) => {
        await openDirection(page);
        expect(await page.locator(proposal).count()).toBeGreaterThan(1);

        const counted = await countGestures(page, async (g) => {
            await g.click(page.locator('.proposals .all'));
            await g.click(page.locator('.proposals .confirm .primary'));
            await expect(page.locator('.grid .cell.busy').first()).toBeVisible();
        });
        budget('tout lancer', counted, 2);
    });

    // « résultat sans score | 2 clics » : la case de la table, puis le nom du vainqueur. Le
    // score est libre, et ne rien en dire ne coûte rien.
    test('un résultat sans score tient en deux clics', async ({ page }) => {
        await openDirection(page);
        await page.locator('.proposals .all').click();
        await page.locator('.proposals .confirm .primary').click();
        await expect(page.locator('.grid .cell.busy').first()).toBeVisible();
        const busyBefore = await page.locator('.grid .cell.busy').count();

        const counted = await countGestures(page, async (g) => {
            await g.click(page.locator('.grid .cell.busy').first());
            await g.click(page.locator('.card .winner').first());
            await expect(page.locator('.grid .cell.busy')).toHaveCount(busyBefore - 1);
        });
        budget('résultat sans score', counted, 2);
    });

    // « corriger une erreur de saisie vue aussitôt | 2 clics ». La dernière décision reste sous
    // les yeux : corriger ne demande pas d'aller la chercher.
    test('corriger la dernière saisie tient en deux clics', async ({ page }) => {
        await openDirection(page);
        await page.locator('.proposals .all').click();
        await page.locator('.proposals .confirm .primary').click();
        await page.locator('.grid .cell.busy').first().click();
        await page.locator('.card .winner').first().click();
        await expect(page.locator('.last')).toBeVisible();
        const wrong = await page.locator('.last .what').textContent();

        const counted = await countGestures(page, async (g) => {
            await g.click(page.locator('.last button').first());
            await g.click(page.locator('.correct .winner').nth(1));
            await expect(page.locator('.last .what')).not.toHaveText(wrong);
        });
        budget('corriger la dernière saisie', counted, 2);
    });

    // « apparier à la main | ≤ 7 clics ». Un `select` natif se déplie puis se choisit : deux
    // gestes pour l'utilisateur, un seul appel pour le pilote (`selectOption` n'émet rien). Le
    // second geste de chaque liste est donc ajouté à la main, sans quoi le budget serait tenu
    // par une commodité de Playwright.
    test('apparier à la main tient dans son budget', async ({ page }) => {
        await openDirection(page);
        const runningBefore = await page.locator('.grid .cell.busy').count();

        const counted = await countGestures(page, async (g) => {
            await g.click(page.locator('.manual .link'));
            const selects = page.locator('.manual select');
            await g.click(selects.nth(0));
            await selects.nth(0).selectOption(ENTRANTS[0].id);
            await g.click(selects.nth(1));
            await selects.nth(1).selectOption(ENTRANTS[1].id);
            await g.click(page.locator('.manual .primary'));
            await expect(page.locator('.grid .cell.busy')).toHaveCount(runningBefore + 1);
        });
        const withPicks = { ...counted, total: counted.total + 2 };
        budget('apparier à la main', withPicks, 7);
    });

    // « retirer un joueur | ≤ 6 clics ».
    test('retirer un joueur tient dans son budget', async ({ page }) => {
        await openDirection(page);

        const counted = await countGestures(page, async (g) => {
            await g.click(page.locator('[data-testid="direction-tab-players"]'));
            await g.click(page.locator('.players tbody tr').first().locator('td.actions button').nth(1));
            await expect(page.locator('.players tbody tr').first()).toContainText(/retiré|withdrawn/i);
        });
        budget('retirer un joueur', counted, 6);
    });

    // « inscrire un retardataire | ≤ 5 clics », hors saisie du nom : le formulaire d'inscription
    // est en haut de l'onglet Joueurs, toujours prêt.
    test('inscrire un retardataire tient dans son budget', async ({ page }) => {
        await openDirection(page);
        const before = ENTRANTS.length;

        const counted = await countGestures(page, async (g) => {
            await g.click(page.locator('[data-testid="direction-tab-players"]'));
            const name = page.locator('.players .entry input').first();
            await g.click(name);
            await name.fill('Yanis Ferrand'); // saisie du nom : hors budget
            await g.click(page.locator('.players .entry button[type="submit"]'));
            await expect(page.locator('.players tbody tr')).toHaveCount(before + 1);
        });
        budget('inscrire un retardataire', counted, 5);
    });

    // « reprendre les inscrits d'un tournoi précédent | ≤ 4 clics pour vingt joueurs ». Le
    // nombre d'inscrits ne change rien au compte, et c'est tout l'intérêt : retaper trente noms
    // tous les mois est le premier abandon possible du logiciel.
    test('reprendre les inscrits du mois dernier tient dans son budget', async ({ page }) => {
        await openDirection(page);

        const counted = await countGestures(page, async (g) => {
            await g.click(page.locator('[data-testid="direction-tab-players"]'));
            await g.click(page.locator('.directory .head'));
            await g.click(page.locator('.directory .sources li button').first());
            await expect(page.locator('.players tbody tr')).toHaveCount(ENTRANTS.length * 2);
        });
        budget('reprendre les inscrits d’un tournoi précédent', counted, 4);
    });

    // « de "nouveau tournoi" à la première ronde lancée | ≤ 8 clics hors saisie des noms ».
    // C'est LE coût d'entrée, mesuré de bout en bout depuis rien : pas de tournoi, pas de
    // direction, pas d'inscrit.
    test('le coût d’entrée, de rien à la première ronde lancée', async ({ page }) => {
        await openTournaments(page, { directed: false });

        const counted = await countGestures(page, async (g) => {
            // Créer le tournoi. Le nom se tape (hors budget) ; Entrée le crée.
            const add = page.locator('#tournamentPanel .add-input.name');
            await g.click(add);
            await add.fill('Open de Lyon');
            await g.press('Enter');
            await expect(page.locator('#tournamentPanel tbody tr')).toHaveCount(1);

            // Ouvrir le tournoi, le diriger : la direction s'ouvre au même clic.
            await g.click(page.locator('#tournamentPanel tbody tr').first());
            await g.click(page.locator('#tournamentPanel .direction-btn'));
            await expect(page.locator('.direction-view')).toBeVisible();

            // Inscrire les entrants. Chaque nom se tape et se valide à Entrée : c'est la
            // saisie des noms, que le budget met hors compte.
            await g.click(page.locator('[data-testid="direction-tab-players"]'));
            const name = page.locator('.players .entry input').first();
            for (const p of ENTRANTS) {
                await name.fill(p.name);
                await page.locator('.players .entry button[type="submit"]').dispatchEvent('click');
            }
            await expect(page.locator('.players tbody tr')).toHaveCount(ENTRANTS.length);

            // Lancer la première ronde.
            await g.click(page.locator('[data-testid="direction-tab-direction"]'));
            await g.click(page.locator('.proposals .all'));
            await g.click(page.locator('.proposals .confirm .primary'));
            await expect(page.locator('.grid .cell.busy').first()).toBeVisible();
        });
        budget('de « nouveau tournoi » à la première ronde', counted, 8);
    });

    // Le même coût d'entrée, mais AVEC l'annuaire : plus un seul nom à taper. C'est la mesure
    // qui compte pour un directeur de club, celui qui dirige les mêmes trente personnes tous
    // les mois.
    test('le coût d’entrée avec l’annuaire, sans taper un seul nom', async ({ page }) => {
        await openTournaments(page, { directed: false });

        const counted = await countGestures(page, async (g) => {
            const add = page.locator('#tournamentPanel .add-input.name');
            await g.click(add);
            await add.fill('Open de Lyon, avril');
            await g.press('Enter');
            await g.click(page.locator('#tournamentPanel tbody tr').first());
            await g.click(page.locator('#tournamentPanel .direction-btn'));

            await g.click(page.locator('[data-testid="direction-tab-players"]'));
            await g.click(page.locator('.directory .head'));
            await g.click(page.locator('.directory .sources li button').first());
            await expect(page.locator('.players tbody tr')).toHaveCount(ENTRANTS.length);

            await g.click(page.locator('[data-testid="direction-tab-direction"]'));
            await g.click(page.locator('.proposals .all'));
            await g.click(page.locator('.proposals .confirm .primary'));
            await expect(page.locator('.grid .cell.busy').first()).toBeVisible();
        });
        budget('coût d’entrée avec l’annuaire', counted, 10);
    });
});
