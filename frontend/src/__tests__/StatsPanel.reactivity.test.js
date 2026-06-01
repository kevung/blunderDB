/**
 * StatsPanel.reactivity.test.js
 *
 * Vérifie que StatsPanel réagit correctement aux changements de statsFilterStore :
 * - refreshStats (via ComputeStats) appelé une fois au montage avec le filtre initial.
 * - refreshStats re-appelé quand statsFilterStore change.
 * - Pas d'appel supplémentaire après démontage (pas de fuite mémoire).
 *
 * Pattern : vi.mock hoisted → render(StatsPanel) → mutate statsFilterStore → await tick() → assert calls.
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { render, cleanup } from '@testing-library/svelte';
import { tick } from 'svelte';

// ── Wails mocks ──────────────────────────────────────────────────────────────
vi.mock('../../wailsjs/go/database/Database.js', () => ({
    ComputeStats: vi.fn(() =>
        Promise.resolve({
            Totals: { NumPositions: 10, NumMatches: 1, NumTournaments: 0, NumDecisions: 10 },
            PRGlobal: 4.0,
            PRChecker: 4.0,
            PRCube: 0,
            MWCGlobal: 0,
            MWCChecker: 0,
            MWCCube: 0,
            PRRolling: {},
            MWCRolling: {},
            ErrorsByType: []
        })
    ),
    GetAllPlayerNames: vi.fn(() => Promise.resolve([])),
    GetAllTournaments: vi.fn(() => Promise.resolve([]))
}));

vi.mock('../../wailsjs/go/main/Config.js', () => ({
    GetStatsFilter: vi.fn(() => Promise.resolve(null)),
    SaveStatsFilter: vi.fn(() => Promise.resolve())
}));

// ── Stores ───────────────────────────────────────────────────────────────────
import { statsFilterStore, statsResultStore, statsLoadingStore, statsErrorStore, statsMetricStore } from '../stores/statsStore.js';
import { databasePathStore } from '../stores/databaseStore.js';

// ── DB mock ref (pour compter les appels) ────────────────────────────────────
import { ComputeStats } from '../../wailsjs/go/database/Database.js';

// ── Composant ────────────────────────────────────────────────────────────────
import StatsPanel from '../components/stats/StatsPanel.svelte';

// ── Helpers ───────────────────────────────────────────────────────────────────

const DEFAULT_FILTER = {
    playerName: '',
    tournamentIDs: [],
    dateFrom: '',
    dateTo: '',
    decisionType: -1,
    matchLength: []
};

function resetStores() {
    // StatsPanel only refreshes stats when a database is open
    // (databaseLoadedStore derives from a non-empty databasePathStore).
    databasePathStore.set('/tmp/test.db');
    statsFilterStore.set({ ...DEFAULT_FILTER });
    statsResultStore.set(null);
    statsLoadingStore.set(false);
    statsErrorStore.set(null);
    statsMetricStore.set('pr');
}

// ── Suite ─────────────────────────────────────────────────────────────────────

describe('StatsPanel — réactivité statsFilterStore', () => {
    beforeEach(() => {
        resetStores();
        vi.clearAllMocks();
    });
    afterEach(cleanup);

    // ── Test 1 : montage → refreshStats appelé une fois ──────────────────────
    test('T1 — montage : ComputeStats appelé au moins une fois avec le filtre initial', async () => {
        render(StatsPanel);
        await tick();
        await tick(); // laisser les effets async se stabiliser (StatsFilterBar.onMount inclus)

        // StatsFilterBar.onMount appelle toujours statsFilterStore.set() une fois résolue,
        // ce qui déclenche l'$effect une deuxième fois : >= 1 appel attendu.
        expect(ComputeStats.mock.calls.length).toBeGreaterThanOrEqual(1);
        // Le premier appel reçoit le filtre initial (sans nom de joueur)
        expect(ComputeStats).toHaveBeenCalledWith(
            expect.objectContaining({
                playerName: '',
                decisionType: -1
            })
        );
    });

    // ── Test 2 : changement de filtre → refreshStats re-appelé ───────────────
    test('T2 — changement de filtre : ComputeStats re-appelé avec le nouveau filtre', async () => {
        render(StatsPanel);
        await tick();
        await tick();

        const callsAfterMount = ComputeStats.mock.calls.length;

        const newFilter = {
            ...DEFAULT_FILTER,
            playerName: 'Alice',
            decisionType: 0
        };
        statsFilterStore.set(newFilter);
        await tick();
        await tick();

        expect(ComputeStats).toHaveBeenCalledTimes(callsAfterMount + 1);
        expect(ComputeStats).toHaveBeenLastCalledWith(
            expect.objectContaining({
                playerName: 'Alice',
                decisionType: 0
            })
        );
    });

    // ── Test 3 : démontage → pas de fuite mémoire ────────────────────────────
    test("T3 — démontage : pas d'appel à ComputeStats après démontage", async () => {
        const { unmount } = render(StatsPanel);
        await tick();
        await tick();

        unmount();
        await tick();

        const callCountAfterUnmount = ComputeStats.mock.calls.length;

        // Changer le filtre après démontage ne doit pas déclencher refreshStats
        statsFilterStore.set({ ...DEFAULT_FILTER, playerName: 'Bob' });
        await tick();
        await tick();

        expect(ComputeStats).toHaveBeenCalledTimes(callCountAfterUnmount);
    });
});
