/**
 * direction-epreuve-tabs.spec.js — D6.4 (#447, ADR-0056 §5) : deux Directions d'une Rencontre
 * côte à côte, sans écran partagé. Ouvrir la Rencontre ouvre toutes ses épreuves ; un onglet par
 * épreuve dans l'en-tête, avec son résumé (propositions en attente, matchs en cours, alerte) ;
 * changer d'épreuve est un clic, sans confirmation, sans rien fermer ni rejouer. Un tournoi hors
 * Rencontre ne voit aucun changement.
 */
import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installDirectionEngine } from './helpers/directionEngine.js';

// Dock à sa hauteur par défaut : sans elle, le mock rend null et le dock prend une hauteur
// automatique qui écrase la page Direction (voir direction-hall-768.spec.js).
const DOCK = { GetPanelPosition: 'bottom' };

const ROOM = {
    id: 5,
    name: 'Festival de Lyon',
    sister: 'Principal',
    busy: [],
    // Une deuxième Direction, vraie et distincte, pour l'onglet d'à côté (D6.4).
    dual: {
        tournamentId: 2,
        state: 'running',
        engineVersion: 'v0.3.0',
        outputDir: '',
        config: { name: 'Principal', tables: { count: 4 }, phases: [{ kind: 'swiss_lives', length: 7, lives: 2, mode: 'continuous', target: 0 }] },
        proposals: [{ kind: 'start_match', phase: 0, key: 'd0', a: 'ha', b: 'lb', length: 7, table: 0, label: { kind: 'swiss_group', losses: 0, match: 1 } }],
        warnings: [{ code: 'slow_match', match: 'm9' }],
        ranking: [],
        players: [],
        running: [{ ID: 'm9', A: 'mc', B: 'nd', a: 'mc', b: 'nd', Length: 7, Table: 3, Status: 1 }],
        phase: 0,
        finished: false,
        eventCount: 3,
        rencontreId: 5,
        busyTables: []
    }
};

test('changer d’épreuve est un clic, sans confirmation ni fermeture', async ({ page }) => {
    const dialogs = [];
    page.on('dialog', (d) => {
        dialogs.push(d.message());
        d.dismiss();
    });

    await installWailsMock(page, openLibraryMock({ config: DOCK }));
    await installDirectionEngine(page, { room: ROOM });
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await page.locator('#tournamentPanel tbody tr').first().dblclick();
    await page.locator('#tournamentPanel .direction-btn').click();
    await expect(page.locator('.direction-view')).toBeVisible();
    await page.locator('[data-testid="direction-tab-direction"]').click();

    // Les deux épreuves de la Rencontre sont ouvertes d'emblée : deux onglets, chacun son
    // résumé, après celui de la Salle.
    const tabs = page.locator('[data-testid="epreuve-tabs"] button');
    await expect(tabs).toHaveCount(3);
    const tab1 = page.locator('[data-testid="epreuve-tab-1"]');
    const tab2 = page.locator('[data-testid="epreuve-tab-2"]');
    await expect(tab1).toHaveClass(/active/);
    await expect(tab1).toContainText('Speed');
    await expect(tab1).toContainText('2'); // 4 inscrits, aucun lancé : deux propositions.
    await expect(tab2).toContainText('Principal');
    await expect(tab2).toContainText('1'); // une proposition en attente.
    await expect(tab2.locator('.badge.warning')).toBeVisible();

    await expect(page.locator('header .name')).toHaveText('Open de Lyon');
    await expect(page.locator('.proposals .queue li:not(.empty)')).toHaveCount(2);

    // Un clic change d'épreuve : ni confirmation, ni fermeture de la première.
    await tab2.click();
    await expect(page.locator('header .name')).toHaveText('Principal');
    await expect(tab2).toHaveClass(/active/);
    await expect(tab1).not.toHaveClass(/active/);
    await expect(page.locator('.proposals .queue li:not(.empty)')).toHaveCount(1);
    expect(dialogs).toEqual([]);

    // Revenir : l'épreuve quittée n'a pas été rejouée depuis rien, elle est intacte.
    await tab1.click();
    await expect(page.locator('header .name')).toHaveText('Open de Lyon');
    await expect(page.locator('.proposals .queue li:not(.empty)')).toHaveCount(2);
    expect(dialogs).toEqual([]);
});

test('un tournoi hors Rencontre ne montre aucun onglet d’épreuve', async ({ page }) => {
    await installWailsMock(page, openLibraryMock({ config: DOCK }));
    await installDirectionEngine(page);
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await page.locator('#tournamentPanel tbody tr').first().dblclick();
    await page.locator('#tournamentPanel .direction-btn').click();
    await expect(page.locator('.direction-view')).toBeVisible();
    await expect(page.locator('[data-testid="epreuve-tabs"]')).toHaveCount(0);
});

test('changer d’onglet de l’app puis revenir rend la Salle quittée', async ({ page }) => {
    await installWailsMock(page, openLibraryMock({ config: DOCK }));
    await installDirectionEngine(page, { room: ROOM });
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await page.locator('#tournamentPanel tbody tr').first().dblclick();
    await page.locator('#tournamentPanel .direction-btn').click();
    await page.locator('[data-testid="epreuve-tab-hall"]').click();
    await expect(page.locator('[data-testid="direction-pane-hall"]')).toBeVisible();

    await page.locator('[data-testid="tab-search"]').click();
    await expect(page.locator('.direction-view')).toHaveCount(0);
    await page.locator('[data-testid="tab-tournaments"]').click();
    await expect(page.locator('[data-testid="direction-pane-hall"]')).toBeVisible();
    await expect(page.locator('[data-testid="epreuve-tab-hall"]')).toHaveClass(/active/);
});
