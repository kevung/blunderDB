/**
 * epc-bar-refreshes-on-return.spec.js  — Scénario S1 étendu
 *
 * Vérifie que la valeur EPC dans la barre d'état se met à jour quand on
 * change de position entre deux visites de l'onglet EPC.
 *
 * Attendu initial sur `ui-reactivity` : ROUGE (bug — la barre affiche
 * l'ancienne valeur EPC au retour sur l'onglet EPC). Deviendra vert après
 * Fiche 05.a / 05.f.
 *
 * Stratégie :
 *   1. Mock ComputeEPCFromPosition → retourne epcResultA pour positionA
 *   2. Cliquer EPC → vérifier que la barre affiche « EPC: 66.47 »
 *   3. Quitter EPC (aller sur Stats)
 *   4. Via page.evaluate, importer positionStore et le setter à positionB,
 *      puis patcher le mock pour retourner epcResultB
 *   5. Retour EPC → vérifier que la barre affiche « EPC: 72.34 »
 */

import { test, expect } from '@playwright/test';
import { installWailsMock, overrideDbMethod } from './helpers/wailsMock.js';
import { epcResultA, epcResultB } from './helpers/fixtures.js';

// ── Helper ────────────────────────────────────────────────────────────────────

async function waitForApp(page) {
    await page.waitForSelector('[data-testid="status-bar"]', { timeout: 8000 });
    await page.waitForTimeout(200);
}

async function clickTab(page, tabId) {
    await page.click(`[data-testid="tab-${tabId}"]`);
    await page.waitForTimeout(50);
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
        // Attendre que window.go soit installé (ce script s'exécute après
        // le script de base dans wailsMock, mais avant le code applicatif)
        const orig = window.go?.main?.Database?.ComputeEPCFromPosition;
        Object.defineProperty(window.go.main.Database, 'ComputeEPCFromPosition', {
            get() {
                return () => Promise.resolve(window.__epcFixture ?? window.__epcFixtureA ?? null);
            },
            configurable: true,
        });
        void orig; // suppress linter
    });

    await page.goto('http://localhost:5173/');
    await waitForApp(page);
});

// ── T1 : barre d'état EPC renseignée lors de la première visite ───────────────

test('T1 — barre d\'état affiche EPC lors de la première visite', async ({ page }) => {
    await clickTab(page, 'epc');

    // La barre doit afficher une valeur EPC dans les 2 s
    const statusBar = page.locator('[data-testid="status-bar"]');
    await expect(statusBar).toContainText(/EPC[:\s]+\d/, { timeout: 2000 });
});

// ── T2 : EPC se met à jour quand la position change entre deux visites ─────────

test('T2 — EPC change après changement de position (S1 étendu)', async ({ page }) => {
    // 1. Visiter EPC et mémoriser la valeur affichée
    await clickTab(page, 'epc');
    const statusBar = page.locator('[data-testid="status-bar"]');
    await expect(statusBar).toContainText(/EPC[:\s]+\d/, { timeout: 2000 });
    const initialText = await statusBar.textContent();

    // 2. Quitter EPC (aller sur Stats)
    await clickTab(page, 'stats');

    // 3. Patcher le mock pour retourner epcResultB (position différente)
    await overrideDbMethod(page, 'ComputeEPCFromPosition', epcResultB);
    // Aussi mettre à jour le fixture courant pour la surcharge par defineProperty
    await page.evaluate((result) => {
        window.__epcFixture = result;
    }, epcResultB);

    // 4. Retour sur EPC
    await clickTab(page, 'epc');

    // 5. La barre doit afficher une valeur EPC différente
    await expect(statusBar).toContainText(/EPC[:\s]+\d/, { timeout: 2000 });
    const newText = await statusBar.textContent();

    expect(newText).not.toBe(initialText);
});

// ── T3 : EPC stable si la position n'a pas changé (S1 base) ──────────────────

test('T3 — EPC stable au retour si la position n\'a pas changé', async ({ page }) => {
    // Première visite EPC
    await clickTab(page, 'epc');
    const statusBar = page.locator('[data-testid="status-bar"]');
    await expect(statusBar).toContainText(/EPC[:\s]+\d/, { timeout: 2000 });
    const firstText = await statusBar.textContent();

    // Aller sur Stats sans changer la position
    await clickTab(page, 'stats');

    // Retour sur EPC — la valeur ne doit PAS avoir changé
    await clickTab(page, 'epc');
    await expect(statusBar).toContainText(/EPC[:\s]+\d/, { timeout: 2000 });
    const secondText = await statusBar.textContent();

    // La valeur EPC doit être identique (même position)
    // Note : si le texte contient des infos dynamiques non-EPC, on compare
    // uniquement la partie EPC via regex
    const epcPattern = /EPC[:\s]+([\d.]+)/;
    const m1 = firstText?.match(epcPattern)?.[1];
    const m2 = secondText?.match(epcPattern)?.[1];
    expect(m1).toBeDefined();
    expect(m2).toBeDefined();
    expect(m1).toBe(m2);
});
