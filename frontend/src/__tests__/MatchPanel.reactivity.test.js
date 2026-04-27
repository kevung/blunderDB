/**
 * MatchPanel.reactivity.test.js
 *
 * Vérifie que MatchPanel réagit correctement aux changements de openPanels :
 * - loadMatches (= GetAllMatches) n'est appelé que lorsque le panel s'ouvre
 * - Plusieurs ouvertures successives déclenchent autant d'appels
 * - Pas de race condition avec des bascules rapides
 *
 * Pattern : vi.mock hoisted → render(MatchPanel) → mutate openPanels → await tick() → assert calls.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';

// ── Wails mock ───────────────────────────────────────────────────────────────
vi.mock('../../wailsjs/go/main/Database.js', () => ({
    GetAllMatches: vi.fn(() => Promise.resolve([])),
    GetAllTournaments: vi.fn(() => Promise.resolve([])),
    DeleteMatch: vi.fn(() => Promise.resolve()),
    UpdateMatch: vi.fn(() => Promise.resolve()),
    UpdateMatchComment: vi.fn(() => Promise.resolve()),
    GetMatchMovePositions: vi.fn(() => Promise.resolve([])),
    GetGamesByMatch: vi.fn(() => Promise.resolve([])),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    SetMatchTournamentByName: vi.fn(() => Promise.resolve()),
    SwapMatchPlayers: vi.fn(() => Promise.resolve()),
    SaveLastVisitedPosition: vi.fn(() => Promise.resolve()),
    LoadCommandHistory: vi.fn(() => Promise.resolve([])),
    SaveCommand: vi.fn(() => Promise.resolve())
}));

// ── Stores ───────────────────────────────────────────────────────────────────
import { openPanels, PANEL, matchPanelRefreshTriggerStore } from '../stores/uiStore.js';
import { lastVisitedMatchStore } from '../stores/positionStore.js';
import { tournamentsStore } from '../stores/tournamentStore.js';

// ── DB mock ref (pour vérifier les appels) ────────────────────────────────────
import { GetAllMatches } from '../../wailsjs/go/main/Database.js';

// ── Composant ────────────────────────────────────────────────────────────────
import MatchPanel from '../components/MatchPanel.svelte';

// ── Reset stores ─────────────────────────────────────────────────────────────
function resetStores() {
    openPanels.set(new Set());
    matchPanelRefreshTriggerStore.set(0);
    tournamentsStore.set([]);
    lastVisitedMatchStore.set({ matchID: null, currentIndex: 0, gameNumber: 1 });
}

// ── Suite ─────────────────────────────────────────────────────────────────────
describe('MatchPanel — réactivité openPanels', () => {
    beforeEach(() => {
        resetStores();
        vi.clearAllMocks();
    });
    afterEach(cleanup);

    // ── Test 1 : panel non visible → loadMatches pas appelé ──────────────────
    test('T1 — panel non visible au montage : GetAllMatches non appelé', async () => {
        openPanels.set(new Set()); // panel fermé
        render(MatchPanel);
        await tick();
        await tick(); // double tick pour laisser les effets async se stabiliser

        expect(GetAllMatches).not.toHaveBeenCalled();
    });

    // ── Test 2 : ouverture du panel → loadMatches appelé une fois ─────────────
    test('T2 — ouverture du panel : GetAllMatches appelé exactement une fois', async () => {
        render(MatchPanel);
        await tick();

        openPanels.set(new Set([PANEL.MATCH]));
        await tick();
        await tick(); // attendre la résolution des promesses

        expect(GetAllMatches).toHaveBeenCalledTimes(1);
    });

    // ── Test 3 : fermeture puis réouverture → deuxième appel ─────────────────
    test('T3 — fermeture puis réouverture : GetAllMatches appelé deux fois', async () => {
        render(MatchPanel);
        await tick();

        // Première ouverture
        openPanels.set(new Set([PANEL.MATCH]));
        await tick();
        await tick();

        expect(GetAllMatches).toHaveBeenCalledTimes(1);

        // Fermeture
        openPanels.set(new Set());
        await tick();

        // Réouverture
        openPanels.set(new Set([PANEL.MATCH]));
        await tick();
        await tick();

        expect(GetAllMatches).toHaveBeenCalledTimes(2);
    });

    // ── Test 4 : bascules rapides → nb appels = nb ouvertures ────────────────
    test('T4 — 5 ouvertures alternées : GetAllMatches appelé exactement 5 fois', async () => {
        render(MatchPanel);
        await tick();

        for (let i = 0; i < 5; i++) {
            openPanels.set(new Set([PANEL.MATCH]));
            await tick();
            await tick(); // stabilisation async
            openPanels.set(new Set());
            await tick();
        }

        expect(GetAllMatches).toHaveBeenCalledTimes(5);
    });
});

// ── Suite : matchPanelRefreshTriggerStore ────────────────────────────────────
describe('MatchPanel — matchPanelRefreshTriggerStore', () => {
    beforeEach(() => {
        resetStores();
        vi.clearAllMocks();
    });
    afterEach(cleanup);

    // ── Test 5 : trigger quand panel visible → loadMatches appelé ────────────
    test('T5 — trigger quand panel ouvert : GetAllMatches rappelé', async () => {
        render(MatchPanel);
        await tick();

        // Ouvrir le panel (1er appel)
        openPanels.set(new Set([PANEL.MATCH]));
        await tick();
        await tick();
        expect(GetAllMatches).toHaveBeenCalledTimes(1);

        // Déclencher le refresh (2e appel)
        matchPanelRefreshTriggerStore.update((n) => n + 1);
        await tick();
        await tick();
        expect(GetAllMatches).toHaveBeenCalledTimes(2);
    });

    // ── Test 6 : trigger quand panel fermé → loadMatches pas appelé ──────────
    test('T6 — trigger quand panel fermé : GetAllMatches non rappelé', async () => {
        render(MatchPanel);
        await tick();

        // Panel fermé — trigger ne doit pas charger
        matchPanelRefreshTriggerStore.update((n) => n + 1);
        await tick();
        await tick();

        expect(GetAllMatches).not.toHaveBeenCalled();
    });
});
