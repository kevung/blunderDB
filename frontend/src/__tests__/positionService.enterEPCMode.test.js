/**
 * positionService.enterEPCMode.test.js
 *
 * Vérifie que :
 *   - enterEPCMode positionne statusBarModeStore à 'EPC' AVANT positionStore.
 *   - exitEPCMode réinitialise statusBarModeStore à 'NORMAL' AVANT de restaurer
 *     positionStore (évite que l'effet EPC se re-déclenche sur la position
 *     restaurée alors que le mode est encore 'EPC').
 *   - enterEPCMode est idempotent (pas de double set si déjà en mode EPC).
 *
 * Stratégie : on importe les vrais stores Svelte (partagés avec le module
 * positionService) et on espionne leur méthode `.set` via vi.spyOn pour
 * capturer l'ordre des appels. Les espions appellent l'implémentation originale
 * pour que get() reflète la bonne valeur. On mock uniquement les dépendances
 * Wails et databaseService (E/S asynchrones).
 */

import { describe, test, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';

// ── Mocks Wails (doivent précéder les imports des modules qui les utilisent) ──
vi.mock('../../wailsjs/go/main/Database.js', () => ({
    LoadAllPositions: vi.fn(() => Promise.resolve([])),
    DeletePosition: vi.fn(),
    DeleteAnalysis: vi.fn(),
    UpdatePosition: vi.fn(),
    SaveAnalysis: vi.fn(),
    LoadAnalysis: vi.fn(),
    LoadPositionsByFilters: vi.fn(() => Promise.resolve([])),
    ComputeEPCFromPosition: vi.fn(() => Promise.resolve({})),
    SaveLastVisitedPosition: vi.fn(),
    GetLastVisitedMatch: vi.fn(() => Promise.resolve(null)),
    GetMatchMovePositions: vi.fn(() => Promise.resolve([])),
    SaveEditPosition: vi.fn(),
    SaveFilter: vi.fn(),
    LoadComment: vi.fn(() => Promise.resolve('')),
}));

// Mock databaseService (tire App.js, Config.js, runtime.js via ses propres imports)
vi.mock('../services/databaseService.js', () => ({
    setStatusBarMessage: vi.fn(),
    warningMessageStore: { subscribe: vi.fn(), set: vi.fn(), update: vi.fn() },
}));

// ── Stores réels (partagés avec positionService via le cache de modules) ──────
import { statusBarModeStore, statusBarTextStore, currentPositionIndexStore } from '../stores/uiStore.js';
import { positionStore, positionsStore } from '../stores/positionStore.js';
import { epcDataStore } from '../stores/epcStore.js';

// ── Module testé ──────────────────────────────────────────────────────────────
import { enterEPCMode, exitEPCMode } from '../services/positionService.js';

// ── Helpers ───────────────────────────────────────────────────────────────────

/** Position factice représentant une vraie partie, à sauvegarder avant EPC. */
function makeRealPosition(id = 99) {
    return {
        id,
        board: {
            points: Array(26).fill(null).map(() => ({ checkers: 0, color: -1 })),
            bearoff: [3, 3],
        },
        cube: { owner: -1, value: 1 },
        dice: [3, 1],
        score: [0, 0],
        player_on_roll: 0,
        decision_type: 0,
        has_jacoby: 0,
        has_beaver: 0,
    };
}

/** Réinitialise les stores à un état cohérent avant chaque test. */
function resetStores() {
    statusBarModeStore.set('NORMAL');
    statusBarTextStore.set('');
    currentPositionIndexStore.set(0);
    positionStore.set(null);
    positionsStore.set([]);
    epcDataStore.set({ bottomEPC: null, topEPC: null, error: null });
}

/**
 * Installe des espions call-through sur les .set de plusieurs stores.
 * Chaque appel à .set enregistre une entrée { store, value } dans callOrder.
 * Les espions appellent l'implémentation originale pour que get() soit à jour.
 * Retourne les spy objects pour permettre mockClear() dans les tests.
 */
function installSetSpies(storeMap, callOrder) {
    const spies = {};
    for (const [name, store] of Object.entries(storeMap)) {
        const origSet = store.set; // référence à l'implémentation originale
        spies[name] = vi.spyOn(store, 'set').mockImplementation((v) => {
            callOrder.push({ store: name, value: v });
            origSet(v); // propagation réelle : get() reste cohérent
        });
    }
    return spies;
}

// ── Tests enterEPCMode ────────────────────────────────────────────────────────

describe('enterEPCMode — ordre des set', () => {
    let callOrder;
    let spies;

    beforeEach(() => {
        resetStores();
        callOrder = [];
        spies = installSetSpies(
            { statusBarModeStore, positionStore, positionsStore },
            callOrder
        );
    });

    afterEach(() => {
        vi.restoreAllMocks();
    });

    test('T1 — statusBarModeStore(EPC) est appelé avant positionStore', () => {
        enterEPCMode();

        const modeIdx = callOrder.findIndex(
            (c) => c.store === 'statusBarModeStore' && c.value === 'EPC'
        );
        const posIdx = callOrder.findIndex((c) => c.store === 'positionStore');

        expect(modeIdx, 'statusBarModeStore.set("EPC") doit avoir eu lieu').toBeGreaterThanOrEqual(0);
        expect(posIdx, 'positionStore.set doit avoir eu lieu').toBeGreaterThanOrEqual(0);
        expect(modeIdx, 'mode doit être défini avant position').toBeLessThan(posIdx);
    });

    test('T2 — statusBarModeStore(EPC) est appelé avant positionsStore', () => {
        enterEPCMode();

        const modeIdx = callOrder.findIndex(
            (c) => c.store === 'statusBarModeStore' && c.value === 'EPC'
        );
        const posListIdx = callOrder.findIndex((c) => c.store === 'positionsStore');

        expect(modeIdx).toBeGreaterThanOrEqual(0);
        expect(posListIdx).toBeGreaterThanOrEqual(0);
        expect(modeIdx).toBeLessThan(posListIdx);
    });

    test('T3 — idempotent : second appel ignoré si déjà en mode EPC', () => {
        // Premier appel réel (via les espions call-through)
        enterEPCMode();
        // Le store reflète maintenant 'EPC' grâce au call-through
        expect(get(statusBarModeStore)).toBe('EPC');

        // Réinitialiser les compteurs
        callOrder.length = 0;
        for (const spy of Object.values(spies)) spy.mockClear();

        // Second appel : doit retourner immédiatement (mode déjà EPC)
        enterEPCMode();

        expect(spies.statusBarModeStore).not.toHaveBeenCalled();
        expect(spies.positionStore).not.toHaveBeenCalled();
        expect(spies.positionsStore).not.toHaveBeenCalled();
    });
});

// ── Tests exitEPCMode ─────────────────────────────────────────────────────────

describe('exitEPCMode — ordre des set', () => {
    let callOrder;
    let spies;

    beforeEach(() => {
        // Préparer un état "en mode EPC avec position sauvegardée"
        resetStores();
        const realPos = makeRealPosition();
        positionStore.set(realPos);
        positionsStore.set([realPos]);
        currentPositionIndexStore.set(0);

        // Entrer en mode EPC via la vraie fonction (sans espions actifs)
        enterEPCMode();
        expect(get(statusBarModeStore)).toBe('EPC');

        // Installer les espions call-through APRÈS enterEPCMode pour ne mesurer
        // que les appels issus de exitEPCMode.
        callOrder = [];
        spies = installSetSpies(
            { statusBarModeStore, positionStore },
            callOrder
        );
    });

    afterEach(() => {
        vi.restoreAllMocks();
        resetStores();
    });

    test('T4 — statusBarModeStore(NORMAL) est appelé avant positionStore lors du exit', () => {
        exitEPCMode();

        const modeIdx = callOrder.findIndex(
            (c) => c.store === 'statusBarModeStore' && c.value === 'NORMAL'
        );
        const posIdx = callOrder.findIndex((c) => c.store === 'positionStore');

        expect(modeIdx, 'statusBarModeStore.set("NORMAL") doit avoir eu lieu').toBeGreaterThanOrEqual(0);
        expect(posIdx, 'positionStore.set doit avoir eu lieu').toBeGreaterThanOrEqual(0);
        expect(modeIdx, 'mode doit être remis à NORMAL avant la restauration de la position').toBeLessThan(posIdx);
    });

    test('T5 — exitEPCMode ignoré si mode !== EPC', () => {
        // Forcer un mode différent (via le spy call-through : store mis à jour)
        spies.statusBarModeStore.mockClear();
        spies.positionStore.mockClear();
        statusBarModeStore.set('NORMAL'); // appel via le spy → store à jour
        callOrder.length = 0;
        spies.statusBarModeStore.mockClear();
        spies.positionStore.mockClear();

        exitEPCMode(); // doit retourner immédiatement (mode !== EPC)

        // positionStore ne doit pas avoir été touché
        expect(spies.positionStore).not.toHaveBeenCalled();
    });
});
