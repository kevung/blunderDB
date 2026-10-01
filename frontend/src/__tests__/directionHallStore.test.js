/**
 * La Salle côté store : une réponse dépassée n'écrase pas la suivante, une erreur remonte au
 * lieu d'un chargement sans fin, et un geste de la Salle réécrit la page de l'épreuve visée.
 */

import { describe, test, expect, vi, beforeEach } from 'vitest';

vi.mock('../../wailsjs/go/database/Database', async (importOriginal) => ({
    ...(await importOriginal()),
    RencontreTableGrid: vi.fn(),
    EnterResult: vi.fn(async () => ({})),
    WriteDirectionPage: vi.fn(async () => ''),
    GetDirection: vi.fn(async () => null)
}));

import { RencontreTableGrid, EnterResult, WriteDirectionPage } from '../../wailsjs/go/database/Database';
import { directionStore, hallGrid, hallEnterResult } from '../stores/directionStore';

/** @returns {{ promise: Promise<any>, resolve: (v: any) => void, reject: (e: any) => void }} */
function deferred() {
    let resolve = (/** @type {any} */ _v) => {};
    let reject = (/** @type {any} */ _e) => {};
    const promise = new Promise((res, rej) => {
        resolve = res;
        reject = rej;
    });
    return { promise, resolve, reject };
}

beforeEach(() => {
    vi.clearAllMocks();
    directionStore.set(/** @type {any} */ ({ tournamentId: 1, rencontreId: 5 }));
});

describe('la Salle, côté store', () => {
    test('une réponse plus ancienne arrivée après la suivante est ignorée', async () => {
        const first = deferred();
        const second = deferred();
        vi.mocked(RencontreTableGrid).mockReturnValueOnce(first.promise).mockReturnValueOnce(second.promise);
        const a = hallGrid();
        const b = hallGrid();
        second.resolve({ name: 'nouvelle' });
        first.resolve({ name: 'ancienne' });
        expect(await b).toEqual({ name: 'nouvelle' });
        expect(await a).toBeUndefined();
    });

    test('une erreur remonte à qui demande la Salle', async () => {
        vi.mocked(RencontreTableGrid).mockRejectedValueOnce(new Error('direction: boom'));
        await expect(hallGrid()).rejects.toThrow('boom');
    });

    test("un geste de la Salle réécrit la page de l'épreuve de la case, pas seulement l'ouverte", async () => {
        await hallEnterResult(2, 'm1', 'a');
        expect(EnterResult).toHaveBeenCalledWith(2, 'm1', 'a', 0, 0, '');
        expect(WriteDirectionPage).toHaveBeenCalledWith(2);
    });
});
