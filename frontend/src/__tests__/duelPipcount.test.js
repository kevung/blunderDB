/**
 * Le Duel joué pipcount masqué : ni le bouton de la barre d'outils ni la touche `p` (tous deux
 * `togglePipcount`) ne le révèlent pendant le Duel ; la préférence revient à sa sortie.
 */
import { describe, test, expect, beforeEach, vi } from 'vitest';
import { get } from 'svelte/store';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    CreateDuel: vi.fn(),
    OpenDuel: vi.fn(),
    SuspendDuel: vi.fn(() => Promise.resolve()),
    PlayDuel: vi.fn(),
    FlagDuel: vi.fn(),
    StopDuel: vi.fn(),
    ForfeitDuel: vi.fn(),
    ListDuels: vi.fn(() => Promise.resolve([])),
    DuelOffer: vi.fn()
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({ LegalMoves: vi.fn(() => Promise.resolve([])), StartGammonNetMatchBatch: vi.fn() }));
vi.mock('../../wailsjs/go/main/Config.js', () => ({ GetGammonNetAnalysisPly: vi.fn(), GetGammonNetPruneK: vi.fn(), GetDuelForm: vi.fn(), SaveDuelForm: vi.fn(() => Promise.resolve()) }));
vi.mock('../services/modeMachine.js', () => ({ enterDuelMode: vi.fn(() => Promise.resolve()), exitDuelMode: vi.fn(() => Promise.resolve()) }));

import { CreateDuel, OpenDuel } from '../../wailsjs/go/database/Database.js';
import { pipcountVisibleStore, showPipcountStore, duelPipcountHiddenStore } from '../stores/uiStore.js';
import { togglePipcount } from '../services/tabToggles.js';
import { startDuel, resumeDuel, suspendDuel } from '../services/duelService.js';
import { DEFAULT_FORM } from '../services/duel.js';

const state = {
    id: 1,
    revision: 1,
    header: { match_length: 5 },
    sides: [{ kind: 'external' }, { kind: 'bot' }],
    awaiting: { side: 0, kind: 'cube', position: { board: { points: [] }, dice: [0, 0] } }
};

beforeEach(() => {
    vi.clearAllMocks();
    showPipcountStore.set(true);
    duelPipcountHiddenStore.set(false);
});

describe('the pipcount during a Duel', () => {
    test('hidden by the form, neither the button nor the shortcut shows it; the preference comes back', async () => {
        vi.mocked(CreateDuel).mockResolvedValue(/** @type {any} */ ({ state, sheet: { actions: [] } }));
        await startDuel({ ...DEFAULT_FORM, pipcount: false }, []);
        expect(get(pipcountVisibleStore)).toBe(false);
        togglePipcount();
        expect(get(pipcountVisibleStore)).toBe(false);
        togglePipcount();
        expect(get(pipcountVisibleStore)).toBe(false);
        await suspendDuel();
        expect(get(pipcountVisibleStore)).toBe(true);
        expect(get(showPipcountStore)).toBe(true);
    });

    test('shown by the form, the preference rules as ever', async () => {
        vi.mocked(OpenDuel).mockResolvedValue(/** @type {any} */ ({ state, sheet: { actions: [] } }));
        await resumeDuel(1, { ...DEFAULT_FORM, pipcount: true });
        expect(get(pipcountVisibleStore)).toBe(true);
        togglePipcount();
        expect(get(pipcountVisibleStore)).toBe(false);
        await suspendDuel();
    });
});
