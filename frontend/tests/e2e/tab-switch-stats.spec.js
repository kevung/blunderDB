/**
 * tab-switch-stats.spec.js  — Scénario S2
 *
 * Vérifie que les transitions d'onglets *impliquant l'onglet Stats* changent
 * effectivement le contenu du panneau (TabbedPanel).
 *
 * Attendu initial sur la branche `ui-reactivity` : au moins une variante
 * ROUGE reproductible 10/10 fois (bug S2 confirmé). Ces specs deviendront
 * vertes après Fiche 05.a.
 *
 * Sélecteurs utilisés :
 *   - [data-testid="tab-<id>"]  → bouton d'onglet (TabbedPanel.svelte)
 *   - [data-testid="tab-content"] → conteneur du panneau actif
 */

import { test, expect } from '@playwright/test';
import { installWailsMock } from './helpers/wailsMock.js';

// ── Helper ────────────────────────────────────────────────────────────────────

/**
 * Attend que la page soit « prête » (app montée, barre d'état visible).
 * @param {import('@playwright/test').Page} page
 */
async function waitForApp(page) {
    await page.waitForSelector('[data-testid="status-bar"]', { timeout: 8000 });
    // Laisser Svelte finir ses effets initiaux
    await page.waitForTimeout(200);
}

/**
 * Clique sur un onglet et attend que le DOM se stabilise.
 * @param {import('@playwright/test').Page} page
 * @param {string} tabId  — identifiant d'onglet (ex: 'stats', 'matches', 'epc')
 */
async function clickTab(page, tabId) {
    await page.click(`[data-testid="tab-${tabId}"]`);
    // Pas de waitForTimeout long : si la transition prend > 2 s, le test échoue
    await page.waitForTimeout(50);
}

// ── Tests ─────────────────────────────────────────────────────────────────────

test.beforeEach(async ({ page }) => {
    await installWailsMock(page);
    await page.goto('http://localhost:5173/');
    await waitForApp(page);
});

// ── Vérification de base : onglet actif visuellement ─────────────────────────

test('T1 — onglet Stats actif quand on clique dessus', async ({ page }) => {
    await clickTab(page, 'stats');

    const statsTab = page.locator('[data-testid="tab-stats"]');
    await expect(statsTab).toHaveClass(/active/);
});

test('T2 — onglet Matchs actif quand on clique dessus', async ({ page }) => {
    await clickTab(page, 'stats');
    await clickTab(page, 'matches');

    const matchTab = page.locator('[data-testid="tab-matches"]');
    await expect(matchTab).toHaveClass(/active/);
});

// ── Vérification du contenu affiché ──────────────────────────────────────────

test('T3 — transition Match → Stats : contenu stats rendu', async ({ page }) => {
    // L'onglet par défaut est 'analysis' ; d'abord aller sur Matches
    await clickTab(page, 'matches');
    await clickTab(page, 'stats');

    // L'onglet Stats doit être actif
    const statsTab = page.locator('[data-testid="tab-stats"]');
    await expect(statsTab).toHaveClass(/active/);

    // Le panneau Stats DOIT être visible (composant monté dans .tab-content)
    await expect(page.locator('.stats-panel')).toBeVisible();
});

test('T4 — transition Stats → Match : contenu match rendu', async ({ page }) => {
    await clickTab(page, 'stats');
    await clickTab(page, 'matches');

    const matchTab = page.locator('[data-testid="tab-matches"]');
    await expect(matchTab).toHaveClass(/active/);

    // Le panneau Stats ne doit plus être dans le DOM
    await expect(page.locator('.stats-panel')).not.toBeVisible();

    const statsTab = page.locator('[data-testid="tab-stats"]');
    await expect(statsTab).not.toHaveClass(/active/);
});

test('T5 — transition EPC → Stats → Match (chaîne tripartite)', async ({ page }) => {
    await clickTab(page, 'epc');
    await clickTab(page, 'stats');
    await clickTab(page, 'matches');

    const matchTab = page.locator('[data-testid="tab-matches"]');
    await expect(matchTab).toHaveClass(/active/);
});

test('T6 — transition Stats → Anki → Stats (aller-retour)', async ({ page }) => {
    await clickTab(page, 'stats');
    await clickTab(page, 'anki');
    await clickTab(page, 'stats');

    const statsTab = page.locator('[data-testid="tab-stats"]');
    await expect(statsTab).toHaveClass(/active/);
});

// ── Robustesse : 5 bascules rapides Match ↔ Stats ────────────────────────────

test('T7 — 5 bascules rapides Match ↔ Stats toutes passent', async ({ page }) => {
    for (let i = 0; i < 5; i++) {
        await clickTab(page, i % 2 === 0 ? 'stats' : 'matches');
    }

    // Après 5 bascules (0=stats, 1=matches, 2=stats, 3=matches, 4=stats),
    // on finit sur Stats
    const statsTab = page.locator('[data-testid="tab-stats"]');
    await expect(statsTab).toHaveClass(/active/);
});

// ── Variante : toutes les transitions impliquant Stats ───────────────────────

test('T8 — séquence Analysis → Stats → EPC → Stats → Anki → Stats', async ({ page }) => {
    const transitions = ['stats', 'epc', 'stats', 'anki', 'stats'];
    for (const tab of transitions) {
        await clickTab(page, tab);
    }

    const statsTab = page.locator('[data-testid="tab-stats"]');
    await expect(statsTab).toHaveClass(/active/);
});
