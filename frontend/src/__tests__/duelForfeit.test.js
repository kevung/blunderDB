/**
 * Abandonner, mettre en pause, annuler : l'abandon du match part après confirmation, au nom du
 * joueur ; l'annulation jette le Duel sans rien écrire.
 */
import { describe, test, expect, beforeEach, vi } from 'vitest';

vi.mock('../../wailsjs/go/database/Database.js', () => ({
    CreateDuel: vi.fn(),
    OpenDuel: vi.fn(),
    SuspendDuel: vi.fn(),
    PlayDuel: vi.fn(),
    FlagDuel: vi.fn(),
    StopDuel: vi.fn(() => Promise.resolve(null)),
    ForfeitDuel: vi.fn(() => Promise.resolve(null)),
    ListDuels: vi.fn(() => Promise.resolve([])),
    DuelOffer: vi.fn()
}));
vi.mock('../../wailsjs/go/gui/App.js', () => ({ LegalMoves: vi.fn(), StartGammonNetMatchBatch: vi.fn() }));
vi.mock('../../wailsjs/go/main/Config.js', () => ({ GetGammonNetAnalysisPly: vi.fn(), GetGammonNetPruneK: vi.fn(), GetDuelForm: vi.fn(), SaveDuelForm: vi.fn() }));
vi.mock('../services/confirmService.js', async (importOriginal) => ({ ...(await importOriginal()), confirmAction: vi.fn() }));

import { StopDuel, ForfeitDuel } from '../../wailsjs/go/database/Database.js';
import { confirmAction } from '../services/confirmService.js';
import { duelStore } from '../stores/duelStore.js';
import { confirmForfeitDuel, confirmCancelDuel } from '../services/duelService.js';

/**
 * @param {number} length
 * @param {any[]} sides
 */
function open(length, sides) {
    duelStore.set({
        state: { id: 4, revision: 7, header: { match_length: length, player1: 'Alice', player2: 'Bot' }, sides, awaiting: { side: 0, kind: 'cube', position: { cube: { value: 2 } } } },
        sheet: { actions: [] }
    });
}

beforeEach(() => {
    vi.clearAllMocks();
});

describe('ending a Duel', () => {
    test('forfeit names the winner, then forfeits for the human Side', async () => {
        open(5, [{ kind: 'bot' }, { kind: 'external' }]);
        vi.mocked(confirmAction).mockResolvedValue(true);
        await confirmForfeitDuel();
        expect(vi.mocked(confirmAction).mock.calls[0][0]).toContain('Alice');
        expect(ForfeitDuel).toHaveBeenCalledWith(4, 7, 1);
    });

    test('in money, the confirmation tells the backgammon at the cube', async () => {
        open(0, [{ kind: 'external' }, { kind: 'bot' }]);
        vi.mocked(confirmAction).mockResolvedValue(true);
        await confirmForfeitDuel();
        expect(vi.mocked(confirmAction).mock.calls[0][0]).toContain('6');
        expect(ForfeitDuel).toHaveBeenCalledWith(4, 7, 0);
    });

    test('nothing leaves when the forfeit is not confirmed', async () => {
        open(5, [{ kind: 'external' }, { kind: 'bot' }]);
        vi.mocked(confirmAction).mockResolvedValue(false);
        await confirmForfeitDuel();
        expect(ForfeitDuel).not.toHaveBeenCalled();
    });

    test('cancel throws the Duel away once confirmed', async () => {
        open(5, [{ kind: 'external' }, { kind: 'bot' }]);
        vi.mocked(confirmAction).mockResolvedValue(true);
        await confirmCancelDuel();
        expect(StopDuel).toHaveBeenCalledWith(4, 7, false);
    });
});
