/**
 * StatusBar.reactivity.test.js
 *
 * Canari de réactivité pour StatusBar.svelte.
 * Vérifie que les mutations de stores se reflètent dans le DOM.
 *
 * Pattern : vi.mock hoisted → render(StatusBar) → mutate store → await tick() → assert DOM.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, screen, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';

// ── Wails mock (doit être déclaré avant l'import du composant) ──────────────
vi.mock('../../../wailsjs/go/main/Database.js', () => ({
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(() => Promise.resolve(undefined)),
}));

// ── Stores ──────────────────────────────────────────────────────────────────
import { statusBarTextStore, currentPositionIndexStore } from '../stores/uiStore.js';
import { positionsStore, matchContextStore } from '../stores/positionStore.js';

// ── Composant ────────────────────────────────────────────────────────────────
import StatusBar from '../components/StatusBar.svelte';

// ── Helpers ──────────────────────────────────────────────────────────────────

/** Stub de position minimal satisfaisant les gardes de StatusBar. */
function makePosition(overrides = {}) {
    return {
        cube: { owner: -1, value: 0 },
        score: [-1, -1],
        move_type: 'checker',
        game_number: 1,
        ...overrides,
    };
}

function resetStores() {
    statusBarTextStore.set('');
    currentPositionIndexStore.set(0);
    positionsStore.set([]);
    matchContextStore.set({
        isMatchMode: false,
        matchID: null,
        movePositions: [],
        currentIndex: 0,
        player1Name: '',
        player2Name: '',
    });
}

// ── Suite ────────────────────────────────────────────────────────────────────

describe('StatusBar — réactivité', () => {
    beforeEach(resetStores);
    afterEach(cleanup);

    // ── Test 1 : rendu initial ───────────────────────────────────────────────
    test('T1 — rendu initial : texte de statut visible dans le DOM', () => {
        statusBarTextStore.set('Ready');
        render(StatusBar);

        // La valeur initiale du store doit apparaître dans .info-message
        expect(screen.getByText('Ready')).toBeInTheDocument();
    });

    // ── Test 2 : réactivité du texte ─────────────────────────────────────────
    test('T2 — mutation de statusBarTextStore reflétée dans le DOM', async () => {
        statusBarTextStore.set('Initial text');
        render(StatusBar);

        statusBarTextStore.set('Hello world');
        await tick();

        expect(screen.getByText('Hello world')).toBeInTheDocument();
    });

    // ── Test 3 : réactivité du compteur de positions ─────────────────────────
    test('T3 — mutation de positionsStore + currentPositionIndexStore affiche X / N', async () => {
        render(StatusBar);

        positionsStore.set([makePosition(), makePosition(), makePosition()]);
        currentPositionIndexStore.set(1); // index 1 → affiche "2 / 3"
        await tick();

        // .position-info doit afficher "2 / 3"
        const posInfo = document.querySelector('.position-info');
        expect(posInfo).not.toBeNull();
        expect(posInfo.textContent.trim()).toBe('2 / 3');
    });

    // ── Test 4 : latence de mise à jour ──────────────────────────────────────
    test('T4 — mutation + flush DOM en moins de 150 ms', async () => {
        render(StatusBar);

        const t0 = performance.now();
        statusBarTextStore.set('Perf check');
        await tick();
        const elapsed = performance.now() - t0;

        expect(elapsed).toBeLessThan(150);
        expect(screen.getByText('Perf check')).toBeInTheDocument();
    });

    // ── Test 5 : canari de régression (deux mutations successives) ───────────
    test('T5 — deux mutations successives sont toutes les deux reflétées (pas de closure stale)', async () => {
        statusBarTextStore.set('State A');
        render(StatusBar);
        await tick();

        expect(screen.getByText('State A')).toBeInTheDocument();

        statusBarTextStore.set('State B');
        await tick();

        expect(screen.queryByText('State A')).not.toBeInTheDocument();
        expect(screen.getByText('State B')).toBeInTheDocument();

        // Troisième mutation pour s'assurer que le canal reste ouvert
        statusBarTextStore.set('State C');
        await tick();

        expect(screen.queryByText('State B')).not.toBeInTheDocument();
        expect(screen.getByText('State C')).toBeInTheDocument();
    });
});
