/**
 * Cibles et visibilité du Tournoi / de la Direction aux échelles réelles : toute commande
 * cliquable mesure au moins 24 px de haut, et rien ne masque le centre de « Lancer » (zoom 150 %,
 * 911×512), des cases de table, ni d'« Enregistrer » des Réglages (1920×1080).
 *
 * La sonde est `elementFromPoint` au centre de la cible : un bouton masqué par le dock ou par un
 * champ y rend un autre élément, là où un clic Playwright ferait défiler jusqu'à lui.
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';
import { openLibraryMock } from './helpers/fixtures.js';
import { installDirectionEngine, S2_HALL } from './helpers/directionEngine.js';

const DOCK = { GetPanelHeight: 250, GetPanelPosition: 'bottom' };
const MIN = 24;

async function open(page, viewport, engine = S2_HALL) {
    await page.setViewportSize(viewport);
    await installWailsMock(page, openLibraryMock({ config: DOCK }));
    await installDirectionEngine(page, engine);
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await expect(page.locator('#tournamentPanel')).toBeVisible();
    await page.locator('#tournamentPanel tbody tr').first().dblclick();
}

async function size(locator) {
    return locator.evaluate((el) => {
        const r = el.getBoundingClientRect();
        return { w: Math.round(r.width), h: Math.round(r.height) };
    });
}

/** Ce que l'utilisateur toucherait au centre de la cible, telle que la page la pose (sans défilement
 * tiers : un conteneur `overflow: hidden` ne se défile pas à la main). */
async function coveredBy(locator, scroll = false) {
    return locator.evaluate((el, scroll) => {
        if (scroll) el.scrollIntoView({ block: 'nearest' });
        const r = el.getBoundingClientRect();
        const hit = document.elementFromPoint(r.left + r.width / 2, r.top + r.height / 2);
        return { ok: !!hit && (hit === el || el.contains(hit)), hit: hit ? `${hit.tagName}.${hit.className}` : null, inView: r.top >= 0 && r.bottom <= window.innerHeight };
    }, scroll);
}

test('panneau Tournoi : « ← » et « Diriger » mesurent au moins 24 px de haut', async ({ page }) => {
    await open(page, { width: 1920, height: 1080 });
    for (const sel of ['#tournamentPanel .back-btn', '#tournamentPanel .direction-btn']) {
        const s = await size(page.locator(sel).first());
        expect(s.h, `${sel} : ${s.w}×${s.h}`).toBeGreaterThanOrEqual(MIN);
        expect(s.w, `${sel} : ${s.w}×${s.h}`).toBeGreaterThanOrEqual(MIN);
    }
    await page.locator('#tournamentPanel .direction-btn').click();
    const open_ = await size(page.locator('[data-testid="tournament-direction-toggle"]'));
    expect(open_.h, `« Ouvrir la direction » : ${open_.w}×${open_.h}`).toBeGreaterThanOrEqual(MIN);
});

test('Historique : « Corriger » mesure au moins 24 px de haut', async ({ page }) => {
    await open(page, { width: 1920, height: 1080 });
    await page.locator('#tournamentPanel .direction-btn').click();
    await expect(page.locator('.direction-view')).toBeVisible();
    await page.locator('[data-testid="direction-tab-direction"]').click();
    await page.locator('[data-testid="direction-table-14"]').click();
    await page.locator('[data-testid="direction-result-winner-a"]').click();
    await expect(page.locator('[data-testid="direction-last"]')).toBeVisible();
    await page.locator('[data-testid="direction-tab-history"]').click();
    const correct = page.locator('[data-testid="direction-history-correct"]').first();
    await expect(correct).toBeVisible();
    const s = await size(correct);
    expect(s.h, `« Corriger » : ${s.w}×${s.h}`).toBeGreaterThanOrEqual(MIN);
});

test('zoom 150 % (911×512) : « Lancer » et les cases de table ne sont masqués par rien', async ({ page }) => {
    await open(page, { width: 911, height: 512 });
    await page.locator('#tournamentPanel .direction-btn').click();
    await expect(page.locator('.direction-view')).toBeVisible();
    await page.locator('[data-testid="direction-tab-direction"]').click();
    await expect(page.locator('.proposals .queue li').first()).toBeVisible();
    const go = page.locator('.proposals .queue li .go').first();
    // Le seul défilement permis est celui du panneau de l'onglet, comme à la main.
    await go.evaluate((el) => {
        const pane = document.querySelector('.direction-view .pane:not([hidden])');
        pane.scrollTop += el.getBoundingClientRect().top - pane.getBoundingClientRect().top - 8;
    });
    const g = await coveredBy(go);
    expect(g.ok, `« Lancer » masqué par ${g.hit}`).toBe(true);
    expect(g.inView, '« Lancer » hors de la fenêtre').toBe(true);
    const cell = page.locator('[data-testid="direction-table-1"]');
    await cell.evaluate((el) => {
        const pane = document.querySelector('.direction-view .pane:not([hidden])');
        pane.scrollTop += el.getBoundingClientRect().top - pane.getBoundingClientRect().top - 8;
    });
    const c = await coveredBy(cell);
    expect(c.ok, `case de table masquée par ${c.hit}`).toBe(true);
    expect(c.inView, 'case de table hors de la fenêtre').toBe(true);
    // Un lancement va au bout.
    await go.click();
    await expect(page.locator('.grid .cell.busy').first()).toBeVisible();
});

test('1920×1080 : « Enregistrer » des Réglages n’est masqué par aucun champ', async ({ page }) => {
    await open(page, { width: 1920, height: 1080 });
    await page.locator('#tournamentPanel .direction-btn').click();
    await expect(page.locator('.direction-view')).toBeVisible();
    await page.locator('[data-testid="direction-tab-settings"]').click();
    const apply = page.locator('[data-testid="direction-settings-apply"]');
    await expect(apply).toBeVisible();
    const a = await coveredBy(apply);
    expect(a.ok, `« Enregistrer » masqué par ${a.hit}`).toBe(true);
});

test('tournoi non dirigé : « Diriger » mesure au moins 24 px de haut', async ({ page }) => {
    await page.setViewportSize({ width: 1920, height: 1080 });
    await installWailsMock(page, openLibraryMock({ config: DOCK }));
    await installDirectionEngine(page, { ...S2_HALL, directed: false });
    await page.goto('/');
    await page.locator('[data-testid="tab-tournaments"]').click();
    await page.locator('#tournamentPanel').getByTestId('panel-new').click();
    await page.locator('#tournamentPanel .add-input.name').fill('Open de Lyon');
    await page.keyboard.press('Enter');
    await page.locator('#tournamentPanel tbody tr').first().dblclick();
    const s = await size(page.locator('[data-testid="tournament-direct"]'));
    expect(s.h, `« Diriger » : ${s.w}×${s.h}`).toBeGreaterThanOrEqual(MIN);
});
