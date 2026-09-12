/**
 * searchEncounterToken.test.js — #362
 *
 * `s n>3` (les rencontres, #282) marchait en ligne de commande et ne faisait
 * rien dans l'application : l'analyseur JS ne produisait pas le champ et la
 * charge utile ne le portait pas, si bien que la recherche partait sans lui et
 * rendait tout. Ce test suit le jeton de la barre de commande jusqu'à l'appel
 * du backend, sans raccourci : processCommand → loadPositionsByFilters →
 * LoadPositionIDsByFilters.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';

const bindings = vi.hoisted(() => ({
    SaveComment: vi.fn(() => Promise.resolve()),
    ClearCommandHistory: vi.fn(() => Promise.resolve()),
    SaveSearchHistory: vi.fn(() => Promise.resolve()),
    ListPositionIDs: vi.fn(() => Promise.resolve([])),
    LoadPositionsByIDs: vi.fn(() => Promise.resolve([])),
    LoadPositionIDsByFilters: vi.fn(() => Promise.resolve([])),
    LoadAnalysis: vi.fn(() => Promise.resolve(null)),
    LoadComment: vi.fn(() => Promise.resolve('')),
    SaveLastVisitedPosition: vi.fn(() => Promise.resolve()),
    GetLastVisitedMatch: vi.fn(() => Promise.resolve(null))
}));

vi.mock('../../wailsjs/go/database/Database.js', () => bindings);
vi.mock('../services/databaseService.js', () => ({
    setStatusBarMessage: vi.fn(),
    warningMessageStore: { subscribe: vi.fn(), set: vi.fn(), update: vi.fn() }
}));
vi.mock('../services/sessionService.js', () => ({ saveSessionState: vi.fn() }));
vi.mock('../services/confirmService.js', () => ({ confirmAction: vi.fn(() => Promise.resolve(true)) }));

import { processCommand, initCommandProcessor } from '../commandProcessor.js';
import { loadPositionsByFilters } from '../services/positionService.js';
import { statusBarModeStore, currentPositionIndexStore } from '../stores/uiStore.js';
import { databasePathStore } from '../stores/databaseStore.js';

async function searchFromCommandBar(command) {
    let pending;
    initCommandProcessor({
        onLoadPositionsByFilters: (opts) => {
            pending = loadPositionsByFilters(opts);
            return pending;
        }
    });
    processCommand(command);
    expect(pending, `${command} never reached loadPositionsByFilters`).toBeDefined();
    await pending;
    expect(bindings.LoadPositionIDsByFilters).toHaveBeenCalledTimes(1);
    return bindings.LoadPositionIDsByFilters.mock.calls[0][0];
}

beforeEach(() => {
    vi.clearAllMocks();
    databasePathStore.set('/tmp/lib.db');
    statusBarModeStore.set('NORMAL');
    currentPositionIndexStore.set(-1);
});

describe('le jeton n, de la barre de commande au backend (#362)', () => {
    test('s n>3 envoie encounterFilter', async () => {
        const payload = await searchFromCommandBar('s n>3');
        expect(payload.encounterFilter).toBe('n>3');
    });

    test('la forme bornée voyage telle quelle, le compte exact est déplié', async () => {
        expect((await searchFromCommandBar('s n2,5')).encounterFilter).toBe('n2,5');
        vi.clearAllMocks();
        expect((await searchFromCommandBar('s n4')).encounterFilter).toBe('n4,4');
    });

    test('nc ne se confond pas avec n', async () => {
        const payload = await searchFromCommandBar('s nc');
        expect(payload.noContactFilter).toBe(true);
        expect(payload.encounterFilter).toBe('');
    });
});
