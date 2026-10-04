/** Un second geste pendant le chargement de la Direction ne la referme pas. */
import { test, expect, vi, afterEach } from 'vitest';
import { get } from 'svelte/store';

let release;
vi.mock('../../wailsjs/go/database/Database.js', async (importOriginal) => {
    const orig = await importOriginal();
    return Object.fromEntries(Object.keys(/** @type {object} */ (orig)).map((k) => [k, vi.fn().mockResolvedValue(null)]));
});

import * as DB from '../../wailsjs/go/database/Database.js';
import { openDirection, closeDirection, directionIsLoading, openDirectionIdStore, directionStore, directionViewLoadedStore } from '../stores/directionStore.js';

afterEach(() => {
    closeDirection();
    directionViewLoadedStore.set(false);
});

test('tant que la vue arrive, le clic de fermeture est un double clic', async () => {
    const gate = new Promise((r) => (release = r));
    vi.mocked(DB.GetDirection).mockReturnValue(/** @type {any} */ (gate.then(() => '{}')));
    const p = openDirection(3);
    expect(directionIsLoading()).toBe(true);
    release();
    await p.catch(() => {});
    directionStore.set(/** @type {any} */ ({ tournamentId: 3, state: 'draft' }));
    expect(directionIsLoading()).toBe(true); // rejouée mais le morceau de la vue n'est pas monté
    directionViewLoadedStore.set(true);
    expect(directionIsLoading()).toBe(false);
    expect(get(openDirectionIdStore)).toBe(3);
});
