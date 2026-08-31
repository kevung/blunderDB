/**
 * epc-bar-refreshes-on-return.spec.js  — Scénario S1 étendu
 *
 * Vérifie que la valeur EPC affichée dans le PANNEAU se met à jour quand on
 * change de position entre deux visites de l'onglet EPC — et que la barre
 * d'état, elle, ne transmet AUCUNE valeur EPC (le mode défi masque le
 * panneau ; une copie dans la barre d'état trahirait les réponses).
 *
 * Stratégie :
 *   1. Mock ComputeEPCFromPosition → retourne epcResultA pour positionA
 *   2. Cliquer EPC → vérifier que le panneau affiche « 66.47 »
 *   3. Quitter EPC (aller sur Stats)
 *   4. Patcher le mock pour retourner epcResultB
 *   5. Retour EPC → vérifier que le panneau affiche « 72.34 »
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, overrideDbMethod } from './helpers/wailsMock.js';
import { epcResultA, epcResultB } from './helpers/fixtures.js';

// ── Helper ────────────────────────────────────────────────────────────────────

async function waitForApp(page) {
    await expect(page.locator('[data-testid="status-bar"]')).toBeVisible({ timeout: 8000 });
}

async function clickTab(page, tabId) {
    await page.click(`[data-testid="tab-${tabId}"]`);
    await expect(page.locator(`[data-testid="tab-${tabId}"]`)).toHaveClass(/active/);
}

// ── Tests ─────────────────────────────────────────────────────────────────────

test.beforeEach(async ({ page }) => {
    // Mock de base avec EPC résultat A pour la première visite
    await installWailsMock(page);
    // Surcharge ComputeEPCFromPosition pour retourner epcResultA
    await page.addInitScript((result) => {
        // Sera exécuté avant le chargement de la page — on surcharge via un
        // flag pour que le mock principal le lise
        window.__epcFixtureA = result;
    }, epcResultA);
    await page.addInitScript(() => {
        Object.defineProperty(window.go.database.Database, 'ComputeEPCFromPosition', {
            get() {
                return () => Promise.resolve(window.__epcFixture ?? window.__epcFixtureA ?? null);
            },
            configurable: true
        });
    });

    await page.goto('/');
    await waitForApp(page);
});

// ── T1 : panneau EPC renseigné lors de la première visite ─────────────────────

test("T1 — le panneau affiche l'EPC lors de la première visite", async ({ page }) => {
    await clickTab(page, 'epc');

    // Le panneau doit afficher la valeur EPC de la fixture dans les 2 s
    const panel = page.locator('.epc-panel');
    await expect(panel).toContainText('66.47', { timeout: 2000 });
});

// ── T2 : EPC se met à jour quand la position change entre deux visites ─────────

test('T2 — EPC change après changement de position (S1 étendu)', async ({ page }) => {
    // 1. Visiter EPC et vérifier la valeur affichée
    await clickTab(page, 'epc');
    const panel = page.locator('.epc-panel');
    await expect(panel).toContainText('66.47', { timeout: 2000 });

    // 2. Quitter EPC (aller sur Stats)
    await clickTab(page, 'stats');

    // 3. Patcher le mock pour retourner epcResultB (position différente)
    await overrideDbMethod(page, 'ComputeEPCFromPosition', epcResultB);
    // Aussi mettre à jour le fixture courant pour la surcharge par defineProperty
    await page.evaluate((result) => {
        window.__epcFixture = result;
    }, epcResultB);

    // 4. Retour sur EPC — le panneau doit refléter la nouvelle valeur
    await clickTab(page, 'epc');
    await expect(panel).toContainText('72.34', { timeout: 2000 });
    await expect(panel).not.toContainText('66.47');
});

// ── T3 : EPC stable si la position n'a pas changé (S1 base) ──────────────────

test("T3 — EPC stable au retour si la position n'a pas changé", async ({ page }) => {
    // Première visite EPC
    await clickTab(page, 'epc');
    const panel = page.locator('.epc-panel');
    await expect(panel).toContainText('66.47', { timeout: 2000 });

    // Aller sur Stats sans changer la position
    await clickTab(page, 'stats');

    // Retour sur EPC — la valeur ne doit PAS avoir changé
    await clickTab(page, 'epc');
    await expect(panel).toContainText('66.47', { timeout: 2000 });
});

// ── T4 : la barre d'état ne transmet aucune valeur EPC (mode défi étanche) ────

test("T4 — la barre d'état ne contient jamais de valeur EPC", async ({ page }) => {
    await clickTab(page, 'epc');

    // Le panneau affiche la valeur…
    const panel = page.locator('.epc-panel');
    await expect(panel).toContainText('66.47', { timeout: 2000 });

    // …mais la barre d'état, jamais : ni la valeur, ni le motif « EPC: <n> ».
    const statusBar = page.locator('[data-testid="status-bar"]');
    await expect(statusBar).not.toContainText('66.47');
    await expect(statusBar).not.toContainText(/EPC[:\s]+\d/);
});
